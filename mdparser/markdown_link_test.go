package mdparser

import (
	"strings"
	"testing"
)

func TestMarkdownNoteLinks(t *testing.T) {
	src := []byte("[一](工作/%E8%AE%A1%E5%88%92.md#章节) [二](./相对.md) [三](笔记) [私有](秘密.md) [外部](https://example.com/file.md) [网站](/search) [附件](文件.pdf) ![图片](a.png) `[[代码]]`\n")
	calls := map[string]string{"工作/计划.md": "工作/计划", "./相对.md": "工作/相对", "笔记": "笔记"}
	resolve := func(target string) (string, bool) { v, ok := calls[target]; return v, ok }
	for _, math := range []bool{false, true} {
		got, _, err := ConvertWithResolver(src, math, "", resolve)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`href="/note/%E5%B7%A5%E4%BD%9C/%E8%AE%A1%E5%88%92#%E7%AB%A0%E8%8A%82"`, `href="/note/%E5%B7%A5%E4%BD%9C/%E7%9B%B8%E5%AF%B9"`, `href="/note/%E7%AC%94%E8%AE%B0"`, `class="unresolved-link"`, `href="https://example.com/file.md"`, `href="/search"`, `href="%E6%96%87%E4%BB%B6.pdf"`} {
			if !strings.Contains(got, want) {
				t.Errorf("math=%v 缺少 %s: %s", math, want, got)
			}
		}
	}
	got := ExtractNoteTargets(src)
	if strings.Join(got, ",") != "工作/计划.md,./相对.md,笔记,秘密.md" {
		t.Fatalf("提取结果: %v", got)
	}
}
