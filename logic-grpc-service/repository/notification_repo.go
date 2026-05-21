package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/pagination"
)

type NotificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// Create inserts a notification record and lets DB unique keys enforce
// idempotency where event_id or business keys are present.
func (r *NotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NotificationRepo) Exists(ctx context.Context, receiverID int64, receiverRole int32, bizType string, bizID int64, notificationType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("receiver_id = ? AND receiver_role = ? AND biz_type = ? AND biz_id = ? AND type = ?",
			receiverID, receiverRole, bizType, bizID, notificationType).
		Count(&count).Error
	return count > 0, err
}

// CreateOnce inserts a notification only if the same receiver/business/type
// notification has not already been created. Use this for idempotent events,
// such as "candidate has submitted this application".
func (r *NotificationRepo) CreateOnce(ctx context.Context, n *model.Notification) error {
	_, err := r.CreateOnceWithResult(ctx, n)
	return err
}

func (r *NotificationRepo) CreateOnceWithResult(ctx context.Context, n *model.Notification) (bool, error) {
	if err := r.Create(ctx, n); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateOrIgnore is kept as a compatibility wrapper for older call sites.
func (r *NotificationRepo) CreateOrIgnore(ctx context.Context, n *model.Notification) error {
	return r.CreateOnce(ctx, n)
}

func (r *NotificationRepo) List(ctx context.Context, receiverID int64, receiverRole int32, page, pageSize int32) ([]model.Notification, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("receiver_id = ? AND receiver_role = ?", receiverID, receiverRole)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Notification
	err := query.Order("created_at DESC, id DESC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Find(&rows).Error
	return rows, total, err
}

// ListCursor returns notifications using cursor-based pagination.
func (r *NotificationRepo) ListCursor(ctx context.Context, receiverID int64, receiverRole int32, cursor string, limit int32) ([]model.Notification, string, bool, error) {
	t, id, err := pagination.DecodeCursor(cursor)
	if err != nil {
		return nil, "", false, err
	}
	query := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("receiver_id = ? AND receiver_role = ?", receiverID, receiverRole)
	if !t.IsZero() || id > 0 {
		query = query.Where("(created_at, id) < (?, ?)", t, id)
	}
	fetchLimit := int(limit) + 1
	var rows []model.Notification
	if err := query.Order("created_at DESC, id DESC").Limit(fetchLimit).Find(&rows).Error; err != nil {
		return nil, "", false, err
	}
	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}
	var nextCursor string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	return rows, nextCursor, hasMore, nil
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, receiverID int64, receiverRole int32) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("receiver_id = ? AND receiver_role = ? AND is_read = ?", receiverID, receiverRole, 0).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepo) Latest(ctx context.Context, receiverID int64, receiverRole int32) (*model.Notification, error) {
	var row model.Notification
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("receiver_id = ? AND receiver_role = ?", receiverID, receiverRole).
		Order("created_at DESC, id DESC").
		Limit(1).
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &row, err
}

func (r *NotificationRepo) MarkRead(ctx context.Context, receiverID int64, receiverRole int32, notificationID int64) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND receiver_id = ? AND receiver_role = ? AND is_read = ?", notificationID, receiverID, receiverRole, 0).
		Updates(map[string]any{"is_read": 1, "read_at": &now})
	return result.RowsAffected, result.Error
}

// MarkAllReadBatch marks up to `limit` unread notifications as read in a single
// UPDATE. Call in a loop until rows returned < limit.
func (r *NotificationRepo) MarkAllReadBatch(ctx context.Context, receiverID int64, receiverRole int32, limit int) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("receiver_id = ? AND receiver_role = ? AND is_read = ?", receiverID, receiverRole, 0).
		Limit(limit).
		Updates(map[string]any{"is_read": 1, "read_at": &now})
	return result.RowsAffected, result.Error
}
