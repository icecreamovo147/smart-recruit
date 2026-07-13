package grpc

import (
	"context"

	gogrpc "google.golang.org/grpc"

	"smart-recruit-domain-go/service"
	"smart-recruit-proto/recruitment/pb"
)

func NewLegacyAIService(hr *service.AIService, candidate *service.CandidateAIService) pb.AIServiceServer {
	return legacyAIService{hr: hr, candidate: candidate}
}

func NewLegacyRecruitingIntelligenceService(service *service.RecruitingIntelligenceService) pb.RecruitingIntelligenceServiceServer {
	return legacyRecruitingIntelligenceService{service: service}
}

type legacyAIService struct {
	pb.UnimplementedAIServiceServer
	hr        *service.AIService
	candidate *service.CandidateAIService
}

func (s legacyAIService) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	return s.hr.Chat(ctx, req)
}

func (s legacyAIService) ChatStream(req *pb.ChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	return s.hr.ChatStream(req, stream)
}

func (s legacyAIService) History(ctx context.Context, req *pb.ChatHistoryRequest) (*pb.ChatHistoryResponse, error) {
	return s.hr.History(ctx, req)
}

func (s legacyAIService) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	return s.hr.AnalyzeApplication(ctx, req)
}

func (s legacyAIService) ListChatSessions(ctx context.Context, req *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error) {
	return s.hr.ListChatSessions(ctx, req)
}

func (s legacyAIService) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error) {
	return s.hr.CreateChatSession(ctx, req)
}

func (s legacyAIService) SessionMessages(ctx context.Context, req *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	return s.hr.SessionMessages(ctx, req)
}

func (s legacyAIService) CreateApplicationAnalysisSession(ctx context.Context, req *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	return s.hr.CreateApplicationAnalysisSession(ctx, req)
}

func (s legacyAIService) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.CommonResponse, error) {
	return s.hr.UpdateSession(ctx, req)
}

func (s legacyAIService) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.CommonResponse, error) {
	return s.hr.DeleteSession(ctx, req)
}

func (s legacyAIService) CandidateChatStream(req *pb.CandidateChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	return s.candidate.StreamChatGRPC(req, stream)
}

func (s legacyAIService) CandidateListSessions(ctx context.Context, req *pb.CandidateSessionListRequest) (*pb.ChatSessionListResponse, error) {
	return s.candidate.ListSessionsGRPC(ctx, req)
}

func (s legacyAIService) CandidateCreateSession(ctx context.Context, req *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error) {
	return s.candidate.CreateSessionGRPC(ctx, req)
}

func (s legacyAIService) CandidateSessionMessages(ctx context.Context, req *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	return s.candidate.SessionMessagesGRPC(ctx, req)
}

func (s legacyAIService) CandidateUpdateSession(ctx context.Context, req *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error) {
	return s.candidate.UpdateSessionGRPC(ctx, req)
}

func (s legacyAIService) CandidateDeleteSession(ctx context.Context, req *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error) {
	return s.candidate.DeleteSessionGRPC(ctx, req)
}

func (s legacyAIService) GetToolTraces(ctx context.Context, req *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error) {
	return s.hr.GetToolTraces(ctx, req)
}

func (s legacyAIService) GetAgentRuns(ctx context.Context, req *pb.GetAgentRunsRequest) (*pb.GetAgentRunsResponse, error) {
	return s.hr.GetAgentRuns(ctx, req)
}

func (s legacyAIService) CreateAgentRun(ctx context.Context, req *pb.CreateAgentRunRequest) (*pb.CreateAgentRunResponse, error) {
	return s.hr.CreateAgentRun(ctx, req)
}

func (s legacyAIService) GetAgentRun(ctx context.Context, req *pb.GetAgentRunRequest) (*pb.GetAgentRunResponse, error) {
	return s.hr.GetAgentRun(ctx, req)
}

func (s legacyAIService) GetActiveAgentRun(ctx context.Context, req *pb.GetActiveAgentRunRequest) (*pb.GetActiveAgentRunResponse, error) {
	return s.hr.GetActiveAgentRun(ctx, req)
}

func (s legacyAIService) SubscribeAgentRunEvents(req *pb.SubscribeAgentRunEventsRequest, stream gogrpc.ServerStreamingServer[pb.AgentRunEvent]) error {
	return s.hr.SubscribeAgentRunEvents(req, stream)
}

func (s legacyAIService) CancelAgentRun(ctx context.Context, req *pb.CancelAgentRunRequest) (*pb.CancelAgentRunResponse, error) {
	return s.hr.CancelAgentRun(ctx, req)
}

func (s legacyAIService) ConfirmAgentRun(ctx context.Context, req *pb.ConfirmAgentRunRequest) (*pb.ConfirmAgentRunResponse, error) {
	return s.hr.ConfirmAgentRun(ctx, req)
}

type legacyRecruitingIntelligenceService struct {
	pb.UnimplementedRecruitingIntelligenceServiceServer
	service *service.RecruitingIntelligenceService
}

func (s legacyRecruitingIntelligenceService) GetResumeProfile(ctx context.Context, req *pb.GetResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return s.service.GetResumeProfile(ctx, req)
}

func (s legacyRecruitingIntelligenceService) ParseResumeProfile(ctx context.Context, req *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return s.service.ParseResumeProfile(ctx, req)
}

func (s legacyRecruitingIntelligenceService) EvaluateCandidateMatch(ctx context.Context, req *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return s.service.EvaluateCandidateMatch(ctx, req)
}

func (s legacyRecruitingIntelligenceService) GetCandidateMatchEvaluation(ctx context.Context, req *pb.GetCandidateMatchEvaluationRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return s.service.GetCandidateMatchEvaluation(ctx, req)
}

func (s legacyRecruitingIntelligenceService) CompareCandidatesForJob(ctx context.Context, req *pb.CompareCandidatesForJobRequest) (*pb.CompareCandidatesForJobResponse, error) {
	return s.service.CompareCandidatesForJob(ctx, req)
}
