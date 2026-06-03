package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/mq"
	"logic-grpc-service/repository"
)

type notificationPayload struct {
	EventID             string `json:"event_id"`
	ReceiverID          int64  `json:"receiver_id"`
	ReceiverRole        int32  `json:"receiver_role"`
	ReceiverAccountType string `json:"receiver_account_type"`
	Type                string `json:"type"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	Link                string `json:"link"`
	BizType             string `json:"biz_type"`
	BizID               int64  `json:"biz_id"`
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

	acctType := p.ReceiverAccountType
	if acctType == "" {
		logger.L().Warn("notification consumer payload missing receiver_account_type, defaulting to candidate",
			zap.Int64("receiver_id", p.ReceiverID))
		acctType = "candidate"
	}

	n := &model.Notification{
		ReceiverID:          p.ReceiverID,
		ReceiverRole:        p.ReceiverRole,
		ReceiverAccountType: acctType,
		Type:                p.Type,
		Title:               p.Title,
		Content:             p.Content,
		Link:                p.Link,
		BizType:             p.BizType,
		BizID:               p.BizID,
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
		return nil
	}

	// Invalidate Redis unread cache and publish SSE event
	if c.cache != nil {
		c.cache.Invalidate(ctx, uint64(p.ReceiverID), acctType)
		unread, _ := c.repo.UnreadCount(ctx, p.ReceiverID, acctType)
		c.cache.SetUnreadCount(ctx, uint64(p.ReceiverID), acctType, unread)
		event, _ := json.Marshal(notificationEvent{
			Type:             "notification_created",
			NotificationType: p.Type,
			NotificationID:   n.ID,
			Unread:           unread,
			Title:            p.Title,
			Content:          p.Content,
			Link:             p.Link,
			CreatedAt:        n.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		})
		c.cache.PublishNotificationEvent(ctx, uint64(p.ReceiverID), acctType, string(event))
	}

	return nil
}
