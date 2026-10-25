package middleware

import (
	"obsidian-web/config"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		isLogin := false

		// 未配置密码时视为不可登录，避免空密码和残留会话绕过访问控制
		if config.Get().Password != "" {
			session := sessions.Default(ctx)
			if v, ok := session.Get("isLogin").(bool); ok {
				isLogin = v
			}
		}

		ctx.Set("isLogin", isLogin)
		ctx.Next()
	}
}
