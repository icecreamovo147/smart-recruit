package parser

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	LevelUnknown          = "UNKNOWN"
	DefaultMaxRecordBytes = 32 * 1024
	DefaultMaxRecordLines = 200
)

type LogRecord struct {
	ID         uint64            `json:"id"`
	Generation uint64            `json:"generation"`
	Service    string            `json:"service"`
	ObservedAt time.Time         `json:"observed_at"`
	SourceTime *time.Time        `json:"source_time,omitempty"`
	Level      string            `json:"level"`
	Caller     string            `json:"caller,omitempty"`
	Message    string            `json:"message"`
	Lines      []string          `json:"lines"`
	Fields     map[string]string `json:"fields,omitempty"`
	RequestID  string            `json:"request_id,omitempty"`
	TraceID    string            `json:"trace_id,omitempty"`
	SpanID     string            `json:"span_id,omitempty"`
	Truncated  bool              `json:"truncated,omitempty"`
}

type Parser struct {
	MaxRecordBytes int
	MaxRecordLines int
}

func New() Parser {
	return Parser{
		MaxRecordBytes: DefaultMaxRecordBytes,
		MaxRecordLines: DefaultMaxRecordLines,
	}
}

func (p Parser) ParseLines(service string, generation uint64, lines []string, observedAt time.Time) []LogRecord {
	if p.MaxRecordBytes <= 0 {
		p.MaxRecordBytes = DefaultMaxRecordBytes
	}
	if p.MaxRecordLines <= 0 {
		p.MaxRecordLines = DefaultMaxRecordLines
	}

	records := []LogRecord{}
	var current *LogRecord
	flush := func() {
		if current == nil {
			return
		}
		finalize(current, p.MaxRecordBytes, p.MaxRecordLines)
		records = append(records, *current)
		current = nil
	}

	for _, rawLine := range lines {
		line := stripANSI(strings.ToValidUTF8(rawLine, "\uFFFD"))
		start := parseStart(service, generation, line, observedAt)
		if start != nil {
			flush()
			current = start
			continue
		}
		if current != nil && isContinuation(line) {
			current.Lines = append(current.Lines, line)
			current.Message += "\n" + line
			continue
		}
		flush()
		current = &LogRecord{
			Service:    service,
			Generation: generation,
			ObservedAt: observedAt,
			Level:      LevelUnknown,
			Message:    line,
			Lines:      []string{line},
		}
	}
	flush()
	return records
}

func parseStart(service string, generation uint64, line string, observedAt time.Time) *LogRecord {
	if record := parseZap(service, generation, line, observedAt); record != nil {
		return record
	}
	if strings.HasPrefix(line, "[GIN]") {
		return &LogRecord{
			Service:    service,
			Generation: generation,
			ObservedAt: observedAt,
			Level:      "INFO",
			Message:    line,
			Lines:      []string{line},
		}
	}
	if strings.Contains(line, " VITE ") || strings.HasPrefix(strings.TrimSpace(line), "VITE ") {
		return &LogRecord{
			Service:    service,
			Generation: generation,
			ObservedAt: observedAt,
			Level:      "INFO",
			Message:    strings.TrimSpace(line),
			Lines:      []string{line},
		}
	}
	return nil
}

func parseZap(service string, generation uint64, line string, observedAt time.Time) *LogRecord {
	parts := strings.Split(line, "\t")
	if len(parts) < 4 {
		return nil
	}
	sourceTime, ok := parseTime(parts[0])
	if !ok || !isLevel(parts[1]) || !strings.Contains(parts[2], ".go:") {
		return nil
	}
	message := strings.Join(parts[3:], "\t")
	record := &LogRecord{
		Service:    service,
		Generation: generation,
		ObservedAt: observedAt,
		SourceTime: &sourceTime,
		Level:      strings.ToUpper(parts[1]),
		Caller:     parts[2],
		Message:    message,
		Lines:      []string{line},
	}
	applyJSONFields(record)
	return record
}

func finalize(record *LogRecord, maxBytes int, maxLines int) {
	applyJSONFields(record)
	if len(record.Lines) > maxLines {
		record.Lines = record.Lines[:maxLines]
		record.Truncated = true
	}
	if len(record.Message) > maxBytes {
		record.Message = trimUTF8(record.Message, maxBytes)
		record.Truncated = true
	}
}

func applyJSONFields(record *LogRecord) {
	start := strings.LastIndex(record.Message, "{")
	end := strings.LastIndex(record.Message, "}")
	if start < 0 || end <= start || end != len(record.Message)-1 {
		return
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(record.Message[start:end+1]), &values); err != nil {
		return
	}
	allowed := map[string]bool{
		"request_id": true,
		"trace_id":   true,
		"span_id":    true,
		"grpc_code":  true,
		"elapsed":    true,
	}
	for key, value := range values {
		if !allowed[key] {
			continue
		}
		scalar, ok := scalarString(value)
		if !ok {
			continue
		}
		if record.Fields == nil {
			record.Fields = map[string]string{}
		}
		record.Fields[key] = scalar
		switch key {
		case "request_id":
			record.RequestID = scalar
		case "trace_id":
			record.TraceID = scalar
		case "span_id":
			record.SpanID = scalar
		}
	}
}

func scalarString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case float64, bool:
		return strings.TrimSpace(strings.ReplaceAll(toJSON(typed), "\"", "")), true
	default:
		return "", false
	}
}

func toJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseTime(value string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z0700",
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05Z0700",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func isLevel(value string) bool {
	switch strings.ToUpper(value) {
	case "DEBUG", "INFO", "WARN", "ERROR", "DPANIC", "PANIC", "FATAL":
		return true
	default:
		return false
	}
}

func isContinuation(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(line, " ") ||
		strings.HasPrefix(line, "\t") ||
		strings.HasPrefix(trimmed, "goroutine ") ||
		strings.Contains(trimmed, ".go:")
}

func trimUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.ValidString(value[:limit]) {
		limit--
	}
	return value[:limit]
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07]*(\x07|\x1b\\)`)

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}
