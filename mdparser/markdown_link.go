package mdparser

import (
	"net/url"
	"path"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// markdownTarget 只接管库内相对笔记链接；外部地址、附件和网站绝对路径保持原样。
func markdownTarget(dest []byte) (*url.URL, bool) {
	u, err := url.Parse(string(dest))
	if err != nil || u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") || u.Path == "" {
		return nil, false
	}
	ext := strings.ToLower(path.Ext(u.Path))
	if isKnownAttachment(u.Path) || (ext != "" && ext != ".md") {
		return nil, false
	}
	return u, true
}

func normalizeFragment(fragment string) string {
	if fragment == "" {
		return ""
	}
	if strings.HasPrefix(fragment, "^") {
		return "block-" + strings.TrimPrefix(fragment, "^")
	}
	return headingID(fragment)
}

func prepareMarkdownLinks(doc ast.Node, resolve func(string) (string, bool)) {
	if resolve == nil {
		return
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		link, ok := n.(*ast.Link)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		// 本页标题和块链接无需查找其他笔记，但锚点规则应与标题 ID 一致。
		if local, err := url.Parse(string(link.Destination)); err == nil && local.Path == "" && local.Scheme == "" && local.Host == "" && local.Fragment != "" {
			local.Fragment = normalizeFragment(local.Fragment)
			link.Destination = []byte(local.String())
			return ast.WalkContinue, nil
		}
		u, ok := markdownTarget(link.Destination)
		if !ok {
			return ast.WalkContinue, nil
		}
		target, found := resolve(u.Path)
		if !found {
			link.SetAttributeString("unresolved-note", true)
			return ast.WalkContinue, nil
		}
		link.Destination = []byte((&url.URL{Path: "/note/" + target, RawQuery: u.RawQuery, Fragment: normalizeFragment(u.Fragment)}).String())
		return ast.WalkContinue, nil
	})
}

type markdownLinkRenderer struct{}

func (*markdownLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, renderMarkdownLink)
}
func renderMarkdownLink(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if _, unresolved := node.AttributeString("unresolved-note"); unresolved {
		if entering {
			_, _ = w.WriteString(`<span class="unresolved-link" title="笔记不存在、存在歧义或无访问权限">`)
		} else {
			_, _ = w.WriteString(`</span>`)
		}
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Link)
	if entering {
		_, _ = w.WriteString(`<a href="`)
		dest := util.URLEscape(n.Destination, true)
		if !html.IsDangerousURL(dest) {
			_, _ = w.Write(util.EscapeHTML(dest))
		}
		_ = w.WriteByte('"')
		if n.Title != nil {
			_, _ = w.WriteString(` title="`)
			_, _ = w.Write(util.EscapeHTML(n.Title))
			_ = w.WriteByte('"')
		}
		if n.Attributes() != nil {
			html.RenderAttributes(w, n, html.LinkAttributeFilter)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString(`</a>`)
	}
	return ast.WalkContinue, nil
}
