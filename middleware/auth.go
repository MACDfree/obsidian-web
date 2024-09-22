package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

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
