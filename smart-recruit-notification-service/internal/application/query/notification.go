package query

import "smart-recruit-notification-service/internal/domain/model"

const (
	CodeOK         int32 = 0
	CodeBadRequest int32 = 400
	CodeForbidden  int32 = 403
)

type ListNotifications struct {
	UserID      int64
	AccountType string
	Page        int32
	PageSize    int32
	Cursor      string
}

type UnreadCount struct {
	UserID      int64
	AccountType string
}

type Summary struct {
	UserID      int64
	AccountType string
}

type ListResult struct {
	Code       int32
	Message    string
	List       []model.Notification
	Total      int64
	NextCursor string
	HasMore    bool
}

type UnreadCountResult struct {
	Unread int64
}

type SummaryResult struct {
	Unread               int64
	LatestNotificationID int64
	LatestCreatedAt      string
}

type CommandResult struct {
	Code    int32
	Message string
}
