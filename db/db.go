package db

import (
	"io"
	"log"
	"obsidian-web/config"
	"os"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func init() {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "logs/gorm.log",
		MaxSize:    100,
		MaxBackups: 0,
		MaxAge:     180,
		Compress:   false,
	}
	out := io.MultiWriter(lumberJackLogger, os.Stdout)
	newLogger := gormlogger.New(
		log.New(out, "\r\n", log.LstdFlags), // io writer
		gormlogger.Config{
			SlowThreshold:             time.Second,     // Slow SQL threshold
			LogLevel:                  gormlogger.Info, // Log level
			IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,            // Don't include params in the SQL log
			Colorful:                  false,           // Disable color
		},
	)

	var err error
	// db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic("failed to connect database")
	}

	// 迁移 schema
	db.AutoMigrate(&Note{}, &AttachInfo{})
}

func DeleteAll() error {
	err := db.Exec("delete from notes").Error
	if err != nil {
		return err
	}

	err = db.Exec("delete from attach_infos").Error
	if err != nil {
		return err
	}

	return nil
}

func InsertNote(note *Note) error {
	result := db.Create(note)
	return result.Error
}

func ListNote(isLogin bool, pageIndex int) ([]Note, error) {
	var notes []Note = make([]Note, 0)
	if isLogin {
		result := db.Order("created desc").Limit(config.Get().Paginate).Offset(config.Get().Paginate * pageIndex).Find(&notes)
		for i := range notes {
			notes[i].FullTitle = strings.TrimSuffix(notes[i].FullTitle, "/_index")
		}
		return notes, result.Error
	} else {
		result := db.Where("publish = ?", true).Order("created desc").Limit(config.Get().Paginate).Offset(config.Get().Paginate * pageIndex).Find(&notes)
		for i := range notes {
			notes[i].FullTitle = strings.TrimSuffix(notes[i].FullTitle, "/_index")
		}
		return notes, result.Error
	}
}

func ListAllNote(isLogin bool) ([]Note, error) {
	var notes []Note = make([]Note, 0)
	if isLogin {
		result := db.Order("created desc").Find(&notes)
		for i := range notes {
			notes[i].FullTitle = strings.TrimSuffix(notes[i].FullTitle, "/_index")
		}
		return notes, result.Error
	} else {
		result := db.Where("publish = ?", true).Order("created desc").Find(&notes)
		for i := range notes {
			notes[i].FullTitle = strings.TrimSuffix(notes[i].FullTitle, "/_index")
		}
		return notes, result.Error
	}
}

func ListTag(isLogin bool) ([]TagCount, error) {
	var tagCounts []TagCount = make([]TagCount, 0)
	if isLogin {
		result := db.Raw("SELECT json_each.value tag, count(*) count FROM notes,json_each(notes.tags) group by json_each.value order by count(*) desc").Scan(&tagCounts)
		return tagCounts, result.Error
	} else {
		result := db.Raw("SELECT json_each.value tag, count(*) count FROM notes,json_each(notes.tags) where notes.publish='1' group by json_each.value order by count(*) desc").Scan(&tagCounts)
		return tagCounts, result.Error
	}
}

func ListNoteByTag(isLogin bool, tag string) ([]Note, error) {
	var notes []Note = make([]Note, 0)
	if isLogin {
		result := db.Where("tags like ?", "%\""+tag+"\"%").Order("created desc").Find(&notes)
		return notes, result.Error
	} else {
		result := db.Where("tags like ? and publish='1'", "%\""+tag+"\"%").Order("created desc").Find(&notes)
		return notes, result.Error
	}
}

func GetNoteByPath(fullPath string) (*Note, error) {
	note := &Note{}
	result := db.Where("full_title in (?,?)", fullPath, fullPath+"/_index").Take(note)
	if result.Error != nil {
		return nil, result.Error
	}
	return note, nil
}

func InsertAttachInfo(attachInfo *AttachInfo) error {
	result := db.Create(attachInfo)
	return result.Error
}

type Note struct {
	ID        uint `gorm:"primaryKey"`
	Title     string
	FullTitle string   `gorm:"unique"`
	Aliases   []string `gorm:"serializer:json"`
	Created   time.Time
	Updated   time.Time `gorm:"index:,sort:desc"`
	Tags      []string  `gorm:"serializer:json"`
	Publish   bool
	Path      string
	MD5       string

	ExtInfo map[string]any `gorm:"serializer:json"`
}

type AttachInfo struct {
	ID         uint   `gorm:"primaryKey"`
	AttachName string `gorm:"unique"`
	Path       string
}

type TagCount struct {
	Tag   string
	Count int
}
