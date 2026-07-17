package parser

import (
	"strings"
	"testing"
	"time"
)

func TestParseZapANSIJSONAndFields(t *testing.T) {
	observed := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	lines := []string{
		"\x1b[32m2026-07-15T09:00:00.123Z\tINFO\tservice/main.go:42\tstarted {\"request_id\":\"req-1\",\"trace_id\":\"trace-1\",\"span_id\":\"span-1\",\"grpc_code\":\"OK\",\"elapsed\":\"2ms\"}\x1b[0m",
	}

	records := New().ParseLines("identity-service", 7, lines, observed)
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	record := records[0]
	if record.Generation != 7 || record.Service != "identity-service" {
		t.Fatalf("unexpected identity: %+v", record)
	}
	if record.Level != "INFO" || record.Caller != "service/main.go:42" {
		t.Fatalf("unexpected parsed metadata: %+v", record)
	}
	if strings.Contains(record.Message, "\x1b") {
		t.Fatalf("ANSI was not stripped: %q", record.Message)
	}
	if record.RequestID != "req-1" || record.TraceID != "trace-1" || record.SpanID != "span-1" {
		t.Fatalf("correlation fields not extracted: %+v", record)
	}
	if record.Fields["grpc_code"] != "OK" || record.Fields["elapsed"] != "2ms" {
		t.Fatalf("allowed scalar fields not extracted: %+v", record.Fields)
	}
	if record.SourceTime == nil {
		t.Fatalf("source time not parsed")
	}
}

func TestParseZapConsoleTimeAndColoredLevel(t *testing.T) {
	observed := time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC)
	lines := []string{
		"2026-07-15 21:36:40\t\x1b[34mINFO\x1b[0m\tlogger/interceptor.go:62\tgrpc\t{\"request_id\":\"req-2\",\"trace_id\":\"trace-2\",\"span_id\":\"span-2\",\"grpc_code\":\"OK\",\"elapsed\":\"958.135µs\"}",
	}

	records := New().ParseLines("identity-service", 1, lines, observed)
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	record := records[0]
	if record.Level != "INFO" || record.Caller != "logger/interceptor.go:62" || record.Message == "" {
		t.Fatalf("console zap line was not parsed: %+v", record)
	}
	if record.SourceTime == nil {
		t.Fatalf("console source time was not parsed")
	}
	if strings.Contains(record.Lines[0], "\x1b") {
		t.Fatalf("ANSI was not stripped: %q", record.Lines[0])
	}
	if record.RequestID != "req-2" || record.TraceID != "trace-2" || record.SpanID != "span-2" {
		t.Fatalf("correlation fields not extracted: %+v", record)
	}
}

func TestParseMultilineStackAndUnknownFallback(t *testing.T) {
	lines := []string{
		"2026-07-15T09:00:00.123Z\tERROR\tservice/handler.go:12\tpanic",
		"    at handler",
		"/app/service/handler.go:12 +0x1",
		"plain vite output",
		"[GIN] 2026/07/15 - 09:00:00 | 200 | 1ms | 127.0.0.1 | GET \"/healthz\"",
	}

	records := New().ParseLines("gateway", 0, lines, time.Now())
	if len(records) != 3 {
		t.Fatalf("records = %#v", records)
	}
	if len(records[0].Lines) != 3 || !strings.Contains(records[0].Message, "handler.go") {
		t.Fatalf("stack lines were not preserved: %+v", records[0])
	}
	if records[1].Level != LevelUnknown || records[1].Message != "plain vite output" {
		t.Fatalf("unknown fallback changed semantics: %+v", records[1])
	}
	if records[2].Level != "INFO" || !strings.HasPrefix(records[2].Message, "[GIN]") {
		t.Fatalf("gin line not preserved: %+v", records[2])
	}
}

func TestParserBoundsLongRecords(t *testing.T) {
	parser := Parser{MaxRecordBytes: 12, MaxRecordLines: 2}
	records := parser.ParseLines("svc", 0, []string{
		"2026-07-15T09:00:00.123Z\tINFO\tsvc/main.go:1\t" + strings.Repeat("x", 100),
		" continuation",
		" another",
	}, time.Now())
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	if !records[0].Truncated {
		t.Fatalf("record was not marked truncated: %+v", records[0])
	}
	if len(records[0].Message) > 12 || len(records[0].Lines) > 2 {
		t.Fatalf("record bounds not enforced: %+v", records[0])
	}
}
