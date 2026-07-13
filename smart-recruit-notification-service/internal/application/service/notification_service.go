package service

import (
	"context"
	"encoding/json"

	"smart-recruit-notification-service/internal/application/command"
	"smart-recruit-notification-service/internal/application/port"
	"smart-recruit-notification-service/internal/application/query"
	"smart-recruit-notification-service/internal/domain/model"
	"smart-recruit-notification-service/internal/domain/repository"
)

const markAllBatchSize = 1000

type Deps struct {
	Notifications repository.NotificationRepository
	EmailLogs     repository.EmailLogRepository
	UnreadCache   port.UnreadCache
	Realtime      port.RealtimePublisher
	ActorVerifier port.ActorVerifier
	Users         port.UserDirectory
	Renderer      port.EmailRenderer
	Sender        port.EmailSender
	Clock         port.Clock
}

type Service struct {
	notifications repository.NotificationRepository
	emailLogs     repository.EmailLogRepository
	unreadCache   port.UnreadCache
	realtime      port.RealtimePublisher
	actorVerifier port.ActorVerifier
	users         port.UserDirectory
	renderer      port.EmailRenderer
	sender        port.EmailSender
	clock         port.Clock
}

func New(deps Deps) *Service {
	clock := deps.Clock
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &Service{
		notifications: deps.Notifications,
		emailLogs:     deps.EmailLogs,
		unreadCache:   deps.UnreadCache,
		realtime:      deps.Realtime,
		actorVerifier: deps.ActorVerifier,
		users:         deps.Users,
		renderer:      deps.Renderer,
		sender:        deps.Sender,
		clock:         clock,
	}
}

func (s *Service) CreateNotification(ctx context.Context, cmd command.CreateNotification) error {
	n := notificationFromCommand(cmd)
	n.EnsureCreatedAt(s.clock.Now())
	if err := s.notifications.Create(ctx, &n); err != nil {
		return err
	}
	s.afterNotificationCreated(ctx, &n)
	return nil
}

func (s *Service) ListNotifications(ctx context.Context, req query.ListNotifications) (query.ListResult, error) {
	if req.UserID == 0 {
		return query.ListResult{Code: query.CodeBadRequest, Message: "user_id is required"}, nil
	}
	if req.AccountType == "" {
		return query.ListResult{Code: query.CodeBadRequest, Message: "account_type is required"}, nil
	}
	if s.actorVerifier != nil {
		if err := s.actorVerifier.VerifyActorMatch(ctx, req.UserID); err != nil {
			return query.ListResult{Code: query.CodeForbidden, Message: err.Error()}, nil
		}
	}
	pageSize := normalizePageSize(req.PageSize)
	if req.Cursor != "" || req.Page <= 0 {
		rows, nextCursor, hasMore, err := s.notifications.ListCursor(ctx, req.UserID, req.AccountType, req.Cursor, pageSize)
		if err != nil {
			return query.ListResult{}, nil
		}
		return query.ListResult{
			Code:       query.CodeOK,
			Message:    "success",
			List:       rows,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		}, nil
	}
	rows, total, err := s.notifications.List(ctx, req.UserID, req.AccountType, req.Page, pageSize)
	if err != nil {
		return query.ListResult{}, nil
	}
	return query.ListResult{List: rows, Total: total}, nil
}

func (s *Service) UnreadNotificationCount(ctx context.Context, req query.UnreadCount) (query.UnreadCountResult, error) {
	unread, err := s.unreadCount(ctx, req.UserID, req.AccountType)
	if err != nil {
		return query.UnreadCountResult{}, nil
	}
	return query.UnreadCountResult{Unread: unread}, nil
}

func (s *Service) NotificationSummary(ctx context.Context, req query.Summary) (query.SummaryResult, error) {
	unread, err := s.unreadCount(ctx, req.UserID, req.AccountType)
	if err != nil {
		unread = 0
	}
	latest, err := s.notifications.Latest(ctx, req.UserID, req.AccountType)
	if err != nil || latest == nil {
		return query.SummaryResult{Unread: unread}, nil
	}
	return query.SummaryResult{
		Unread:               unread,
		LatestNotificationID: latest.ID,
		LatestCreatedAt:      formatTime(latest.CreatedAt),
	}, nil
}

func (s *Service) MarkNotificationRead(ctx context.Context, cmd command.MarkRead) (query.CommandResult, error) {
	rows, err := s.notifications.MarkRead(ctx, cmd.UserID, cmd.AccountType, cmd.NotificationID)
	if err != nil {
		return query.CommandResult{Message: "success"}, nil
	}
	if rows == 0 {
		return query.CommandResult{Code: query.CodeForbidden, Message: "无权限或通知不存在"}, nil
	}
	if s.unreadCache != nil {
		s.unreadCache.Invalidate(ctx, uint64(cmd.UserID), cmd.AccountType)
	}
	return query.CommandResult{Message: "success"}, nil
}

func (s *Service) MarkAllNotificationsRead(ctx context.Context, cmd command.MarkAllRead) (query.CommandResult, error) {
	for {
		rows, err := s.notifications.MarkAllReadBatch(ctx, cmd.UserID, cmd.AccountType, markAllBatchSize)
		if err != nil || rows < markAllBatchSize {
			break
		}
	}
	if s.unreadCache != nil {
		s.unreadCache.Invalidate(ctx, uint64(cmd.UserID), cmd.AccountType)
	}
	return query.CommandResult{Message: "success"}, nil
}

func (s *Service) HandleNotificationMessage(ctx context.Context, msg command.NotificationMessage) error {
	n := notificationFromCommand(msg)
	n.EnsureCreatedAt(s.clock.Now())
	created, err := s.notifications.CreateOnceWithResult(ctx, &n)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	s.afterNotificationCreated(ctx, &n)
	return nil
}

func (s *Service) afterNotificationCreated(ctx context.Context, n *model.Notification) {
	n.NormalizeAccountType()
	if s.unreadCache != nil {
		s.unreadCache.Invalidate(ctx, uint64(n.ReceiverID), n.ReceiverAccountType)
	}
	if s.realtime == nil {
		return
	}
	unread, err := s.unreadCount(ctx, n.ReceiverID, n.ReceiverAccountType)
	if err != nil {
		unread = 0
	}
	payload, err := json.Marshal(model.RealtimeEvent{
		Type:             "notification_created",
		NotificationType: n.Type,
		NotificationID:   n.ID,
		Unread:           unread,
		Title:            n.Title,
		Content:          n.Content,
		Link:             n.Link,
		CreatedAt:        formatTime(n.CreatedAt),
	})
	if err != nil {
		return
	}
	_ = s.realtime.PublishNotificationEvent(ctx, uint64(n.ReceiverID), n.ReceiverAccountType, string(payload))
}

func (s *Service) unreadCount(ctx context.Context, userID int64, accountType string) (int64, error) {
	if s.unreadCache != nil {
		if count, ok := s.unreadCache.GetUnreadCount(ctx, uint64(userID), accountType); ok {
			return count, nil
		}
	}
	count, err := s.notifications.UnreadCount(ctx, userID, accountType)
	if err != nil {
		return 0, err
	}
	if s.unreadCache != nil {
		s.unreadCache.SetUnreadCount(ctx, uint64(userID), accountType, count)
	}
	return count, nil
}

func notificationFromCommand(cmd command.CreateNotification) model.Notification {
	n := model.Notification{
		EventID:             cmd.EventID,
		ReceiverID:          cmd.ReceiverID,
		ReceiverAccountType: cmd.ReceiverAccountType,
		ReceiverRole:        cmd.ReceiverRole,
		Type:                cmd.Type,
		Title:               cmd.Title,
		Content:             cmd.Content,
		Link:                cmd.Link,
		BizType:             cmd.BizType,
		BizID:               cmd.BizID,
	}
	n.NormalizeAccountType()
	return n
}

func normalizePageSize(pageSize int32) int32 {
	if pageSize < 1 || pageSize > 50 {
		return 20
	}
	return pageSize
}
