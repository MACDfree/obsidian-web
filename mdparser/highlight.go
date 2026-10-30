package mdparser

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var highlightKind = ast.NewNodeKind("ObsidianHighlight")

type highlightNode struct{ ast.BaseInline }

func (*highlightNode) Kind() ast.NodeKind           { return highlightKind }
func (n *highlightNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type highlightDelimiters struct{}

func (*highlightDelimiters) IsDelimiter(b byte) bool                   { return b == '=' }
func (*highlightDelimiters) CanOpenCloser(a, b *parser.Delimiter) bool { return a.Char == b.Char }
func (*highlightDelimiters) OnMatch(int) ast.Node                      { return &highlightNode{} }

type highlightParser struct{}

func (*highlightParser) Trigger() []byte { return []byte{'='} }
func (*highlightParser) Parse(_ ast.Node, block text.Reader, ctx parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, &highlightDelimiters{})
	if node == nil || node.OriginalLength != 2 || before == '=' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + 2)
	block.Advance(2)
	ctx.PushDelimiter(node)
	return node
}
func (*highlightParser) CloseBlock(ast.Node, parser.Context) {}

type highlightRenderer struct{}

func (*highlightRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(highlightKind, func(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString("<mark>")
		} else {
			_, _ = w.WriteString("</mark>")
		}
		return ast.WalkContinue, nil
	})
}

type highlightExtension struct{}

func (*highlightExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(&highlightParser{}, 500)))
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&highlightRenderer{}, 500)))
}
