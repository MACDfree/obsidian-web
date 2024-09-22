package noteloader

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"io"
	"obsidian-web/config"
	"obsidian-web/db"
	"obsidian-web/log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

func Load() {
	loadNoteBook()
}

func loadNoteBook() {
	start := time.Now()
	err := db.DeleteAll()
	if err != nil {
		log.Error(errors.WithStack(err))
		return
	}
	rootPath := config.Get().NotePath
	ignorePaths := config.Get().IgnorePaths
	attachPath := config.Get().AttachmentPath
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
				log.Error(errors.WithStack(err))
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
						log.Error(errors.WithStack(err))
					}
					note.Created = t
					note.Updated = t
				}
			}

			md, err := fileMD5(path)
			if err != nil {
				log.Error(errors.WithStack(err))
				return nil
			}
			note.MD5 = md

			err = db.InsertNote(note)
			if err != nil {
				log.Error(errors.WithStack(err))
			}
		} else if strings.HasPrefix(path[len(rootPath)+1:], attachPath) {
			// 处理附件元信息
			attachInfo := &db.AttachInfo{
				Path:       path,
				AttachName: filepath.Base(path),
			}
			err := db.InsertAttachInfo(attachInfo)
			if err != nil {
				log.Error(errors.WithStack(err))
			}
		}
		return nil
	})
	if err != nil {
		log.Error(errors.WithStack(err))
	}
	log.Infof("init notebook cost: %v", time.Since(start))
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

func (cTime CTime) MarshalYAML() (interface{}, error) {
	return yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.TaggedStyle,
		Value: cTime.Format("2006-01-02T15:04"),
	}, nil
}

func (cTime *CTime) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		t, err := time.Parse("2006-01-02T15:04", value.Value)
		if err != nil {
			log.Warnf("再试一次，%+v", err)
			t, err = time.Parse("2006-01-02T15:04:05", value.Value)
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
