package server

import (
	"html/template"
	"io"
	"net/http"
	"obsidian-web/handler"
	"obsidian-web/logger"
	"obsidian-web/middleware"
	"os"
	"path/filepath"

	"github.com/Masterminds/sprig/v3"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewRouter() *gin.Engine {
	gin.DisableConsoleColor()
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "logs/access.log",
		MaxSize:    100,
		MaxBackups: 0,
		MaxAge:     180,
		Compress:   false,
	}
	gin.DefaultWriter = io.MultiWriter(lumberJackLogger, os.Stdout)

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.Session("webauth"))
	r.Use(middleware.Auth())
	r.Use(middleware.Error())

	r.HTMLRender = loadTemplates("templates")

	r.StaticFS("/assets", http.Dir("assets"))
	r.GET("/auth", handler.AuthPage)
	r.POST("/auth", handler.Auth)
	r.GET("/:static", handler.Static)
	r.GET("/", handler.ListNotes)
	r.GET("/page/:num", handler.ListNotes)
	r.GET("/tag/*tag", handler.ListTags)
	r.GET("/note/*path", handler.NotePage)
	r.GET("/attachment/:attach", handler.Attachment)
	r.GET("/search", handler.SearchNotes)
	r.GET("/gitpull", handler.GitPullPage)
	r.POST("/gitpull", handler.GitPull)

	return r
}

func loadTemplates(templatesPath string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	partials, err := filepath.Glob(templatesPath + "/partials/*.html")
	if err != nil {
		logger.Fatal(errors.WithStack(err))
	}

	views, err := filepath.Glob(templatesPath + "/views/*.html")
	if err != nil {
		logger.Fatal(errors.WithStack(err))
	}

	funcMap := sprig.FuncMap()
	funcMap["SafeHTML"] = func(s string) template.HTML {
		return template.HTML(s)
	}

	for _, view := range views {
		files := []string{}
		files = append(files, templatesPath+"/base.html", view)
		files = append(files, partials...)
		r.AddFromFilesFuncs(filepath.Base(view), funcMap, files...)
	}

	return r
}
