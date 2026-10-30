package mdparser

import (
	"strings"
	"testing"
)

func TestCallouts(t *testing.T) {
	for _, math := range []bool{false, true} {
		input := "> [!WARNING]- 注意 <script>\n> 内容 **加粗**\n\n> 普通引用\n\n```text\n> [!note] 代码\n```\n"
		got, _, err := ConvertToHTML([]byte(input), math)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`<details class="callout" data-callout="warning">`, `注意 &lt;script&gt;`, `<strong>加粗</strong>`, `</details>`, `<blockquote>`, `[!note]`} {
			if !strings.Contains(got, want) {
				t.Errorf("math=%v: 缺少 %q: %s", math, want, got)
			}
		}
	}
}
