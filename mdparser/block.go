package mdparser

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var anchoredBlockKind = ast.NewNodeKind("AnchoredBlock")

type anchoredBlock struct{ ast.BaseBlock }

func (*anchoredBlock) Kind() ast.NodeKind           { return anchoredBlockKind }
func (n *anchoredBlock) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type blockRenderer struct{}

func (*blockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(anchoredBlockKind, renderAnchoredBlock)
}
func renderAnchoredBlock(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="obsidian-block" id="`)
		id, _ := n.AttributeString("id")
		_, _ = w.Write(util.EscapeHTML(id.([]byte)))
		_, _ = w.WriteString(`">`)
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

// Standalone IDs following a block identify the entire preceding block.
func prepareStandaloneBlockIDs(doc ast.Node, src []byte) {
	var markers []ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		p, ok := n.(*ast.Paragraph)
		if !entering || !ok || p.PreviousSibling() == nil || p.Lines().Len() != 1 {
			return ast.WalkContinue, nil
		}
		line := p.Lines().At(0)
		value := []byte(strings.TrimSpace(string(line.Value(src))))
		loc := blockIDPattern.FindSubmatchIndex(value)
		if loc != nil && loc[0] == 0 {
			markers = append(markers, p)
		}
		return ast.WalkContinue, nil
	})
	for _, marker := range markers {
		if marker.Parent() == nil || marker.PreviousSibling() == nil {
			continue
		}
		line := marker.Lines().At(0)
		value := []byte(strings.TrimSpace(string(line.Value(src))))
		loc := blockIDPattern.FindSubmatchIndex(value)
		parent, prev := marker.Parent(), marker.PreviousSibling()
		wrap := &anchoredBlock{}
		wrap.SetAttributeString("id", append([]byte("block-"), value[loc[2]:loc[3]]...))
		parent.ReplaceChild(parent, prev, wrap)
		wrap.AppendChild(wrap, prev)
		parent.RemoveChild(parent, marker)
	}
}
