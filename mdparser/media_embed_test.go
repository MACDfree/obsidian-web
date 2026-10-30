package mdparser

import (
	"strings"
	"testing"
)

func TestMediaEmbedAndDottedNote(t *testing.T) {
	s := []byte("![[paper.pdf#page=2]] ![[sound.mp3]] ![[movie.mp4]] ![[版本1.0]] [[版本1.0]]")
	for _, math := range []bool{false, true} {
		got, _, err := ConvertWithResolver(s, math, "", func(target string) (string, bool) {
			if target == "版本1.0" {
				return "目录/版本1.0", true
			}
			return "", false
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range []string{`<iframe class="media-embed"`, `/attachment/paper.pdf#page=2`, `<audio class="media-embed"`, `<video class="media-embed"`, `data-target="版本1.0"`, `href="/note/`} {
			if !strings.Contains(got, v) {
				t.Errorf("缺少 %q: %s", v, got)
			}
		}
	}
}
