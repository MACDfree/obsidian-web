package handler

import (
	"encoding/xml"
	"net/http"
	"obsidian-web/config"
	"obsidian-web/db"
	"obsidian-web/logger"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	LastBuild   string    `xml:"lastBuildDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Guid        string `xml:"guid"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
}

const rssTimeLayout = time.RFC1123Z

func RSS(ctx *gin.Context) {
	notes, err := db.ListRecentPublicNotes(20)
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}

	scheme := "http"
	if proto := ctx.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if ctx.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + ctx.Request.Host

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       config.Get().Title,
			Link:        baseURL,
			Description: "以文字记录时间",
			Items:       make([]rssItem, 0, len(notes)),
		},
	}

	for _, note := range notes {
		link := baseURL + "/note/" + note.FullTitle
		item := rssItem{
			Title: note.Title,
			Link:  link,
			Guid:  note.FullTitle,
		}
		if !note.Updated.IsZero() {
			item.PubDate = note.Updated.Format(rssTimeLayout)
			if feed.Channel.LastBuild == "" {
				feed.Channel.LastBuild = item.PubDate
			}
		}
		item.Description = noteExcerpt(note.Path)
		feed.Channel.Items = append(feed.Channel.Items, item)
	}

	out, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		logger.Error(errors.WithStack(err))
		ctx.String(500, "error: %v", err)
		return
	}
	ctx.Data(http.StatusOK, "application/rss+xml; charset=utf-8", append([]byte(xml.Header), out...))
}

// noteExcerpt 提取笔记正文开头一小段纯文本作为摘要
func noteExcerpt(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(db.StripFrontMatter(content)))
	text = strings.ReplaceAll(text, "\r\n", "\n")
	// 去掉 markdown 语法符号，保留纯文本
	replacer := strings.NewReplacer("**", "", "__", "", "`", "", "]]", "", "[[", "")
	text = replacer.Replace(text)
	runes := []rune(text)
	if len(runes) > 200 {
		text = string(runes[:200]) + "…"
	}
	return text
}
