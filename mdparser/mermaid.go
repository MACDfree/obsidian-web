package mdparser

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"strings"
)

var mermaidKind = ast.NewNodeKind("Mermaid")

type mermaidNode struct {
	ast.BaseBlock
	lines *text.Segments
}

func (*mermaidNode) Kind() ast.NodeKind           { return mermaidKind }
func (n *mermaidNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type mermaidRenderer struct{}

func (*mermaidRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(mermaidKind, renderMermaid)
}
func renderMermaid(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	_, _ = w.WriteString(`<pre><code class="language-mermaid">`)
	lines := node.(*mermaidNode).lines
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		_, _ = w.Write(util.EscapeHTML(segment.Value(src)))
	}
	_, _ = w.WriteString("</code></pre>\n")
	return ast.WalkSkipChildren, nil
}
func prepareMermaid(doc ast.Node, src []byte) {
	var pending []struct {
		old ast.Node
		new *mermaidNode
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if code, ok := n.(*ast.FencedCodeBlock); entering && ok && strings.EqualFold(string(code.Language(src)), "mermaid") {
			node := &mermaidNode{lines: code.Lines()}
			pending = append(pending, struct {
				old ast.Node
				new *mermaidNode
			}{n, node})
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	for _, p := range pending {
		p.old.Parent().ReplaceChild(p.old.Parent(), p.old, p.new)
	}
}
