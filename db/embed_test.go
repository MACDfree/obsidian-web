package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestResolveNoteLinkAccess(t *testing.T) {
	previous := db
	memory, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db = memory
	defer func() { db = previous }()
	if err := db.AutoMigrate(&Note{}); err != nil {
		t.Fatal(err)
	}
	for _, n := range []Note{
		{FullTitle: "文件夹/源", Title: "源", Publish: true},
		{FullTitle: "文件夹/公开", Title: "公开", Publish: true, Aliases: []string{"别名"}},
		{FullTitle: "文件夹/私有", Title: "私有", Publish: false},
	} {
		if err := db.Create(&n).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		target       string
		login, found bool
	}{
		{"公开", false, true}, {"别名", false, true}, {"私有", false, false}, {"私有", true, true}, {"", false, true},
	} {
		_, err := ResolveNoteLink("文件夹/源", tc.target, tc.login)
		if (err == nil) != tc.found {
			t.Errorf("target=%s login=%v: %v", tc.target, tc.login, err)
		}
	}
}

func TestAttachmentLookupRequiresUniqueFilename(t *testing.T) {
	previous := db
	memory, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db = memory
	defer func() { db = previous }()
	if err := db.AutoMigrate(&AttachInfo{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a/shared.png", "b/shared.png", "a/unique_1.png"} {
		if err := db.Create(&AttachInfo{AttachName: name}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := GetAttachInfoByName("shared.png"); err == nil {
		t.Fatal("同名附件不应随机选择")
	}
	if _, err := GetAttachInfoByName("a/shared.png"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetAttachInfoByName("unique_1.png"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetAttachInfoByName("unique%1.png"); err == nil {
		t.Fatal("通配符不能匹配其他文件")
	}
}

func TestResolveNoteConsistentAcrossVisibility(t *testing.T) {
	previous := db
	memory, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db = memory
	defer func() { db = previous }()
	if err := db.AutoMigrate(&Note{}); err != nil {
		t.Fatal(err)
	}
	notes := []Note{
		{FullTitle: "工作/源", Title: "源", Publish: true},
		{FullTitle: "工作/计划", Title: "工作计划", Publish: false},
		{FullTitle: "生活/计划", Title: "生活计划", Publish: true},
		{FullTitle: "其他/命名", Title: "不同标题", Aliases: []string{"别名"}, Publish: true},
		{FullTitle: "有歧义/一", Title: "重复", Publish: true},
		{FullTitle: "有歧义/二", Title: "重复", Publish: false},
		{FullTitle: "目录", Title: "目录", Publish: true},
		{FullTitle: "目录/_index", Title: "目录索引", Publish: true},
	}
	for i := range notes {
		if err := db.Create(&notes[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		source, target string
		login          bool
		want           string
	}{
		{"工作/源", "计划", true, "工作/计划"}, {"工作/源", "计划", false, ""},
		{"其他/源", "计划", true, ""}, {"其他/源", "计划", false, ""},
		{"工作/源", "生活/计划", false, "生活/计划"}, {"工作/源", "别名", false, "其他/命名"},
		{"工作/源", "命名.md", false, "其他/命名"}, {"工作/源", "重复", false, ""},
		{"工作/源", "目录", true, "目录"},
	} {
		note, err := ResolveNoteLink(tc.source, tc.target, tc.login)
		if tc.want == "" {
			if err == nil {
				t.Errorf("%s 不应解析为 %s", tc.target, note.FullTitle)
			}
			continue
		}
		if err != nil || note.FullTitle != tc.want {
			t.Errorf("%s: 得到 %v，错误 %v，期望 %s", tc.target, note, err, tc.want)
		}
	}
}

func TestResolvePathPriority(t *testing.T) {
	previous := db
	memory, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db = memory
	defer func() { db = previous }()
	if err := db.AutoMigrate(&Note{}); err != nil {
		t.Fatal(err)
	}
	for _, n := range []Note{{FullTitle: "工作/源", Title: "源", Publish: true}, {FullTitle: "工作/计划", Title: "根路径", Publish: true}, {FullTitle: "工作/工作/计划", Title: "相对路径", Publish: true}, {FullTitle: "公共/私有", Title: "私有", Publish: false}} {
		if err := db.Create(&n).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		target string
		login  bool
		want   string
	}{
		{"工作/计划", false, "工作/计划"}, {"./工作/计划", false, "工作/工作/计划"}, {"../工作/计划", false, "工作/计划"}, {"../../工作/计划", true, ""}, {"公共/私有", false, ""}, {"公共/私有", true, "公共/私有"},
	} {
		n, e := ResolveNoteLink("工作/源", tc.target, tc.login)
		if tc.want == "" {
			if e == nil {
				t.Errorf("%s 不应解析为 %s", tc.target, n.FullTitle)
			}
		} else if e != nil || n.FullTitle != tc.want {
			t.Errorf("%s: %v %v", tc.target, n, e)
		}
	}
}
