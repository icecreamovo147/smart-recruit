package application

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"logic-grpc-service/repository"
)

func TestCurrentAnalyticsRepositorySatisfiesReadModelPort(t *testing.T) {
	t.Parallel()

	var _ ReportingReadModelRepository = (*repository.AnalyticsRepo)(nil)
}

func TestAnalyticsBoundaryDoesNotIntroduceServiceReadDependencies(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate analytics boundary test")
	}
	analyticsRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
	err := filepath.WalkDir(analyticsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.Contains(filepath.Base(path), "service_read") {
			t.Fatalf("analytics boundary introduced forbidden service_read file: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "\"logic-grpc-service/service\"") {
			t.Fatalf("analytics boundary production file imports source service package: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
