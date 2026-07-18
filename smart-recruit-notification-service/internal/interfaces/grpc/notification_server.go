package grpc

import (
	"context"

	appservice "smart-recruit-notification-service/internal/application/service"
	"smart-recruit-notification-service/internal/interfaces/mapper"
	"smart-recruit-proto/recruitment/pb"
)

type Server struct {
	pb.UnimplementedNotificationServiceServer
	notifications *appservice.Service
}

func NewServer(notifications *appservice.Service) *Server {
	return &Server{notifications: notifications}
}

func (s *Server) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	result, err := s.notifications.ListNotifications(ctx, mapper.ListRequest(req))
	if err != nil {
		return nil, err
	}
	return mapper.ListResponse(result), nil
}

func (s *Server) UnreadNotificationCount(ctx context.Context, req *pb.UnreadNotificationCountRequest) (*pb.UnreadNotificationCountResponse, error) {
	result, err := s.notifications.UnreadNotificationCount(ctx, mapper.UnreadRequest(req))
	if err != nil {
		return nil, err
	}
	return mapper.UnreadResponse(result), nil
}

func (s *Server) NotificationSummary(ctx context.Context, req *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error) {
	result, err := s.notifications.NotificationSummary(ctx, mapper.SummaryRequest(req))
	if err != nil {
		return nil, err
	}
	return mapper.SummaryResponse(result), nil
}

func (s *Server) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationReadRequest) (*pb.CommonResponse, error) {
	result, err := s.notifications.MarkNotificationRead(ctx, mapper.MarkReadRequest(req))
	if err != nil {
		return nil, err
	}
	return mapper.CommonResponse(result), nil
}

func (s *Server) MarkAllNotificationsRead(ctx context.Context, req *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error) {
	result, err := s.notifications.MarkAllNotificationsRead(ctx, mapper.MarkAllReadRequest(req))
	if err != nil {
		return nil, err
	}
	return mapper.CommonResponse(result), nil
}
