package pb

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGeneratedProtoCopiesMatchLogicService(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	for _, name := range []string{"recruitment.pb.go", "recruitment_grpc.pb.go"} {
		webPath := filepath.Join(repoRoot, "web-gin-service", "recruitment", "pb", name)
		logicPath := filepath.Join(repoRoot, "logic-grpc-service", "recruitment", "pb", name)
		webBytes, err := os.ReadFile(webPath)
		if err != nil {
			t.Fatalf("read web generated proto %s: %v", name, err)
		}
		logicBytes, err := os.ReadFile(logicPath)
		if err != nil {
			t.Fatalf("read logic generated proto %s: %v", name, err)
		}
		if !bytes.Equal(webBytes, logicBytes) {
			t.Fatalf("generated proto copy differs between web-gin-service and logic-grpc-service: %s", name)
		}
	}
}
