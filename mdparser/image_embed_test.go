package mdparser

import (
	"strings"
	"testing"
)

func TestImageEmbedSize(t *testing.T) {
	input := []byte("![[sub/test.png|300x200]] ![[test.jpg|说明]] ![[test.png|300]]")
	for _, math := range []bool{false, true} {
		got, _, err := ConvertToHTML(input, math)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`src="/attachment/sub/test.png" alt="" width="300" height="200" style="height:200px;max-width:100%;object-fit:fill"`, `alt="说明"`, `width="300"`} {
			if !strings.Contains(got, want) {
				t.Errorf("math=%v missing %s: %s", math, want, got)
			}
		}
	}
}
func TestMathAndCodeHighlight(t *testing.T) {
	got, _, err := ConvertToHTML([]byte("```go\nfmt.Println(1)\n```\n"), true)
	if err != nil || !strings.Contains(got, `class="code-block"`) || !strings.Contains(got, `class="code-copy-btn"`) {
		t.Fatalf("%s: %v", got, err)
	}
}
