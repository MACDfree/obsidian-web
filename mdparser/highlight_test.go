package mdparser

import (
	"strings"
	"testing"
)

func TestObsidianHighlight(t *testing.T) {
	src := []byte("==重点 **加粗**== `==代码==` ===普通===\n")
	for _, math := range []bool{false, true} {
		got, _, err := ConvertToHTML(src, math)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "<mark>重点 <strong>加粗</strong></mark>") || !strings.Contains(got, "<code>==代码==</code>") {
			t.Errorf("%s", got)
		}
	}
}
