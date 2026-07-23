package candidate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

func TestCandidateChatAggregatesStreamResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	aiClient := &candidateChatAIClient{
		stream: &candidateChatStream{
			ctx: context.Background(),
			responses: []*pb.ChatStreamResponse{
				{Code: 0, Msg: "success", Delta: "hello ", SessionId: 123, CreatedAt: "2026-07-15T10:00:00Z"},
				{Code: 0, Msg: "success", Delta: "candidate", Done: true, SessionId: 123, CreatedAt: "2026-07-15T10:00:01Z", SuggestedQuestions: []string{"Q1", "Q2", "Q3"}},
			},
		},
	}
	handler := NewAIHandler(&rpc.Clients{AI: aiClient})
	router := gin.New()
	router.POST("/api/v1/candidate/ai/chat", func(c *gin.Context) {
		c.Set("user_id", int64(55))
		handler.Chat(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/candidate/ai/chat", strings.NewReader(`{"message":"hi","session_id":123}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if aiClient.request.GetUserId() != 55 || aiClient.request.GetSessionId() != 123 || aiClient.request.GetMessage() != "hi" {
		t.Fatalf("candidate chat request = %#v", aiClient.request)
	}
	var envelope struct {
		Code int32 `json:"code"`
		Data struct {
			Reply              string   `json:"reply"`
			SessionID          int64    `json:"session_id"`
			CreatedAt          string   `json:"created_at"`
			SuggestedQuestions []string `json:"suggested_questions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Code != 0 || envelope.Data.Reply != "hello candidate" || envelope.Data.SessionID != 123 || envelope.Data.CreatedAt != "2026-07-15T10:00:01Z" {
		t.Fatalf("response envelope = %#v", envelope)
	}
	if len(envelope.Data.SuggestedQuestions) != 3 || envelope.Data.SuggestedQuestions[2] != "Q3" {
		t.Fatalf("suggested questions = %#v", envelope.Data.SuggestedQuestions)
	}
}

func TestMapContextUsageIncludesBudgetAndBreakdownFields(t *testing.T) {
	mapped := mapContextUsage(&pb.ContextUsageInfo{
		InputBudgetTokens:    6758,
		SafetyMarginTokens:   410,
		BudgetUsageRatio:     0.25,
		BudgetStatus:         "within_budget",
		IncludedMessageCount: 12,
		OmittedMessageCount:  3,
		SummaryApplied:       true,
		Breakdown: &pb.ContextUsageBreakdown{
			ToolSchemaTokens:       44,
			ProtocolOverheadTokens: 50,
		},
	})
	if mapped["input_budget_tokens"] != int32(6758) || mapped["budget_status"] != "within_budget" {
		t.Fatalf("budget mapping incomplete: %#v", mapped)
	}
	if mapped["included_message_count"] != int32(12) || mapped["omitted_message_count"] != int32(3) || mapped["summary_applied"] != true {
		t.Fatalf("message/summary mapping incomplete: %#v", mapped)
	}
	breakdown, ok := mapped["breakdown"].(map[string]any)
	if !ok {
		t.Fatalf("breakdown type = %T, want map[string]any", mapped["breakdown"])
	}
	if breakdown["tool_schema_tokens"] != int32(44) || breakdown["protocol_overhead_tokens"] != int32(50) {
		t.Fatalf("breakdown mapping incomplete: %#v", breakdown)
	}
}

func TestCandidateChatForwardsModelID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	aiClient := &candidateChatAIClient{
		stream: &candidateChatStream{
			ctx: context.Background(),
			responses: []*pb.ChatStreamResponse{
				{
					Code: 0, Msg: "success", EventType: "model_info",
					ContextUsage: &pb.ContextUsageInfo{ModelId: 9, ModelName: "qwen-candidate"},
				},
				{Code: 0, Msg: "success", Delta: "hello", Done: true, SessionId: 123},
			},
		},
	}
	handler := NewAIHandler(&rpc.Clients{AI: aiClient})
	router := gin.New()
	router.POST("/api/v1/candidate/ai/chat", func(c *gin.Context) {
		c.Set("user_id", int64(55))
		handler.Chat(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/candidate/ai/chat", strings.NewReader(`{"message":"hi","session_id":123,"model_id":9}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if aiClient.request.GetModelId() != 9 {
		t.Fatalf("candidate chat model_id = %d, want 9", aiClient.request.GetModelId())
	}
	var envelope struct {
		Code int32 `json:"code"`
		Data struct {
			Reply     string `json:"reply"`
			ModelName string `json:"model_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Data.ModelName != "qwen-candidate" {
		t.Fatalf("model_name = %q, want qwen-candidate", envelope.Data.ModelName)
	}
}

func TestCandidateChatStreamIncludesContextUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	aiClient := &candidateChatAIClient{
		stream: &candidateChatStream{
			ctx: context.Background(),
			responses: []*pb.ChatStreamResponse{
				{
					Code: 0, Msg: "success", EventType: "model_info",
					ContextUsage: &pb.ContextUsageInfo{ModelId: 9, ModelName: "qwen-candidate"},
				},
				{Code: 0, Msg: "success", Done: true},
			},
		},
	}
	handler := NewAIHandler(&rpc.Clients{AI: aiClient})
	router := gin.New()
	router.POST("/api/v1/candidate/ai/chat/stream", func(c *gin.Context) {
		c.Set("user_id", int64(55))
		handler.ChatStream(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/candidate/ai/chat/stream", strings.NewReader(`{"message":"hi","model_id":9}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"model_name":"qwen-candidate"`) {
		t.Fatalf("stream body missing context_usage model_name: %s", rec.Body.String())
	}
	if aiClient.request.GetModelId() != 9 {
		t.Fatalf("candidate chat model_id = %d, want 9", aiClient.request.GetModelId())
	}
}

type candidateBillingClient struct {
	pb.BillingServiceClient
}

func (c *candidateBillingClient) CheckAIAccess(_ context.Context, _ *pb.CheckAIAccessRequest, _ ...grpc.CallOption) (*pb.CheckAIAccessResponse, error) {
	return &pb.CheckAIAccessResponse{Code: 0, Msg: "success", Allowed: true, CapabilityVersionId: 77}, nil
}

type candidatePlatformAIClient struct {
	pb.PlatformAIControlPlaneServiceClient
}

func (c *candidatePlatformAIClient) ListPlatformAIRuntimeModels(_ context.Context, req *pb.ListPlatformAIRuntimeModelsRequest, _ ...grpc.CallOption) (*pb.ListPlatformAIRuntimeModelsResponse, error) {
	return &pb.ListPlatformAIRuntimeModelsResponse{
		Code: 0, Msg: "success", CapabilityVersionId: req.GetCapabilityVersionId(), SnapshotHash: "snapshot-77",
		List: []*pb.PlatformAIRuntimeModelInfo{{Id: 1, ModelName: "candidate-model", DisplayName: "Candidate", IsDefault: true}},
	}, nil
}

func TestCandidateListAvailableModelsUsesPublishedCandidatePool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAIHandler(&rpc.Clients{Billing: &candidateBillingClient{}, PlatformAI: &candidatePlatformAIClient{}})
	router := gin.New()
	router.GET("/api/v1/candidate/ai/models", func(c *gin.Context) {
		c.Set("user_id", int64(55))
		handler.ListAvailableModels(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/candidate/ai/models", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Code int32 `json:"code"`
		Data struct {
			Total               int64 `json:"total"`
			CapabilityVersionID int64 `json:"capability_version_id"`
			List                []struct {
				ID        int64  `json:"id"`
				ModelName string `json:"model_name"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Data.Total != 1 || len(envelope.Data.List) != 1 || envelope.Data.List[0].ID != 1 || envelope.Data.CapabilityVersionID != 77 {
		t.Fatalf("response envelope = %#v", envelope)
	}
}

type candidateChatAIClient struct {
	pb.AIServiceClient
	stream  grpc.ServerStreamingClient[pb.ChatStreamResponse]
	request *pb.CandidateChatRequest
}

func (c *candidateChatAIClient) CandidateChatStream(_ context.Context, req *pb.CandidateChatRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.ChatStreamResponse], error) {
	c.request = req
	return c.stream, nil
}

type candidateChatStream struct {
	grpc.ClientStream
	ctx       context.Context
	responses []*pb.ChatStreamResponse
	index     int
}

func (s *candidateChatStream) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *candidateChatStream) Recv() (*pb.ChatStreamResponse, error) {
	if s.index >= len(s.responses) {
		return nil, io.EOF
	}
	response := s.responses[s.index]
	s.index++
	return response, nil
}
