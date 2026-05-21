package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/mq"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

type notificationPayload struct {
	EventID      string `json:"event_id"`
	ReceiverID   int64  `json:"receiver_id"`
	ReceiverRole int32  `json:"receiver_role"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Link         string `json:"link"`
	BizType      string `json:"biz_type"`
	BizID        int64  `json:"biz_id"`
}

type NotificationConsumer struct {
	repo  *repository.NotificationRepo
	cache *cache.NotificationCache
}

func NewNotificationConsumer(repo *repository.NotificationRepo, c *cache.NotificationCache) *NotificationConsumer {
	return &NotificationConsumer{repo: repo, cache: c}
}

func (c *NotificationConsumer) Start(ctx context.Context, mqConn *mq.Conn) error {
	return mqConn.Consume(ctx, mqConn.NotificationQueue(), func(ctx context.Context, body []byte) error {
		return c.handle(ctx, body)
	})
}

func (c *NotificationConsumer) handle(ctx context.Context, body []byte) error {
	var p notificationPayload
	if err := json.Unmarshal(body, &p); err != nil {
		logger.L().Error("notification consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}

	n := &model.Notification{
		ReceiverID:   p.ReceiverID,
		ReceiverRole: p.ReceiverRole,
		Type:         p.Type,
		Title:        p.Title,
		Content:      p.Content,
		Link:         p.Link,
		BizType:      p.BizType,
		BizID:        p.BizID,
	}
	if p.EventID != "" {
		n.EventID = &p.EventID
	}

	created, err := c.repo.CreateOnceWithResult(ctx, n)
	if err != nil {
		logger.L().Error("notification consumer: create failed", zap.Error(err))
		return err
	}
	if !created {
		logger.L().Debug("notification consumer: duplicate skipped",
			zap.String("type", p.Type), zap.Int64("biz_id", p.BizID))
		return nil
	}

	// Invalidate Redis unread cache and publish SSE event
	if c.cache != nil {
		c.cache.Invalidate(ctx, uint64(p.ReceiverID), p.ReceiverRole)
		unread, _ := c.repo.UnreadCount(ctx, p.ReceiverID, p.ReceiverRole)
		c.cache.SetUnreadCount(ctx, uint64(p.ReceiverID), p.ReceiverRole, unread)
		event, _ := json.Marshal(notificationEvent{
			Type:           "notification_created",
			NotificationID: n.ID,
			Unread:         unread,
			Title:          p.Title,
			Content:        p.Content,
			Link:           p.Link,
			CreatedAt:      n.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		})
		c.cache.PublishNotificationEvent(ctx, uint64(p.ReceiverID), p.ReceiverRole, string(event))
	}

	return nil
}
