package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type aiSessionSummaryRecord struct {
	ID               uint64    `gorm:"primaryKey"`
	TenantID         *int64    `gorm:"column:tenant_id"`
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
	summary, _, _, found, err := s.GetSessionSummaryState(ctx, ownerID, sessionID)
	return summary, found, err
}

// GetSessionSummaryState returns the summary and its monotonic coverage cursor.
func (s *NativeStore) GetSessionSummaryState(ctx context.Context, ownerID, sessionID int64) (string, int64, int, bool, error) {
	if s == nil || s.db == nil {
		return "", 0, 0, false, gorm.ErrInvalidDB
	}
	var row aiSessionSummaryRecord
	err := s.db.WithContext(ctx).
		Where("session_id = ? AND hr_id = ?", sessionID, ownerID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, 0, false, nil
	}
	if err != nil {
		return "", 0, 0, false, err
	}
	return row.Summary, int64(row.CoveredMessageID), row.MessageCount, true, nil
}

// UpsertSessionSummary inserts or updates the rolling summary for a session.
func (s *NativeStore) UpsertSessionSummary(ctx context.Context, ownerID, sessionID int64, summary string, coveredMessageID int64, messageCount int) error {
	_, err := s.UpsertSessionSummaryIfNewer(ctx, ownerID, sessionID, summary, coveredMessageID, messageCount)
	return err
}

// UpsertSessionSummaryIfNewer prevents a stale asynchronous summary from
// replacing a summary that already covers a newer message.
func (s *NativeStore) UpsertSessionSummaryIfNewer(ctx context.Context, ownerID, sessionID int64, summary string, coveredMessageID int64, messageCount int) (bool, error) {
	if s == nil || s.db == nil {
		return false, gorm.ErrInvalidDB
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
	updates := map[string]any{
		"hr_id":              row.HrID,
		"summary":            row.Summary,
		"covered_message_id": row.CoveredMessageID,
		"message_count":      row.MessageCount,
		"updated_at":         now,
	}
	result := s.db.WithContext(ctx).Model(&aiSessionSummaryRecord{}).
		Where("session_id = ? AND covered_message_id < ?", sessionID, coveredMessageID).
		Updates(updates)
	if result.Error != nil || result.RowsAffected > 0 {
		return result.RowsAffected > 0, result.Error
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&aiSessionSummaryRecord{}).Where("session_id = ?", sessionID).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		// A concurrent insert may have won. Retry the conditional monotonic update.
		result = s.db.WithContext(ctx).Model(&aiSessionSummaryRecord{}).
			Where("session_id = ? AND covered_message_id < ?", sessionID, coveredMessageID).
			Updates(updates)
		return result.RowsAffected > 0, result.Error
	}
	return true, nil
}
