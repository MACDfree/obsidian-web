package main

import (
	"bytes"
	"flag"
	"obsidian-web/config"
	"obsidian-web/job"
	"obsidian-web/noteloader"
	"obsidian-web/server"
	"os/exec"

	"obsidian-web/log"

	"github.com/pkg/errors"
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
	flag.Parse()
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
			log.Errorf("pre script error: %s. pre script stdout: %s.", serr.String(), sout.String())
			log.Fatalf("%+v", errors.WithStack(err))
		}
		log.Infof("pre script error: %s. pre script stdout: %s.", serr.String(), sout.String())
	}

	noteloader.Load()

	r := server.NewRouter()

	// r.GET("/error/404", func(ctx *gin.Context) {
	// 	isLogin := ctx.MustGet("isLogin").(bool)
	// 	ctx.HTML(http.StatusNotFound, "404.html", gin.H{
	// 		"Auth": gin.H{
	// 			"IsLogin": isLogin,
	// 		},
	// 		"Site": gin.H{
	// 			"Title": config.Get().Title,
	// 		},
	// 	})
	// })

	// r.GET("/error/500", func(ctx *gin.Context) {
	// 	isLogin := ctx.MustGet("isLogin").(bool)
	// 	ctx.HTML(http.StatusInternalServerError, "500.html", gin.H{
	// 		"Auth": gin.H{
	// 			"IsLogin": isLogin,
	// 		},
	// 		"Site": gin.H{
	// 			"Title": config.Get().Title,
	// 		},
	// 	})
	// })

	// r.NoRoute(func(ctx *gin.Context) {
	// 	isLogin := ctx.MustGet("isLogin").(bool)
	// 	ctx.HTML(http.StatusOK, "404.html", gin.H{
	// 		"Auth": gin.H{
	// 			"IsLogin": isLogin,
	// 		},
	// 		"Site": gin.H{
	// 			"Title": config.Get().Title,
	// 		},
	// 	})
	// })

	// r.NoRoute(func(ctx *gin.Context) {
	// 	ctx.Redirect(302, "/error/404")
	// })

	job.Start()

	r.Run(config.Get().BindAddr)
}
