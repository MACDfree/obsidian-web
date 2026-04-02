package handler

import (
	"net/http"
	"obsidian-web/config"
	"obsidian-web/db"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateCommentRequest struct {
	NoteFullTitle string `json:"note_full_title" binding:"required,max=500"`
	ParentID      *uint  `json:"parent_id"`
	Nickname      string `json:"nickname" binding:"required,max=50"`
	Content       string `json:"content" binding:"required,max=2000"`
}

// CommentResponse 评论响应结构
type CommentResponse struct {
	ID            uint              `json:"id"`
	Nickname      string            `json:"nickname"`
	Content       string            `json:"content"`
	CreatedAt     time.Time         `json:"created_at"`
	NoteFullTitle string            `json:"note_full_title,omitempty"`
	Status        int               `json:"status,omitempty"`
	IP            string            `json:"ip,omitempty"`
	Replies       []*CommentResponse `json:"replies"`
}

// CreateComment 创建评论
func CreateComment(ctx *gin.Context) {
	var req CreateCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Content = strings.TrimSpace(req.Content)
	req.NoteFullTitle = strings.TrimSpace(req.NoteFullTitle)

	if req.Nickname == "" || req.Content == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "昵称和内容不能为空"})
		return
	}

	// 验证笔记存在且已发布
	note, err := db.GetNoteByPath(req.NoteFullTitle)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "笔记不存在"})
		return
	}
	if !note.Publish {
		ctx.JSON(http.StatusForbidden, gin.H{"code": 1, "msg": "该笔记不支持评论"})
		return
	}

	// 限流：同 IP 每10分钟最多3条
	ip := ctx.ClientIP()
	recentCount, _ := db.CountRecentCommentsByIP(ip, 10)
	if recentCount >= 3 {
		ctx.JSON(http.StatusTooManyRequests, gin.H{"code": 1, "msg": "评论过于频繁，请稍后再试"})
		return
	}

	// 敏感词检查
	if db.IsContentBlocked(req.Content) || db.IsContentBlocked(req.Nickname) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "内容包含不当词汇，请修改后重试"})
		return
	}

	// 验证父评论
	if req.ParentID != nil {
		parent, err := db.GetCommentByID(*req.ParentID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "回复的评论不存在"})
			return
		}
		if parent.NoteFullTitle != req.NoteFullTitle {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "回复的评论不存在"})
			return
		}
	}

	comment := &db.Comment{
		NoteFullTitle: req.NoteFullTitle,
		ParentID:      req.ParentID,
		Nickname:      req.Nickname,
		Content:       req.Content,
		Status:        0,
		IP:            ip,
		UserAgent:     ctx.GetHeader("User-Agent"),
		CreatedAt:     time.Now(),
	}

	if err := db.InsertComment(comment); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "评论失败，请稍后再试"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "评论成功",
		"data": gin.H{
			"id":         comment.ID,
			"nickname":   comment.Nickname,
			"content":    comment.Content,
			"created_at": comment.CreatedAt,
		},
	})
}

// ListComments 获取指定笔记的评论列表
func ListComments(ctx *gin.Context) {
	fullTitle := ctx.Query("note_full_title")
	if fullTitle == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "缺少参数"})
		return
	}

	comments, err := db.ListCommentsByNoteFullTitle(fullTitle)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "获取评论失败"})
		return
	}

	count := int64(len(comments))
	tree := buildCommentTree(comments)

	ctx.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  tree,
		"count": count,
	})
}

// buildCommentTree 将扁平评论列表构建为树形结构
func buildCommentTree(comments []db.Comment) []*CommentResponse {
	idMap := make(map[uint]*CommentResponse)
	var roots []*CommentResponse

	// 先创建所有响应对象
	for i := range comments {
		resp := &CommentResponse{
			ID:        comments[i].ID,
			Nickname:  comments[i].Nickname,
			Content:   comments[i].Content,
			CreatedAt: comments[i].CreatedAt,
			Replies:   []*CommentResponse{},
		}
		idMap[comments[i].ID] = resp
	}

	// 建立父子关系
	for i := range comments {
		resp := idMap[comments[i].ID]
		if comments[i].ParentID == nil {
			roots = append(roots, resp)
		} else {
			if parent, ok := idMap[*comments[i].ParentID]; ok {
				parent.Replies = append(parent.Replies, resp)
			} else {
				// 父评论不存在或已隐藏，作为顶级评论显示
				roots = append(roots, resp)
			}
		}
	}

	return roots
}

// DeleteComment 删除评论（管理员）
func DeleteComment(ctx *gin.Context) {
	isLogin := ctx.MustGet("isLogin").(bool)
	if !isLogin {
		ctx.JSON(http.StatusForbidden, gin.H{"code": 1, "msg": "无权限"})
		return
	}

	commentID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || commentID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "参数错误"})
		return
	}

	if err := db.DeleteComment(uint(commentID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "删除失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// AdminComments 管理员评论管理页面
func AdminComments(ctx *gin.Context) {
	isLogin := ctx.MustGet("isLogin").(bool)
	if !isLogin {
		ctx.Status(403)
		return
	}

	pageSize := 20
	pageIndex := 0
	if p := ctx.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			pageIndex = parsed
		}
	}

	comments, err := db.ListAllComments(pageSize, pageIndex)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "error: %v", err)
		return
	}

	total, _ := db.CountAllComments()

	// 计算分页
	var prePageIndex, nexPageIndex int
	if pageIndex > 0 {
		prePageIndex = pageIndex - 1
	}
	if int64((pageIndex+1)*pageSize) < total {
		nexPageIndex = pageIndex + 1
	}

	ctx.HTML(http.StatusOK, "admin_comments.html", gin.H{
		"Auth":         gin.H{"IsLogin": isLogin},
		"Site":         gin.H{"Title": config.Get().Title},
		"CurrentPath":  "/admin/comments",
		"Comments":     comments,
		"Total":        total,
		"PrePageIndex": prePageIndex,
		"NexPageIndex": nexPageIndex,
		"PageIndex":    pageIndex,
	})
}
