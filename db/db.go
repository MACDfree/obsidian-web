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
	db, err = gorm.Open(sqlite.Open("tmp.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic("failed to connect database")
	}

	// 启用 WAL 模式
	err = db.Exec("PRAGMA journal_mode=WAL;").Error
	if err != nil {
		panic("failed to set wal mode")
	}

	// 还可以设置其他并发相关参数
	// db.Exec("PRAGMA busy_timeout=5000;")  // 5秒超时
	// db.Exec("PRAGMA synchronous=NORMAL;") // 平衡性能和安全性

	// 迁移 schema
	db.AutoMigrate(&Note{}, &AttachInfo{}, &NoteAttachment{})
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
	AttachName string `gorm:"unique"` // 相对路径，如 "folder1/image.png"
	Path       string // 完整文件系统路径
}

// NoteAttachment 记录笔记与附件的关联关系（多对多）
type NoteAttachment struct {
	ID         uint `gorm:"primaryKey"`
	NoteID     uint `gorm:"index;not null"`
	AttachID   uint `gorm:"index;not null"`
	AttachName string
}

type TagCount struct {
	Tag   string
	Count int
}

// DeleteAllNoteAttachments 删除所有笔记-附件关联
func DeleteAllNoteAttachments() error {
	return db.Exec("DELETE FROM note_attachments").Error
}

// GetAttachInfoByName 根据附件名称查询附件（支持精确匹配和文件名匹配）
func GetAttachInfoByName(name string) (*AttachInfo, error) {
	attachInfo := &AttachInfo{}
	// 先尝试精确匹配
	result := db.Where("attach_name = ?", name).Take(attachInfo)
	if result.Error == nil {
		return attachInfo, nil
	}
	// 如果精确匹配失败，尝试文件名匹配
	result = db.Where("attach_name LIKE ?", "%/"+name).Take(attachInfo)
	if result.Error == nil {
		return attachInfo, nil
	}
	// 最后尝试直接文件名匹配（无路径）
	result = db.Where("attach_name = ?", name).Take(attachInfo)
	if result.Error != nil {
		return nil, result.Error
	}
	return attachInfo, nil
}

// CreateNoteAttachment 创建笔记-附件关联
func CreateNoteAttachment(noteID, attachID uint, attachName string) error {
	// 检查是否已存在
	var existing NoteAttachment
	result := db.Where("note_id = ? AND attach_id = ?", noteID, attachID).Take(&existing)
	if result.Error == nil {
		return nil // 已存在，跳过
	}

	noteAttachment := &NoteAttachment{
		NoteID:     noteID,
		AttachID:   attachID,
		AttachName: attachName,
	}
	return db.Create(noteAttachment).Error
}

// GetAttachPublishStatus 判断附件是否可公开访问
// 返回值: isPublic (是否可公开), exists (附件是否存在)
func GetAttachPublishStatus(attachName string) (isPublic bool, exists bool) {
	// 查询附件是否存在
	attachInfo, err := GetAttachInfoByName(attachName)
	if err != nil {
		return false, false // 附件不存在
	}

	// 查询所有引用此附件的笔记
	var noteIDs []uint
	db.Model(&NoteAttachment{}).
		Where("attach_id = ?", attachInfo.ID).
		Pluck("note_id", &noteIDs)

	// 如果没有任何笔记引用此附件
	if len(noteIDs) == 0 {
		return false, true // 存在但不可公开（保守策略）
	}

	// 检查是否有任意公开笔记引用
	var publicCount int64
	db.Model(&Note{}).
		Where("id IN ? AND publish = ?", noteIDs, true).
		Count(&publicCount)

	return publicCount > 0, true
}
