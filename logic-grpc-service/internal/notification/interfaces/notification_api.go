package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// NotificationAPI is the Notification-owned gRPC method contract implemented by current services.
type NotificationAPI interface {
	ListNotifications(context.Context, *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error)
	UnreadNotificationCount(context.Context, *pb.UnreadNotificationCountRequest) (*pb.UnreadNotificationCountResponse, error)
	NotificationSummary(context.Context, *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error)
	MarkNotificationRead(context.Context, *pb.MarkNotificationReadRequest) (*pb.CommonResponse, error)
	MarkAllNotificationsRead(context.Context, *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error)
}
