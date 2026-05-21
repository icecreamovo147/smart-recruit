package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type ChatRepo struct {
	db *gorm.DB
}

func NewChatRepo(db *gorm.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

func (r *ChatRepo) Add(ctx context.Context, history *model.AIChatHistory) error {
	ensureHROwner(history)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if history.SessionID > 0 {
			// Verify session ownership via COUNT to avoid MySQL
			// "RowsAffected = 0 when value unchanged" behavior.
			var cnt int64
			if err := tx.Model(&model.AIChatSession{}).
				Where("id = ? AND hr_id = ? AND deleted_at IS NULL", history.SessionID, history.HrID).
				Count(&cnt).Error; err != nil {
				return err
			}
			if cnt == 0 {
				return gorm.ErrRecordNotFound
			}
			// Touch updated_at (best-effort; ignore rows affected).
			tx.Model(&model.AIChatSession{}).
				Where("id = ? AND hr_id = ? AND deleted_at IS NULL", history.SessionID, history.HrID).
				Update("updated_at", time.Now())
		}
		return tx.Create(history).Error
	})
}

func (r *ChatRepo) List(ctx context.Context, hrID int64, page, pageSize int32) ([]model.AIChatHistory, error) {
	var rows []model.AIChatHistory
	err := r.db.WithContext(ctx).Where("hr_id = ?", hrID).
		Where("session_id = 0 OR EXISTS (SELECT 1 FROM ai_chat_sessions s WHERE s.id = ai_chat_history.session_id AND s.hr_id = ? AND s.deleted_at IS NULL)", hrID).
		Order("created_at ASC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Find(&rows).Error
	return rows, err
}

func (r *ChatRepo) CreateSession(ctx context.Context, session *model.AIChatSession) error {
	ensureHRSessionOwner(session)
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *ChatRepo) ListSessions(ctx context.Context, hrID int64, page, pageSize int32) ([]model.AIChatSession, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.AIChatSession{}).Where("hr_id = ? AND deleted_at IS NULL", hrID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AIChatSession
	err := query.Order("updated_at DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error
	return rows, total, err
}

func (r *ChatRepo) GetSessionOwned(ctx context.Context, hrID, sessionID int64) (*model.AIChatSession, error) {
	var session model.AIChatSession
	err := r.db.WithContext(ctx).Where("id = ? AND hr_id = ? AND deleted_at IS NULL", sessionID, hrID).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &session, err
}

func (r *ChatRepo) ListBySession(ctx context.Context, hrID, sessionID int64, page, pageSize int32) ([]model.AIChatHistory, error) {
	var rows []model.AIChatHistory
	err := r.db.WithContext(ctx).Where("hr_id = ? AND session_id = ?", hrID, sessionID).
		Where("EXISTS (SELECT 1 FROM ai_chat_sessions s WHERE s.id = ? AND s.hr_id = ? AND s.deleted_at IS NULL)", sessionID, hrID).
		Order("created_at ASC, id ASC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Find(&rows).Error
	return rows, err
}

func (r *ChatRepo) UpdateSessionTitle(ctx context.Context, hrID, sessionID int64, title string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.AIChatSession{}).
		Where("id = ? AND hr_id = ? AND deleted_at IS NULL", sessionID, hrID).
		Update("title", title)
	return result.RowsAffected, result.Error
}

func (r *ChatRepo) DeleteSession(ctx context.Context, hrID, sessionID int64) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.AIChatSession{}).
		Where("id = ? AND hr_id = ? AND deleted_at IS NULL", sessionID, hrID).
		Updates(map[string]any{"deleted_at": &now})
	return result.RowsAffected, result.Error
}

// ListRecentBySession returns the most recent N messages for a session in chronological order.
// It queries the last `limit` messages by created_at DESC then reverses to ascending order.
func (r *ChatRepo) ListRecentBySession(ctx context.Context, hrID, sessionID int64, limit int) ([]model.AIChatHistory, error) {
	var rows []model.AIChatHistory
	err := r.db.WithContext(ctx).
		Where("hr_id = ? AND session_id = ?", hrID, sessionID).
		Where("EXISTS (SELECT 1 FROM ai_chat_sessions s WHERE s.id = ? AND s.hr_id = ? AND s.deleted_at IS NULL)", sessionID, hrID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	// Reverse to chronological order.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}

// CountBySession returns the total number of chat messages in a session.
func (r *ChatRepo) CountBySession(ctx context.Context, hrID, sessionID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AIChatHistory{}).
		Where("hr_id = ? AND session_id = ?", hrID, sessionID).
		Where("EXISTS (SELECT 1 FROM ai_chat_sessions s WHERE s.id = ? AND s.hr_id = ? AND s.deleted_at IS NULL)", sessionID, hrID).
		Count(&count).Error
	return count, err
}

// ---- Owner-based methods for candidate AI ----

// AddOwned creates a chat history record with owner fields populated from the session.
func (r *ChatRepo) AddOwned(ctx context.Context, ownerRole int32, ownerID int64, history *model.AIChatHistory) error {
	history.OwnerRole = ownerRole
	history.OwnerID = ownerID
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if history.SessionID > 0 {
			// Verify session ownership first — COUNT is immune to MySQL
			// "RowsAffected = 0 when value unchanged" behavior.
			var cnt int64
			if err := tx.Model(&model.AIChatSession{}).
				Where("id = ? AND owner_role = ? AND owner_id = ? AND deleted_at IS NULL", history.SessionID, ownerRole, ownerID).
				Count(&cnt).Error; err != nil {
				return err
			}
			if cnt == 0 {
				return gorm.ErrRecordNotFound
			}
			// Touch updated_at (best-effort; ignore rows affected).
			tx.Model(&model.AIChatSession{}).
				Where("id = ? AND owner_role = ? AND owner_id = ? AND deleted_at IS NULL", history.SessionID, ownerRole, ownerID).
				Update("updated_at", time.Now())
		}
		return tx.Create(history).Error
	})
}

// CreateSessionOwned creates a chat session with owner fields.
func (r *ChatRepo) CreateSessionOwned(ctx context.Context, ownerRole int32, ownerID int64, session *model.AIChatSession) error {
	session.OwnerRole = ownerRole
	session.OwnerID = ownerID
	return r.db.WithContext(ctx).Create(session).Error
}

// ListSessionsOwned lists chat sessions by owner.
func (r *ChatRepo) ListSessionsOwned(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32) ([]model.AIChatSession, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.AIChatSession{}).Where("owner_role = ? AND owner_id = ? AND deleted_at IS NULL", ownerRole, ownerID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AIChatSession
	err := query.Order("updated_at DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error
	return rows, total, err
}

// GetSessionOwnedBy retrieves a session by owner and session ID.
func (r *ChatRepo) GetSessionOwnedBy(ctx context.Context, ownerRole int32, ownerID, sessionID int64) (*model.AIChatSession, error) {
	var session model.AIChatSession
	err := r.db.WithContext(ctx).Where("id = ? AND owner_role = ? AND owner_id = ? AND deleted_at IS NULL", sessionID, ownerRole, ownerID).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &session, err
}

// ListBySessionOwned lists chat history by owner and session.
func (r *ChatRepo) ListBySessionOwned(ctx context.Context, ownerRole int32, ownerID, sessionID int64, page, pageSize int32) ([]model.AIChatHistory, error) {
	var rows []model.AIChatHistory
	err := r.db.WithContext(ctx).Where("owner_role = ? AND owner_id = ? AND session_id = ?", ownerRole, ownerID, sessionID).
		Where("EXISTS (SELECT 1 FROM ai_chat_sessions s WHERE s.id = ? AND s.owner_role = ? AND s.owner_id = ? AND s.deleted_at IS NULL)", sessionID, ownerRole, ownerID).
		Order("created_at ASC, id ASC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Find(&rows).Error
	return rows, err
}

// UpdateSessionTitleOwned updates a session title by owner.
func (r *ChatRepo) UpdateSessionTitleOwned(ctx context.Context, ownerRole int32, ownerID, sessionID int64, title string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.AIChatSession{}).
		Where("id = ? AND owner_role = ? AND owner_id = ? AND deleted_at IS NULL", sessionID, ownerRole, ownerID).
		Update("title", title)
	return result.RowsAffected, result.Error
}

// DeleteSessionOwned deletes a session by owner.
func (r *ChatRepo) DeleteSessionOwned(ctx context.Context, ownerRole int32, ownerID, sessionID int64) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.AIChatSession{}).
		Where("id = ? AND owner_role = ? AND owner_id = ? AND deleted_at IS NULL", sessionID, ownerRole, ownerID).
		Updates(map[string]any{"deleted_at": &now})
	return result.RowsAffected, result.Error
}

// MaxMessageIDBySession returns the maximum message ID in a session, or 0 if empty.
func (r *ChatRepo) MaxMessageIDBySession(ctx context.Context, hrID, sessionID int64) (int64, error) {
	var maxID int64
	err := r.db.WithContext(ctx).Model(&model.AIChatHistory{}).
		Where("hr_id = ? AND session_id = ?", hrID, sessionID).
		Where("EXISTS (SELECT 1 FROM ai_chat_sessions s WHERE s.id = ? AND s.hr_id = ? AND s.deleted_at IS NULL)", sessionID, hrID).
		Select("COALESCE(MAX(id), 0)").
		Scan(&maxID).Error
	return maxID, err
}

func ensureHRSessionOwner(session *model.AIChatSession) {
	if session == nil || session.HrID <= 0 {
		return
	}
	if session.OwnerRole == 0 {
		session.OwnerRole = 2
	}
	if session.OwnerID == 0 {
		session.OwnerID = session.HrID
	}
}

func ensureHROwner(history *model.AIChatHistory) {
	if history == nil || history.HrID <= 0 {
		return
	}
	if history.OwnerRole == 0 {
		history.OwnerRole = 2
	}
	if history.OwnerID == 0 {
		history.OwnerID = history.HrID
	}
}
