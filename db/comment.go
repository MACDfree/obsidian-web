package db

import (
	"io"
	"log"
	"os"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var commentDB *gorm.DB

// Comment 评论
type Comment struct {
	ID            uint      `gorm:"primaryKey"`
	NoteFullTitle string    `gorm:"index;not null;size:500"` // 笔记的 FullTitle（相对路径，稳定标识）
	ParentID      *uint     `gorm:"index"`                   // 父评论ID，nil 表示顶级评论
	Nickname      string    `gorm:"size:50;not null"`
	Content       string    `gorm:"type:text;not null"`
	Status        int       `gorm:"default:0;not null"` // 0=正常, 2=隐藏
	IP            string    `gorm:"size:45"`
	UserAgent     string    `gorm:"size:500"`
	CreatedAt     time.Time `gorm:"index"`
}

// InitCommentDB 初始化评论数据库（独立 SQLite 文件）
func InitCommentDB() {
	os.MkdirAll("data", 0755)

	lumberJackLogger := &lumberjack.Logger{
		Filename:   "logs/gorm_comment.log",
		MaxSize:    100,
		MaxBackups: 0,
		MaxAge:     180,
		Compress:   false,
	}
	out := io.MultiWriter(lumberJackLogger, os.Stdout)
	newLogger := gormlogger.New(
		log.New(out, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)

	var err error
	commentDB, err = gorm.Open(sqlite.Open("data/comments.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic("failed to connect comment database")
	}

	commentDB.Exec("PRAGMA journal_mode=WAL;")
	commentDB.AutoMigrate(&Comment{})
}

// InsertComment 插入评论
func InsertComment(comment *Comment) error {
	return commentDB.Create(comment).Error
}

// ListCommentsByNoteFullTitle 获取指定笔记的所有可见评论，按创建时间升序
func ListCommentsByNoteFullTitle(fullTitle string) ([]Comment, error) {
	var comments []Comment
	err := commentDB.Where("note_full_title = ? AND status = 0", fullTitle).
		Order("created_at asc").
		Find(&comments).Error
	return comments, err
}

// CountCommentsByNoteFullTitle 获取指定笔记的可见评论数
func CountCommentsByNoteFullTitle(fullTitle string) (int64, error) {
	var count int64
	err := commentDB.Model(&Comment{}).
		Where("note_full_title = ? AND status = 0", fullTitle).
		Count(&count).Error
	return count, err
}

// DeleteComment 删除评论（软删除：设置为隐藏状态）
func DeleteComment(id uint) error {
	return commentDB.Model(&Comment{}).Where("id = ?", id).Update("status", 2).Error
}

// ListAllComments 管理员获取所有评论（分页）
func ListAllComments(pageSize int, pageIndex int) ([]Comment, error) {
	var comments []Comment
	err := commentDB.Order("created_at desc").
		Limit(pageSize).
		Offset(pageSize * pageIndex).
		Find(&comments).Error
	return comments, err
}

// CountAllComments 管理员获取评论总数
func CountAllComments() (int64, error) {
	var count int64
	err := commentDB.Model(&Comment{}).Count(&count).Error
	return count, err
}

// GetCommentByID 根据ID获取评论
func GetCommentByID(id uint) (*Comment, error) {
	var comment Comment
	err := commentDB.First(&comment, id).Error
	return &comment, err
}

// CountRecentCommentsByIP 统计指定IP在最近N分钟内的评论数（用于限流）
func CountRecentCommentsByIP(ip string, minutes int) (int64, error) {
	var count int64
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	err := commentDB.Model(&Comment{}).
		Where("ip = ? AND created_at > ?", ip, cutoff).
		Count(&count).Error
	return count, err
}
