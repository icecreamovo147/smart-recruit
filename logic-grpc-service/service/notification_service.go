package service

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type NotificationService struct {
	repo  *repository.NotificationRepo
	cache *cache.NotificationCache
}

type notificationEvent struct {
	Type           string `json:"type"`
	NotificationID int64  `json:"notification_id"`
	Unread         int64  `json:"unread"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Link           string `json:"link"`
	CreatedAt      string `json:"created_at"`
}

func NewNotificationService(repo *repository.NotificationRepo, c *cache.NotificationCache) *NotificationService {
	return &NotificationService{repo: repo, cache: c}
}

func (s *NotificationService) Create(ctx context.Context, n *model.Notification) error {
	if err := s.repo.Create(ctx, n); err != nil {
		return err
	}
	// Invalidate cached unread count so next poll picks up the new notification.
	if s.cache != nil {
		s.cache.Invalidate(ctx, uint64(n.ReceiverID), n.ReceiverRole)
	}
	s.publishCreatedEvent(ctx, n)
	return nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	ps := req.PageSize
	if ps < 1 || ps > 50 {
		ps = 20
	}
	if req.Cursor != "" || req.Page <= 0 {
		rows, cursor, hasMore, err := s.repo.ListCursor(ctx, req.UserId, req.Role, req.Cursor, ps)
		if err != nil {
			logger.L().Warn("list notifications cursor failed, returning empty", zap.Error(err))
			return &pb.ListNotificationsResponse{List: nil}, nil
		}
		list := make([]*pb.Notification, 0, len(rows))
		for _, row := range rows {
			var readAt string
			if row.ReadAt != nil {
				readAt = row.ReadAt.Format("2006-01-02T15:04:05-07:00")
			}
			list = append(list, &pb.Notification{
				NotificationId: row.ID,
				Type:           row.Type,
				Title:          row.Title,
				Content:        row.Content,
				Link:           row.Link,
				BizType:        row.BizType,
				BizId:          row.BizID,
				IsRead:         row.IsRead == 1,
				CreatedAt:      row.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
				ReadAt:         readAt,
			})
		}
		return &pb.ListNotificationsResponse{Code: errs.OK, Msg: "success", List: list, NextCursor: cursor, HasMore: hasMore}, nil
	}
	page := req.Page
	if page < 1 {
		page = 1
	}
	rows, total, err := s.repo.List(ctx, req.UserId, req.Role, page, ps)
	if err != nil {
		logger.L().Warn("list notifications failed, returning empty", zap.Error(err))
		return &pb.ListNotificationsResponse{Total: 0, List: nil}, nil
	}
	logger.L().Debug("list notifications from db",
		zap.Int64("user_id", req.UserId),
		zap.Int32("role", req.Role),
		zap.Int64("total", total),
		zap.Int("returned", len(rows)),
	)
	list := make([]*pb.Notification, 0, len(rows))
	for _, row := range rows {
		var readAt string
		if row.ReadAt != nil {
			readAt = row.ReadAt.Format("2006-01-02T15:04:05-07:00")
		}
		list = append(list, &pb.Notification{
			NotificationId: row.ID,
			Type:           row.Type,
			Title:          row.Title,
			Content:        row.Content,
			Link:           row.Link,
			BizType:        row.BizType,
			BizId:          row.BizID,
			IsRead:         row.IsRead == 1,
			CreatedAt:      row.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
			ReadAt:         readAt,
		})
	}
	return &pb.ListNotificationsResponse{Total: total, List: list}, nil
}

func (s *NotificationService) UnreadNotificationCount(ctx context.Context, req *pb.UnreadNotificationCountRequest) (*pb.UnreadNotificationCountResponse, error) {
	// P0: Try Redis cache first.
	if s.cache != nil {
		if count, ok := s.cache.GetUnreadCount(ctx, uint64(req.UserId), req.Role); ok {
			logger.L().Debug("unread count from cache",
				zap.Int64("user_id", req.UserId),
				zap.Int32("role", req.Role),
				zap.Int64("count", count),
			)
			return &pb.UnreadNotificationCountResponse{Unread: count}, nil
		}
	}
	count, err := s.repo.UnreadCount(ctx, req.UserId, req.Role)
	if err != nil {
		logger.L().Warn("unread count query failed, returning 0", zap.Error(err))
		return &pb.UnreadNotificationCountResponse{Unread: 0}, nil
	}
	logger.L().Debug("unread count from db",
		zap.Int64("user_id", req.UserId),
		zap.Int32("role", req.Role),
		zap.Int64("count", count),
	)
	// Backfill cache.
	if s.cache != nil {
		s.cache.SetUnreadCount(ctx, uint64(req.UserId), req.Role, count)
	}
	return &pb.UnreadNotificationCountResponse{Unread: count}, nil
}

func (s *NotificationService) NotificationSummary(ctx context.Context, req *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error) {
	unread, err := s.unreadCount(ctx, req.UserId, req.Role)
	if err != nil {
		logger.L().Warn("notification summary unread count failed, returning 0", zap.Error(err))
		unread = 0
	}
	latest, err := s.repo.Latest(ctx, req.UserId, req.Role)
	if err != nil {
		logger.L().Warn("notification summary latest query failed", zap.Error(err))
		return &pb.NotificationSummaryResponse{Unread: unread}, nil
	}
	if latest == nil {
		return &pb.NotificationSummaryResponse{Unread: unread}, nil
	}
	return &pb.NotificationSummaryResponse{
		Unread:               unread,
		LatestNotificationId: latest.ID,
		LatestCreatedAt:      latest.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
	}, nil
}

func (s *NotificationService) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationReadRequest) (*pb.CommonResponse, error) {
	rows, err := s.repo.MarkRead(ctx, req.UserId, req.Role, req.NotificationId)
	if err != nil {
		logger.L().Warn("mark read failed", zap.Error(err))
		return &pb.CommonResponse{Msg: "success"}, nil
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: 403, Msg: "无权限或通知不存在"}, nil
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, uint64(req.UserId), req.Role)
	}
	return &pb.CommonResponse{Msg: "success"}, nil
}

func (s *NotificationService) unreadCount(ctx context.Context, userID int64, role int32) (int64, error) {
	if s.cache != nil {
		if count, ok := s.cache.GetUnreadCount(ctx, uint64(userID), role); ok {
			return count, nil
		}
	}
	count, err := s.repo.UnreadCount(ctx, userID, role)
	if err != nil {
		return 0, err
	}
	if s.cache != nil {
		s.cache.SetUnreadCount(ctx, uint64(userID), role, count)
	}
	return count, nil
}

func (s *NotificationService) publishCreatedEvent(ctx context.Context, n *model.Notification) {
	if s.cache == nil {
		return
	}
	unread, err := s.unreadCount(ctx, n.ReceiverID, n.ReceiverRole)
	if err != nil {
		logger.L().Warn("notification event unread count failed", zap.Error(err))
	}
	payload, err := json.Marshal(notificationEvent{
		Type:           "notification_created",
		NotificationID: n.ID,
		Unread:         unread,
		Title:          n.Title,
		Content:        n.Content,
		Link:           n.Link,
		CreatedAt:      n.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
	})
	if err != nil {
		logger.L().Warn("notification event marshal failed", zap.Error(err))
		return
	}
	s.cache.PublishNotificationEvent(ctx, uint64(n.ReceiverID), n.ReceiverRole, string(payload))
}

func (s *NotificationService) MarkAllNotificationsRead(ctx context.Context, req *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error) {
	// P2: Batch loop to avoid long row locks.
	var total int64
	for {
		rows, err := s.repo.MarkAllReadBatch(ctx, req.UserId, req.Role, 1000)
		if err != nil {
			logger.L().Warn("mark all read batch failed", zap.Error(err))
			break
		}
		total += rows
		if rows < 1000 {
			break
		}
	}
	logger.L().Info("mark all read done",
		zap.Int64("user_id", req.UserId),
		zap.Int32("role", req.Role),
		zap.Int64("rows", total),
	)
	if s.cache != nil {
		s.cache.Invalidate(ctx, uint64(req.UserId), req.Role)
	}
	return &pb.CommonResponse{Msg: "success"}, nil
}
