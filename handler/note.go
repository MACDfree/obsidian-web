package handler

import (
	"bufio"
	"html/template"
	"obsidian-web/config"
	"obsidian-web/db"
	"obsidian-web/logger"
	"obsidian-web/mdparser"
	"os"
	"path/filepath"
	"regexp"
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
			"list": tagCounts,
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
			"tag":  tag,
			"list": notes,
		})
	}
}

func NotePage(ctx *gin.Context) {
	path := ctx.Param("path")
	path = strings.Trim(path, "/")
	isLogin := ctx.MustGet("isLogin").(bool)
	if strings.Contains(path, "/assets/") {
		// 兼容思源导出的附件情况
		// 去对应路径下找附件并返回
		assetPath := filepath.Join(config.Get().NotePath, path)
		ctx.File(assetPath)
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

	htmlStr, err := mdparser.ConvertToHTML(source, withMathJax)

	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	ctx.Header("ETag", "\""+note.MD5+isLoginStr+"\"")
	ctx.HTML(200, "note.html", gin.H{
		"Auth": gin.H{
			"IsLogin": isLogin,
		},
		"Site": gin.H{
			"Title": config.Get().Title,
		},
		"IsNote":  true,
		"ExtInfo": note.ExtInfo,
		"info":    note,
		"content": template.HTML(htmlStr),
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
		})
		return
	}

	// 进行关键词的搜索，搜索范围为说有md文件的名称及内容，进行多线程处理
	// 先从数据库中查找当前可以访问的文件
	notes, err := db.ListAllNote(isLogin)
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	searchReg := regexp.MustCompile("(?i)" + keyword)
	results := make([]*SearchResult, 0)
	for _, note := range notes {
		result := SearchResult{Note: &note}
		if searchReg.MatchString(note.FullTitle) {
			result.HitTitle = searchReg.ReplaceAllString(note.FullTitle, "<mark>$0</mark>")
		}
		// 下面需要打开文档进行匹配了
		mdFile, err := os.Open(note.Path)
		if err != nil {
			logger.Error(errors.WithStack(err))
			ctx.String(500, "error: %v", err)
			return
		}
		defer mdFile.Close()

		scanner := bufio.NewScanner(mdFile)
		for scanner.Scan() {
			line := scanner.Text()
			if searchReg.MatchString(line) {
				result.HitText = append(result.HitText, searchReg.ReplaceAllString(line, "<mark>$0</mark>"))
			}
		}

		if len(result.HitTitle) > 0 || len(result.HitText) > 0 {
			results = append(results, &result)
		}
	}

	ctx.HTML(200, "search.html", gin.H{
		"Auth": gin.H{
			"IsLogin": isLogin,
		},
		"Site": gin.H{
			"Title": config.Get().Title,
		},
		"SearchResults": results,
	})
}

type SearchResult struct {
	*db.Note
	HitTitle string
	HitText  []string
}
