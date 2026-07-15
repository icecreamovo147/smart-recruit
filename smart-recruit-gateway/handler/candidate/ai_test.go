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
