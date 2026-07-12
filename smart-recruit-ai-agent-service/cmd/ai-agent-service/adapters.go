package main

import (
	"context"

	"google.golang.org/grpc"

	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/service"
)

type aiServer struct {
	pb.UnimplementedAIServiceServer
	hr        *service.AIService
	candidate *service.CandidateAIService
}

func (s aiServer) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	return s.hr.Chat(ctx, req)
}

func (s aiServer) ChatStream(req *pb.ChatRequest, stream grpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	return s.hr.ChatStream(req, stream)
}

func (s aiServer) History(ctx context.Context, req *pb.ChatHistoryRequest) (*pb.ChatHistoryResponse, error) {
	return s.hr.History(ctx, req)
}

func (s aiServer) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	return s.hr.AnalyzeApplication(ctx, req)
}

func (s aiServer) ListChatSessions(ctx context.Context, req *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error) {
	return s.hr.ListChatSessions(ctx, req)
}

func (s aiServer) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error) {
	return s.hr.CreateChatSession(ctx, req)
}

func (s aiServer) SessionMessages(ctx context.Context, req *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	return s.hr.SessionMessages(ctx, req)
}

func (s aiServer) CreateApplicationAnalysisSession(ctx context.Context, req *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	return s.hr.CreateApplicationAnalysisSession(ctx, req)
}

func (s aiServer) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.CommonResponse, error) {
	return s.hr.UpdateSession(ctx, req)
}

func (s aiServer) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.CommonResponse, error) {
	return s.hr.DeleteSession(ctx, req)
}

func (s aiServer) CandidateChatStream(req *pb.CandidateChatRequest, stream grpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	return s.candidate.StreamChatGRPC(req, stream)
}

func (s aiServer) CandidateListSessions(ctx context.Context, req *pb.CandidateSessionListRequest) (*pb.ChatSessionListResponse, error) {
	return s.candidate.ListSessionsGRPC(ctx, req)
}

func (s aiServer) CandidateCreateSession(ctx context.Context, req *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error) {
	return s.candidate.CreateSessionGRPC(ctx, req)
}

func (s aiServer) CandidateSessionMessages(ctx context.Context, req *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	return s.candidate.SessionMessagesGRPC(ctx, req)
}

func (s aiServer) CandidateUpdateSession(ctx context.Context, req *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error) {
	return s.candidate.UpdateSessionGRPC(ctx, req)
}

func (s aiServer) CandidateDeleteSession(ctx context.Context, req *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error) {
	return s.candidate.DeleteSessionGRPC(ctx, req)
}

func (s aiServer) GetToolTraces(ctx context.Context, req *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error) {
	return s.hr.GetToolTraces(ctx, req)
}

func (s aiServer) GetAgentRuns(ctx context.Context, req *pb.GetAgentRunsRequest) (*pb.GetAgentRunsResponse, error) {
	return s.hr.GetAgentRuns(ctx, req)
}

func (s aiServer) CreateAgentRun(ctx context.Context, req *pb.CreateAgentRunRequest) (*pb.CreateAgentRunResponse, error) {
	return s.hr.CreateAgentRun(ctx, req)
}

func (s aiServer) GetAgentRun(ctx context.Context, req *pb.GetAgentRunRequest) (*pb.GetAgentRunResponse, error) {
	return s.hr.GetAgentRun(ctx, req)
}

func (s aiServer) GetActiveAgentRun(ctx context.Context, req *pb.GetActiveAgentRunRequest) (*pb.GetActiveAgentRunResponse, error) {
	return s.hr.GetActiveAgentRun(ctx, req)
}

func (s aiServer) SubscribeAgentRunEvents(req *pb.SubscribeAgentRunEventsRequest, stream grpc.ServerStreamingServer[pb.AgentRunEvent]) error {
	return s.hr.SubscribeAgentRunEvents(req, stream)
}

func (s aiServer) CancelAgentRun(ctx context.Context, req *pb.CancelAgentRunRequest) (*pb.CancelAgentRunResponse, error) {
	return s.hr.CancelAgentRun(ctx, req)
}

func (s aiServer) ConfirmAgentRun(ctx context.Context, req *pb.ConfirmAgentRunRequest) (*pb.ConfirmAgentRunResponse, error) {
	return s.hr.ConfirmAgentRun(ctx, req)
}

type recruitingIntelligenceServer struct {
	pb.UnimplementedRecruitingIntelligenceServiceServer
	service *service.RecruitingIntelligenceService
}

func (s recruitingIntelligenceServer) GetResumeProfile(ctx context.Context, req *pb.GetResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return s.service.GetResumeProfile(ctx, req)
}

func (s recruitingIntelligenceServer) ParseResumeProfile(ctx context.Context, req *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return s.service.ParseResumeProfile(ctx, req)
}

func (s recruitingIntelligenceServer) EvaluateCandidateMatch(ctx context.Context, req *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return s.service.EvaluateCandidateMatch(ctx, req)
}

func (s recruitingIntelligenceServer) GetCandidateMatchEvaluation(ctx context.Context, req *pb.GetCandidateMatchEvaluationRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return s.service.GetCandidateMatchEvaluation(ctx, req)
}

func (s recruitingIntelligenceServer) CompareCandidatesForJob(ctx context.Context, req *pb.CompareCandidatesForJobRequest) (*pb.CompareCandidatesForJobResponse, error) {
	return s.service.CompareCandidatesForJob(ctx, req)
}
