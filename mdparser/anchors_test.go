package mdparser

import (
	"strings"
	"testing"
)

func TestObsidianAnchors(t *testing.T) {
	for _, math := range []bool{false, true} {
		got, toc, err := ConvertToHTML([]byte("## 中文 标题\n\n段落内容 ^abc-12\n\n[[#中文 标题]] [[笔记#^abc-12]]\n"), math)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`id="中文-标题"`, `id="block-abc-12"`, `href="#%E4%B8%AD%E6%96%87-%E6%A0%87%E9%A2%98"`, `href="/note/`, `#block-abc-12`} {
			if !strings.Contains(got, want) {
				t.Errorf("math=%v 缺少 %q: %s", math, want, got)
			}
		}
		if len(toc) != 1 || toc[0].ID != "中文-标题" {
			t.Errorf("目录错误: %#v", toc)
		}
	}
}
