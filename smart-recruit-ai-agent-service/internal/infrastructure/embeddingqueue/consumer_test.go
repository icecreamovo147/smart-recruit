package embeddingqueue

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type processorStub struct {
	versionIDs []int64
	err        error
}

func (p *processorStub) UpsertAgentSkillVersion(_ context.Context, versionID int64) error {
	p.versionIDs = append(p.versionIDs, versionID)
	return p.err
}

func TestConsumerProcessesAgentSkillVersionOnce(t *testing.T) {
	store, db := newConsumerTestStore(t)
	processor := &processorStub{}
	consumer := NewConsumer(nil, store, processor, Options{})
	body := []byte(`{
		"event_id":"evt-101",
		"event_type":"embedding.upsert",
		"aggregate_type":"agent_skill_version",
		"aggregate_id":"101",
		"idempotency_key":"agent_skill_version:101:embedding.upsert:hash",
		"payload":{"object_type":"agent_skill_version","object_id":101}
	}`)

	if err := consumer.handleMessage(context.Background(), body); err != nil {
		t.Fatalf("first delivery returned error: %v", err)
	}
	if err := consumer.handleMessage(context.Background(), body); err != nil {
		t.Fatalf("duplicate delivery returned error: %v", err)
	}
	if len(processor.versionIDs) != 1 || processor.versionIDs[0] != 101 {
		t.Fatalf("processed version ids = %#v, want [101]", processor.versionIDs)
	}

	var inbox inboxRecord
	if err := db.First(&inbox).Error; err != nil {
		t.Fatalf("load inbox: %v", err)
	}
	if inbox.Status != inboxStatusProcessed || inbox.AttemptCount != 1 {
		t.Fatalf("inbox = %#v, want processed single attempt", inbox)
	}
}

func TestConsumerReclaimsFailedDelivery(t *testing.T) {
	store, db := newConsumerTestStore(t)
	processor := &processorStub{err: errors.New("provider unavailable")}
	consumer := NewConsumer(nil, store, processor, Options{})
	body := []byte(`{
		"event_id":"evt-202",
		"event_type":"embedding.upsert",
		"object_type":"agent_skill_version",
		"object_id":202
	}`)

	if err := consumer.handleMessage(context.Background(), body); err == nil {
		t.Fatal("first delivery succeeded, want provider error")
	}
	processor.err = nil
	if err := consumer.handleMessage(context.Background(), body); err != nil {
		t.Fatalf("retry delivery returned error: %v", err)
	}
	if len(processor.versionIDs) != 2 {
		t.Fatalf("processor calls = %#v, want two attempts", processor.versionIDs)
	}

	var inbox inboxRecord
	if err := db.First(&inbox).Error; err != nil {
		t.Fatalf("load inbox: %v", err)
	}
	if inbox.Status != inboxStatusProcessed || inbox.AttemptCount != 2 {
		t.Fatalf("inbox = %#v, want processed after two attempts", inbox)
	}
}

func TestDecodeMessageRejectsUnsupportedObjectType(t *testing.T) {
	processor := &processorStub{}
	consumer := NewConsumer(nil, &memoryInboxStore{}, processor, Options{})
	err := consumer.handleMessage(context.Background(), []byte(`{
		"event_id":"evt-memory",
		"event_type":"embedding.upsert",
		"payload":{"object_type":"ai_memory","object_id":9}
	}`))
	if err == nil {
		t.Fatal("unsupported object type was accepted")
	}
	if len(processor.versionIDs) != 0 {
		t.Fatalf("processor calls = %#v, want none", processor.versionIDs)
	}
}

func newConsumerTestStore(t *testing.T) (*Store, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&inboxRecord{}); err != nil {
		t.Fatalf("migrate inbox: %v", err)
	}
	return NewStore(db), db
}

type memoryInboxStore struct{}

func (*memoryInboxStore) ClaimInbox(context.Context, string, string, string, string) (uint64, bool, error) {
	return 1, true, nil
}

func (*memoryInboxStore) MarkInboxProcessed(context.Context, uint64) error { return nil }
func (*memoryInboxStore) MarkInboxFailed(context.Context, uint64, string) error {
	return nil
}
