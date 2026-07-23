package mapper

import (
	"time"

	"smart-recruit-notification-service/internal/application/command"
	"smart-recruit-notification-service/internal/application/query"
	"smart-recruit-notification-service/internal/domain/model"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-proto/recruitment/pb"
)

const successMessage = "success"

func ListRequest(req *pb.ListNotificationsRequest) query.ListNotifications {
	return query.ListNotifications{
		UserID:      req.GetUserId(),
		AccountType: req.GetAccountType(),
		Page:        req.GetPage(),
		PageSize:    req.GetPageSize(),
		Cursor:      req.GetCursor(),
	}
}

func UnreadRequest(req *pb.UnreadNotificationCountRequest) query.UnreadCount {
	return query.UnreadCount{UserID: req.GetUserId(), AccountType: req.GetAccountType()}
}

func SummaryRequest(req *pb.NotificationSummaryRequest) query.Summary {
	return query.Summary{UserID: req.GetUserId(), AccountType: req.GetAccountType()}
}

func MarkReadRequest(req *pb.MarkNotificationReadRequest) command.MarkRead {
	return command.MarkRead{
		UserID:         req.GetUserId(),
		AccountType:    req.GetAccountType(),
		NotificationID: req.GetNotificationId(),
	}
}

func MarkAllReadRequest(req *pb.MarkAllNotificationsReadRequest) command.MarkAllRead {
	return command.MarkAllRead{UserID: req.GetUserId(), AccountType: req.GetAccountType()}
}

func ListResponse(result query.ListResult) *pb.ListNotificationsResponse {
	code, msg := codeMessage(result.Code, result.Message)
	return &pb.ListNotificationsResponse{
		Code:       code,
		Msg:        msg,
		Total:      result.Total,
		List:       Notifications(result.List),
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
	}
}

func UnreadResponse(result query.UnreadCountResult) *pb.UnreadNotificationCountResponse {
	return &pb.UnreadNotificationCountResponse{Code: query.CodeOK, Msg: successMessage, Unread: result.Unread}
}

func SummaryResponse(result query.SummaryResult) *pb.NotificationSummaryResponse {
	return &pb.NotificationSummaryResponse{
		Code:                 query.CodeOK,
		Msg:                  successMessage,
		Unread:               result.Unread,
		LatestNotificationId: result.LatestNotificationID,
		LatestCreatedAt:      result.LatestCreatedAt,
	}
}

func CommonResponse(result query.CommandResult) *pb.CommonResponse {
	code, msg := codeMessage(result.Code, result.Message)
	return &pb.CommonResponse{Code: code, Msg: msg}
}

func Notifications(rows []model.Notification) []*pb.Notification {
	out := make([]*pb.Notification, 0, len(rows))
	for _, row := range rows {
		out = append(out, Notification(row))
	}
	return out
}

func Notification(row model.Notification) *pb.Notification {
	return &pb.Notification{
		NotificationId: row.ID,
		Type:           row.Type,
		Title:          row.Title,
		Content:        row.Content,
		Link:           row.Link,
		BizType:        row.BizType,
		BizId:          row.BizID,
		IsRead:         row.IsRead,
		CreatedAt:      formatTime(row.CreatedAt),
		ReadAt:         formatOptionalTime(row.ReadAt),
	}
}

func codeMessage(code int32, msg string) (int32, string) {
	if msg == "" {
		msg = successMessage
	}
	return code, msg
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return businessclock.FormatRFC3339(t)
}

func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}
