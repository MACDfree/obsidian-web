package mdparser

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"go.abhg.dev/goldmark/wikilink"
)

var blockIDPattern = regexp.MustCompile(`(?:^|[ \t]+)\^([a-zA-Z0-9-]+)[ \t]*$`)

func headingID(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "heading"
	}
	return b.String()
}

func prepareAnchors(doc ast.Node, source []byte) {
	used := map[string]bool{}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok {
			base := headingID(nodeText(h, source))
			id := base
			for count := 1; used[id]; count++ {
				id = base + "-" + strconv.Itoa(count)
			}
			used[id] = true
			h.SetAttributeString("id", []byte(id))
		}
		if n.Kind() == ast.KindParagraph || n.Kind() == ast.KindTextBlock {
			if last, ok := n.LastChild().(*ast.Text); ok {
				value := last.Segment.Value(source)
				loc := blockIDPattern.FindSubmatchIndex(value)
				if loc != nil {
					target := n
					if _, ok := n.Parent().(*ast.ListItem); ok {
						target = n.Parent()
					}
					target.SetAttributeString("id", append([]byte("block-"), value[loc[2]:loc[3]]...))
					last.Segment.Stop = last.Segment.Start + loc[0]
				}
			}
		}
		if w, ok := n.(*wikilink.Node); ok && len(w.Fragment) > 0 && !isKnownAttachment(string(w.Target)) {
			frag := string(w.Fragment)
			if strings.HasPrefix(frag, "^") {
				w.Fragment = []byte("block-" + strings.TrimPrefix(frag, "^"))
			} else {
				w.Fragment = []byte(headingID(frag))
			}
		}
		return ast.WalkContinue, nil
	})
}
