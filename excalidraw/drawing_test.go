package excalidraw

import (
	"strings"
	"testing"
)

func TestExtract(t *testing.T) {
	for _, kind := range []string{"json", "compressed-json"} {
		src := []byte("---\nexcalidraw-plugin: parsed\n---\n文字\n# Excalidraw Data\n\n```" + kind + "\nhello\n```\n")
		prefix, html, ok := Extract(src, true)
		if !ok || strings.Contains(string(prefix), "Excalidraw Data") || !strings.Contains(html, `data-encoding="`+kind+`"`) || !strings.Contains(html, "aGVsbG8=") {
			t.Fatalf("%s %s %v", prefix, html, ok)
		}
		_, _, ok = Extract(src, false)
		if ok {
			t.Fatal("普通笔记不应处理")
		}
	}
	src := []byte("# Excalidraw Data\n```json\nmissing closing fence")
	result, _, ok := Extract(src, true)
	if ok || string(result) != string(src) {
		t.Fatal("不完整文件应保留原文")
	}
}

func TestRemovePluginWarningOnly(t *testing.T) {
	src := []byte("---\nexcalidraw-plugin: parsed\n---\n==⚠  Switch to EXCALIDRAW VIEW in the MORE OPTIONS menu of this document. ⚠==\n\n我的说明\n\n# Excalidraw Data\n```json\n{}\n```\n")
	prefix, _, ok := Extract(src, true)
	if !ok || strings.Contains(string(prefix), "Switch to EXCALIDRAW VIEW") || !strings.Contains(string(prefix), "我的说明") {
		t.Fatalf("提示过滤错误: %s", prefix)
	}
	prefix, _, ok = Extract(src, false)
	if ok || !strings.Contains(string(prefix), "Switch to EXCALIDRAW VIEW") {
		t.Fatal("普通笔记应保留原文")
	}
}
