package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoundedContextSkeletonsExist(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	internalRoot := filepath.Clean(filepath.Join(cwd, ".."))

	contexts := []string{
		"identity",
		"recruitment",
		"interview",
		"offer",
		"notification",
		"aiagent",
		"analytics",
		filepath.Join("platform", "workers"),
	}
	layers := []string{"domain", "application", "infrastructure", "interfaces"}

	for _, contextPath := range contexts {
		for _, layer := range layers {
			docPath := filepath.Join(internalRoot, contextPath, layer, "doc.go")
			if _, err := os.Stat(docPath); err != nil {
				t.Fatalf("missing %s layer skeleton for %s: %v", layer, contextPath, err)
			}
		}
	}
}
