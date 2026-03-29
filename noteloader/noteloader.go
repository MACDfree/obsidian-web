package noteloader

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"io"
	"obsidian-web/config"
	"obsidian-web/db"
	"obsidian-web/logger"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

// attachmentExtensions 支持的附件扩展名
var attachmentExtensions = []string{
	".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", // 图片
	".pdf", ".zip", ".doc", ".docx", ".xls", ".xlsx", // 文档
	".mp3", ".mp4", ".wav", ".webm", // 音视频
}

func Load() {
	loadNoteBook()
}

func loadNoteBook() {
	start := time.Now()
	err := db.DeleteAll()
	if err != nil {
		logger.Error(errors.WithStack(err))
		return
	}
	err = db.DeleteAllNoteAttachments()
	if err != nil {
		logger.Error(errors.WithStack(err))
		return
	}
	err = db.DeleteAllNoteLinks()
	if err != nil {
		logger.Error(errors.WithStack(err))
		return
	}

	rootPath := config.Get().NotePath
	ignorePaths := config.Get().IgnorePaths
	attachPath := config.Get().AttachmentPath

	// 用于保存笔记信息和附件引用
	type noteWithRefs struct {
		note      *db.Note
		refs      []string
		content   []byte // 保存内容用于后续提取笔记链接
	}
	var notesWithRefs []noteWithRefs

	// 笔记查找表：用于解析 wikilink 目标
	// key: FullTitle -> NoteID
	// key: "title:"+Title -> NoteID
	// key: "alias:"+Alias -> NoteID
	noteLookup := make(map[string]uint)

	err = filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootPath {
			return nil
		}
		if d.IsDir() {
			if lo.Contains(ignorePaths, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			// 执行解析逻辑
			note, err := parseFrontMatter(path)
			if err != nil {
				logger.Error(errors.WithStack(err))
				return nil
			}
			if note.Tags == nil {
				note.Tags = []string{}
			}
			if note.Aliases == nil {
				note.Aliases = []string{}
			}
			if note.Title == "" {
				title := filepath.Base(path)
				title = strings.TrimSuffix(title, ".md")
				note.Title = title
			}
			fullTitle := path[len(rootPath)+1 : len(path)-3]
			fullTitle = strings.ReplaceAll(fullTitle, "\\", "/")
			note.FullTitle = fullTitle

			if note.Created.IsZero() {
				if strings.HasPrefix(fullTitle, "daily note/") {
					str := fullTitle[len("daily note/"):]
					t, err := time.Parse("2006-01-02", str)
					if err != nil {
						logger.Error(errors.WithStack(err))
					}
					note.Created = t
					note.Updated = t
				}
			}

			md, err := fileMD5(path)
			if err != nil {
				logger.Error(errors.WithStack(err))
				return nil
			}
			note.MD5 = md

			// 读取完整内容并提取附件引用
			content, err := os.ReadFile(path)
			if err != nil {
				logger.Error(errors.WithStack(err))
				return nil
			}
			refs := extractAttachmentRefs(content)

			err = db.InsertNote(note)
			if err != nil {
				logger.Error(errors.WithStack(err))
			} else {
				// 只在插入成功后保存引用信息和内容
				notesWithRefs = append(notesWithRefs, noteWithRefs{note: note, refs: refs, content: content})
				// 构建查找表
				noteLookup[note.FullTitle] = note.ID
				noteLookup["title:"+note.Title] = note.ID
				for _, alias := range note.Aliases {
					noteLookup["alias:"+alias] = note.ID
				}
			}
		} else if strings.HasPrefix(path[len(rootPath)+1:], attachPath) {
			// 处理附件元信息
			// 计算相对路径（相对于 attachment_path）
			relativePath := path[len(rootPath)+1+len(attachPath)+1:]
			attachInfo := &db.AttachInfo{
				Path:       path,
				AttachName: relativePath,
			}
			err := db.InsertAttachInfo(attachInfo)
			if err != nil {
				logger.Error(errors.WithStack(err))
			}
		}
		return nil
	})
	if err != nil {
		logger.Error(errors.WithStack(err))
	}

	// 建立笔记-附件关联
	for _, nwr := range notesWithRefs {
		for _, ref := range nwr.refs {
			attachInfo, err := db.GetAttachInfoByName(ref)
			if err != nil {
				// 附件不存在，跳过（可能是外部链接或引用错误）
				continue
			}
			err = db.CreateNoteAttachment(nwr.note.ID, attachInfo.ID, ref)
			if err != nil {
				logger.Error(errors.WithStack(err))
			}
		}
	}

	// 建立笔记链接关系
	for _, nwr := range notesWithRefs {
		noteLinks := extractNoteLinks(nwr.content, noteLookup)
		for _, targetNoteID := range noteLinks {
			err = db.CreateNoteLink(nwr.note.ID, targetNoteID)
			if err != nil {
				logger.Error(errors.WithStack(err))
			}
		}
	}

	logger.Infof("init notebook cost: %v", time.Since(start))
}

func parseFrontMatter(mdPath string) (*db.Note, error) {
	// 1. 读取md文件的前几行，不需要全读
	mdFile, err := os.Open(mdPath)
	if err != nil {
		return nil, err
	}
	defer mdFile.Close()

	scanner := bufio.NewScanner(mdFile)
	sb := &strings.Builder{}
	if scanner.Scan() {
		line := scanner.Text()
		if line != "---" {
			// 没有front matter的话，存储基本的信息
			return &db.Note{
				Path: mdPath,
			}, nil
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		sb.WriteString(line)
		sb.WriteString("\r\n")
	}

	frontMatter := &FrontMatter{}
	err = yaml.Unmarshal([]byte(sb.String()), frontMatter)
	if err != nil {
		return nil, err
	}
	note := &db.Note{
		Title:   frontMatter.Title,
		Tags:    frontMatter.Tags,
		Aliases: frontMatter.Aliases,
		Created: frontMatter.Created.Time,
		Updated: frontMatter.Updated.Time,
		Publish: frontMatter.Publish,
		Path:    mdPath,
		ExtInfo: frontMatter.ExtInfo,
	}
	return note, nil
}

type FrontMatter struct {
	Title   string         `yaml:"title"`
	Tags    []string       `yaml:"tags"`
	Aliases []string       `yaml:"aliases"`
	Created CTime          `yaml:"created"`
	Updated CTime          `yaml:"updated"`
	Publish bool           `yaml:"publish"`
	ExtInfo map[string]any `yaml:",inline"`
}

type CTime struct {
	time.Time
}

func (cTime CTime) MarshalYAML() (any, error) {
	return yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.TaggedStyle,
		Value: cTime.Format("2006-01-02T15:04"),
	}, nil
}

func (cTime *CTime) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		t, err := time.ParseInLocation("2006-01-02T15:04", value.Value, time.Local)
		if err != nil {
			logger.Warnf("再试一次，%+v", err)
			t, err = time.ParseInLocation("2006-01-02T15:04:05", value.Value, time.Local)
			if err != nil {
				return err
			}
		}
		cTime.Time = t
	}
	return nil
}

func fileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// extractNoteLinks 从 markdown 内容中提取所有笔记链接（非附件）
// 返回目标笔记 ID 列表（已去重）
func extractNoteLinks(content []byte, noteLookup map[string]uint) []uint {
	links := make(map[uint]bool) // 使用 map 去重

	// 提取 wikilink 格式: [[noteName]] 或 [[noteName|显示文本]] 或 [[noteName#heading]]
	wikilinkRe := regexp.MustCompile(`\[\[([^\]|#]+)(?:[|#][^\]]+)?\]\]`)
	for _, match := range wikilinkRe.FindAllSubmatch(content, -1) {
		target := strings.TrimSpace(string(match[1]))

		// 排除附件链接（根据扩展名判断）
		if isAttachmentLink(target) {
			continue
		}

		// 解析笔记目标
		noteID := resolveNoteTarget(target, noteLookup)
		if noteID > 0 {
			links[noteID] = true
		}
	}

	// 转换为 slice
	result := make([]uint, 0, len(links))
	for noteID := range links {
		result = append(result, noteID)
	}
	return result
}

// isAttachmentLink 判断是否为附件链接
func isAttachmentLink(target string) bool {
	for _, ext := range attachmentExtensions {
		if strings.HasSuffix(strings.ToLower(target), ext) {
			return true
		}
	}
	return false
}

// resolveNoteTarget 解析笔记目标，返回笔记ID
// 支持通过 FullTitle、Title、Aliases 匹配
func resolveNoteTarget(target string, noteLookup map[string]uint) uint {
	// 1. 精确匹配 FullTitle
	if id, ok := noteLookup[target]; ok {
		return id
	}
	// 2. 匹配 Title（不含路径）
	if id, ok := noteLookup["title:"+target]; ok {
		return id
	}
	// 3. 匹配 Alias
	if id, ok := noteLookup["alias:"+target]; ok {
		return id
	}
	return 0
}

// extractAttachmentRefs 从 markdown 内容中提取所有附件引用
func extractAttachmentRefs(content []byte) []string {
	refs := make(map[string]bool) // 使用 map 去重

	// 1. 提取 wikilink 格式: [[filename.ext]] 或 [[path/filename.ext]]
	wikilinkRe := regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
	for _, match := range wikilinkRe.FindAllSubmatch(content, -1) {
		target := string(match[1])
		// 判断是否为附件（根据扩展名）
		for _, ext := range attachmentExtensions {
			if strings.HasSuffix(strings.ToLower(target), ext) {
				refs[target] = true
				break
			}
		}
	}

	// 2. 提取 markdown 图片/链接: ![alt](path) 或 [text](path)
	mdLinkRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)|\[[^\]]+\]\(([^)]+)\)`)
	for _, match := range mdLinkRe.FindAllSubmatch(content, -1) {
		// match[2] 是图片路径，match[3] 是链接路径
		target := string(match[2])
		if target == "" {
			target = string(match[3])
		}
		if target == "" {
			continue
		}
		// 处理路径，提取文件名或相对路径
		for _, ext := range attachmentExtensions {
			if strings.HasSuffix(strings.ToLower(target), ext) {
				refs[target] = true
				break
			}
		}
	}

	// 转换为 slice
	result := make([]string, 0, len(refs))
	for ref := range refs {
		result = append(result, ref)
	}
	return result
}
