package mdparser

import (
	"bytes"
	"path/filepath"
	"sync"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/samber/lo"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/wikilink"
)

var markdown goldmark.Markdown
var markdownWithMathJax goldmark.Markdown

// var cssStr = bytes.NewBuffer([]byte{})

func init() {
	sync.OnceFunc(func() {
		markdown = goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				&frontmatter.Extender{},
				&wikilink.Extender{
					Resolver: myResolver{},
				},
				highlighting.NewHighlighting(
					highlighting.WithStyle("monokai"),
					highlighting.WithFormatOptions(
						chromahtml.ClassPrefix(""),
						chromahtml.WithClasses(true),
						chromahtml.WithLineNumbers(true),
						chromahtml.LineNumbersInTable(true),
						chromahtml.TabWidth(4),
					),
					// highlighting.WithCSSWriter(cssStr),
					highlighting.WithWrapperRenderer(func(w util.BufWriter, context highlighting.CodeBlockContext, entering bool) {
						if entering {
							if lang, ok := context.Language(); ok {
								w.WriteString(`<div class="highlight language-`)
								w.Write(lang)
								w.WriteString(`">`)
							} else {
								w.WriteString(`<div class="highlight language-plaintext">`)
							}
						} else {
							w.WriteString("</div>")
						}
					}),
				),
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
				renderer.WithNodeRenderers(
					util.Prioritized(&ObsidianRenderer{}, 999),
				),
			),
		)

		markdownWithMathJax = goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				&frontmatter.Extender{},
				&wikilink.Extender{
					Resolver: myResolver{},
				},
				mathjax.MathJax,
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
				renderer.WithNodeRenderers(
					util.Prioritized(&ObsidianRenderer{}, 999),
				),
			),
		)
	})()
}

var exts = []string{
	".png",
	".pdf",
	".jpg",
	".zip",
}

type myResolver struct {
}

func (myResolver) ResolveWikilink(n *wikilink.Node) (destination []byte, err error) {
	_hash := []byte{'#'}
	dest := make([]byte, lo.Max([]int{len([]byte("/note/")), len([]byte("/attachment/"))})+len(n.Target)+len("#")+len(n.Fragment))
	var i int
	if len(n.Target) > 0 {
		ext := filepath.Ext(string(n.Target))
		if lo.Contains(exts, ext) {
			i += copy(dest[i:], []byte("/attachment/"))
			i += copy(dest[i:], n.Target)
		} else {
			i += copy(dest[i:], []byte("/note/"))
			i += copy(dest[i:], n.Target)
		}
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}
	return dest[:i], nil
}

type ObsidianRenderer struct {
	html.Renderer
}

func (r *ObsidianRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *ObsidianRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)
	_, _ = w.WriteString("<img src=\"")
	if r.Unsafe || !html.IsDangerousURL(n.Destination) {
		// 对于图片，如果路径只有文件名称，这默认到attachment文件夹下
		if !bytes.Contains(n.Destination, []byte("/")) {
			_, _ = w.WriteString("/attachment/")
		}
		_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
	}
	_, _ = w.WriteString(`" alt="`)
	_, _ = w.Write(nodeToHTMLText(n, source))
	_ = w.WriteByte('"')
	if n.Title != nil {
		_, _ = w.WriteString(` title="`)
		r.Writer.Write(w, n.Title)
		_ = w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, html.ImageAttributeFilter)
	}
	if r.XHTML {
		_, _ = w.WriteString(" />")
	} else {
		_, _ = w.WriteString(">")
	}
	return ast.WalkSkipChildren, nil
}

func nodeToHTMLText(n ast.Node, source []byte) []byte {
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if s, ok := c.(*ast.String); ok && s.IsCode() {
			buf.Write(s.Text(source))
		} else if !c.HasChildren() {
			buf.Write(util.EscapeHTML(c.Text(source)))
			if t, ok := c.(*ast.Text); ok && t.SoftLineBreak() {
				buf.WriteByte('\n')
			}
		} else {
			buf.Write(nodeToHTMLText(c, source))
		}
	}
	return buf.Bytes()
}

func ConvertToHTML(src []byte, withMathJax bool) (string, error) {
	var htmlBuf bytes.Buffer
	var err error
	if withMathJax {
		err = markdownWithMathJax.Convert(src, &htmlBuf)

	} else {
		err = markdown.Convert(src, &htmlBuf)
	}
	if err != nil {
		return "", err
	}
	// logger.Info(cssStr.String())
	return htmlBuf.String(), nil
}
