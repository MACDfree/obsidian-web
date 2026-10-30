package mdparser

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/wikilink"
	"strings"
)

// ExtractNoteTargets 提取正文中的笔记 wikilink 和 Markdown 链接，跳过代码及附件。
func ExtractNoteTargets(src []byte) []string {
	doc := markdown.Parser().Parse(text.NewReader(src))
	seen := map[string]bool{}
	var result []string
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if link, ok := n.(*wikilink.Node); entering && ok {
			target := strings.TrimSpace(string(link.Target))
			if target != "" && !isKnownAttachment(target) && !seen[target] {
				seen[target] = true
				result = append(result, target)
			}
		}
		if link, ok := n.(*ast.Link); entering && ok {
			if u, ok := markdownTarget(link.Destination); ok && !seen[u.Path] {
				seen[u.Path] = true
				result = append(result, u.Path)
			}
		}
		return ast.WalkContinue, nil
	})
	return result
}
