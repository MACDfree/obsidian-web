// Package excalidraw 提取 Obsidian Excalidraw 笔记中的画布数据。
package excalidraw

import (
	"encoding/base64"
	"regexp"
	"strings"
)

var drawingSection = regexp.MustCompile(`(?m)^# Excalidraw Data\s*\r?$`)
var pluginWarning = regexp.MustCompile(`(?m)^==⚠[ \t]+Switch to EXCALIDRAW VIEW in the MORE OPTIONS menu of this document\.[ \t]+⚠==[ \t]*\r?\n?`)
var drawingFence = regexp.MustCompile("(?ms)^```(compressed-json|json)[ \\t]*\\r?\\n(.*?)\\r?\\n```[ \\t]*\\r?$")

// Extract 只处理明确标记为 Excalidraw 的笔记；无效或不完整文件保留原文。
func Extract(src []byte, marked bool) (markdown []byte, placeholder string, ok bool) {
	if !marked {
		return src, "", false
	}
	loc := drawingSection.FindIndex(src)
	if loc == nil {
		return src, "", false
	}
	fence := drawingFence.FindSubmatch(src[loc[1]:])
	if fence == nil {
		return src, "", false
	}
	kind := string(fence[1])
	payload := strings.TrimSpace(string(fence[2]))
	if payload == "" {
		return src, "", false
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	html := `<div class="excalidraw-viewer" data-encoding="` + kind + `" data-drawing="` + encoded + `"><p>正在加载画布…</p></div>`
	return pluginWarning.ReplaceAll(src[:loc[0]], nil), html, true
}
