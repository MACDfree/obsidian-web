package handler

import (
	"obsidian-web/config"
	"obsidian-web/db"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// safeJoin 安全地连接路径，防止路径遍历攻击
// 返回清理后的绝对路径和是否在允许的基础目录内
func safeJoin(baseDir, userPath string) (string, bool) {
	// 清理路径，移除 . 和 .. 等相对路径元素
	cleanPath := filepath.Clean(filepath.Join(baseDir, userPath))

	// 获取基础目录的绝对路径
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", false
	}

	// 获取目标路径的绝对路径
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", false
	}

	// 检查目标路径是否在基础目录内
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", false
	}

	return cleanPath, true
}

func Attachment(ctx *gin.Context) {
	fileName := ctx.Param("attach")

	// 检查路径遍历攻击
	if strings.Contains(fileName, "..") {
		ctx.Status(400)
		return
	}

	// 安全连接路径
	baseDir := filepath.Join(config.Get().NotePath, config.Get().AttachmentPath)
	safePath, ok := safeJoin(baseDir, fileName)
	if !ok {
		ctx.Status(400)
		return
	}

	// 获取附件的公开状态
	isPublic, exists := db.GetAttachPublishStatus(fileName)

	if !exists {
		ctx.Status(404)
		return
	}

	// 如果附件不公开，需要检查登录状态
	if !isPublic {
		isLogin := ctx.MustGet("isLogin").(bool)
		if !isLogin {
			ctx.Status(404)
			return
		}
	}

	ctx.File(safePath)
}