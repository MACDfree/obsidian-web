// Package linkutil 提供笔记索引与渲染共用的链接分类规则。
package linkutil

import (
	"path"
	"strings"
)

func IsAttachment(target string) bool {
	switch strings.ToLower(path.Ext(target)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".pdf", ".zip", ".doc", ".docx", ".xls", ".xlsx", ".mp3", ".mp4", ".wav", ".webm":
		return true
	default:
		return false
	}
}
