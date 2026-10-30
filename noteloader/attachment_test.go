package noteloader

import "testing"

func TestAttachmentRefsWithFragment(t *testing.T) {
	refs := extractAttachmentRefs([]byte("![[dir/paper.pdf#page=2]] ![[img.png|300]]"))
	seen := map[string]bool{}
	for _, r := range refs {
		seen[r] = true
	}
	if !seen["dir/paper.pdf"] || !seen["img.png"] {
		t.Fatalf("引用提取错误: %v", refs)
	}
}
