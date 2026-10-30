package handler

import (
	"html/template"
	"obsidian-web/config"
	"obsidian-web/db"
	"obsidian-web/excalidraw"
	"obsidian-web/logger"
	"obsidian-web/mdparser"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func ListNotes(ctx *gin.Context) {
	pageIndexStr := ctx.Param("num")
	if pageIndexStr == "" {
		pageIndexStr = "0"
	}
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}
	isLogin := ctx.MustGet("isLogin").(bool)
	notes, err := db.ListNote(isLogin, pageIndex)
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}
	prePageIndex := pageIndex - 1
	nextPageIndex := -1
	if len(notes) == config.Get().Paginate {
		nextPageIndex = pageIndex + 1
	}

	ctx.HTML(200, "index.html", gin.H{
		"Auth": gin.H{
			"IsLogin": isLogin,
		},
		"Site": gin.H{
			"Title": config.Get().Title,
		},
		"CurrentPath":  "/",
		"list":         notes,
		"nexPageIndex": nextPageIndex,
		"prePageIndex": prePageIndex,
	})
}

func ListTags(ctx *gin.Context) {
	tag := ctx.Param("tag")
	isLogin := ctx.MustGet("isLogin").(bool)
	if tag == "/" {
		// 展示所有的tag
		tagCounts, err := db.ListTag(isLogin)
		if err != nil {
			logger.Error(errors.WithStack(err))
			ctx.String(500, "error: %v", err)
			return
		}

		ctx.HTML(200, "tag.html", gin.H{
			"Auth": gin.H{
				"IsLogin": isLogin,
			},
			"Site": gin.H{
				"Title": config.Get().Title,
			},
			"CurrentPath": "/tag",
			"list":        tagCounts,
		})
	} else {
		tag = strings.TrimPrefix(tag, "/")
		notes, err := db.ListNoteByTag(isLogin, tag)
		if err != nil {
			logger.Error(errors.WithStack(err))
			ctx.String(500, "error: %v", err)
			return
		}

		ctx.HTML(200, "tag_item.html", gin.H{
			"Auth": gin.H{
				"IsLogin": isLogin,
			},
			"Site": gin.H{
				"Title": config.Get().Title,
			},
			"CurrentPath": "/tag",
			"tag":         tag,
			"list":        notes,
		})
	}
}

func NotePage(ctx *gin.Context) {
	path := ctx.Param("path")
	path = strings.Trim(path, "/")
	isLogin := ctx.MustGet("isLogin").(bool)
	if strings.Contains(path, "/assets/") {
		// 兼容思源导出的附件。这类文件不在附件索引中，无法判断发布状态，
		// 只对已登录用户开放，避免匿名读取笔记库中的任意文件
		if !isLogin {
			ctx.Status(404)
			return
		}

		// 仅服务 NotePath 内的普通文件，拒绝目录和符号链接逃逸
		safePath, ok := resolveRegularFile(config.Get().NotePath, path)
		if !ok {
			ctx.Status(404)
			return
		}

		ctx.File(safePath)
		return
	}
	note, err := db.GetNoteByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.Status(404)
			return
		}
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	if !isLogin && !note.Publish {
		ctx.Status(404)
		return
	}

	// 笔记链接解析依赖其他笔记的路径和权限，不能只用当前文件的哈希判断页面是否过期。
	if isLogin {
		ctx.Header("Cache-Control", "private, no-store")
	} else {
		ctx.Header("Cache-Control", "no-cache")
	}

	source, err := os.ReadFile(note.Path)
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	withMathJax, _ := note.ExtInfo["mathjax"].(bool)
	_, marked := note.ExtInfo["excalidraw-plugin"]
	source, drawing, _ := excalidraw.Extract(source, marked || strings.HasSuffix(strings.ToLower(note.FullTitle), ".excalidraw"))

	htmlStr, toc, err := mdparser.ConvertWithResolver(source, withMathJax, "", func(target string) (string, bool) {
		resolved, err := db.ResolveNoteLink(note.FullTitle, target, isLogin)
		if err != nil {
			return "", false
		}
		return resolved.FullTitle, true
	})

	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	htmlStr += drawing

	// 获取反向链接
	backlinks, err := db.GetBacklinks(note.ID, isLogin)
	if err != nil {
		logger.Error(errors.WithStack(err))
		backlinks = []db.BacklinkInfo{}
	}

	ctx.HTML(200, "note.html", gin.H{
		"Auth": gin.H{
			"IsLogin": isLogin,
		},
		"Site": gin.H{
			"Title": config.Get().Title,
		},
		"CurrentPath": "/note",
		"IsNote":      true,
		"ExtInfo":     note.ExtInfo,
		"info":        note,
		"content":     template.HTML(htmlStr),
		"Toc":         toc,
		"Backlinks":   backlinks,
	})
}

func SearchNotes(ctx *gin.Context) {
	isLogin := ctx.MustGet("isLogin").(bool)

	keyword := ctx.Query("keyword")
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		ctx.HTML(200, "search.html", gin.H{
			"Auth": gin.H{
				"IsLogin": isLogin,
			},
			"Site": gin.H{
				"Title": config.Get().Title,
			},
			"CurrentPath": "/search",
			"Keyword":     keyword,
		})
		return
	}

	// 全文搜索：命中 FTS5 索引（trigram），关键词按字面量匹配
	hits, err := db.SearchNotes(isLogin, keyword)
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}
	results := make([]*SearchResult, 0, len(hits))
	for _, hit := range hits {
		results = append(results, &SearchResult{
			FullTitle: hit.FullTitle,
			HitTitle:  hit.HitTitle,
			HitText:   hit.HitText,
		})
	}

	ctx.HTML(200, "search.html", gin.H{
		"Auth": gin.H{
			"IsLogin": isLogin,
		},
		"Site": gin.H{
			"Title": config.Get().Title,
		},
		"CurrentPath":   "/search",
		"Keyword":       keyword,
		"SearchResults": results,
	})
}

type SearchResult struct {
	FullTitle string
	HitTitle  string
	HitText   []string
}

// EmbeddedNote 只返回当前访问者可读取的笔记片段。
func EmbeddedNote(ctx *gin.Context) {
	source := ctx.Query("source")
	target := ctx.Query("target")
	isLogin := ctx.MustGet("isLogin").(bool)
	note, err := db.ResolveNoteLink(source, target, isLogin)
	if err != nil {
		ctx.Status(404)
		return
	}
	content, err := os.ReadFile(note.Path)
	if err != nil {
		ctx.Status(404)
		return
	}
	math, _ := note.ExtInfo["mathjax"].(bool)
	_, marked := note.ExtInfo["excalidraw-plugin"]
	content, drawing, _ := excalidraw.Extract(content, marked || strings.HasSuffix(strings.ToLower(note.FullTitle), ".excalidraw"))
	html, _, err := mdparser.ConvertWithResolver(content, math, ctx.Query("fragment"), func(target string) (string, bool) {
		resolved, err := db.ResolveNoteLink(note.FullTitle, target, isLogin)
		if err != nil {
			return "", false
		}
		return resolved.FullTitle, true
	})
	if err != nil {
		ctx.Status(404)
		return
	}
	if ctx.Query("fragment") == "" {
		html += drawing
	}
	ctx.Header("Cache-Control", "private, no-store")
	ctx.JSON(200, gin.H{"path": note.FullTitle, "html": html, "mathjax": math})
}
