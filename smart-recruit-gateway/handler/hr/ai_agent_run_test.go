package hr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

// mockAIServiceClient implements pb.AIServiceClient for durable-run handler tests.
type mockAIServiceClient struct {
	createFn    func(context.Context, *pb.CreateAgentRunRequest, ...grpc.CallOption) (*pb.CreateAgentRunResponse, error)
	getFn       func(context.Context, *pb.GetAgentRunRequest, ...grpc.CallOption) (*pb.GetAgentRunResponse, error)
	activeFn    func(context.Context, *pb.GetActiveAgentRunRequest, ...grpc.CallOption) (*pb.GetActiveAgentRunResponse, error)
	subscribeFn func(context.Context, *pb.SubscribeAgentRunEventsRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[pb.AgentRunEvent], error)
	cancelFn    func(context.Context, *pb.CancelAgentRunRequest, ...grpc.CallOption) (*pb.CancelAgentRunResponse, error)
	confirmFn   func(context.Context, *pb.ConfirmAgentRunRequest, ...grpc.CallOption) (*pb.ConfirmAgentRunResponse, error)

	cancelCalled atomic.Bool
}

func (m *mockAIServiceClient) Chat(context.Context, *pb.ChatRequest, ...grpc.CallOption) (*pb.ChatResponse, error) {
	return &pb.ChatResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) ChatStream(context.Context, *pb.ChatRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[pb.ChatStreamResponse], error) {
	return nil, status.Error(codes.Unimplemented, "not used")
}
func (m *mockAIServiceClient) History(context.Context, *pb.ChatHistoryRequest, ...grpc.CallOption) (*pb.ChatHistoryResponse, error) {
	return &pb.ChatHistoryResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) AnalyzeApplication(context.Context, *pb.AnalyzeApplicationRequest, ...grpc.CallOption) (*pb.AnalyzeApplicationResponse, error) {
	return &pb.AnalyzeApplicationResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) ListChatSessions(context.Context, *pb.ChatSessionListRequest, ...grpc.CallOption) (*pb.ChatSessionListResponse, error) {
	return &pb.ChatSessionListResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) CreateChatSession(context.Context, *pb.CreateChatSessionRequest, ...grpc.CallOption) (*pb.CreateChatSessionResponse, error) {
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) SessionMessages(context.Context, *pb.SessionMessagesRequest, ...grpc.CallOption) (*pb.ChatHistoryResponse, error) {
	return &pb.ChatHistoryResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) CreateApplicationAnalysisSession(context.Context, *pb.CreateApplicationAnalysisSessionRequest, ...grpc.CallOption) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	return &pb.CreateApplicationAnalysisSessionResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) UpdateSession(context.Context, *pb.UpdateSessionRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) DeleteSession(context.Context, *pb.DeleteSessionRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) CandidateChatStream(context.Context, *pb.CandidateChatRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[pb.ChatStreamResponse], error) {
	return nil, status.Error(codes.Unimplemented, "not used")
}
func (m *mockAIServiceClient) CandidateListSessions(context.Context, *pb.CandidateSessionListRequest, ...grpc.CallOption) (*pb.ChatSessionListResponse, error) {
	return &pb.ChatSessionListResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) CandidateCreateSession(context.Context, *pb.CandidateCreateSessionRequest, ...grpc.CallOption) (*pb.CreateChatSessionResponse, error) {
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) CandidateSessionMessages(context.Context, *pb.CandidateSessionMessagesRequest, ...grpc.CallOption) (*pb.ChatHistoryResponse, error) {
	return &pb.ChatHistoryResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) CandidateUpdateSession(context.Context, *pb.CandidateUpdateSessionRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) CandidateDeleteSession(context.Context, *pb.CandidateDeleteSessionRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) GetToolTraces(context.Context, *pb.GetToolTracesRequest, ...grpc.CallOption) (*pb.GetToolTracesResponse, error) {
	return &pb.GetToolTracesResponse{Code: 0, Msg: "ok"}, nil
}
func (m *mockAIServiceClient) GetAgentRuns(context.Context, *pb.GetAgentRunsRequest, ...grpc.CallOption) (*pb.GetAgentRunsResponse, error) {
	return &pb.GetAgentRunsResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAIServiceClient) CreateAgentRun(ctx context.Context, req *pb.CreateAgentRunRequest, opts ...grpc.CallOption) (*pb.CreateAgentRunResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req, opts...)
	}
	return &pb.CreateAgentRunResponse{
		Code: 0,
		Msg:  "ok",
		Run:  &pb.AgentRunSnapshot{RunId: 1, SessionId: req.SessionId, HrId: req.HrId, Status: "queued"},
	}, nil
}

func (m *mockAIServiceClient) GetAgentRun(ctx context.Context, req *pb.GetAgentRunRequest, opts ...grpc.CallOption) (*pb.GetAgentRunResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req, opts...)
	}
	return &pb.GetAgentRunResponse{
		Code: 0,
		Msg:  "ok",
		Run:  &pb.AgentRunSnapshot{RunId: req.RunId, HrId: req.HrId, Status: "running"},
	}, nil
}

func (m *mockAIServiceClient) GetActiveAgentRun(ctx context.Context, req *pb.GetActiveAgentRunRequest, opts ...grpc.CallOption) (*pb.GetActiveAgentRunResponse, error) {
	if m.activeFn != nil {
		return m.activeFn(ctx, req, opts...)
	}
	return &pb.GetActiveAgentRunResponse{Code: 0, Msg: "ok", HasActiveRun: false}, nil
}

func (m *mockAIServiceClient) SubscribeAgentRunEvents(ctx context.Context, req *pb.SubscribeAgentRunEventsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.AgentRunEvent], error) {
	if m.subscribeFn != nil {
		return m.subscribeFn(ctx, req, opts...)
	}
	return &mockAgentRunEventStream{ctx: ctx, events: nil}, nil
}

func (m *mockAIServiceClient) CancelAgentRun(ctx context.Context, req *pb.CancelAgentRunRequest, opts ...grpc.CallOption) (*pb.CancelAgentRunResponse, error) {
	m.cancelCalled.Store(true)
	if m.cancelFn != nil {
		return m.cancelFn(ctx, req, opts...)
	}
	return &pb.CancelAgentRunResponse{
		Code: 0,
		Msg:  "ok",
		Run:  &pb.AgentRunSnapshot{RunId: req.RunId, HrId: req.HrId, Status: "cancel_requested"},
	}, nil
}

func (m *mockAIServiceClient) ConfirmAgentRun(ctx context.Context, req *pb.ConfirmAgentRunRequest, opts ...grpc.CallOption) (*pb.ConfirmAgentRunResponse, error) {
	if m.confirmFn != nil {
		return m.confirmFn(ctx, req, opts...)
	}
	return &pb.ConfirmAgentRunResponse{
		Code: 0,
		Msg:  "ok",
		Run:  &pb.AgentRunSnapshot{RunId: req.RunId, HrId: req.HrId, Status: "running"},
	}, nil
}

// mockAgentRunEventStream is a minimal gRPC server-streaming client for tests.
type mockAgentRunEventStream struct {
	ctx    context.Context
	events []*pb.AgentRunEvent
	err    error
	idx    int
	mu     sync.Mutex
	// blockUntilCancel when true makes Recv wait until ctx is cancelled after events are drained.
	blockUntilCancel bool
}

func (m *mockAgentRunEventStream) Recv() (*pb.AgentRunEvent, error) {
	m.mu.Lock()
	if m.idx < len(m.events) {
		e := m.events[m.idx]
		m.idx++
		m.mu.Unlock()
		return e, nil
	}
	m.mu.Unlock()

	if m.blockUntilCancel {
		<-m.ctx.Done()
		return nil, m.ctx.Err()
	}
	if m.err != nil {
		return nil, m.err
	}
	return nil, io.EOF
}

func (m *mockAgentRunEventStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockAgentRunEventStream) Trailer() metadata.MD         { return nil }
func (m *mockAgentRunEventStream) CloseSend() error             { return nil }
func (m *mockAgentRunEventStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}
func (m *mockAgentRunEventStream) SendMsg(any) error { return nil }
func (m *mockAgentRunEventStream) RecvMsg(any) error { return io.EOF }

func withAuthUser(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func newAgentRunTestRouter(mock *mockAIServiceClient, userID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewAIHandler(&rpc.Clients{AI: mock})
	r := gin.New()
	if userID > 0 {
		r.Use(withAuthUser(userID))
	}
	r.POST("/api/v1/hr/ai/runs", handler.CreateAgentRun)
	r.GET("/api/v1/hr/ai/runs/:run_id", handler.GetAgentRun)
	r.GET("/api/v1/hr/ai/sessions/:session_id/active-run", handler.GetActiveAgentRun)
	r.GET("/api/v1/hr/ai/runs/:run_id/events", handler.SubscribeAgentRunEvents)
	r.POST("/api/v1/hr/ai/runs/:run_id/cancel", handler.CancelAgentRun)
	r.POST("/api/v1/hr/ai/runs/:run_id/confirm", handler.ConfirmAgentRun)
	return r
}

func TestCreateAgentRun_Validation(t *testing.T) {
	mock := &mockAIServiceClient{}
	r := newAgentRunTestRouter(mock, 42)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/ai/runs", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 envelope, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, _ := body["code"].(float64); code != 400 {
		t.Fatalf("expected business code 400 for missing session_id, got %v body=%s", body["code"], w.Body.String())
	}
}

func TestCreateAgentRun_RejectsBlankMessageBeforeRPC(t *testing.T) {
	var called atomic.Bool
	mock := &mockAIServiceClient{createFn: func(_ context.Context, _ *pb.CreateAgentRunRequest, _ ...grpc.CallOption) (*pb.CreateAgentRunResponse, error) {
		called.Store(true)
		return &pb.CreateAgentRunResponse{Code: 0, Msg: "ok"}, nil
	}}
	r := newAgentRunTestRouter(mock, 42)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/ai/runs", strings.NewReader(`{"session_id":7,"message":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, _ := body["code"].(float64); code != 400 {
		t.Fatalf("expected business code 400, got %v body=%s", body["code"], w.Body.String())
	}
	if called.Load() {
		t.Fatal("blank message must be rejected before CreateAgentRun RPC")
	}
}

func TestCreateAgentRun_ForwardsHrIDAndFields(t *testing.T) {
	var captured *pb.CreateAgentRunRequest
	mock := &mockAIServiceClient{
		createFn: func(_ context.Context, req *pb.CreateAgentRunRequest, _ ...grpc.CallOption) (*pb.CreateAgentRunResponse, error) {
			captured = req
			return &pb.CreateAgentRunResponse{
				Code:             0,
				Msg:              "ok",
				IdempotentReplay: false,
				Run: &pb.AgentRunSnapshot{
					RunId:     99,
					SessionId: req.SessionId,
					HrId:      req.HrId,
					Status:    "queued",
				},
			}, nil
		},
	}
	r := newAgentRunTestRouter(mock, 42)
	body := `{"session_id":7,"client_request_id":"req-1","message":"hello","action_type":"submit","model_id":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/ai/runs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if captured == nil {
		t.Fatal("expected CreateAgentRun RPC call")
	}
	if captured.HrId != 42 || captured.SessionId != 7 || captured.ClientRequestId != "req-1" || captured.Message != "hello" {
		t.Fatalf("unexpected RPC request: %+v", captured)
	}
	if !strings.Contains(w.Body.String(), `"run_id":99`) {
		t.Fatalf("expected run_id in response, got %s", w.Body.String())
	}
}

func TestGetAgentRun_InvalidID(t *testing.T) {
	r := newAgentRunTestRouter(&mockAIServiceClient{}, 42)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/ai/runs/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if code, _ := body["code"].(float64); code != 400 {
		t.Fatalf("expected 400, got %v body=%s", body["code"], w.Body.String())
	}
}

func TestGetAgentRun_AuthFailureMappedFromPermissionDenied(t *testing.T) {
	mock := &mockAIServiceClient{
		getFn: func(_ context.Context, _ *pb.GetAgentRunRequest, _ ...grpc.CallOption) (*pb.GetAgentRunResponse, error) {
			return nil, status.Error(codes.PermissionDenied, "not owner")
		},
	}
	r := newAgentRunTestRouter(mock, 42)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/ai/runs/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, _ := body["code"].(float64); code != 403 {
		t.Fatalf("expected mapped 403, got %v body=%s", body["code"], w.Body.String())
	}
}

func TestGetActiveAgentRun_ForwardsSessionAndHr(t *testing.T) {
	var captured *pb.GetActiveAgentRunRequest
	mock := &mockAIServiceClient{
		activeFn: func(_ context.Context, req *pb.GetActiveAgentRunRequest, _ ...grpc.CallOption) (*pb.GetActiveAgentRunResponse, error) {
			captured = req
			return &pb.GetActiveAgentRunResponse{
				Code:         0,
				Msg:          "ok",
				HasActiveRun: true,
				Run:          &pb.AgentRunSnapshot{RunId: 5, SessionId: req.SessionId, Status: "running", LastEventSeq: 12},
			}, nil
		},
	}
	r := newAgentRunTestRouter(mock, 77)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/ai/sessions/3/active-run", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if captured == nil || captured.HrId != 77 || captured.SessionId != 3 {
		t.Fatalf("unexpected active request: %+v", captured)
	}
	if !strings.Contains(w.Body.String(), `"has_active_run":true`) {
		t.Fatalf("expected has_active_run true, got %s", w.Body.String())
	}
}

func TestSubscribeAgentRunEvents_AfterSeqQuery(t *testing.T) {
	var captured *pb.SubscribeAgentRunEventsRequest
	mock := &mockAIServiceClient{
		subscribeFn: func(ctx context.Context, req *pb.SubscribeAgentRunEventsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.AgentRunEvent], error) {
			captured = req
			return &mockAgentRunEventStream{
				ctx: ctx,
				events: []*pb.AgentRunEvent{
					{RunId: req.RunId, Seq: 11, EventType: "assistant.delta", Delta: "hi"},
				},
			}, nil
		},
	}
	r := newAgentRunTestRouter(mock, 42)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/ai/runs/9/events?after_seq=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if captured == nil || captured.AfterSeq != 10 || captured.RunId != 9 || captured.HrId != 42 {
		t.Fatalf("unexpected subscribe request: %+v", captured)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "id: 11\n") {
		t.Fatalf("expected SSE id line, got %q", body)
	}
	if !strings.Contains(body, `"event_type":"assistant.delta"`) {
		t.Fatalf("expected event payload, got %q", body)
	}
}

func TestSubscribeAgentRunEvents_LastEventIDTakesPrecedence(t *testing.T) {
	var captured *pb.SubscribeAgentRunEventsRequest
	mock := &mockAIServiceClient{
		subscribeFn: func(ctx context.Context, req *pb.SubscribeAgentRunEventsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.AgentRunEvent], error) {
			captured = req
			return &mockAgentRunEventStream{ctx: ctx, events: nil}, nil
		},
	}
	r := newAgentRunTestRouter(mock, 42)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/ai/runs/9/events?after_seq=10", nil)
	req.Header.Set("Last-Event-ID", "25")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if captured == nil || captured.AfterSeq != 25 {
		t.Fatalf("expected Last-Event-ID to win, got after_seq=%v captured=%+v", captured, captured)
	}
}

func TestSubscribeAgentRunEvents_InvalidAfterSeq(t *testing.T) {
	r := newAgentRunTestRouter(&mockAIServiceClient{}, 42)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/ai/runs/9/events?after_seq=nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if code, _ := body["code"].(float64); code != 400 {
		t.Fatalf("expected 400 for invalid after_seq, got %v body=%s", body["code"], w.Body.String())
	}
}

func TestSubscribeAgentRunEvents_DisconnectDoesNotCancelRun(t *testing.T) {
	mock := &mockAIServiceClient{}
	subscribed := make(chan struct{})
	mock.subscribeFn = func(ctx context.Context, req *pb.SubscribeAgentRunEventsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.AgentRunEvent], error) {
		close(subscribed)
		return &mockAgentRunEventStream{ctx: ctx, blockUntilCancel: true}, nil
	}

	r := newAgentRunTestRouter(mock, 42)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/hr/ai/runs/3/events", nil)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.ServeHTTP(w, req)
	}()

	select {
	case <-subscribed:
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe never started")
	}

	// Simulate HTTP client disconnect.
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after disconnect")
	}

	if mock.cancelCalled.Load() {
		t.Fatal("HTTP disconnect must not call CancelAgentRun")
	}
}

func TestCancelAgentRun_ForwardsExplicitCommand(t *testing.T) {
	var captured *pb.CancelAgentRunRequest
	mock := &mockAIServiceClient{
		cancelFn: func(_ context.Context, req *pb.CancelAgentRunRequest, _ ...grpc.CallOption) (*pb.CancelAgentRunResponse, error) {
			captured = req
			return &pb.CancelAgentRunResponse{
				Code: 0,
				Msg:  "ok",
				Run:  &pb.AgentRunSnapshot{RunId: req.RunId, Status: "cancel_requested"},
			}, nil
		},
	}
	r := newAgentRunTestRouter(mock, 42)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/ai/runs/15/cancel", strings.NewReader(`{"client_request_id":"c1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if captured == nil || captured.RunId != 15 || captured.HrId != 42 || captured.ClientRequestId != "c1" {
		t.Fatalf("unexpected cancel request: %+v", captured)
	}
	if mock.cancelCalled.Load() != true {
		t.Fatal("expected cancel RPC")
	}
}

func TestConfirmAgentRun_ForwardsSkills(t *testing.T) {
	var captured *pb.ConfirmAgentRunRequest
	mock := &mockAIServiceClient{
		confirmFn: func(_ context.Context, req *pb.ConfirmAgentRunRequest, _ ...grpc.CallOption) (*pb.ConfirmAgentRunResponse, error) {
			captured = req
			return &pb.ConfirmAgentRunResponse{
				Code: 0,
				Msg:  "ok",
				Run:  &pb.AgentRunSnapshot{RunId: req.RunId, Status: "running"},
			}, nil
		},
	}
	r := newAgentRunTestRouter(mock, 5)
	body := `{"agent_skill_ids":[1,2],"agent_skill_selection_confirmed":true,"client_request_id":"cf1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/ai/runs/8/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if captured == nil || captured.RunId != 8 || captured.HrId != 5 || !captured.AgentSkillSelectionConfirmed {
		t.Fatalf("unexpected confirm request: %+v", captured)
	}
	if len(captured.AgentSkillIds) != 2 {
		t.Fatalf("expected skill ids, got %+v", captured.AgentSkillIds)
	}
}

func TestResolveAfterSeq_Helpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("default zero", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/events", nil)
		seq, ok := resolveAfterSeq(c)
		if !ok || seq != 0 {
			t.Fatalf("got seq=%d ok=%v", seq, ok)
		}
	})

	t.Run("query", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/events?after_seq=4", nil)
		seq, ok := resolveAfterSeq(c)
		if !ok || seq != 4 {
			t.Fatalf("got seq=%d ok=%v", seq, ok)
		}
	})

	t.Run("header wins", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/events?after_seq=4", nil)
		c.Request.Header.Set("Last-Event-ID", "9")
		seq, ok := resolveAfterSeq(c)
		if !ok || seq != 9 {
			t.Fatalf("got seq=%d ok=%v", seq, ok)
		}
	})
}

func TestHRContextUsageMappingCoversBudgetFieldsForAllTransports(t *testing.T) {
	usage := &pb.ContextUsageInfo{
		InputBudgetTokens: 6758, SafetyMarginTokens: 410, BudgetUsageRatio: 0.3,
		BudgetStatus: "within_budget", IncludedMessageCount: 9, OmittedMessageCount: 2,
		SummaryApplied: true,
		Breakdown:      &pb.ContextUsageBreakdown{ToolSchemaTokens: 33, ProtocolOverheadTokens: 38},
	}
	mapped := mapHRContextUsage(usage)
	if mapped["input_budget_tokens"] != int32(6758) || mapped["safety_margin_tokens"] != int32(410) || mapped["budget_status"] != "within_budget" {
		t.Fatalf("budget mapping incomplete: %#v", mapped)
	}
	if mapped["included_message_count"] != int32(9) || mapped["omitted_message_count"] != int32(2) || mapped["summary_applied"] != true {
		t.Fatalf("message/summary mapping incomplete: %#v", mapped)
	}
	breakdown, ok := mapped["breakdown"].(map[string]any)
	if !ok || breakdown["tool_schema_tokens"] != int32(33) || breakdown["protocol_overhead_tokens"] != int32(38) {
		t.Fatalf("breakdown mapping incomplete: %#v", mapped["breakdown"])
	}
	metadata := agentRunResultMetadataPayload(&pb.AgentRunResultMetadata{ContextUsage: usage})
	if got, ok := metadata["context_usage"].(map[string]any); !ok || got["input_budget_tokens"] != int32(6758) {
		t.Fatalf("agent-run metadata context mapping incomplete: %#v", metadata)
	}
}
