package mdparser

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var calloutHeader = regexp.MustCompile(`(?i)^\[!([a-z][a-z0-9_-]*)\]([+-]?)(?:[ \t]+(.*))?[ \t]*$`)

// prepareCallouts 将引用块首行的 Obsidian 标记转换为渲染属性。
func prepareCallouts(doc ast.Node, source []byte) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		quote, ok := n.(*ast.Blockquote)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		p, ok := quote.FirstChild().(*ast.Paragraph)
		if !ok || p.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}
		line := p.Lines().At(0)
		match := calloutHeader.FindStringSubmatch(strings.TrimSpace(string(line.Value(source))))
		if match == nil {
			return ast.WalkContinue, nil
		}
		kind := strings.ToLower(match[1])
		title := strings.TrimSpace(match[3])
		if title == "" {
			title = strings.ToUpper(kind[:1]) + kind[1:]
		}
		quote.SetAttributeString("callout-type", []byte(kind))
		quote.SetAttributeString("callout-title", []byte(title))
		quote.SetAttributeString("callout-fold", []byte(match[2]))
		for child := p.FirstChild(); child != nil; {
			next := child.NextSibling()
			last := false
			if t, ok := child.(*ast.Text); ok {
				last = t.SoftLineBreak() || t.HardLineBreak()
			}
			p.RemoveChild(p, child)
			if last {
				break
			}
			child = next
		}
		if !p.HasChildren() {
			quote.RemoveChild(quote, p)
		}
		return ast.WalkContinue, nil
	})
}

type calloutRenderer struct{}

func (*calloutRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindBlockquote, renderCallout)
}
func renderCallout(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	kind, ok := node.AttributeString("callout-type")
	if !ok {
		if entering {
			_, _ = w.WriteString("<blockquote>\n")
		} else {
			_, _ = w.WriteString("</blockquote>\n")
		}
		return ast.WalkContinue, nil
	}
	fold, _ := node.AttributeString("callout-fold")
	folded := string(fold.([]byte))
	if entering {
		tag := "div"
		if folded != "" {
			tag = "details"
		}
		_, _ = w.WriteString("<" + tag + ` class="callout" data-callout="`)
		_, _ = w.Write(util.EscapeHTML(kind.([]byte)))
		_, _ = w.WriteString(`"`)
		if folded == "+" {
			_, _ = w.WriteString(" open")
		}
		_, _ = w.WriteString(">\n")
		title, _ := node.AttributeString("callout-title")
		titleTag := "div"
		if folded != "" {
			titleTag = "summary"
		}
		_, _ = w.WriteString("<" + titleTag + ` class="callout-title">`)
		_, _ = w.Write(util.EscapeHTML(title.([]byte)))
		_, _ = w.WriteString("</" + titleTag + ">\n")
	} else {
		if folded != "" {
			_, _ = w.WriteString("</details>\n")
		} else {
			_, _ = w.WriteString("</div>\n")
		}
	}
	return ast.WalkContinue, nil
}
