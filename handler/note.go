package handler

import (
	"html/template"
	"obsidian-web/config"
	"obsidian-web/db"
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

	// 解决登录后页面缓存导致不刷新的问题
	isLoginStr := "0"
	if isLogin {
		isLoginStr = "1"
	}
	ifNoneMatch := ctx.GetHeader("If-None-Match")
	if ifNoneMatch == "\""+note.MD5+isLoginStr+"\"" {
		ctx.Status(304)
		return
	}

	source, err := os.ReadFile(note.Path)
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	withMathJax := note.ExtInfo != nil && note.ExtInfo["mathjax"] != nil && note.ExtInfo["mathjax"].(bool)

	htmlStr, toc, err := mdparser.ConvertToHTML(source, withMathJax)

	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	ctx.Header("ETag", "\""+note.MD5+isLoginStr+"\"")

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
