package mdparser

import (
	"errors"
	"obsidian-web/linkutil"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

var embedKind = ast.NewNodeKind("NoteEmbed")

type embedNode struct {
	ast.BaseInline
	target, fragment string
}

func (*embedNode) Kind() ast.NodeKind           { return embedKind }
func (n *embedNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type embedRenderer struct{}

func (*embedRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(embedKind, renderEmbed)
}
func renderEmbed(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	e := n.(*embedNode)
	_, _ = w.WriteString(`<span class="note-embed" data-target="`)
	_, _ = w.Write(util.EscapeHTML([]byte(e.target)))
	_, _ = w.WriteString(`" data-fragment="`)
	_, _ = w.Write(util.EscapeHTML([]byte(e.fragment)))
	_, _ = w.WriteString(`">正在加载嵌入笔记…</span>`)
	return ast.WalkSkipChildren, nil
}
func prepareEmbeds(doc ast.Node) {
	var pending []struct {
		old ast.Node
		new *embedNode
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if w, ok := n.(*wikilink.Node); entering && ok && w.Embed && (!isKnownAttachment(string(w.Target))) {
			pending = append(pending, struct {
				old ast.Node
				new *embedNode
			}{n, &embedNode{target: string(w.Target), fragment: string(w.Fragment)}})
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	for _, p := range pending {
		p.old.Parent().ReplaceChild(p.old.Parent(), p.old, p.new)
	}
}

// selectFragment selects a heading and its section or an identified paragraph.
func selectFragment(doc ast.Node, fragment string) error {
	if fragment == "" {
		return nil
	}
	id := headingID(fragment)
	if strings.HasPrefix(fragment, "^") {
		id = "block-" + strings.TrimPrefix(fragment, "^")
	}
	var found ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if value, ok := n.AttributeString("id"); ok && string(value.([]byte)) == id {
				found = n
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	if found == nil {
		return errors.New("fragment not found")
	}
	// Sections are limited to document-level headings. Paragraph blocks can be nested.
	selected := []ast.Node{found}
	if item, ok := found.(*ast.ListItem); ok {
		if parent, ok := item.Parent().(*ast.List); ok {
			wrapper := ast.NewList(parent.Marker)
			wrapper.Start, wrapper.IsTight = parent.Start, parent.IsTight
			parent.RemoveChild(parent, item)
			wrapper.AppendChild(wrapper, item)
			selected = []ast.Node{wrapper}
		}
	}
	if heading, ok := found.(*ast.Heading); ok {
		for next := found.NextSibling(); next != nil; next = next.NextSibling() {
			if h, ok := next.(*ast.Heading); ok && h.Level <= heading.Level {
				break
			}
			selected = append(selected, next)
		}
	}
	for _, n := range selected {
		if n.Parent() != nil {
			n.Parent().RemoveChild(n.Parent(), n)
		}
	}
	for doc.FirstChild() != nil {
		doc.RemoveChild(doc, doc.FirstChild())
	}
	for _, n := range selected {
		doc.AppendChild(doc, n)
	}
	return nil
}

// prepareNoteLinks 将可解析的笔记标题和别名转换为实际路径。
func prepareNoteLinks(doc ast.Node, resolve func(string) (string, bool)) {
	if resolve == nil {
		return
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		link, ok := n.(*wikilink.Node)
		if !entering || !ok || len(link.Target) == 0 {
			return ast.WalkContinue, nil
		}
		if isKnownAttachment(string(link.Target)) {
			return ast.WalkContinue, nil
		}
		if target, ok := resolve(string(link.Target)); ok {
			link.Target = []byte(target)
		} else {
			link.SetAttributeString("unresolved-note", true)
		}
		return ast.WalkContinue, nil
	})
}

func isKnownAttachment(target string) bool { return linkutil.IsAttachment(target) }
