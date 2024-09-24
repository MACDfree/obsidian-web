package handler

import (
	"bytes"
	"obsidian-web/config"
	"obsidian-web/logger"
	"obsidian-web/noteloader"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func GitPullPage(ctx *gin.Context) {
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
}

func GitPull(ctx *gin.Context) {
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
		logger.Error(errors.WithStack(err))
		ctx.JSON(500, gin.H{
			"msg":  "git pull 报错",
			"sout": sout.String(),
			"serr": serr.String(),
		})
		return
	}

	// 需要重新解析文件
	noteloader.Load()

	ctx.JSON(200, gin.H{
		"msg":  "git pull 成功",
		"sout": sout.String(),
		"serr": serr.String(),
	})
}
