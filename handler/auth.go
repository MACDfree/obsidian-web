package handler

import (
	"net/http"
	"obsidian-web/config"
	"obsidian-web/logger"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func AuthPage(ctx *gin.Context) {
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
		"CurrentPath": "/auth",
	})
}

func Auth(ctx *gin.Context) {
	isLogin := ctx.MustGet("isLogin").(bool)
	if isLogin {
		ctx.Redirect(302, "/")
		return
	}

	session := sessions.Default(ctx)
	session.Options(sessions.Options{
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   false,
	})

	if session.Get("error") != nil && session.Get("error").(int) > 3 {
		logger.Warn("max wrong password tims")
		ctx.Redirect(302, "/auth")
		return
	}

	password := ctx.PostForm("password")
	if password != config.Get().Password {
		if session.Get("error") == nil {
			logger.Warn("wrong password: 1")
			session.Set("error", 1)
		} else {
			logger.Warn("wrong password: " + strconv.Itoa(session.Get("error").(int)))
			session.Set("error", session.Get("error").(int)+1)
		}
		err := session.Save()
		if err != nil {
			logger.Error(errors.WithStack(err))
		}
		ctx.Redirect(302, "/auth")
		return
	}

	session.Set("error", 0)

	session.Set("isLogin", true)
	err := session.Save()
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.Redirect(302, "/auth")
		return
	}
	ctx.Redirect(302, "/")
}
