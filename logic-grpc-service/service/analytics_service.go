package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/pkg/metadata"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

// AnalyticsService provides scope-aware analytics queries for the dashboard and reports.
type AnalyticsService struct {
	analytics   *repository.AnalyticsRepo
	authzRepo   *repository.AuthzRepo
	serviceAuth *ServiceAuthorizer
}

func NewAnalyticsService(analytics *repository.AnalyticsRepo, authzRepo *repository.AuthzRepo, serviceAuth *ServiceAuthorizer) *AnalyticsService {
	return &AnalyticsService{
		analytics:   analytics,
		authzRepo:   authzRepo,
		serviceAuth: serviceAuth,
	}
}

// getServiceAuth returns the ServiceAuthorizer, creating a nil-safe one if not configured.
func (s *AnalyticsService) getServiceAuth() *ServiceAuthorizer {
	if s.serviceAuth != nil {
		return s.serviceAuth
	}
	return NewServiceAuthorizer(nil, nil)
}

// getUserScopeData loads the user's scope keys, department IDs, and location IDs.
func (s *AnalyticsService) getUserScopeData(ctx context.Context, userID uint64) ([]string, []uint64, []uint64, error) {
	if s.authzRepo == nil {
		return nil, nil, nil, nil
	}
	scopeKeys, err := s.authzRepo.GetUserScopeKeys(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	deptIDs, err := s.authzRepo.GetUserDepartmentIDs(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	locIDs, err := s.authzRepo.GetUserLocationIDs(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	return scopeKeys, deptIDs, locIDs, nil
}

// verifyStaffPermission checks that the authenticated user has the required staff permission.
// Returns the actor user ID and nil on success.
func (s *AnalyticsService) verifyStaffPermission(ctx context.Context, reqUserID int64, permKey string) (uint64, error) {
	actorID := metadata.GetAuthUserID(ctx)
	if actorID == 0 {
		return 0, fmt.Errorf("authenticated user not found in gRPC context")
	}
	if err := s.getServiceAuth().AuthorizePermission(ctx, uint64(actorID), permKey); err != nil {
		return 0, err
	}
	// Verify actor matches request for safety
	if reqUserID > 0 && reqUserID != actorID {
		// Grant if the actor has system_all scope
		if s.authzRepo != nil {
			scopeKeys, err := s.authzRepo.GetUserScopeKeys(ctx, uint64(actorID))
			if err == nil {
				for _, sk := range scopeKeys {
					if sk == authz.ScopeSystemAll {
						return uint64(actorID), nil
					}
				}
			}
		}
		return 0, fmt.Errorf("actor mismatch: authenticated user %d != requested user %d", actorID, reqUserID)
	}
	return uint64(actorID), nil
}

// GetDashboardReport returns the scoped dashboard KPI and distribution data.
func (s *AnalyticsService) GetDashboardReport(ctx context.Context, req *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	actorID, err := s.verifyStaffPermission(ctx, req.StaffUserId, authz.PermJobRead)
	if err != nil {
		return &pb.GetDashboardReportResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	scopeKeys, deptIDs, locIDs, err := s.getUserScopeData(ctx, actorID)
	if err != nil {
		logger.L().Error("analytics: scope lookup failed", zap.Error(err))
		return nil, err
	}

	// Dashboard KPI
	kpi, err := s.analytics.GetDashboardKPI(ctx, actorID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		logger.L().Error("analytics: dashboard KPI query failed", zap.Error(err))
		return &pb.GetDashboardReportResponse{Code: errs.ErrInternal, Msg: "查询工作台数据失败"}, nil
	}

	// Unread notifications
	unreadCount, err := s.analytics.GetUnreadNotificationCount(ctx, int64(actorID), "staff")
	if err != nil {
		logger.L().Warn("analytics: unread count query failed", zap.Error(err))
		unreadCount = 0
	}

	// Trend
	trend, err := s.analytics.GetTrend(ctx, actorID, scopeKeys, deptIDs, locIDs, 7)
	if err != nil {
		logger.L().Warn("analytics: trend query failed", zap.Error(err))
		trend = nil
	}

	// Stage distribution
	stageDist, err := s.analytics.GetStageDistribution(ctx, actorID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		logger.L().Warn("analytics: stage distribution query failed", zap.Error(err))
		stageDist = nil
	}

	// Build response
	trendPoints := make([]*pb.DashboardTrendPoint, len(trend))
	for i, tp := range trend {
		trendPoints[i] = &pb.DashboardTrendPoint{
			Date:         tp.Date,
			Applications: tp.Applications,
		}
	}

	stageItems := make([]*pb.DashboardReportItem, len(stageDist))
	for i, s := range stageDist {
		stageItems[i] = &pb.DashboardReportItem{
			StageKey:   s.StageKey,
			StageLabel: stageLabel(s.StageKey),
			Count:      s.Count,
		}
	}

	return &pb.GetDashboardReportResponse{
		Code:               errs.OK,
		Msg:                "success",
		OnlineJobs:         kpi.OnlineJobs,
		OfflineJobs:        kpi.OfflineJobs,
		TotalApplications:  kpi.TotalApplications,
		TodayApplications:  kpi.TodayApplications,
		UnreadNotifications: unreadCount,
		PendingActions:     kpi.PendingActions,
		Trend:              trendPoints,
		StageDistribution:   stageItems,
	}, nil
}

// GetFunnelReport returns the scoped recruitment funnel with conversion rates.
func (s *AnalyticsService) GetFunnelReport(ctx context.Context, req *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	actorID, err := s.verifyStaffPermission(ctx, req.StaffUserId, authz.PermApplicationRead)
	if err != nil {
		return &pb.GetFunnelReportResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	scopeKeys, deptIDs, locIDs, err := s.getUserScopeData(ctx, actorID)
	if err != nil {
		logger.L().Error("analytics: scope lookup failed", zap.Error(err))
		return nil, err
	}

	var startDate, endDate *time.Time
	if req.StartDate != "" {
		t, err := time.Parse(time.RFC3339, req.StartDate)
		if err == nil {
			startDate = &t
		}
	}
	if req.EndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EndDate)
		if err == nil {
			endDate = &t
		}
	}

	stages, err := s.analytics.GetFunnelReport(ctx, actorID, scopeKeys, deptIDs, locIDs, startDate, endDate, req.JobId)
	if err != nil {
		logger.L().Error("analytics: funnel query failed", zap.Error(err))
		return &pb.GetFunnelReportResponse{Code: errs.ErrInternal, Msg: "查询漏斗数据失败"}, nil
	}

	// Define funnel order
	funnelOrder := []string{"applied", "viewed", "screening", "screen_passed", "interview_pending", "interviewing", "interview_passed", "offer_pending", "offer_sent", "hired"}
	stageMap := make(map[string]int64)
	for _, s := range stages {
		stageMap[s.StageKey] = s.Count
	}

	var funnelStages []*pb.FunnelStage
	var prevCount int64
	first := true
	for _, key := range funnelOrder {
		count, ok := stageMap[key]
		if !ok {
			continue
		}
		var rate float64
		if first {
			rate = 100.0
			prevCount = count
			first = false
		} else if prevCount > 0 {
			rate = float64(count) / float64(prevCount) * 100.0
			prevCount = count
		}
		funnelStages = append(funnelStages, &pb.FunnelStage{
			StageKey:       key,
			StageLabel:     stageLabel(key),
			Count:          count,
			ConversionRate: rate,
		})
	}

	return &pb.GetFunnelReportResponse{
		Code:   errs.OK,
		Msg:    "success",
		Stages: funnelStages,
	}, nil
}

// GetTimeInStageReport returns the average time spent in each pipeline stage.
func (s *AnalyticsService) GetTimeInStageReport(ctx context.Context, req *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	actorID, err := s.verifyStaffPermission(ctx, req.StaffUserId, authz.PermApplicationRead)
	if err != nil {
		return &pb.GetTimeInStageReportResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	scopeKeys, deptIDs, locIDs, err := s.getUserScopeData(ctx, actorID)
	if err != nil {
		logger.L().Error("analytics: scope lookup failed", zap.Error(err))
		return nil, err
	}

	var startDate, endDate *time.Time
	if req.StartDate != "" {
		t, err := time.Parse(time.RFC3339, req.StartDate)
		if err == nil {
			startDate = &t
		}
	}
	if req.EndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EndDate)
		if err == nil {
			endDate = &t
		}
	}

	durations, err := s.analytics.GetTimeInStage(ctx, actorID, scopeKeys, deptIDs, locIDs, startDate, endDate, req.JobId)
	if err != nil {
		logger.L().Error("analytics: time-in-stage query failed", zap.Error(err))
		return &pb.GetTimeInStageReportResponse{Code: errs.ErrInternal, Msg: "查询阶段耗时失败"}, nil
	}

	stageItems := make([]*pb.StageDuration, len(durations))
	for i, d := range durations {
		stageItems[i] = &pb.StageDuration{
			StageKey:        d.FromStatus,
			StageLabel:      stageLabel(d.FromStatus),
			AvgHours:        d.AvgDurationSecs / 3600.0,
			TransitionCount: d.TransitionCount,
		}
	}

	return &pb.GetTimeInStageReportResponse{
		Code:      errs.OK,
		Msg:       "success",
		Durations: stageItems,
	}, nil
}

// GetInterviewOfferMetrics returns interview pass rate and offer acceptance rate.
func (s *AnalyticsService) GetInterviewOfferMetrics(ctx context.Context, req *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	actorID, err := s.verifyStaffPermission(ctx, req.StaffUserId, authz.PermApplicationRead)
	if err != nil {
		return &pb.GetInterviewOfferMetricsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	scopeKeys, deptIDs, locIDs, err := s.getUserScopeData(ctx, actorID)
	if err != nil {
		logger.L().Error("analytics: scope lookup failed", zap.Error(err))
		return nil, err
	}

	var startDate, endDate *time.Time
	if req.StartDate != "" {
		t, err := time.Parse(time.RFC3339, req.StartDate)
		if err == nil {
			startDate = &t
		}
	}
	if req.EndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EndDate)
		if err == nil {
			endDate = &t
		}
	}

	interviewMetrics, err := s.analytics.GetInterviewMetrics(ctx, actorID, scopeKeys, deptIDs, locIDs, startDate, endDate, req.JobId)
	if err != nil {
		logger.L().Error("analytics: interview metrics query failed", zap.Error(err))
		return &pb.GetInterviewOfferMetricsResponse{Code: errs.ErrInternal, Msg: "查询面试指标失败"}, nil
	}

	offerMetrics, err := s.analytics.GetOfferMetrics(ctx, actorID, scopeKeys, deptIDs, locIDs, startDate, endDate, req.JobId)
	if err != nil {
		logger.L().Error("analytics: offer metrics query failed", zap.Error(err))
		return &pb.GetInterviewOfferMetricsResponse{Code: errs.ErrInternal, Msg: "查询Offer指标失败"}, nil
	}

	passRate := 0.0
	if interviewMetrics.CompletedInterviews > 0 {
		passRate = float64(interviewMetrics.PositiveFeedbacks) / float64(interviewMetrics.CompletedInterviews) * 100.0
	}

	acceptanceRate := 0.0
	if offerMetrics.TotalOffers > 0 {
		acceptanceRate = float64(offerMetrics.AcceptedOffers) / float64(offerMetrics.TotalOffers) * 100.0
	}

	return &pb.GetInterviewOfferMetricsResponse{
		Code:                errs.OK,
		Msg:                 "success",
		TotalInterviews:     interviewMetrics.TotalInterviews,
		CompletedInterviews: interviewMetrics.CompletedInterviews,
		PositiveFeedbacks:   interviewMetrics.PositiveFeedbacks,
		PassRate:            passRate,
		TotalOffers:         offerMetrics.TotalOffers,
		AcceptedOffers:      offerMetrics.AcceptedOffers,
		RejectedOffers:      offerMetrics.RejectedOffers,
		AcceptanceRate:      acceptanceRate,
	}, nil
}

// QueryAuthAuditLogs returns authorization audit logs.
func (s *AnalyticsService) QueryAuthAuditLogs(ctx context.Context, req *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	actorID := metadata.GetAuthUserID(ctx)
	if actorID == 0 {
		return &pb.QueryAuthAuditLogsResponse{Code: errs.ErrForbidden, Msg: "authenticated user not found in gRPC context"}, nil
	}
	if err := s.getServiceAuth().AuthorizePermission(ctx, uint64(actorID), authz.PermAuditSecurityRead); err != nil {
		return &pb.QueryAuthAuditLogsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var actorUserID *uint64
	if req.ActorUserId != 0 {
		uid := uint64(req.ActorUserId)
		actorUserID = &uid
	}

	logs, total, err := s.authzRepo.QueryAuthAuditLogs(ctx, actorUserID, req.PermissionKey, req.Decision, (page-1)*pageSize, pageSize)
	if err != nil {
		logger.L().Error("analytics: auth audit query failed", zap.Error(err))
		return &pb.QueryAuthAuditLogsResponse{Code: errs.ErrInternal, Msg: "查询安全审计日志失败"}, nil
	}

	list := make([]*pb.AuthAuditLogItem, len(logs))
	for i, l := range logs {
		list[i] = &pb.AuthAuditLogItem{
			Id:            l.ID,
			ActorUserId:   int64(l.ActorUserID),
			ActorRoles:    l.ActorRoles,
			PermissionKey: l.PermissionKey,
			ResourceType:  l.ResourceType,
			ResourceId:    l.ResourceID,
			Decision:      l.Decision,
			Reason:        l.Reason,
			RequestId:     l.RequestID,
			ClientIp:      l.ClientIP,
			CreatedAt:     l.CreatedAt.Format(time.RFC3339),
		}
	}

	return &pb.QueryAuthAuditLogsResponse{
		Code:  errs.OK,
		Msg:   "success",
		Total: total,
		List:  list,
	}, nil
}

// stageLabel returns a Chinese label for the given status key.
func stageLabel(key string) string {
	switch key {
	case "applied":
		return "已投递"
	case "viewed":
		return "已查看"
	case "screening":
		return "筛选中"
	case "screen_passed":
		return "筛选通过"
	case "interview_pending":
		return "待安排面试"
	case "interviewing":
		return "面试中"
	case "interview_passed":
		return "面试通过"
	case "offer_pending":
		return "待发Offer"
	case "offer_sent":
		return "Offer已发"
	case "hired":
		return "已入职"
	case "rejected":
		return "淘汰"
	case "withdrawn":
		return "候选人撤回"
	default:
		return key
	}
}
