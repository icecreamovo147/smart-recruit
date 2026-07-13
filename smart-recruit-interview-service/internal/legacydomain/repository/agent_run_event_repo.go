package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-interview-service/internal/legacydomain/model"
)

// AgentRunEventRepo persists ordered durable events for resumable agent runs.
type AgentRunEventRepo struct {
	db *gorm.DB
}

func NewAgentRunEventRepo(db *gorm.DB) *AgentRunEventRepo {
	return &AgentRunEventRepo{db: db}
}

// AppendEvent allocates the next per-run sequence number, inserts the event, and
// advances agent_runs.last_event_seq in the same transaction.
func (r *AgentRunEventRepo) AppendEvent(ctx context.Context, runID uint64, eventType, payloadJSON string) (*model.AgentRunEvent, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("agent run event repo is nil")
	}
	if runID == 0 {
		return nil, fmt.Errorf("run_id is required")
	}
	if eventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}
	if strings.TrimSpace(payloadJSON) == "" {
		payloadJSON = "{}"
	}

	var event model.AgentRunEvent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run model.AgentRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, runID).Error; err != nil {
			return err
		}
		nextSeq := run.LastEventSeq + 1
		event = model.AgentRunEvent{
			RunID:       runID,
			Seq:         nextSeq,
			EventType:   eventType,
			PayloadJSON: payloadJSON,
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return tx.Model(&model.AgentRun{}).Where("id = ?", runID).Update("last_event_seq", nextSeq).Error
	})
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// AppendEventAtSeq inserts an event at an explicit sequence.
// Duplicate (run_id, seq) rows are treated as safe no-ops and return the existing row.
// Stale sequences (seq <= current last_event_seq without an existing row) are rejected.
func (r *AgentRunEventRepo) AppendEventAtSeq(ctx context.Context, runID uint64, seq int64, eventType, payloadJSON string) (*model.AgentRunEvent, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, fmt.Errorf("agent run event repo is nil")
	}
	if runID == 0 {
		return nil, false, fmt.Errorf("run_id is required")
	}
	if seq <= 0 {
		return nil, false, fmt.Errorf("seq must be positive")
	}
	if eventType == "" {
		return nil, false, fmt.Errorf("event_type is required")
	}
	if strings.TrimSpace(payloadJSON) == "" {
		payloadJSON = "{}"
	}

	var (
		event     model.AgentRunEvent
		duplicate bool
	)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.AgentRunEvent
		findErr := tx.Where("run_id = ? AND seq = ?", runID, seq).First(&existing).Error
		if findErr == nil {
			duplicate = true
			event = existing
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		var run model.AgentRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, runID).Error; err != nil {
			return err
		}
		if seq <= run.LastEventSeq {
			return fmt.Errorf("stale event seq %d for run %d (last_event_seq=%d)", seq, runID, run.LastEventSeq)
		}
		if seq != run.LastEventSeq+1 {
			return fmt.Errorf("out-of-order event seq %d for run %d (expected %d)", seq, runID, run.LastEventSeq+1)
		}

		event = model.AgentRunEvent{
			RunID:       runID,
			Seq:         seq,
			EventType:   eventType,
			PayloadJSON: payloadJSON,
		}
		if err := tx.Create(&event).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				dupErr := tx.Where("run_id = ? AND seq = ?", runID, seq).First(&existing).Error
				if dupErr != nil {
					return err
				}
				duplicate = true
				event = existing
				return nil
			}
			return err
		}
		return tx.Model(&model.AgentRun{}).Where("id = ?", runID).Update("last_event_seq", seq).Error
	})
	if err != nil {
		return nil, false, err
	}
	return &event, duplicate, nil
}

// ListEventsAfter returns events for runID with seq > afterSeq, ordered by seq ascending.
func (r *AgentRunEventRepo) ListEventsAfter(ctx context.Context, runID uint64, afterSeq int64) ([]model.AgentRunEvent, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("agent run event repo is nil")
	}
	var rows []model.AgentRunEvent
	err := r.db.WithContext(ctx).
		Where("run_id = ? AND seq > ?", runID, afterSeq).
		Order("seq ASC").
		Find(&rows).Error
	return rows, err
}

// GetEventBySeq loads a single event by run_id and seq.
func (r *AgentRunEventRepo) GetEventBySeq(ctx context.Context, runID uint64, seq int64) (*model.AgentRunEvent, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("agent run event repo is nil")
	}
	var event model.AgentRunEvent
	err := r.db.WithContext(ctx).Where("run_id = ? AND seq = ?", runID, seq).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}
