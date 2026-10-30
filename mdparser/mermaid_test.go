package mdparser

import (
	"strings"
	"testing"
)

func TestMermaidCodeBlock(t *testing.T) {
	for _, math := range []bool{false, true} {
		got, _, err := ConvertToHTML([]byte("```mermaid\ngraph TD\nA-->B\n```\n"), math)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, `<code class="language-mermaid">graph TD`) || !strings.Contains(got, "A--&gt;B") {
			t.Errorf("math=%v: %s", math, got)
		}
	}
}
