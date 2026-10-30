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

var imageEmbedKind = ast.NewNodeKind("ImageEmbed")
var imageSize = regexp.MustCompile(`^([1-9][0-9]{0,3})(?:x([1-9][0-9]{0,3}))?$`)

type imageEmbedNode struct {
	ast.BaseInline
	target, alt, width, height string
}

func (*imageEmbedNode) Kind() ast.NodeKind           { return imageEmbedKind }
func (n *imageEmbedNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type imageEmbedRenderer struct{}

func (*imageEmbedRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(imageEmbedKind, renderImageEmbed)
}
func renderImageEmbed(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*imageEmbedNode)
	_, _ = w.WriteString(`<img src="`)
	_, _ = w.Write(util.EscapeHTML(util.URLEscape([]byte("/attachment/"+n.target), true)))
	_, _ = w.WriteString(`" alt="`)
	_, _ = w.Write(util.EscapeHTML([]byte(n.alt)))
	_, _ = w.WriteString(`"`)
	if n.width != "" {
		_, _ = w.WriteString(` width="` + n.width + `"`)
	}
	if n.height != "" {
		_, _ = w.WriteString(` height="` + n.height + `" style="height:` + n.height + `px;max-width:100%;object-fit:fill"`)
	}
	_, _ = w.WriteString(` />`)
	return ast.WalkSkipChildren, nil
}
func prepareImageEmbeds(doc ast.Node, src []byte) {
	var pending []struct {
		old ast.Node
		new *imageEmbedNode
	}
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		link, ok := node.(*wikilink.Node)
		if !entering || !ok || !link.Embed {
			return ast.WalkContinue, nil
		}
		switch strings.ToLower(filepath.Ext(string(link.Target))) {
		case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg":
		default:
			return ast.WalkContinue, nil
		}
		label := nodeText(link, src)
		if label == string(link.Target) {
			label = ""
		}
		n := &imageEmbedNode{target: string(link.Target), alt: label}
		if match := imageSize.FindStringSubmatch(strings.TrimSpace(label)); match != nil {
			n.width, n.height, n.alt = match[1], match[2], ""
		}
		pending = append(pending, struct {
			old ast.Node
			new *imageEmbedNode
		}{node, n})
		return ast.WalkSkipChildren, nil
	})
	for _, p := range pending {
		p.old.Parent().ReplaceChild(p.old.Parent(), p.old, p.new)
	}
}
