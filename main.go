package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"html/template"
	"io"
	"net/http"
	"obsidian-web/config"
	"obsidian-web/db"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"obsidian-web/log"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/wikilink"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

/*
目标是将obsidian中的笔记发布为网页访问，并且，其中非公开部分可以通过密码访问

实现逻辑上：
1. 扫描obsidian中的笔记，构建笔记元信息，存储在sqlite中
2. 实现一个web服务，提供笔记的访问和鉴权逻辑
*/

// preScript用于执行启动前的一系列脚本
var preScript = flag.String("pre", "", "pre script")

func main() {
	if *preScript != "" {
		log.Infof("start pre script: %s", *preScript)
		cmd := exec.Command("sh", "-c", *preScript)
		cmd.Dir = ""
		sout := bytes.NewBuffer(nil)
		cmd.Stdout = sout
		serr := bytes.NewBuffer(nil)
		cmd.Stderr = serr
		err := cmd.Run()
		if err != nil {
			log.Error("pre script error: %s. pre script stdout: %s.", serr.String(), sout.String())
			log.Fatalf("%+v", errors.WithStack(err))
		}
		log.Error("pre script error: %s. pre script stdout: %s.", serr.String(), sout.String())
	}

	loadNoteBook()

	r := gin.Default()
	r.HTMLRender = loadTemplates("templates")

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("webauth", store))

	r.Use(Auth())

	r.StaticFS("/assets", http.Dir("assets"))

	r.GET("/auth", func(ctx *gin.Context) {
		isLogin := ctx.MustGet("isLogin").(bool)
		if isLogin {
			ctx.Redirect(302, "/")
			return
		}

		ctx.HTML(http.StatusOK, "auth.html", gin.H{
			"Auth": gin.H{
				"IsLogin": isLogin,
			},
			"Site": gin.H{
				"Title": config.Get().Title,
			},
		})
	})
	r.POST("/auth", func(ctx *gin.Context) {
		isLogin := ctx.MustGet("isLogin").(bool)
		if isLogin {
			ctx.Redirect(302, "/")
			return
		}

		password := ctx.PostForm("password")
		if password != config.Get().Password {
			ctx.Redirect(302, "/auth")
			return
		}

		session := sessions.Default(ctx)
		session.Options(sessions.Options{
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   false,
		})
		session.Set("isLogin", true)
		err := session.Save()
		if err != nil {
			log.Error(errors.WithStack(err))
			ctx.String(500, "error: %v", err)
			return
		}
		ctx.Redirect(302, "/")
	})

	r.GET("/:static", func(ctx *gin.Context) {
		static := ctx.Param("static")
		if static == "tag" ||
			static == "note" {
			ctx.Redirect(302, "/"+static+"/")
			return
		}
		ctx.File(filepath.Join("static", static))
	})

	r.GET("/", func(ctx *gin.Context) {
		pageIndex := 0
		isLogin := ctx.MustGet("isLogin").(bool)
		notes, err := db.ListNote(isLogin, pageIndex)
		if err != nil {
			log.Error(errors.WithStack(err))
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
	})
	r.GET("/page/:num", func(ctx *gin.Context) {
		pageIndexStr := ctx.Param("num")
		pageIndex, err := strconv.Atoi(pageIndexStr)
		if err != nil {
			log.Error(errors.WithStack(err))
			ctx.String(500, "error: %v", err)
			return
		}
		isLogin := ctx.MustGet("isLogin").(bool)
		notes, err := db.ListNote(isLogin, pageIndex)
		if err != nil {
			log.Error(errors.WithStack(err))
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
	})
	r.GET("/tag/*tag", func(ctx *gin.Context) {
		tag := ctx.Param("tag")
		isLogin := ctx.MustGet("isLogin").(bool)
		if tag == "/" {
			// 展示所有的tag
			tagCounts, err := db.ListTag(isLogin)
			if err != nil {
				log.Error(errors.WithStack(err))
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
				log.Error(errors.WithStack(err))
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
	})
	r.GET("/archive", func(ctx *gin.Context) {
		isLogin := ctx.MustGet("isLogin").(bool)
		ctx.HTML(200, "archive.html", gin.H{
			"Auth": gin.H{
				"IsLogin": isLogin,
			},
			"Site": gin.H{
				"Title": config.Get().Title,
			},
		})
	})
	r.GET("/note/*path", func(ctx *gin.Context) {
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
			log.Error(errors.WithStack(err))
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

		markExt := make([]goldmark.Extender, 0)
		markExt = append(markExt,
			extension.GFM,
			&frontmatter.Extender{},
			&wikilink.Extender{
				Resolver: myResolver{},
			},
		)

		if note.ExtInfo != nil && note.ExtInfo["mathjax"] != nil && note.ExtInfo["mathjax"].(bool) {
			markExt = append(markExt, mathjax.MathJax)
		}

		md := goldmark.New(
			goldmark.WithExtensions(markExt...),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
			),
		)
		var buf bytes.Buffer
		source, err := os.ReadFile(note.Path)
		if err != nil {
			log.Error(errors.WithStack(err))
			ctx.String(500, "error: %v", err)
			return
		}
		if err := md.Convert(source, &buf); err != nil {
			log.Error(errors.WithStack(err))
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
			"content": template.HTML(buf.String()),
		})
	})
	r.GET("/attachment/:attach", func(ctx *gin.Context) {
		fileName := ctx.Param("attach")
		ctx.File(filepath.Join(config.Get().NotePath, config.Get().AttachmentPath, fileName))
	})
	r.GET("/search", func(ctx *gin.Context) {
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
			log.Error(errors.WithStack(err))
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
				log.Error(errors.WithStack(err))
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

	})

	r.GET("/gitpull", func(ctx *gin.Context) {
		isLogin := ctx.MustGet("isLogin").(bool)
		if !isLogin {
			ctx.Redirect(302, "/")
			return
		}

		ctx.HTML(200, "gitpull.html", gin.H{
			"Auth": gin.H{
				"IsLogin": isLogin,
			},
			"Site": gin.H{
				"Title": config.Get().Title,
			},
		})
	})

	r.POST("/gitpull", func(ctx *gin.Context) {
		isLogin := ctx.MustGet("isLogin").(bool)
		if !isLogin {
			ctx.JSON(403, gin.H{
				"msg": "未登录",
			})
			return
		}

		cmd := exec.Command("git", "pull")
		cmd.Dir = config.Get().NotePath
		sout := bytes.NewBuffer(nil)
		cmd.Stdout = sout
		serr := bytes.NewBuffer(nil)
		cmd.Stderr = serr
		err := cmd.Run()
		if err != nil {
			log.Error(errors.WithStack(err))
			ctx.JSON(500, gin.H{
				"msg":  "git pull 报错",
				"sout": sout.String(),
				"serr": serr.String(),
			})
			return
		}

		// 需要重新解析文件
		loadNoteBook()

		ctx.JSON(200, gin.H{
			"msg":  "git pull 成功",
			"sout": sout.String(),
			"serr": serr.String(),
		})
	})

	r.Run(config.Get().BindAddr)
}

func loadTemplates(templatesPath string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	partials, err := filepath.Glob(templatesPath + "/partials/*.html")
	if err != nil {
		log.Fatal(errors.WithStack(err))
	}

	views, err := filepath.Glob(templatesPath + "/views/*.html")
	if err != nil {
		log.Fatal(errors.WithStack(err))
	}

	funcMap := template.FuncMap{
		"DateStr": func(d time.Time) string {
			return d.Format("2006-01-02")
		},
		"SafeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	for _, view := range views {
		files := []string{}
		files = append(files, templatesPath+"/base.html", view)
		files = append(files, partials...)
		r.AddFromFilesFuncs(filepath.Base(view), funcMap, files...)
	}

	return r
}

func Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		if session.Get("isLogin") == nil {
			ctx.Set("isLogin", false)
		} else {
			isLogin := session.Get("isLogin").(bool)
			ctx.Set("isLogin", isLogin)
		}
		ctx.Next()
	}
}

var exts = []string{
	".png",
	".pdf",
	".jpg",
	".zip",
}

type myResolver struct {
}

func (myResolver) ResolveWikilink(n *wikilink.Node) (destination []byte, err error) {
	_hash := []byte{'#'}
	dest := make([]byte, lo.Max([]int{len([]byte("/note/")), len([]byte("/attachment/"))})+len(n.Target)+len("#")+len(n.Fragment))
	var i int
	if len(n.Target) > 0 {
		ext := filepath.Ext(string(n.Target))
		if lo.Contains(exts, ext) {
			i += copy(dest[i:], []byte("/attachment/"))
			i += copy(dest[i:], n.Target)
		} else {
			i += copy(dest[i:], []byte("/note/"))
			i += copy(dest[i:], n.Target)
		}
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}
	return dest[:i], nil
}

func loadNoteBook() {
	start := time.Now()
	err := db.DeleteAll()
	if err != nil {
		log.Error(errors.WithStack(err))
		return
	}
	rootPath := config.Get().NotePath
	ignorePaths := config.Get().IgnorePaths
	attachPath := config.Get().AttachmentPath
	err = filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootPath {
			return nil
		}
		if d.IsDir() {
			if lo.Contains(ignorePaths, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			// 执行解析逻辑
			note, err := parseFrontMatter(path)
			if err != nil {
				log.Error(errors.WithStack(err))
				return nil
			}
			if note.Tags == nil {
				note.Tags = []string{}
			}
			if note.Aliases == nil {
				note.Aliases = []string{}
			}
			if note.Title == "" {
				title := filepath.Base(path)
				title = strings.TrimSuffix(title, ".md")
				note.Title = title
			}
			fullTitle := path[len(rootPath)+1 : len(path)-3]
			fullTitle = strings.ReplaceAll(fullTitle, "\\", "/")
			note.FullTitle = fullTitle

			if note.Created.IsZero() {
				if strings.HasPrefix(fullTitle, "daily note/") {
					str := fullTitle[len("daily note/"):]
					t, err := time.Parse("2006-01-02", str)
					if err != nil {
						log.Error(errors.WithStack(err))
					}
					note.Created = t
					note.Updated = t
				}
			}

			md, err := fileMD5(path)
			if err != nil {
				log.Error(errors.WithStack(err))
				return nil
			}
			note.MD5 = md

			err = db.InsertNote(note)
			if err != nil {
				log.Error(errors.WithStack(err))
			}
		} else if strings.HasPrefix(path[len(rootPath)+1:], attachPath) {
			// 处理附件元信息
			attachInfo := &db.AttachInfo{
				Path:       path,
				AttachName: filepath.Base(path),
			}
			err := db.InsertAttachInfo(attachInfo)
			if err != nil {
				log.Error(errors.WithStack(err))
			}
		}
		return nil
	})
	if err != nil {
		log.Error(errors.WithStack(err))
	}
	log.Infof("init notebook cost: %v", time.Since(start))
}

func parseFrontMatter(mdPath string) (*db.Note, error) {
	// 1. 读取md文件的前几行，不需要全读
	mdFile, err := os.Open(mdPath)
	if err != nil {
		return nil, err
	}
	defer mdFile.Close()

	scanner := bufio.NewScanner(mdFile)
	sb := &strings.Builder{}
	if scanner.Scan() {
		line := scanner.Text()
		if line != "---" {
			// 没有front matter的话，存储基本的信息
			return &db.Note{
				Path: mdPath,
			}, nil
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		sb.WriteString(line)
		sb.WriteString("\r\n")
	}

	frontMatter := &FrontMatter{}
	err = yaml.Unmarshal([]byte(sb.String()), frontMatter)
	if err != nil {
		return nil, err
	}
	note := &db.Note{
		Title:   frontMatter.Title,
		Tags:    frontMatter.Tags,
		Aliases: frontMatter.Aliases,
		Created: frontMatter.Created.Time,
		Updated: frontMatter.Updated.Time,
		Publish: frontMatter.Publish,
		Path:    mdPath,
		ExtInfo: frontMatter.ExtInfo,
	}
	return note, nil
}

type FrontMatter struct {
	Title   string         `yaml:"title"`
	Tags    []string       `yaml:"tags"`
	Aliases []string       `yaml:"aliases"`
	Created CTime          `yaml:"created"`
	Updated CTime          `yaml:"updated"`
	Publish bool           `yaml:"publish"`
	ExtInfo map[string]any `yaml:",inline"`
}

type CTime struct {
	time.Time
}

func (cTime CTime) MarshalYAML() (interface{}, error) {
	return yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.TaggedStyle,
		Value: cTime.Format("2006-01-02T15:04"),
	}, nil
}

func (cTime *CTime) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		t, err := time.Parse("2006-01-02T15:04", value.Value)
		if err != nil {
			log.Warnf("再试一次，%+v", err)
			t, err = time.Parse("2006-01-02T15:04:05", value.Value)
			if err != nil {
				return err
			}
		}
		cTime.Time = t
	}
	return nil
}

type SearchResult struct {
	*db.Note
	HitTitle string
	HitText  []string
}

func fileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
