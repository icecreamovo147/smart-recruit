package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBackendBoundaryEnforcementScript(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
	scriptPath := filepath.Join(repoRoot, "scripts", "check-backend-boundaries.mjs")

	cmd := exec.Command("node", scriptPath, repoRoot)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("backend boundary enforcement failed: %v\n%s", err, string(output))
	}
}
