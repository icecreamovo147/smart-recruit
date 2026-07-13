package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-domain-go/pkg/pagination"
	"smart-recruit-notification-service/internal/domain/model"
	"smart-recruit-notification-service/internal/domain/repository"
)

const (
	eventInboxStatusProcessing int32 = 0
	eventInboxStatusProcessed  int32 = 1
	eventInboxStatusFailed     int32 = 2
	eventInboxStatusDead       int32 = 3
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *model.Notification) error {
	row := notificationToRow(n)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	applyNotificationRow(n, row)
	return nil
}

func (r *NotificationRepository) CreateOnceWithResult(ctx context.Context, n *model.Notification) (bool, error) {
	if err := r.Create(ctx, n); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *NotificationRepository) List(ctx context.Context, receiverID int64, accountType string, page, pageSize int32) ([]model.Notification, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&notificationRow{}).
		Where("receiver_id = ? AND receiver_account_type = ?", receiverID, accountType)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []notificationRow
	err := query.Order("created_at DESC, id DESC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Find(&rows).Error
	return notificationRowsToDomain(rows), total, err
}

func (r *NotificationRepository) ListCursor(ctx context.Context, receiverID int64, accountType string, cursor string, limit int32) ([]model.Notification, string, bool, error) {
	t, id, err := pagination.DecodeCursor(cursor)
	if err != nil {
		return nil, "", false, err
	}
	query := r.db.WithContext(ctx).Model(&notificationRow{}).
		Where("receiver_id = ? AND receiver_account_type = ?", receiverID, accountType)
	if !t.IsZero() || id > 0 {
		query = query.Where("(created_at, id) < (?, ?)", t, id)
	}
	fetchLimit := int(limit) + 1
	var rows []notificationRow
	if err := query.Order("created_at DESC, id DESC").Limit(fetchLimit).Find(&rows).Error; err != nil {
		return nil, "", false, err
	}
	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}
	nextCursor := ""
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	return notificationRowsToDomain(rows), nextCursor, hasMore, nil
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, receiverID int64, accountType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&notificationRow{}).
		Where("receiver_id = ? AND receiver_account_type = ? AND is_read = ?", receiverID, accountType, 0).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepository) Latest(ctx context.Context, receiverID int64, accountType string) (*model.Notification, error) {
	var row notificationRow
	err := r.db.WithContext(ctx).Model(&notificationRow{}).
		Where("receiver_id = ? AND receiver_account_type = ?", receiverID, accountType).
		Order("created_at DESC, id DESC").
		Limit(1).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n := notificationFromRow(row)
	return &n, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, receiverID int64, accountType string, notificationID int64) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&notificationRow{}).
		Where("id = ? AND receiver_id = ? AND receiver_account_type = ? AND is_read = ?", notificationID, receiverID, accountType, 0).
		Updates(map[string]any{"is_read": 1, "read_at": &now})
	return result.RowsAffected, result.Error
}

func (r *NotificationRepository) MarkAllReadBatch(ctx context.Context, receiverID int64, accountType string, limit int) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&notificationRow{}).
		Where("receiver_id = ? AND receiver_account_type = ? AND is_read = ?", receiverID, accountType, 0).
		Limit(limit).
		Updates(map[string]any{"is_read": 1, "read_at": &now})
	return result.RowsAffected, result.Error
}

type EmailLogRepository struct {
	db *gorm.DB
}

func NewEmailLogRepository(db *gorm.DB) *EmailLogRepository {
	return &EmailLogRepository{db: db}
}

func (r *EmailLogRepository) Create(ctx context.Context, log *model.EmailLog) error {
	row := emailLogToRow(log)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	log.ID = row.ID
	log.CreatedAt = row.CreatedAt
	return nil
}

func (r *EmailLogRepository) ExistsByEventID(ctx context.Context, eventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&emailLogRow{}).Where("event_id = ?", eventID).Count(&count).Error
	return count > 0, err
}

type InboxRepository struct {
	db *gorm.DB
}

func NewInboxRepository(db *gorm.DB) *InboxRepository {
	return &InboxRepository{db: db}
}

func (r *InboxRepository) Claim(ctx context.Context, claim repository.InboxClaim) (*repository.InboxRecord, bool, error) {
	now := time.Now()
	claimed := true
	row := &inboxRow{
		EventID:        claim.EventID,
		EventType:      claim.EventType,
		ConsumerName:   claim.ConsumerName,
		IdempotencyKey: claim.IdempotencyKey,
		Status:         eventInboxStatusProcessing,
		AttemptCount:   1,
		ReceivedAt:     now,
		ProcessingAt:   &now,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "consumer_name"}, {Name: "event_id"}},
			DoNothing: true,
		}).Create(row)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			return nil
		}
		var existing inboxRow
		if err := tx.Where("consumer_name = ? AND event_id = ?", claim.ConsumerName, claim.EventID).First(&existing).Error; err != nil {
			return err
		}
		*row = existing
		if existing.Status == eventInboxStatusProcessed || existing.Status == eventInboxStatusDead {
			claimed = false
			return nil
		}
		if err := tx.Model(&inboxRow{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"status":           eventInboxStatusProcessing,
				"attempt_count":    gorm.Expr("attempt_count + 1"),
				"last_error":       "",
				"processing_at":    now,
				"dead_lettered_at": nil,
			}).Error; err != nil {
			return err
		}
		return tx.First(row, existing.ID).Error
	})
	if err != nil {
		return nil, false, err
	}
	return &repository.InboxRecord{ID: row.ID}, claimed, nil
}

func (r *InboxRepository) MarkProcessed(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&inboxRow{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": eventInboxStatusProcessed, "last_error": "", "processed_at": now}).Error
}

func (r *InboxRepository) MarkFailed(ctx context.Context, id uint64, errMsg string) error {
	return r.db.WithContext(ctx).Model(&inboxRow{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": eventInboxStatusFailed, "last_error": errMsg}).Error
}

type UserDirectory struct {
	db *gorm.DB
}

func NewUserDirectory(db *gorm.DB) *UserDirectory {
	return &UserDirectory{db: db}
}

func (r *UserDirectory) GetEmailRecipient(ctx context.Context, userID int64) (*model.EmailRecipient, error) {
	var row userRow
	err := r.db.WithContext(ctx).Where("id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &model.EmailRecipient{UserID: row.ID, Email: row.Email, Username: row.Username}, nil
}

type notificationRow struct {
	ID                  int64      `gorm:"primaryKey"`
	EventID             *string    `gorm:"column:event_id;uniqueIndex:uk_notification_event_id"`
	ReceiverID          int64      `gorm:"column:receiver_id;uniqueIndex:uk_notification_once,priority:1"`
	ReceiverAccountType string     `gorm:"column:receiver_account_type;uniqueIndex:uk_notification_once,priority:2"`
	ReceiverRole        int32      `gorm:"column:receiver_role"`
	Type                string     `gorm:"column:type;uniqueIndex:uk_notification_once,priority:5"`
	Title               string     `gorm:"column:title"`
	Content             string     `gorm:"column:content"`
	Link                string     `gorm:"column:link"`
	BizType             string     `gorm:"column:biz_type;uniqueIndex:uk_notification_once,priority:3"`
	BizID               int64      `gorm:"column:biz_id;uniqueIndex:uk_notification_once,priority:4"`
	IsRead              int32      `gorm:"column:is_read"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	ReadAt              *time.Time `gorm:"column:read_at"`
}

func (notificationRow) TableName() string { return "notifications" }

type emailLogRow struct {
	ID        int64     `gorm:"primaryKey"`
	EventID   string    `gorm:"column:event_id;uniqueIndex:uk_email_event_id"`
	UserID    int64     `gorm:"column:user_id"`
	Email     string    `gorm:"column:email"`
	Type      string    `gorm:"column:type"`
	Subject   string    `gorm:"column:subject"`
	Status    string    `gorm:"column:status;default:sent"`
	ErrorMsg  *string   `gorm:"column:error_msg"`
	SentAt    time.Time `gorm:"column:sent_at"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (emailLogRow) TableName() string { return "email_logs" }

type inboxRow struct {
	ID             uint64     `gorm:"primaryKey"`
	EventID        string     `gorm:"column:event_id;size:128;uniqueIndex:uk_event_inbox_consumer_event,priority:2"`
	EventType      string     `gorm:"column:event_type;size:128"`
	ConsumerName   string     `gorm:"column:consumer_name;size:128;uniqueIndex:uk_event_inbox_consumer_event,priority:1;index:idx_event_inbox_consumer_status,priority:1"`
	IdempotencyKey string     `gorm:"column:idempotency_key;size:255;index:idx_event_inbox_idempotency_key"`
	Status         int32      `gorm:"column:status;default:0;index:idx_event_inbox_consumer_status,priority:2"`
	AttemptCount   int32      `gorm:"column:attempt_count;default:0"`
	LastError      string     `gorm:"column:last_error"`
	ReceivedAt     time.Time  `gorm:"column:received_at"`
	ProcessingAt   *time.Time `gorm:"column:processing_at"`
	ProcessedAt    *time.Time `gorm:"column:processed_at"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (inboxRow) TableName() string { return "event_inbox" }

type userRow struct {
	ID       int64  `gorm:"primaryKey"`
	Username string `gorm:"column:username"`
	Email    string `gorm:"column:email"`
}

func (userRow) TableName() string { return "users" }

func notificationToRow(n *model.Notification) notificationRow {
	var eventID *string
	if n.EventID != "" {
		value := n.EventID
		eventID = &value
	}
	isRead := int32(0)
	if n.IsRead {
		isRead = 1
	}
	return notificationRow{
		ID:                  n.ID,
		EventID:             eventID,
		ReceiverID:          n.ReceiverID,
		ReceiverAccountType: n.ReceiverAccountType,
		ReceiverRole:        n.ReceiverRole,
		Type:                n.Type,
		Title:               n.Title,
		Content:             n.Content,
		Link:                n.Link,
		BizType:             n.BizType,
		BizID:               n.BizID,
		IsRead:              isRead,
		CreatedAt:           n.CreatedAt,
		ReadAt:              n.ReadAt,
	}
}

func notificationFromRow(row notificationRow) model.Notification {
	eventID := ""
	if row.EventID != nil {
		eventID = *row.EventID
	}
	return model.Notification{
		ID:                  row.ID,
		EventID:             eventID,
		ReceiverID:          row.ReceiverID,
		ReceiverAccountType: row.ReceiverAccountType,
		ReceiverRole:        row.ReceiverRole,
		Type:                row.Type,
		Title:               row.Title,
		Content:             row.Content,
		Link:                row.Link,
		BizType:             row.BizType,
		BizID:               row.BizID,
		IsRead:              row.IsRead != 0,
		CreatedAt:           row.CreatedAt,
		ReadAt:              row.ReadAt,
	}
}

func applyNotificationRow(n *model.Notification, row notificationRow) {
	*n = notificationFromRow(row)
}

func notificationRowsToDomain(rows []notificationRow) []model.Notification {
	out := make([]model.Notification, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationFromRow(row))
	}
	return out
}

func emailLogToRow(log *model.EmailLog) emailLogRow {
	var errMsg *string
	if log.ErrorMsg != "" {
		value := log.ErrorMsg
		errMsg = &value
	}
	return emailLogRow{
		ID:        log.ID,
		EventID:   log.EventID,
		UserID:    log.UserID,
		Email:     log.Email,
		Type:      log.Type,
		Subject:   log.Subject,
		Status:    log.Status,
		ErrorMsg:  errMsg,
		SentAt:    log.SentAt,
		CreatedAt: log.CreatedAt,
	}
}

func offset(page, pageSize int32) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return int((page - 1) * pageSize)
}
