package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRegularFile(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "a.png"), []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("no"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path    string
		allowed bool
	}{{"nested/a.png", true}, {"../secret", false}, {"escape", false}, {"nested", false}} {
		_, ok := resolveRegularFile(root, tc.path)
		if ok != tc.allowed {
			t.Errorf("%s: got %v", tc.path, ok)
		}
	}
}
