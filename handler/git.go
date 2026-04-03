package handler

import (
	"obsidian-web/config"
	"obsidian-web/gitutil"
	"obsidian-web/logger"
	"obsidian-web/noteloader"

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
		"CurrentPath": "/gitpull",
		"GitStatus":   gitutil.StatusText(),
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

	result, err := gitutil.Pull()
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.JSON(500, gin.H{
			"msg": "git pull 报错: " + err.Error(),
		})
		return
	}

	if result.Changed {
		noteloader.Load()
	}

	ctx.JSON(200, gin.H{
		"msg": result.Msg,
	})
}
