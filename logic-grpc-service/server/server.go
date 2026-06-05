package server

import (
	"context"

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
