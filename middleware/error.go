package middleware

import (
	"obsidian-web/config"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Error() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if ctx.Writer.Status() == 500 || ctx.Writer.Status() == 404 {
			isLogin := ctx.MustGet("isLogin").(bool)
			ctx.HTML(ctx.Writer.Status(), strconv.Itoa(ctx.Writer.Status())+".html", gin.H{
				"Auth": gin.H{
					"IsLogin": isLogin,
				},
				"Site": gin.H{
					"Title": config.Get().Title,
				},
			})
			ctx.AbortWithStatus(ctx.Writer.Status())
		}
	}
}
