package mdparser

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

var mediaEmbedKind = ast.NewNodeKind("MediaEmbed")
var pdfPage = regexp.MustCompile(`^page=[1-9][0-9]{0,5}$`)

type mediaEmbedNode struct {
	ast.BaseInline
	target, kind, fragment string
}

func (*mediaEmbedNode) Kind() ast.NodeKind           { return mediaEmbedKind }
func (n *mediaEmbedNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type mediaEmbedRenderer struct{}

func (*mediaEmbedRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(mediaEmbedKind, renderMediaEmbed)
}
func renderMediaEmbed(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*mediaEmbedNode)
	url := "/attachment/" + n.target
	if n.kind == "iframe" && pdfPage.MatchString(n.fragment) {
		url += "#" + n.fragment
	}
	_, _ = w.WriteString("<" + n.kind + ` class="media-embed"`)
	if n.kind == "iframe" {
		_, _ = w.WriteString(` title="PDF" loading="lazy"`)
	} else {
		_, _ = w.WriteString(` controls preload="metadata"`)
	}
	_, _ = w.WriteString(` src="`)
	_, _ = w.Write(util.EscapeHTML(util.URLEscape([]byte(url), true)))
	_, _ = w.WriteString(`"></` + n.kind + `>`)
	return ast.WalkSkipChildren, nil
}
func prepareMediaEmbeds(doc ast.Node) {
	var pending []struct {
		old ast.Node
		new *mediaEmbedNode
	}
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		link, ok := node.(*wikilink.Node)
		if !entering || !ok || !link.Embed {
			return ast.WalkContinue, nil
		}
		kind := ""
		switch strings.ToLower(filepath.Ext(string(link.Target))) {
		case ".pdf":
			kind = "iframe"
		case ".mp3", ".wav":
			kind = "audio"
		case ".mp4", ".webm":
			kind = "video"
		}
		if kind != "" {
			pending = append(pending, struct {
				old ast.Node
				new *mediaEmbedNode
			}{node, &mediaEmbedNode{target: string(link.Target), fragment: string(link.Fragment), kind: kind}})
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	for _, p := range pending {
		p.old.Parent().ReplaceChild(p.old.Parent(), p.old, p.new)
	}
}
