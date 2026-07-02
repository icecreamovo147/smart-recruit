package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/service"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	pb.UnimplementedJobServiceServer
	pb.UnimplementedCandidateServiceServer
	pb.UnimplementedApplicationServiceServer
	pb.UnimplementedAIServiceServer
	pb.UnimplementedNotificationServiceServer
	pb.UnimplementedInterviewServiceServer
	pb.UnimplementedOfferServiceServer
	pb.UnimplementedAdminServiceServer
	pb.UnimplementedCollaborationServiceServer
	pb.UnimplementedLlmConfigServiceServer
	pb.UnimplementedPromptServiceServer
	pb.UnimplementedAgentConfigServiceServer
	pb.UnimplementedMCPServiceServer
	pb.UnimplementedSkillServiceServer
	pb.UnimplementedAgentSkillServiceServer
	pb.UnimplementedRecruitingIntelligenceServiceServer
	svc *service.Services
}

func New(svc *service.Services) *Server {
	return &Server{svc: svc}
}

// Auth

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return s.svc.Auth.Register(ctx, req)
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.svc.Auth.Login(ctx, req)
}

func (s *Server) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return s.svc.Auth.RefreshToken(ctx, req)
}

func (s *Server) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return s.svc.Auth.RevokeRefreshToken(ctx, req)
}

func (s *Server) RecordAuthDecision(ctx context.Context, req *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return s.svc.Auth.RecordAuthDecision(ctx, req)
}

func (s *Server) GetPrincipal(ctx context.Context, req *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return s.svc.Auth.GetPrincipal(ctx, req)
}

func (s *Server) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return s.svc.Auth.UpdateEmail(ctx, req)
}

// Job

func (s *Server) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	return s.svc.Job.CreateJob(ctx, req)
}

func (s *Server) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	return s.svc.Job.UpdateJob(ctx, req)
}

func (s *Server) OfflineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return s.svc.Job.OfflineJob(ctx, req)
}

func (s *Server) OnlineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return s.svc.Job.OnlineJob(ctx, req)
}

func (s *Server) ListHRJobs(ctx context.Context, req *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return s.svc.Job.ListHRJobs(ctx, req)
}

func (s *Server) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	return s.svc.Job.ListPublicJobs(ctx, req)
}

func (s *Server) GetJobDetail(ctx context.Context, req *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return s.svc.Job.GetJobDetail(ctx, req)
}

func (s *Server) ListJobOptions(ctx context.Context, req *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	return s.svc.Taxonomy.ListJobOptions(ctx, req)
}

func (s *Server) ListDepartmentLocations(ctx context.Context, req *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	return s.svc.Taxonomy.ListDepartmentLocations(ctx, req)
}

// Candidate

func (s *Server) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return s.svc.Candidate.GetProfile(ctx, req)
}

func (s *Server) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	return s.svc.Candidate.UpdateProfile(ctx, req)
}

func (s *Server) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	return s.svc.Candidate.GetResume(ctx, req)
}

func (s *Server) PresignResumeUpload(ctx context.Context, req *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	return s.svc.Candidate.PresignResumeUpload(ctx, req)
}

func (s *Server) ConfirmResumeUpload(ctx context.Context, req *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	return s.svc.Candidate.ConfirmResumeUpload(ctx, req)
}

// Application

func (s *Server) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	return s.svc.Application.ApplyJob(ctx, req)
}

func (s *Server) ListMyApplications(ctx context.Context, req *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	return s.svc.Application.ListMyApplications(ctx, req)
}

func (s *Server) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return s.svc.Application.ListJobApplications(ctx, req)
}

func (s *Server) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	return s.svc.Application.UpdateApplicationStatus(ctx, req)
}

func (s *Server) ListApplicationStatusTransitions(ctx context.Context, req *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	return s.svc.Application.ListApplicationStatusTransitions(ctx, req)
}

// AI

func (s *Server) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	return s.svc.AI.Chat(ctx, req)
}

func (s *Server) ChatStream(req *pb.ChatRequest, stream pb.AIService_ChatStreamServer) error {
	return s.svc.AI.ChatStream(req, stream)
}

func (s *Server) History(ctx context.Context, req *pb.ChatHistoryRequest) (*pb.ChatHistoryResponse, error) {
	return s.svc.AI.History(ctx, req)
}

func (s *Server) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	return s.svc.AI.AnalyzeApplication(ctx, req)
}

func (s *Server) ListChatSessions(ctx context.Context, req *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error) {
	return s.svc.AI.ListChatSessions(ctx, req)
}

func (s *Server) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error) {
	return s.svc.AI.CreateChatSession(ctx, req)
}

func (s *Server) SessionMessages(ctx context.Context, req *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	return s.svc.AI.SessionMessages(ctx, req)
}

func (s *Server) CreateApplicationAnalysisSession(ctx context.Context, req *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	return s.svc.AI.CreateApplicationAnalysisSession(ctx, req)
}

func (s *Server) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.CommonResponse, error) {
	return s.svc.AI.UpdateSession(ctx, req)
}

func (s *Server) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.CommonResponse, error) {
	return s.svc.AI.DeleteSession(ctx, req)
}

func (s *Server) GetToolTraces(ctx context.Context, req *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error) {
	return s.svc.AI.GetToolTraces(ctx, req)
}

func (s *Server) GetAgentRuns(ctx context.Context, req *pb.GetAgentRunsRequest) (*pb.GetAgentRunsResponse, error) {
	return s.svc.AI.GetAgentRuns(ctx, req)
}

// Candidate AI

func (s *Server) CandidateChatStream(req *pb.CandidateChatRequest, stream pb.AIService_CandidateChatStreamServer) error {
	return s.svc.CandidateAI.StreamChatGRPC(req, stream)
}

func (s *Server) CandidateListSessions(ctx context.Context, req *pb.CandidateSessionListRequest) (*pb.ChatSessionListResponse, error) {
	return s.svc.CandidateAI.ListSessionsGRPC(ctx, req)
}

func (s *Server) CandidateCreateSession(ctx context.Context, req *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error) {
	return s.svc.CandidateAI.CreateSessionGRPC(ctx, req)
}

func (s *Server) CandidateSessionMessages(ctx context.Context, req *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	return s.svc.CandidateAI.SessionMessagesGRPC(ctx, req)
}

func (s *Server) CandidateUpdateSession(ctx context.Context, req *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error) {
	return s.svc.CandidateAI.UpdateSessionGRPC(ctx, req)
}

func (s *Server) CandidateDeleteSession(ctx context.Context, req *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error) {
	return s.svc.CandidateAI.DeleteSessionGRPC(ctx, req)
}

// Notification

func (s *Server) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	return s.svc.Notification.ListNotifications(ctx, req)
}

func (s *Server) UnreadNotificationCount(ctx context.Context, req *pb.UnreadNotificationCountRequest) (*pb.UnreadNotificationCountResponse, error) {
	return s.svc.Notification.UnreadNotificationCount(ctx, req)
}

func (s *Server) NotificationSummary(ctx context.Context, req *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error) {
	return s.svc.Notification.NotificationSummary(ctx, req)
}

func (s *Server) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationReadRequest) (*pb.CommonResponse, error) {
	return s.svc.Notification.MarkNotificationRead(ctx, req)
}

func (s *Server) MarkAllNotificationsRead(ctx context.Context, req *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error) {
	return s.svc.Notification.MarkAllNotificationsRead(ctx, req)
}

// Offer

func (s *Server) CreateOffer(ctx context.Context, req *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	return s.svc.Offer.CreateOffer(ctx, req)
}

func (s *Server) UpdateOffer(ctx context.Context, req *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	return s.svc.Offer.UpdateOffer(ctx, req)
}

func (s *Server) GetOffer(ctx context.Context, req *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	return s.svc.Offer.GetOffer(ctx, req)
}

func (s *Server) ListOffersByApplication(ctx context.Context, req *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	return s.svc.Offer.ListOffersByApplication(ctx, req)
}

func (s *Server) SendOffer(ctx context.Context, req *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	return s.svc.Offer.SendOffer(ctx, req)
}

func (s *Server) WithdrawOffer(ctx context.Context, req *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	return s.svc.Offer.WithdrawOffer(ctx, req)
}

func (s *Server) AcceptOffer(ctx context.Context, req *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	return s.svc.Offer.AcceptOffer(ctx, req)
}

func (s *Server) RejectOffer(ctx context.Context, req *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	return s.svc.Offer.RejectOffer(ctx, req)
}

func (s *Server) ListMyOffers(ctx context.Context, req *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	return s.svc.Offer.ListMyOffers(ctx, req)
}

func (s *Server) ListOfferEvents(ctx context.Context, req *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	return s.svc.Offer.ListOfferEvents(ctx, req)
}

// Interview

func (s *Server) ScheduleInterview(ctx context.Context, req *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	return s.svc.Interview.ScheduleInterview(ctx, req)
}

func (s *Server) UpdateInterview(ctx context.Context, req *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	return s.svc.Interview.UpdateInterview(ctx, req)
}

func (s *Server) CancelInterview(ctx context.Context, req *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	return s.svc.Interview.CancelInterview(ctx, req)
}

func (s *Server) BatchCancelInterviews(ctx context.Context, req *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error) {
	return s.svc.Interview.BatchCancelInterviews(ctx, req)
}

func (s *Server) GetInterview(ctx context.Context, req *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	return s.svc.Interview.GetInterview(ctx, req)
}

func (s *Server) ListInterviewers(ctx context.Context, req *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error) {
	return s.svc.Interview.ListInterviewers(ctx, req)
}

func (s *Server) ListApplicationInterviews(ctx context.Context, req *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	return s.svc.Interview.ListApplicationInterviews(ctx, req)
}

func (s *Server) ListMyInterviews(ctx context.Context, req *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	return s.svc.Interview.ListMyInterviews(ctx, req)
}

func (s *Server) ListCandidateInterviews(ctx context.Context, req *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	return s.svc.Interview.ListCandidateInterviews(ctx, req)
}

func (s *Server) SubmitFeedback(ctx context.Context, req *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	return s.svc.Interview.SubmitFeedback(ctx, req)
}

func (s *Server) GetFeedback(ctx context.Context, req *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	return s.svc.Interview.GetFeedback(ctx, req)
}

// Admin

func (s *Server) CreateInviteCode(ctx context.Context, req *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error) {
	return s.svc.Admin.CreateInviteCode(ctx, req)
}

func (s *Server) ListInviteCodes(ctx context.Context, req *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error) {
	return s.svc.Admin.ListInviteCodes(ctx, req)
}

func (s *Server) ExtendInviteCode(ctx context.Context, req *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error) {
	return s.svc.Admin.ExtendInviteCode(ctx, req)
}

func (s *Server) RevokeInviteCode(ctx context.Context, req *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error) {
	return s.svc.Admin.RevokeInviteCode(ctx, req)
}

func (s *Server) ReactivateInviteCode(ctx context.Context, req *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error) {
	return s.svc.Admin.ReactivateInviteCode(ctx, req)
}

func (s *Server) ValidateInviteCode(ctx context.Context, req *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {
	return s.svc.Admin.ValidateInviteCode(ctx, req)
}

// Admin — Department taxonomy

func (s *Server) ListDepartments(ctx context.Context, req *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	return s.svc.Taxonomy.ListDepartments(ctx, req)
}

func (s *Server) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return s.svc.Taxonomy.CreateDepartment(ctx, req)
}

func (s *Server) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return s.svc.Taxonomy.UpdateDepartment(ctx, req)
}

func (s *Server) UpdateDepartmentStatus(ctx context.Context, req *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error) {
	return s.svc.Taxonomy.UpdateDepartmentStatus(ctx, req)
}

func (s *Server) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error) {
	return s.svc.Taxonomy.DeleteDepartment(ctx, req)
}

// Admin — Job location taxonomy

func (s *Server) ListJobLocations(ctx context.Context, req *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error) {
	return s.svc.Taxonomy.ListJobLocations(ctx, req)
}

func (s *Server) CreateJobLocation(ctx context.Context, req *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return s.svc.Taxonomy.CreateJobLocation(ctx, req)
}

func (s *Server) UpdateJobLocation(ctx context.Context, req *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return s.svc.Taxonomy.UpdateJobLocation(ctx, req)
}

func (s *Server) UpdateJobLocationStatus(ctx context.Context, req *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error) {
	return s.svc.Taxonomy.UpdateJobLocationStatus(ctx, req)
}

func (s *Server) DeleteJobLocation(ctx context.Context, req *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error) {
	return s.svc.Taxonomy.DeleteJobLocation(ctx, req)
}

// Admin — Department location config

func (s *Server) GetDepartmentLocationConfig(ctx context.Context, req *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return s.svc.Taxonomy.GetDepartmentLocationConfig(ctx, req)
}

func (s *Server) UpdateDepartmentLocationConfig(ctx context.Context, req *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return s.svc.Taxonomy.UpdateDepartmentLocationConfig(ctx, req)
}

func (s *Server) ListDepartmentsLocationMap(ctx context.Context, req *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error) {
	return s.svc.Taxonomy.ListDepartmentsLocationMap(ctx, req)
}

// Admin — Usage Audit

func (s *Server) QueryUsageLogs(ctx context.Context, req *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error) {
	return s.svc.Admin.QueryUsageLogs(ctx, req)
}

// ── RBAC Admin dispatches ───────────────────────────────────────────────

func (s *Server) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return s.svc.Admin.ListRoles(ctx, req)
}

func (s *Server) ListPermissions(ctx context.Context, req *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	return s.svc.Admin.ListPermissions(ctx, req)
}

func (s *Server) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return s.svc.Admin.GetUserRoles(ctx, req)
}

func (s *Server) AssignUserRole(ctx context.Context, req *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	return s.svc.Admin.AssignUserRole(ctx, req)
}

func (s *Server) RevokeUserRole(ctx context.Context, req *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	return s.svc.Admin.RevokeUserRole(ctx, req)
}

func (s *Server) AssignDataScope(ctx context.Context, req *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	return s.svc.Admin.AssignDataScope(ctx, req)
}

func (s *Server) RevokeDataScope(ctx context.Context, req *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	return s.svc.Admin.RevokeDataScope(ctx, req)
}

func (s *Server) ListStaffUsers(ctx context.Context, req *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	return s.svc.Admin.ListStaffUsers(ctx, req)
}

func (s *Server) CreateStaffUser(ctx context.Context, req *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	return s.svc.Admin.CreateStaffUser(ctx, req)
}

// ── Phase 6: Analytics & AI Audit ────────────────────────────────────

func (s *Server) QueryAuthAuditLogs(ctx context.Context, req *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return s.svc.Analytics.QueryAuthAuditLogs(ctx, req)
}

func (s *Server) GetDashboardReport(ctx context.Context, req *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	return s.svc.Analytics.GetDashboardReport(ctx, req)
}

func (s *Server) GetFunnelReport(ctx context.Context, req *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	return s.svc.Analytics.GetFunnelReport(ctx, req)
}

func (s *Server) GetTimeInStageReport(ctx context.Context, req *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	return s.svc.Analytics.GetTimeInStageReport(ctx, req)
}

func (s *Server) GetInterviewOfferMetrics(ctx context.Context, req *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	return s.svc.Analytics.GetInterviewOfferMetrics(ctx, req)
}

// ── Phase 4: Collaboration ────────────────────────────────────────────

func (s *Server) GetCandidateWorkspace(ctx context.Context, req *pb.GetCandidateWorkspaceRequest) (*pb.GetCandidateWorkspaceResponse, error) {
	return s.svc.Collaboration.GetCandidateWorkspace(ctx, req)
}

func (s *Server) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.CreateNoteResponse, error) {
	return s.svc.Collaboration.CreateNote(ctx, req)
}

func (s *Server) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	return s.svc.Collaboration.ListNotes(ctx, req)
}

func (s *Server) CreateTag(ctx context.Context, req *pb.CreateTagRequest) (*pb.CreateTagResponse, error) {
	return s.svc.Collaboration.CreateTag(ctx, req)
}

func (s *Server) ListTags(ctx context.Context, req *pb.ListTagsRequest) (*pb.ListTagsResponse, error) {
	return s.svc.Collaboration.ListTags(ctx, req)
}

func (s *Server) AssignTag(ctx context.Context, req *pb.AssignTagRequest) (*pb.CommonResponse, error) {
	return s.svc.Collaboration.AssignTag(ctx, req)
}

func (s *Server) UnassignTag(ctx context.Context, req *pb.UnassignTagRequest) (*pb.CommonResponse, error) {
	return s.svc.Collaboration.UnassignTag(ctx, req)
}

func (s *Server) ListCandidateTags(ctx context.Context, req *pb.ListCandidateTagsRequest) (*pb.ListCandidateTagsResponse, error) {
	return s.svc.Collaboration.ListCandidateTags(ctx, req)
}

// Recruiting Intelligence

func (s *Server) GetResumeProfile(ctx context.Context, req *pb.GetResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return s.svc.RecruitingIntelligence.GetResumeProfile(ctx, req)
}

func (s *Server) ParseResumeProfile(ctx context.Context, req *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return s.svc.RecruitingIntelligence.ParseResumeProfile(ctx, req)
}

func (s *Server) EvaluateCandidateMatch(ctx context.Context, req *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return s.svc.RecruitingIntelligence.EvaluateCandidateMatch(ctx, req)
}

func (s *Server) GetCandidateMatchEvaluation(ctx context.Context, req *pb.GetCandidateMatchEvaluationRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return s.svc.RecruitingIntelligence.GetCandidateMatchEvaluation(ctx, req)
}

func (s *Server) CompareCandidatesForJob(ctx context.Context, req *pb.CompareCandidatesForJobRequest) (*pb.CompareCandidatesForJobResponse, error) {
	return s.svc.RecruitingIntelligence.CompareCandidatesForJob(ctx, req)
}

func (s *Server) CreateFollowUpTask(ctx context.Context, req *pb.CreateFollowUpTaskRequest) (*pb.CreateFollowUpTaskResponse, error) {
	return s.svc.Collaboration.CreateFollowUpTask(ctx, req)
}

func (s *Server) ListFollowUpTasks(ctx context.Context, req *pb.ListFollowUpTasksRequest) (*pb.ListFollowUpTasksResponse, error) {
	return s.svc.Collaboration.ListFollowUpTasks(ctx, req)
}

func (s *Server) CompleteFollowUpTask(ctx context.Context, req *pb.CompleteFollowUpTaskRequest) (*pb.CommonResponse, error) {
	return s.svc.Collaboration.CompleteFollowUpTask(ctx, req)
}

func (s *Server) GetFollowUpTask(ctx context.Context, req *pb.GetFollowUpTaskRequest) (*pb.GetFollowUpTaskResponse, error) {
	return s.svc.Collaboration.GetFollowUpTask(ctx, req)
}

func (s *Server) ListTimelineEvents(ctx context.Context, req *pb.ListTimelineEventsRequest) (*pb.ListTimelineEventsResponse, error) {
	return s.svc.Collaboration.ListTimelineEvents(ctx, req)
}

// ── LlmConfig ───────────────────────────────────────────────────────────

func (s *Server) ListProviders(ctx context.Context, req *pb.ListProvidersRequest) (*pb.ListProvidersResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.ListProviders(ctx, req)
}

func (s *Server) CreateProvider(ctx context.Context, req *pb.CreateProviderRequest) (*pb.ProviderResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.CreateProvider(ctx, req)
}

func (s *Server) UpdateProvider(ctx context.Context, req *pb.UpdateProviderRequest) (*pb.ProviderResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.UpdateProvider(ctx, req)
}

func (s *Server) DeleteProvider(ctx context.Context, req *pb.DeleteProviderRequest) (*pb.CommonResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.DeleteProvider(ctx, req)
}

func (s *Server) TestProviderConnection(ctx context.Context, req *pb.TestProviderConnectionRequest) (*pb.TestProviderConnectionResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.TestProviderConnection(ctx, req)
}

func (s *Server) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.ListModels(ctx, req)
}

func (s *Server) CreateModel(ctx context.Context, req *pb.CreateModelRequest) (*pb.ModelResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.CreateModel(ctx, req)
}

func (s *Server) UpdateModel(ctx context.Context, req *pb.UpdateModelRequest) (*pb.ModelResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.UpdateModel(ctx, req)
}

func (s *Server) DeleteModel(ctx context.Context, req *pb.DeleteModelRequest) (*pb.CommonResponse, error) {
	if s.svc.LlmConfig == nil {
		return nil, status.Error(codes.Unavailable, "llm config service not available (ENCRYPTION_KEY not set)")
	}
	return s.svc.LlmConfig.DeleteModel(ctx, req)
}

// ── PromptService ─────────────────────────────────────────────────────────

func (s *Server) ListPromptTemplates(ctx context.Context, req *pb.ListPromptTemplatesRequest) (*pb.ListPromptTemplatesResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.ListPromptTemplates(ctx, req)
}

func (s *Server) CreatePromptTemplate(ctx context.Context, req *pb.CreatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.CreatePromptTemplate(ctx, req)
}

func (s *Server) UpdatePromptTemplate(ctx context.Context, req *pb.UpdatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.UpdatePromptTemplate(ctx, req)
}

func (s *Server) DeletePromptTemplate(ctx context.Context, req *pb.DeletePromptTemplateRequest) (*pb.CommonResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.DeletePromptTemplate(ctx, req)
}

func (s *Server) GetPromptVersionHistory(ctx context.Context, req *pb.GetPromptVersionHistoryRequest) (*pb.GetPromptVersionHistoryResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.GetPromptVersionHistory(ctx, req)
}

func (s *Server) RollbackPromptVersion(ctx context.Context, req *pb.RollbackPromptVersionRequest) (*pb.PromptTemplateResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.RollbackPromptVersion(ctx, req)
}

func (s *Server) RenderPrompt(ctx context.Context, req *pb.RenderPromptRequest) (*pb.RenderPromptResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.RenderPrompt(ctx, req)
}

func (s *Server) GetActivePromptByAgentType(ctx context.Context, req *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error) {
	if s.svc.Prompt == nil {
		return nil, status.Error(codes.Unavailable, "prompt service not available")
	}
	return s.svc.Prompt.GetActivePromptByAgentType(ctx, req)
}

func (s *Server) GetUsageStats(ctx context.Context, req *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error) {
	if s.svc.UsageStats == nil {
		return nil, status.Error(codes.Unavailable, "usage stats service not available")
	}
	return s.svc.UsageStats.GetUsageStats(ctx, req)
}

func (s *Server) GetUsageTrend(ctx context.Context, req *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error) {
	if s.svc.UsageStats == nil {
		return nil, status.Error(codes.Unavailable, "usage stats service not available")
	}
	return s.svc.UsageStats.GetUsageTrend(ctx, req)
}

// ── AgentConfigService ───────────────────────────────────────────────────

func (s *Server) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	if s.svc.AgentConfig == nil {
		return nil, status.Error(codes.Unavailable, "agent config service not available")
	}
	return s.svc.AgentConfig.ListAgents(ctx, req)
}

func (s *Server) ListCapabilities(ctx context.Context, req *pb.ListCapabilitiesRequest) (*pb.ListCapabilitiesResponse, error) {
	if s.svc.AgentConfig == nil {
		return nil, status.Error(codes.Unavailable, "agent config service not available")
	}
	return s.svc.AgentConfig.ListCapabilities(ctx, req)
}

func (s *Server) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.AgentConfigResponse, error) {
	if s.svc.AgentConfig == nil {
		return nil, status.Error(codes.Unavailable, "agent config service not available")
	}
	return s.svc.AgentConfig.CreateAgent(ctx, req)
}

func (s *Server) UpdateAgent(ctx context.Context, req *pb.UpdateAgentRequest) (*pb.AgentConfigResponse, error) {
	if s.svc.AgentConfig == nil {
		return nil, status.Error(codes.Unavailable, "agent config service not available")
	}
	return s.svc.AgentConfig.UpdateAgent(ctx, req)
}

func (s *Server) DeleteAgent(ctx context.Context, req *pb.DeleteAgentRequest) (*pb.CommonResponse, error) {
	if s.svc.AgentConfig == nil {
		return nil, status.Error(codes.Unavailable, "agent config service not available")
	}
	return s.svc.AgentConfig.DeleteAgent(ctx, req)
}

func (s *Server) GetAgentConfig(ctx context.Context, req *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error) {
	if s.svc.AgentConfig == nil {
		return nil, status.Error(codes.Unavailable, "agent config service not available")
	}
	return s.svc.AgentConfig.GetAgentConfig(ctx, req)
}

// --- MCPService ---------------------------------------------------------

func (s *Server) ListMCPServers(ctx context.Context, req *pb.ListMCPServersRequest) (*pb.ListMCPServersResponse, error) {
	return s.svc.MCP.ListMCPServers(ctx, req)
}

func (s *Server) CreateMCPServer(ctx context.Context, req *pb.CreateMCPServerRequest) (*pb.MCPServerResponse, error) {
	return s.svc.MCP.CreateMCPServer(ctx, req)
}

func (s *Server) UpdateMCPServer(ctx context.Context, req *pb.UpdateMCPServerRequest) (*pb.MCPServerResponse, error) {
	return s.svc.MCP.UpdateMCPServer(ctx, req)
}

func (s *Server) DeleteMCPServer(ctx context.Context, req *pb.DeleteMCPServerRequest) (*pb.CommonResponse, error) {
	return s.svc.MCP.DeleteMCPServer(ctx, req)
}

func (s *Server) ListMCPToolPolicies(ctx context.Context, req *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error) {
	return s.svc.MCP.ListMCPToolPolicies(ctx, req)
}

func (s *Server) CreateMCPToolPolicy(ctx context.Context, req *pb.CreateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	return s.svc.MCP.CreateMCPToolPolicy(ctx, req)
}

func (s *Server) UpdateMCPToolPolicy(ctx context.Context, req *pb.UpdateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	return s.svc.MCP.UpdateMCPToolPolicy(ctx, req)
}

func (s *Server) DeleteMCPToolPolicy(ctx context.Context, req *pb.DeleteMCPToolPolicyRequest) (*pb.CommonResponse, error) {
	return s.svc.MCP.DeleteMCPToolPolicy(ctx, req)
}

func (s *Server) TestMCPConnection(ctx context.Context, req *pb.TestMCPConnectionRequest) (*pb.TestMCPConnectionResponse, error) {
	return s.svc.MCP.TestMCPConnection(ctx, req)
}

func (s *Server) ListMCPTools(ctx context.Context, req *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error) {
	return s.svc.MCP.ListMCPTools(ctx, req)
}

func (s *Server) CallMCPTool(ctx context.Context, req *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error) {
	return s.svc.MCP.CallMCPTool(ctx, req)
}

// --- SkillService --------------------------------------------------------

func (s *Server) ListSkills(ctx context.Context, req *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	return s.svc.Skill.ListSkills(ctx, req)
}

func (s *Server) CreateSkill(ctx context.Context, req *pb.CreateSkillRequest) (*pb.SkillResponse, error) {
	return s.svc.Skill.CreateSkill(ctx, req)
}

func (s *Server) UpdateSkill(ctx context.Context, req *pb.UpdateSkillRequest) (*pb.SkillResponse, error) {
	return s.svc.Skill.UpdateSkill(ctx, req)
}

func (s *Server) CreateSkillVersion(ctx context.Context, req *pb.CreateSkillVersionRequest) (*pb.SkillVersionResponse, error) {
	return s.svc.Skill.CreateSkillVersion(ctx, req)
}

func (s *Server) ListSkillVersions(ctx context.Context, req *pb.ListSkillVersionsRequest) (*pb.ListSkillVersionsResponse, error) {
	return s.svc.Skill.ListSkillVersions(ctx, req)
}

func (s *Server) ActivateSkillVersion(ctx context.Context, req *pb.ActivateSkillVersionRequest) (*pb.SkillResponse, error) {
	return s.svc.Skill.ActivateSkillVersion(ctx, req)
}

func (s *Server) ListSkillTools(ctx context.Context, req *pb.ListSkillToolsRequest) (*pb.ListSkillToolsResponse, error) {
	return s.svc.Skill.ListSkillTools(ctx, req)
}

func (s *Server) UpdateSkillTool(ctx context.Context, req *pb.UpdateSkillToolRequest) (*pb.SkillToolResponse, error) {
	return s.svc.Skill.UpdateSkillTool(ctx, req)
}

// --- AgentSkillService --------------------------------------------------

func (s *Server) ListAgentSkills(ctx context.Context, req *pb.ListAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	return s.svc.AgentSkill.ListAgentSkills(ctx, req)
}

func (s *Server) GetAgentSkill(ctx context.Context, req *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	return s.svc.AgentSkill.GetAgentSkill(ctx, req)
}

func (s *Server) CreateAgentSkill(ctx context.Context, req *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	return s.svc.AgentSkill.CreateAgentSkill(ctx, req)
}

func (s *Server) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	return s.svc.AgentSkill.UpdateAgentSkill(ctx, req)
}

func (s *Server) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error) {
	return s.svc.AgentSkill.CreateAgentSkillVersion(ctx, req)
}

func (s *Server) ListAgentSkillVersions(ctx context.Context, req *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error) {
	return s.svc.AgentSkill.ListAgentSkillVersions(ctx, req)
}

func (s *Server) ActivateAgentSkillVersion(ctx context.Context, req *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error) {
	return s.svc.AgentSkill.ActivateAgentSkillVersion(ctx, req)
}

func (s *Server) UpdateAgentSkillStatus(ctx context.Context, req *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error) {
	return s.svc.AgentSkill.UpdateAgentSkillStatus(ctx, req)
}

func (s *Server) PreviewAgentSkill(ctx context.Context, req *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error) {
	return s.svc.AgentSkill.PreviewAgentSkill(ctx, req)
}

func (s *Server) ListAvailableAgentSkills(ctx context.Context, req *pb.ListAvailableAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	return s.svc.AgentSkill.ListAvailableAgentSkills(ctx, req)
}
