package mdparser

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

type noteLinkRenderer struct{ fallback *wikilink.Renderer }

func (r *noteLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(wikilink.Kind, r.render)
}
func (r *noteLinkRenderer) render(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if _, ok := node.AttributeString("unresolved-note"); ok {
		if entering {
			_, _ = w.WriteString(`<span class="unresolved-link" title="笔记不存在、存在歧义或无访问权限">`)
		} else {
			_, _ = w.WriteString(`</span>`)
		}
		return ast.WalkContinue, nil
	}
	return r.fallback.Render(w, src, node, entering)
}
