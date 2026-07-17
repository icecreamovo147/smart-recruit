package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type aiSessionSummaryRecord struct {
	ID               uint64    `gorm:"primaryKey"`
	SessionID        uint64    `gorm:"column:session_id;uniqueIndex:uk_session_id"`
	HrID             uint64    `gorm:"column:hr_id"` // owner id (HR or candidate user)
	Summary          string    `gorm:"column:summary"`
	CoveredMessageID uint64    `gorm:"column:covered_message_id"`
	MessageCount     int       `gorm:"column:message_count"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (aiSessionSummaryRecord) TableName() string { return "ai_session_summaries" }

// GetSessionSummary returns the rolling summary for a session owned by ownerID.
// The hr_id column stores the chat owner id (candidate user id for candidate chats).
func (s *NativeStore) GetSessionSummary(ctx context.Context, ownerID, sessionID int64) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, gorm.ErrInvalidDB
	}
	var row aiSessionSummaryRecord
	err := s.db.WithContext(ctx).
		Where("session_id = ? AND hr_id = ?", sessionID, ownerID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.Summary, true, nil
}

// UpsertSessionSummary inserts or updates the rolling summary for a session.
func (s *NativeStore) UpsertSessionSummary(ctx context.Context, ownerID, sessionID int64, summary string, coveredMessageID int64, messageCount int) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	now := time.Now()
	row := aiSessionSummaryRecord{
		SessionID:        uint64(sessionID),
		HrID:             uint64(ownerID),
		Summary:          summary,
		CoveredMessageID: uint64(coveredMessageID),
		MessageCount:     messageCount,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"hr_id":              row.HrID,
			"summary":            row.Summary,
			"covered_message_id": row.CoveredMessageID,
			"message_count":      row.MessageCount,
			"updated_at":         now,
		}),
	}).Create(&row).Error
}
