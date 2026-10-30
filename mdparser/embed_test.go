package mdparser

import (
	"strings"
	"testing"
)

func TestEmbeddedNotesAndFragments(t *testing.T) {
	src := []byte("## 首节\n\n正文 ^abc\n\n![[子笔记#标题]]\n\n### 子标题\n内容\n\n## 末节\n末尾\n")
	got, _, err := ConvertToHTML(src, false)
	if err != nil || !strings.Contains(got, `data-target="子笔记" data-fragment="标题"`) {
		t.Fatalf("嵌入标记: %s %v", got, err)
	}
	got, _, err = ConvertFragmentToHTML(src, false, "首节")
	if err != nil || !strings.Contains(got, "子标题") || strings.Contains(got, "末节") {
		t.Fatalf("标题片段: %s %v", got, err)
	}
	got, _, err = ConvertFragmentToHTML(src, false, "^abc")
	if err != nil || !strings.Contains(got, "正文") || strings.Contains(got, "子标题") {
		t.Fatalf("块片段: %s %v", got, err)
	}
	if _, _, err := ConvertFragmentToHTML(src, false, "不存在"); err == nil {
		t.Fatal("不存在的片段应返回错误")
	}
}

func TestEmbeddedBlockTypes(t *testing.T) {
	src := []byte("- 一项 ^item\n- 二项\n\n> 引文\n\n^quote\n\n```go\nx := 1\n```\n\n^code\n\n| A | B |\n|---|---|\n| C | D |\n\n^table\n")
	for _, tc := range []struct{ id, want string }{{"item", "<li id=\"block-item\">一项"}, {"quote", "<blockquote>"}, {"code", `class="code-block"`}, {"table", "<table>"}} {
		got, _, err := ConvertFragmentToHTML(src, false, "^"+tc.id)
		if err != nil || !strings.Contains(got, tc.want) {
			t.Errorf("%s: %s %v", tc.id, got, err)
		}
		if tc.id == "item" && (!strings.Contains(got, "<ul>") || strings.Contains(got, "二项")) {
			t.Errorf("列表块: %s", got)
		}
	}
}

func TestResolveWikiLinks(t *testing.T) {
	input := []byte("[[别名#标题]] ![[别名]] ![[目录/图.png]]\n")
	got, _, err := ConvertWithResolver(input, false, "", func(target string) (string, bool) {
		if target == "别名" {
			return "目录/真实笔记", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/note/`, `#%E6%A0%87%E9%A2%98"`, `data-target="别名"`, `/attachment/`} {
		if !strings.Contains(got, want) {
			t.Errorf("缺少 %q: %s", want, got)
		}
	}
	if !strings.Contains(got, "%E7%9B%AE%E5%BD%95/%E7%9C%9F%E5%AE%9E%E7%AC%94%E8%AE%B0") {
		t.Errorf("未解析别名：%s", got)
	}
}

func TestUnresolvedWikiLinksAndCodeExamples(t *testing.T) {
	src := []byte("[[存在]] [[缺失]] ![[附注]] [[photo.png]]\n\n`[[行内示例]]`\n\n```md\n[[代码示例]]\n```\n")
	targets := ExtractNoteTargets(src)
	if strings.Join(targets, ",") != "存在,缺失,附注" {
		t.Fatalf("提取结果错误: %v", targets)
	}
	got, _, err := ConvertWithResolver(src, false, "", func(target string) (string, bool) {
		if target == "存在" {
			return "目录/存在", true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `class="unresolved-link"`) || !strings.Contains(got, `href="/attachment/photo.png"`) || !strings.Contains(got, `href="/note/`) {
		t.Fatalf("链接输出错误: %s", got)
	}
}
