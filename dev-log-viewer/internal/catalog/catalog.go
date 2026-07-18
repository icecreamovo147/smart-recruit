package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	ProcessRunning = "running"
	ProcessOffline = "offline"
	ProcessUnknown = "unknown"

	LogReady      = "ready"
	LogMissing    = "missing"
	LogUnreadable = "unreadable"
)

type ServiceDefinition struct {
	ID      string
	Name    string
	Group   string
	Port    int
	LogFile string
	PIDFile string
}

type ServiceStatus struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Group        string     `json:"group"`
	Port         int        `json:"port"`
	ProcessState string     `json:"process_state"`
	LogState     string     `json:"log_state"`
	LogSizeBytes int64      `json:"log_size_bytes"`
	LogUpdatedAt *time.Time `json:"log_updated_at,omitempty"`
}

type Catalog struct {
	root        string
	definitions []ServiceDefinition
}

func New(root string) (*Catalog, error) {
	if root == "" {
		return nil, errors.New("repository root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Catalog{
		root:        filepath.Clean(absolute),
		definitions: DefaultDefinitions(),
	}, nil
}

func DefaultDefinitions() []ServiceDefinition {
	return []ServiceDefinition{
		service("backend", "identity-service", 50061),
		service("backend", "recruitment-service", 50062),
		service("backend", "interview-service", 50063),
		service("backend", "offer-service", 50064),
		service("backend", "notification-service", 50065),
		service("backend", "ai-agent-service", 50066),
		service("backend", "analytics-service", 50067),
		service("backend", "worker-service", 50068),
		service("gateway", "smart-recruit-gateway", 8080),
		service("frontend", "hr-frontend", 5173),
		service("frontend", "user-frontend", 5174),
		service("frontend", "interviewer-frontend", 5175),
	}
}

func (c *Catalog) Definitions() []ServiceDefinition {
	copied := make([]ServiceDefinition, len(c.definitions))
	copy(copied, c.definitions)
	return copied
}

func (c *Catalog) Statuses() []ServiceStatus {
	statuses := make([]ServiceStatus, 0, len(c.definitions))
	for _, definition := range c.definitions {
		status := ServiceStatus{
			ID:           definition.ID,
			Name:         definition.Name,
			Group:        definition.Group,
			Port:         definition.Port,
			ProcessState: c.processState(definition.PIDFile),
		}
		status.LogState, status.LogSizeBytes, status.LogUpdatedAt = c.logState(definition.LogFile)
		statuses = append(statuses, status)
	}
	return statuses
}

func service(group string, id string, port int) ServiceDefinition {
	return ServiceDefinition{
		ID:      id,
		Name:    id,
		Group:   group,
		Port:    port,
		LogFile: filepath.ToSlash(filepath.Join(".dev", "logs", id+".log")),
		PIDFile: filepath.ToSlash(filepath.Join(".dev", "pids", id+".pid")),
	}
}

func (c *Catalog) processState(relativePID string) string {
	content, err := os.ReadFile(c.containedPath(relativePID))
	if err != nil {
		if os.IsNotExist(err) {
			return ProcessOffline
		}
		return ProcessUnknown
	}
	pidText := strings.TrimSpace(string(content))
	if pidText == "" {
		return ProcessUnknown
	}
	pid, err := strconv.Atoi(pidText)
	if err != nil || pid <= 0 {
		return ProcessUnknown
	}
	return probeProcess(pid)
}

func (c *Catalog) logState(relativeLog string) (string, int64, *time.Time) {
	path := c.containedPath(relativeLog)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LogMissing, 0, nil
		}
		return LogUnreadable, 0, nil
	}
	if info.IsDir() {
		return LogUnreadable, 0, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return LogUnreadable, 0, nil
	}
	_ = file.Close()
	updated := info.ModTime().UTC()
	return LogReady, info.Size(), &updated
}

func (c *Catalog) containedPath(relative string) string {
	clean := filepath.Clean(relative)
	return filepath.Join(c.root, clean)
}

func probeProcess(pid int) string {
	err := syscall.Kill(pid, 0)
	switch {
	case err == nil:
		return ProcessRunning
	case errors.Is(err, syscall.ESRCH):
		return ProcessOffline
	case errors.Is(err, syscall.EPERM):
		return ProcessUnknown
	default:
		return ProcessUnknown
	}
}
