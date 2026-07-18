package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestDefaultDefinitions(t *testing.T) {
	definitions := DefaultDefinitions()
	if len(definitions) != 12 {
		t.Fatalf("definition count = %d, want 12", len(definitions))
	}

	want := map[string]struct {
		group string
		port  int
	}{
		"identity-service":      {"backend", 50061},
		"recruitment-service":   {"backend", 50062},
		"interview-service":     {"backend", 50063},
		"offer-service":         {"backend", 50064},
		"notification-service":  {"backend", 50065},
		"ai-agent-service":      {"backend", 50066},
		"analytics-service":     {"backend", 50067},
		"worker-service":        {"backend", 50068},
		"smart-recruit-gateway": {"gateway", 8080},
		"hr-frontend":           {"frontend", 5173},
		"user-frontend":         {"frontend", 5174},
		"interviewer-frontend":  {"frontend", 5175},
	}
	seen := map[string]bool{}
	for _, definition := range definitions {
		if definition.ID == "" || definition.Name != definition.ID {
			t.Fatalf("invalid definition identity: %+v", definition)
		}
		expected, ok := want[definition.ID]
		if !ok {
			t.Fatalf("unexpected service id %q", definition.ID)
		}
		if definition.Group != expected.group || definition.Port != expected.port {
			t.Fatalf("definition %+v, want group=%s port=%d", definition, expected.group, expected.port)
		}
		if seen[definition.ID] {
			t.Fatalf("duplicate service id %q", definition.ID)
		}
		seen[definition.ID] = true
		if filepath.IsAbs(definition.LogFile) || filepath.IsAbs(definition.PIDFile) {
			t.Fatalf("definition exposes absolute path: %+v", definition)
		}
	}

	if !seen["smart-recruit-gateway"] || !seen["hr-frontend"] || !seen["worker-service"] {
		t.Fatalf("expected gateway, frontend, and backend service ids in catalog")
	}
}

func TestStatusesUseFixedCatalogAndDoNotExposePaths(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root)
	writeFile(t, root, ".dev/pids/identity-service.pid", strconv.Itoa(os.Getpid()))
	writeFile(t, root, ".dev/logs/identity-service.log", "ready\n")
	writeFile(t, root, ".dev/pids/recruitment-service.pid", "99999999")
	writeFile(t, root, ".dev/pids/interview-service.pid", "")
	writeFile(t, root, ".dev/pids/offer-service.pid", "not-a-number")
	mkdir(t, root, ".dev/pids/notification-service.pid")
	mkdir(t, root, ".dev/logs/offer-service.log")

	catalog, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	statuses := catalog.Statuses()
	if len(statuses) != 12 {
		t.Fatalf("status count = %d, want 12", len(statuses))
	}
	byID := map[string]ServiceStatus{}
	for _, status := range statuses {
		byID[status.ID] = status
	}

	if got := byID["identity-service"].ProcessState; got != ProcessRunning {
		t.Fatalf("identity process state = %q, want %q", got, ProcessRunning)
	}
	if got := byID["identity-service"].LogState; got != LogReady {
		t.Fatalf("identity log state = %q, want %q", got, LogReady)
	}
	if byID["identity-service"].LogSizeBytes == 0 || byID["identity-service"].LogUpdatedAt == nil {
		t.Fatalf("identity log metadata was not populated: %+v", byID["identity-service"])
	}
	if got := byID["recruitment-service"].ProcessState; got != ProcessOffline {
		t.Fatalf("stale pid process state = %q, want %q", got, ProcessOffline)
	}
	if got := byID["interview-service"].ProcessState; got != ProcessUnknown {
		t.Fatalf("empty pid process state = %q, want %q", got, ProcessUnknown)
	}
	if got := byID["offer-service"].ProcessState; got != ProcessUnknown {
		t.Fatalf("invalid pid process state = %q, want %q", got, ProcessUnknown)
	}
	if got := byID["notification-service"].ProcessState; got != ProcessUnknown {
		t.Fatalf("pid read error process state = %q, want %q", got, ProcessUnknown)
	}
	if got := byID["offer-service"].LogState; got != LogUnreadable {
		t.Fatalf("directory log state = %q, want %q", got, LogUnreadable)
	}
	if got := byID["worker-service"].LogState; got != LogMissing {
		t.Fatalf("missing log state = %q, want %q", got, LogMissing)
	}

	payload, err := json.Marshal(statuses)
	if err != nil {
		t.Fatal(err)
	}
	if containsString(string(payload), root) || containsString(string(payload), ".dev/logs") || containsString(string(payload), ".dev/pids") {
		t.Fatalf("status payload leaked path details: %s", payload)
	}
}

func mkdirs(t *testing.T, root string) {
	t.Helper()
	mkdir(t, root, ".dev/logs")
	mkdir(t, root, ".dev/pids")
}

func mkdir(t *testing.T, root string, relative string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(relative)), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, root string, relative string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsString(haystack string, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && filepath.ToSlash(haystack) != "" && stringContains(haystack, needle)
}

func stringContains(haystack string, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
