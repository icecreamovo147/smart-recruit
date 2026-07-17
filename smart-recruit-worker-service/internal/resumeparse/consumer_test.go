package resumeparse

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type memoryStore struct {
	claimed       map[string]bool
	processed     map[uint64]bool
	failed        map[uint64]string
	parsed        map[int64]string
	hasParsed     map[int64]bool
	nextID        uint64
	claimErr      error
	updateErr     error
	claimRejected bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		claimed:   map[string]bool{},
		processed: map[uint64]bool{},
		failed:    map[uint64]string{},
		parsed:    map[int64]string{},
		hasParsed: map[int64]bool{},
		nextID:    1,
	}
}

func (s *memoryStore) ClaimInbox(_ context.Context, eventID, _, _, _ string) (uint64, bool, error) {
	if s.claimErr != nil {
		return 0, false, s.claimErr
	}
	if s.claimRejected {
		return 0, false, nil
	}
	if s.claimed[eventID] {
		return 0, false, nil
	}
	s.claimed[eventID] = true
	id := s.nextID
	s.nextID++
	return id, true, nil
}

func (s *memoryStore) MarkInboxProcessed(_ context.Context, id uint64) error {
	s.processed[id] = true
	return nil
}

func (s *memoryStore) MarkInboxFailed(_ context.Context, id uint64, errMsg string) error {
	s.failed[id] = errMsg
	return nil
}

func (s *memoryStore) UpdateParsedText(_ context.Context, resumeID int64, text string) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.parsed[resumeID] = text
	s.hasParsed[resumeID] = true
	return nil
}

func (s *memoryStore) HasParsedText(_ context.Context, resumeID int64) (bool, error) {
	return s.hasParsed[resumeID], nil
}

type fakeDownloader struct {
	data map[string][]byte
	err  error
}

func (d *fakeDownloader) DownloadObject(_ context.Context, ossKey string) ([]byte, error) {
	if d.err != nil {
		return nil, d.err
	}
	data, ok := d.data[ossKey]
	if !ok {
		return nil, errors.New("object not found")
	}
	return data, nil
}

type fakeParser struct {
	text string
	err  error
}

func (p *fakeParser) ExtractText(context.Context, []byte) (string, error) {
	return p.text, p.err
}

func (p *fakeParser) SupportedExtensions() []string { return []string{"pdf"} }

func TestHandleMessageSkipsWhenAlreadyParsed(t *testing.T) {
	store := newMemoryStore()
	store.hasParsed[42] = true
	body, _ := json.Marshal(Payload{
		EventID:   "evt-1",
		EventType: "resume.parse",
		ResumeID:  42,
		FileType:  "pdf",
		OSSKey:    "resumes/1/a.pdf",
	})
	c := NewConsumer(nil, store, &fakeDownloader{data: map[string][]byte{"resumes/1/a.pdf": []byte("%PDF-1.4 fake")}}, Options{})
	if err := c.handleMessage(context.Background(), body); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
	if len(store.parsed) != 0 {
		t.Fatalf("should not re-parse when text exists")
	}
	if !store.processed[1] {
		t.Fatalf("expected inbox processed")
	}
}

func TestHandleMessageDownloadFailureMarksFailed(t *testing.T) {
	store := newMemoryStore()
	c := NewConsumer(nil, store, &fakeDownloader{err: errors.New("oss down")}, Options{})
	body, _ := json.Marshal(Payload{EventID: "evt-2", ResumeID: 7, FileType: "pdf", OSSKey: "k"})
	err := c.handleMessage(context.Background(), body)
	if err == nil || !strings.Contains(err.Error(), "download") {
		t.Fatalf("error = %v", err)
	}
	if _, ok := store.failed[1]; !ok {
		t.Fatalf("expected inbox failed")
	}
}

func TestHandleMessageSkipsDuplicateClaim(t *testing.T) {
	store := newMemoryStore()
	store.claimRejected = true
	c := NewConsumer(nil, store, &fakeDownloader{}, Options{})
	body, _ := json.Marshal(Payload{EventID: "evt-3", ResumeID: 7, FileType: "pdf", OSSKey: "k"})
	if err := c.handleMessage(context.Background(), body); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
}

func TestDecodePayloadFlexibleResumeID(t *testing.T) {
	body := []byte(`{"resume_id": 99.0, "file_type": "docx", "oss_key": "a/b.docx", "event_id": "e"}`)
	p, err := decodePayload(body)
	if err != nil {
		t.Fatal(err)
	}
	if p.ResumeID != 99 || p.FileType != "docx" || p.OSSKey != "a/b.docx" {
		t.Fatalf("payload = %+v", p)
	}
}

func TestDecodePayloadMissingResumeID(t *testing.T) {
	if _, err := decodePayload([]byte(`{"file_type":"pdf","oss_key":"x"}`)); err == nil {
		t.Fatal("expected error")
	}
}
