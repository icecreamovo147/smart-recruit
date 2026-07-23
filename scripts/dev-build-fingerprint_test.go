package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFilesTracksContentAndDeletion(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "service", "main.go")
	if err := os.MkdirAll(filepath.Dir(input), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := map[string]struct{}{input: {}}
	initial := testFingerprint(files, root)

	if err := os.WriteFile(input, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed := testFingerprint(files, root)
	if initial == changed {
		t.Fatal("content change did not change fingerprint")
	}

	if err := os.Remove(input); err != nil {
		t.Fatal(err)
	}
	deleted := testFingerprint(files, root)
	if changed == deleted {
		t.Fatal("file deletion did not change fingerprint")
	}
}

func TestCollectExtraSkipsGeneratedFrontendDirectories(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "web", "src", "App.tsx"),
		filepath.Join(root, "web", "dist", "index.js"),
		filepath.Join(root, "web", "node_modules", "dependency.js"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files := make(map[string]struct{})
	collectExtra(files, filepath.Join(root, "web"))
	if _, ok := files[paths[0]]; !ok {
		t.Fatal("frontend source was not collected")
	}
	if _, ok := files[paths[1]]; ok {
		t.Fatal("generated dist file was collected")
	}
	if _, ok := files[paths[2]]; ok {
		t.Fatal("node_modules file was collected")
	}
}

func testFingerprint(files map[string]struct{}, root string) string {
	h := sha256.New()
	hashFiles(h, files, root)
	return hex.EncodeToString(h.Sum(nil))
}
