package handler

import (
	"obsidian-web/config"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func Attachment(ctx *gin.Context) {
	fileName := ctx.Param("attach")
	ctx.File(filepath.Join(config.Get().NotePath, config.Get().AttachmentPath, fileName))
}
