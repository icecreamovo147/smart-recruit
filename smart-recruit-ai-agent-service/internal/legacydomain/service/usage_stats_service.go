package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"smart-recruit-ai-agent-service/internal/legacydomain/repository"
	"smart-recruit-domain-go/pkg/authz"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/logger"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

// UsageStatsService provides aggregated AI usage statistics and trends.
type UsageStatsService struct {
	repo        *repository.UsageStatsRepo
	serviceAuth *ServiceAuthorizer
}

func NewUsageStatsService(repo *repository.UsageStatsRepo, serviceAuth *ServiceAuthorizer) *UsageStatsService {
	return &UsageStatsService{repo: repo, serviceAuth: serviceAuth}
}

// verifyPermission checks that the caller has AUDIT_USAGE_READ permission.
func (s *UsageStatsService) verifyPermission(ctx context.Context) error {
	actorID := metadata.GetAuthUserID(ctx)
	if actorID == 0 {
		return fmt.Errorf("authenticated user not found in context — gRPC metadata x-authenticated-user-id is required")
	}
	if s.serviceAuth == nil {
		return nil
	}
	return s.serviceAuth.AuthorizePermission(ctx, uint64(actorID), authz.PermAuditUsageRead)
}

// GetUsageStats returns aggregated usage stats by the specified dimension.
func (s *UsageStatsService) GetUsageStats(ctx context.Context, req *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error) {
	if err := s.verifyPermission(ctx); err != nil {
		return &pb.GetUsageStatsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	startTime, endTime, err := parseTimeRange(req.StartTime, req.EndTime, 30)
	if err != nil {
		return &pb.GetUsageStatsResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	dimension := req.Dimension
	if dimension == "" {
		dimension = "model"
	}

	var rows []repository.UsageStatsRow
	switch dimension {
	case "user":
		rows, err = s.repo.GetStatsByUser(ctx, startTime, endTime)
	case "session":
		rows, err = s.repo.GetStatsBySession(ctx, startTime, endTime)
	default: // model
		rows, err = s.repo.GetStatsByModel(ctx, startTime, endTime)
	}
	if err != nil {
		logger.L().Error("get usage stats failed", zap.String("dimension", dimension), zap.Error(err))
		return nil, err
	}

	items := make([]*pb.UsageStatsItem, len(rows))
	for i, r := range rows {
		items[i] = &pb.UsageStatsItem{
			Name:          r.Name,
			TotalTokens:   r.TotalTokens,
			CallCount:     r.CallCount,
			AvgCostMs:     r.AvgCostMs,
			EstimatedCost: r.EstimatedCost,
		}
	}
	return &pb.GetUsageStatsResponse{Code: errs.OK, Msg: "success", List: items}, nil
}

// GetUsageTrend returns time-series usage data.
func (s *UsageStatsService) GetUsageTrend(ctx context.Context, req *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error) {
	if err := s.verifyPermission(ctx); err != nil {
		return &pb.GetUsageTrendResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	startTime, endTime, err := parseTimeRange(req.StartTime, req.EndTime, 30)
	if err != nil {
		return &pb.GetUsageTrendResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	granularity := req.Granularity
	if granularity == "" {
		granularity = "day"
	}

	rows, err := s.repo.GetTrend(ctx, startTime, endTime, granularity)
	if err != nil {
		logger.L().Error("get usage trend failed", zap.String("granularity", granularity), zap.Error(err))
		return nil, err
	}

	points := make([]*pb.UsageTrendPoint, len(rows))
	for i, r := range rows {
		points[i] = &pb.UsageTrendPoint{
			Date:          r.Date,
			TotalTokens:   r.TotalTokens,
			CallCount:     r.CallCount,
			AvgCostMs:     r.AvgCostMs,
			EstimatedCost: r.EstimatedCost,
		}
	}
	return &pb.GetUsageTrendResponse{Code: errs.OK, Msg: "success", List: points}, nil
}

// parseTimeRange parses start/end time with a default look-back period.
func parseTimeRange(startStr, endStr string, defaultDays int) (time.Time, time.Time, error) {
	now := time.Now()
	var startTime, endTime time.Time
	var err error

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return startTime, endTime, err
		}
	} else {
		endTime = now
	}

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			return startTime, endTime, err
		}
	} else {
		startTime = endTime.AddDate(0, 0, -defaultDays)
	}

	return startTime, endTime, nil
}
