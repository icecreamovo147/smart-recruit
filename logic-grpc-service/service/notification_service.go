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
	repo   *repository.NotificationRepo
	cache  *cache.NotificationCache
	authz  *ServiceAuthorizer
}

type notificationEvent struct {
	Type             string `json:"type"`
	NotificationType string `json:"notification_type"`
	NotificationID   int64  `json:"notification_id"`
	Unread           int64  `json:"unread"`
	Title            string `json:"title"`
	Content          string `json:"content"`
	Link             string `json:"link"`
	CreatedAt        string `json:"created_at"`
}

func NewNotificationService(repo *repository.NotificationRepo, c *cache.NotificationCache, authz *ServiceAuthorizer) *NotificationService {
	return &NotificationService{repo: repo, cache: c, authz: authz}
}

func (s *NotificationService) Create(ctx context.Context, n *model.Notification) error {
	if err := s.repo.Create(ctx, n); err != nil {
		return err
	}
	acctType := n.ReceiverAccountType
	if acctType == "" {
		logger.L().Warn("notification missing receiver_account_type, defaulting to candidate",
			zap.Int64("receiver_id", n.ReceiverID))
		acctType = "candidate"
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, uint64(n.ReceiverID), acctType)
	}
	s.publishCreatedEvent(ctx, n)
	return nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	if req.UserId == 0 {
		return &pb.ListNotificationsResponse{Code: errs.ErrBadRequest, Msg: "user_id is required"}, nil
	}
	if req.AccountType == "" {
		return &pb.ListNotificationsResponse{Code: errs.ErrBadRequest, Msg: "account_type is required"}, nil
	}
	// Verify the request user_id matches the authenticated actor from gRPC context.
	// This closes the gap where a compromised web-gin handler or direct gRPC call
	// could read another user's notifications.
	if s.authz != nil {
		if err := s.authz.VerifyActorMatch(ctx, req.UserId); err != nil {
			return &pb.ListNotificationsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
		}
	}
	ps := req.PageSize
	if ps < 1 || ps > 50 {
		ps = 20
	}
	if req.Cursor != "" || req.Page <= 0 {
		rows, cursor, hasMore, err := s.repo.ListCursor(ctx, req.UserId, req.AccountType, req.Cursor, ps)
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
	rows, total, err := s.repo.List(ctx, req.UserId, req.AccountType, page, ps)
	if err != nil {
		logger.L().Warn("list notifications failed, returning empty", zap.Error(err))
		return &pb.ListNotificationsResponse{Total: 0, List: nil}, nil
	}
	logger.L().Debug("list notifications from db",
		zap.Int64("user_id", req.UserId),
		zap.String("account_type", req.AccountType),
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
	if s.cache != nil {
		if count, ok := s.cache.GetUnreadCount(ctx, uint64(req.UserId), req.AccountType); ok {
			logger.L().Debug("unread count from cache",
				zap.Int64("user_id", req.UserId),
				zap.String("account_type", req.AccountType),
				zap.Int64("count", count),
			)
			return &pb.UnreadNotificationCountResponse{Unread: count}, nil
		}
	}
	count, err := s.repo.UnreadCount(ctx, req.UserId, req.AccountType)
	if err != nil {
		logger.L().Warn("unread count query failed, returning 0", zap.Error(err))
		return &pb.UnreadNotificationCountResponse{Unread: 0}, nil
	}
	logger.L().Debug("unread count from db",
		zap.Int64("user_id", req.UserId),
		zap.String("account_type", req.AccountType),
		zap.Int64("count", count),
	)
	if s.cache != nil {
		s.cache.SetUnreadCount(ctx, uint64(req.UserId), req.AccountType, count)
	}
	return &pb.UnreadNotificationCountResponse{Unread: count}, nil
}

func (s *NotificationService) NotificationSummary(ctx context.Context, req *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error) {
	unread, err := s.unreadCount(ctx, req.UserId, req.AccountType)
	if err != nil {
		logger.L().Warn("notification summary unread count failed, returning 0", zap.Error(err))
		unread = 0
	}
	latest, err := s.repo.Latest(ctx, req.UserId, req.AccountType)
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
	rows, err := s.repo.MarkRead(ctx, req.UserId, req.AccountType, req.NotificationId)
	if err != nil {
		logger.L().Warn("mark read failed", zap.Error(err))
		return &pb.CommonResponse{Msg: "success"}, nil
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: 403, Msg: "无权限或通知不存在"}, nil
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, uint64(req.UserId), req.AccountType)
	}
	return &pb.CommonResponse{Msg: "success"}, nil
}

func (s *NotificationService) unreadCount(ctx context.Context, userID int64, accountType string) (int64, error) {
	if s.cache != nil {
		if count, ok := s.cache.GetUnreadCount(ctx, uint64(userID), accountType); ok {
			return count, nil
		}
	}
	count, err := s.repo.UnreadCount(ctx, userID, accountType)
	if err != nil {
		return 0, err
	}
	if s.cache != nil {
		s.cache.SetUnreadCount(ctx, uint64(userID), accountType, count)
	}
	return count, nil
}

func (s *NotificationService) publishCreatedEvent(ctx context.Context, n *model.Notification) {
	if s.cache == nil {
		return
	}
	acctType := n.ReceiverAccountType
	if acctType == "" {
		logger.L().Warn("notification event missing receiver_account_type, defaulting to candidate",
			zap.Int64("receiver_id", n.ReceiverID))
		acctType = "candidate"
	}
	unread, err := s.unreadCount(ctx, n.ReceiverID, acctType)
	if err != nil {
		logger.L().Warn("notification event unread count failed", zap.Error(err))
	}
	payload, err := json.Marshal(notificationEvent{
		Type:             "notification_created",
		NotificationType: n.Type,
		NotificationID:   n.ID,
		Unread:           unread,
		Title:            n.Title,
		Content:          n.Content,
		Link:             n.Link,
		CreatedAt:        n.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
	})
	if err != nil {
		logger.L().Warn("notification event marshal failed", zap.Error(err))
		return
	}
	s.cache.PublishNotificationEvent(ctx, uint64(n.ReceiverID), acctType, string(payload))
}

func (s *NotificationService) MarkAllNotificationsRead(ctx context.Context, req *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error) {
	var total int64
	for {
		rows, err := s.repo.MarkAllReadBatch(ctx, req.UserId, req.AccountType, 1000)
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
		zap.String("account_type", req.AccountType),
		zap.Int64("rows", total),
	)
	if s.cache != nil {
		s.cache.Invalidate(ctx, uint64(req.UserId), req.AccountType)
	}
	return &pb.CommonResponse{Msg: "success"}, nil
}
