package resumeparse

import (
	"context"
	"strings"
	"testing"
)

func TestProcessUnsupportedFormat(t *testing.T) {
	store := newMemoryStore()
	c := NewConsumer(nil, store, &fakeDownloader{data: map[string][]byte{"k": []byte("hello")}}, Options{})
	err := c.process(context.Background(), Payload{ResumeID: 1, FileType: "doc", OSSKey: "k"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unsupported") {
		t.Fatalf("error = %v", err)
	}
}

func TestProcessMagicMismatch(t *testing.T) {
	store := newMemoryStore()
	c := NewConsumer(nil, store, &fakeDownloader{data: map[string][]byte{"k": []byte("%PDF-1.4xxxx")}}, Options{})
	err := c.process(context.Background(), Payload{ResumeID: 1, FileType: "docx", OSSKey: "k"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "magic") {
		t.Fatalf("error = %v", err)
	}
}

func TestProcessAlreadyParsed(t *testing.T) {
	store := newMemoryStore()
	store.hasParsed[9] = true
	c := NewConsumer(nil, store, &fakeDownloader{err: &simpleErr{msg: "should not download"}}, Options{})
	if err := c.process(context.Background(), Payload{ResumeID: 9, FileType: "pdf", OSSKey: "k"}); err != nil {
		t.Fatalf("process: %v", err)
	}
}

func TestProcessMissingFields(t *testing.T) {
	c := NewConsumer(nil, newMemoryStore(), &fakeDownloader{}, Options{})
	if err := c.process(context.Background(), Payload{}); err == nil {
		t.Fatal("expected error")
	}
}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
