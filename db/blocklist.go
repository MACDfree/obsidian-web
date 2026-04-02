package db

import "strings"

// blocklist 敏感词列表（涉黄、涉政、涉赌、诈骗等常见违规内容）
var blocklist = []string{
	// 涉黄
	"色情", "裸聊", "约炮", "卖淫", "嫖娼", "乱伦", "强奸", "迷药",
	"春药", "伟哥", "成人电影", "毛片", "性爱", "情色",
	// 涉赌
	"赌博", "博彩", "时时彩", "六合彩", "赌场", "百家乐", "老虎机",
	"彩票代理", "下注", "赌球",
	// 涉政/敏感
	"法轮功", "法轮大法", "台独", "藏独", "疆独", "六四", "天安门事件",
	"推翻政府", "反党", "反华",
	// 诈骗/垃圾
	"代开发票", "办证", "贷款加微信", "刷单", "兼职日结",
	"日赚千元", "月入百万", "免费领红包", "扫码领红包",
	// 违禁品
	"枪支", "弹药", "毒品", "大麻", "海洛因", "冰毒", "假钞",
}

// IsContentBlocked 检查内容是否包含敏感词
func IsContentBlocked(content string) bool {
	lower := strings.ToLower(content)
	for _, word := range blocklist {
		if strings.Contains(lower, strings.ToLower(word)) {
			return true
		}
	}
	return false
}
