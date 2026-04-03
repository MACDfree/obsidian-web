package main

import (
	"flag"
	"obsidian-web/config"
	"obsidian-web/db"
	"obsidian-web/gitutil"
	"obsidian-web/job"
	"obsidian-web/logger"
	"obsidian-web/noteloader"
	"obsidian-web/server"

	"github.com/pkg/errors"
)

/*
目标是将obsidian中的笔记发布为网页访问，并且，其中非公开部分可以通过密码访问

实现逻辑上：
1. 扫描obsidian中的笔记，构建笔记元信息，存储在sqlite中
2. 实现一个web服务，提供笔记的访问和鉴权逻辑
*/

// gitpull 控制启动时是否先拉取仓库
var gitpull = flag.Bool("gitpull", false, "启动时先执行 git pull 拉取最新笔记")

func main() {
	flag.Parse()

	if *gitpull && gitutil.IsConfigured() {
		logger.Info("启动时拉取仓库...")
		result, err := gitutil.Pull()
		if err != nil {
			logger.Fatalf("%+v", errors.WithStack(err))
		}
		logger.Infof("拉取完成: %s", result.Msg)
	}

	noteloader.Load()

	db.InitCommentDB()

	r := server.NewRouter()

	job.Start()

	r.Run(config.Get().BindAddr)
}
