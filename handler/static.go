package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func Static(ctx *gin.Context) {
	static := ctx.Param("static")
	if static == "tag" ||
		static == "note" {
		ctx.Redirect(302, "/"+static+"/")
		return
	}

	if strings.Contains(static, "..") {
		ctx.Status(http.StatusNotFound)
		return
	}

	_, err := os.Stat(filepath.Join("static", static))
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	ctx.File(filepath.Join("static", static))
}
