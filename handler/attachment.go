package handler

import (
	"obsidian-web/config"
	"obsidian-web/db"
	"os"
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

// resolveRegularFile 在 baseDir 内解析真实路径并确保是普通文件，
// 拒绝目录以及指向 baseDir 之外的符号链接
func resolveRegularFile(baseDir, userPath string) (string, bool) {
	safePath, ok := safeJoin(baseDir, userPath)
	if !ok {
		return "", false
	}

	realBase, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		return "", false
	}

	realPath, err := filepath.EvalSymlinks(safePath)
	if err != nil {
		return "", false
	}

	if realPath != realBase && !strings.HasPrefix(realPath, realBase+string(filepath.Separator)) {
		return "", false
	}

	info, err := os.Stat(realPath)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}

	return realPath, true
}

func Attachment(ctx *gin.Context) {
	fileName := strings.TrimPrefix(ctx.Param("attach"), "/")
	if fileName == "" || filepath.IsAbs(fileName) || filepath.Clean(fileName) != fileName {
		ctx.Status(404)
		return
	}
	info, err := db.GetAttachInfoByName(fileName)
	if err != nil {
		ctx.Status(404)
		return
	}
	isPublic, exists := db.GetAttachPublishStatus(info.AttachName)
	if !exists || (!isPublic && !ctx.MustGet("isLogin").(bool)) {
		ctx.Status(404)
		return
	}
	baseDir := filepath.Join(config.Get().NotePath, config.Get().AttachmentPath)
	safePath, ok := resolveRegularFile(baseDir, info.AttachName)
	if !ok {
		ctx.Status(404)
		return
	}
	ctx.File(safePath)
}
