package db

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

// SearchDoc 搜索索引文档
type SearchDoc struct {
	NoteID  uint
	Title   string
	Content string
}

// SearchHit 搜索命中结果（HitTitle/HitText 已转义，仅保留 <mark> 高亮标记）
type SearchHit struct {
	FullTitle string
	HitTitle  string
	HitText   []string
}

// RebuildSearchIndex 全量重建 FTS5 索引
func RebuildSearchIndex(docs []SearchDoc) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM notes_fts").Error; err != nil {
			return err
		}
		for _, doc := range docs {
			if err := tx.Exec("INSERT INTO notes_fts(note_id, title, content) VALUES (?,?,?)",
				doc.NoteID, doc.Title, doc.Content).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SearchNotes 全文搜索，关键词按字面量子串匹配（非正则）
func SearchNotes(isLogin bool, keyword string) ([]SearchHit, error) {
	if utf8.RuneCountInString(keyword) >= 3 {
		return searchFts(isLogin, keyword)
	}
	// trigram 索引要求至少 3 个字符，短关键词退化为 LIKE 扫描（仍在数据库内，不读文件）
	return searchLike(isLogin, keyword)
}

func searchFts(isLogin bool, keyword string) ([]SearchHit, error) {
	// FTS5 字符串字面量：双引号包裹，内部双引号翻倍转义
	match := `"` + strings.ReplaceAll(keyword, `"`, `""`) + `"`
	q := `SELECT n.full_title,
			highlight(notes_fts, 1, '<mark>', '</mark>') AS hit_title,
			snippet(notes_fts, 2, '<mark>', '</mark>', '…', 16) AS hit_text
		FROM notes_fts JOIN notes n ON n.id = notes_fts.note_id
		WHERE notes_fts MATCH ?`
	if !isLogin {
		q += ` AND n.publish = 1`
	}
	q += ` ORDER BY rank LIMIT 100`

	var rows []struct {
		FullTitle string
		HitTitle  string
		HitText   string
	}
	err := db.Raw(q, match).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	hits := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		hit := SearchHit{FullTitle: r.FullTitle}
		if strings.Contains(r.HitTitle, "<mark>") {
			hit.HitTitle = sanitizeHighlight(r.HitTitle)
		}
		if strings.Contains(r.HitText, "<mark>") {
			hit.HitText = []string{sanitizeHighlight(r.HitText)}
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

func searchLike(isLogin bool, keyword string) ([]SearchHit, error) {
	like := "%" + escapeLike(keyword) + "%"
	q := `SELECT n.full_title, f.title, f.content
		FROM notes_fts f JOIN notes n ON n.id = f.note_id
		WHERE (f.title LIKE ? ESCAPE '\' OR f.content LIKE ? ESCAPE '\')`
	if !isLogin {
		q += ` AND n.publish = 1`
	}
	q += ` ORDER BY n.updated DESC LIMIT 100`

	var rows []struct {
		FullTitle string
		Title     string
		Content   string
	}
	err := db.Raw(q, like, like).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	hits := make([]SearchHit, 0, len(rows))
	for _, r := range rows {
		hit := SearchHit{FullTitle: r.FullTitle}
		if title := markSubstring(r.Title, keyword); title != "" {
			hit.HitTitle = title
		}
		if snippet := snippetAround(r.Content, keyword, 40); snippet != "" {
			hit.HitText = []string{snippet}
		}
		hits = append(hits, hit)
	}
	return hits, nil
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// markSubstring 大小写不敏感地高亮所有出现位置；未命中返回空串。返回值是 HTML，需转义原文。
func markSubstring(s, keyword string) string {
	lowerS, lowerK := strings.ToLower(s), strings.ToLower(keyword)
	if !strings.Contains(lowerS, lowerK) {
		return ""
	}
	var buf bytes.Buffer
	pos := 0
	for {
		idx := strings.Index(lowerS[pos:], lowerK)
		if idx < 0 {
			buf.WriteString(htmlEscape(s[pos:]))
			break
		}
		buf.WriteString(htmlEscape(s[pos : pos+idx]))
		buf.WriteString("<mark>")
		buf.WriteString(htmlEscape(s[pos+idx : pos+idx+len(keyword)]))
		buf.WriteString("</mark>")
		pos += idx + len(keyword)
	}
	return buf.String()
}

// snippetAround 取首个命中位置前后的上下文片段（带 <mark> 高亮，HTML 已转义）
func snippetAround(content, keyword string, radius int) string {
	lowerC, lowerK := strings.ToLower(content), strings.ToLower(keyword)
	idx := strings.Index(lowerC, lowerK)
	if idx < 0 {
		return ""
	}
	start := idx - radius
	if start < 0 {
		start = 0
	}
	// 对齐到 rune 边界
	for start > 0 && !utf8.RuneStart(content[start]) {
		start--
	}
	end := idx + len(keyword) + radius
	if end > len(content) {
		end = len(content)
	}
	for end < len(content) && !utf8.RuneStart(content[end]) {
		end++
	}
	prefix, suffix := "", ""
	if start > 0 {
		prefix = "…"
	}
	if end < len(content) {
		suffix = "…"
	}
	return prefix + markSubstring(strings.ReplaceAll(content[start:end], "\n", " "), keyword) + suffix
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// sanitizeHighlight 转义 FTS highlight/snippet 的输出，仅放行 <mark> 标记
func sanitizeHighlight(s string) string {
	s = htmlEscape(s)
	s = strings.ReplaceAll(s, "&lt;mark&gt;", "<mark>")
	s = strings.ReplaceAll(s, "&lt;/mark&gt;", "</mark>")
	return s
}

// StripFrontMatter 去掉 markdown 开头的 YAML frontmatter 块
func StripFrontMatter(content []byte) []byte {
	if !bytes.HasPrefix(content, []byte("---")) {
		return content
	}
	rest := content[3:]
	// 跳过第一行的换行
	if i := bytes.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[i+1:]
	} else {
		return content
	}
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return content
	}
	after := rest[end+4:]
	if i := bytes.IndexByte(after, '\n'); i >= 0 {
		return after[i+1:]
	}
	return nil
}
