package grpc

import (
	"context"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

func TestCandidateChatStreamPersistsMessagesWithCandidateOwnerRole(t *testing.T) {
	store := newFakeAIStore()
	provider := &fakeChatProvider{
		reply: "assistant reply",
		onComplete: func(prompt string) {
			if prompt != "candidate asks" {
				t.Fatalf("provider prompt = %q, want candidate asks", prompt)
			}
			if len(store.messages) != 1 {
				t.Fatalf("messages before provider = %d, want 1", len(store.messages))
			}
			userMessage := store.messages[0]
			if userMessage.OwnerRole != ownerRoleCandidate || userMessage.OwnerID != 55 || userMessage.Role != "user" || userMessage.Content != "candidate asks" {
				t.Fatalf("user message before provider = %#v", userMessage)
			}
		},
	}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, Message: "candidate asks"}, stream); err != nil {
		t.Fatalf("CandidateChatStream returned error: %v", err)
	}

	if len(store.ensureCalls) != 1 {
		t.Fatalf("ensure calls = %d, want 1", len(store.ensureCalls))
	}
	if call := store.ensureCalls[0]; call.ownerRole != ownerRoleCandidate || call.ownerID != 55 {
		t.Fatalf("ensure call = %#v, want candidate role/user", call)
	}
	if len(store.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(store.messages))
	}
	if got := store.messages[1]; got.OwnerRole != ownerRoleCandidate || got.OwnerID != 55 || got.Role != "assistant" || got.Content != "assistant reply" {
		t.Fatalf("assistant message = %#v", got)
	}
	if len(stream.responses) != 1 {
		t.Fatalf("stream responses = %d, want 1", len(stream.responses))
	}
	if resp := stream.responses[0]; !resp.Done || resp.EventType != "done" || resp.Delta != "assistant reply" || resp.SessionId != store.sessions[0].ID {
		t.Fatalf("stream response = %#v", resp)
	}
}

func TestHRChatPersistsMessagesWithHROwnerRole(t *testing.T) {
	store := newFakeAIStore()
	provider := &fakeChatProvider{reply: "hr reply"}
	service := &nativeAIService{store: store, provider: provider}

	resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "hr asks", ApplicationId: 99, ModelId: 123})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.GetCode() != 0 || resp.GetReply() != "hr reply" || resp.GetSessionId() == 0 {
		t.Fatalf("chat response = %#v", resp)
	}
	if len(store.ensureCalls) != 1 {
		t.Fatalf("ensure calls = %d, want 1", len(store.ensureCalls))
	}
	if call := store.ensureCalls[0]; call.ownerRole != ownerRoleHR || call.ownerID != 77 || call.applicationID != 99 {
		t.Fatalf("ensure call = %#v, want hr role/owner/application", call)
	}
	if len(store.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(store.messages))
	}
	for _, message := range store.messages {
		if message.OwnerRole != ownerRoleHR || message.OwnerID != 77 || message.SessionID != resp.GetSessionId() || message.ModelID != 123 {
			t.Fatalf("hr message = %#v", message)
		}
	}
	if store.messages[0].Role != "user" || store.messages[0].Content != "hr asks" {
		t.Fatalf("hr user message = %#v", store.messages[0])
	}
	if store.messages[1].Role != "assistant" || store.messages[1].Content != "hr reply" {
		t.Fatalf("hr assistant message = %#v", store.messages[1])
	}
}

type fakeChatProvider struct {
	reply      string
	onComplete func(prompt string)
}

func (p *fakeChatProvider) Complete(_ context.Context, prompt string) (string, error) {
	if p.onComplete != nil {
		p.onComplete(prompt)
	}
	return p.reply, nil
}

type captureChatStream struct {
	gogrpc.ServerStream
	ctx       context.Context
	responses []*pb.ChatStreamResponse
}

func (s *captureChatStream) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *captureChatStream) Send(resp *pb.ChatStreamResponse) error {
	s.responses = append(s.responses, resp)
	return nil
}

type ensureChatSessionCall struct {
	ownerRole     int32
	ownerID       int64
	title         string
	applicationID int64
}

type fakeAIStore struct {
	nextSessionID int64
	nextMessageID int64
	ensureCalls   []ensureChatSessionCall
	sessions      []ChatSessionRow
	messages      []ChatMessageRow
}

func newFakeAIStore() *fakeAIStore {
	return &fakeAIStore{nextSessionID: 100, nextMessageID: 200}
}

func (s *fakeAIStore) EnsureChatSession(_ context.Context, ownerRole int32, ownerID int64, title string, applicationID int64) (ChatSessionRow, error) {
	s.nextSessionID++
	now := time.Now()
	row := ChatSessionRow{ID: s.nextSessionID, Title: title, ApplicationID: applicationID, CreatedAt: now, UpdatedAt: now}
	s.ensureCalls = append(s.ensureCalls, ensureChatSessionCall{ownerRole: ownerRole, ownerID: ownerID, title: title, applicationID: applicationID})
	s.sessions = append(s.sessions, row)
	return row, nil
}

func (s *fakeAIStore) ListChatSessions(context.Context, int32, int64, int32, int32) ([]ChatSessionRow, int64, error) {
	return s.sessions, int64(len(s.sessions)), nil
}

func (s *fakeAIStore) UpdateChatSessionTitle(context.Context, int32, int64, int64, string) error {
	return nil
}

func (s *fakeAIStore) DeleteChatSession(context.Context, int32, int64, int64) error {
	return nil
}

func (s *fakeAIStore) AppendChatMessage(_ context.Context, message ChatMessageRow) (ChatMessageRow, error) {
	s.nextMessageID++
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	message.ID = s.nextMessageID
	s.messages = append(s.messages, message)
	return message, nil
}

func (s *fakeAIStore) ListChatMessages(context.Context, int32, int64, int64, int32, int32) ([]ChatMessageRow, error) {
	return s.messages, nil
}

func (s *fakeAIStore) ListToolTraces(context.Context, int64, int64) ([]ToolTraceRow, error) {
	return nil, nil
}

func (s *fakeAIStore) CreateAgentRun(context.Context, AgentRunRow) (AgentRunRow, bool, error) {
	return AgentRunRow{}, false, nil
}

func (s *fakeAIStore) ListAgentRuns(context.Context, int64, int64) ([]AgentRunRow, error) {
	return nil, nil
}

func (s *fakeAIStore) GetAgentRun(context.Context, int64, int64) (AgentRunRow, bool, error) {
	return AgentRunRow{}, false, nil
}

func (s *fakeAIStore) GetActiveAgentRun(context.Context, int64, int64) (AgentRunRow, bool, error) {
	return AgentRunRow{}, false, nil
}

func (s *fakeAIStore) UpdateAgentRunStatus(context.Context, int64, int64, string) (AgentRunRow, bool, error) {
	return AgentRunRow{}, false, nil
}

func (s *fakeAIStore) CompleteAgentRun(context.Context, int64, int64, string, string, string, string) (AgentRunRow, bool, error) {
	return AgentRunRow{}, false, nil
}

func (s *fakeAIStore) AppendAgentRunEvent(context.Context, int64, string, string) (AgentRunEventRow, error) {
	return AgentRunEventRow{}, nil
}

func (s *fakeAIStore) ListAgentRunEvents(context.Context, int64, int64, int64) ([]AgentRunEventRow, error) {
	return nil, nil
}

func (s *fakeAIStore) ListLlmProviders(context.Context, int32, int32) ([]*pb.LlmProviderInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListLlmModels(context.Context, int32, int32, int64) ([]*pb.LlmModelInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListPromptTemplates(context.Context, int32, int32, string) ([]*pb.PromptTemplateInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListAgentConfigs(context.Context, int32, int32, string) ([]*pb.AgentConfigInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListMCPServers(context.Context, int32, int32) ([]*pb.MCPServerInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListAgentSkills(context.Context, int32, int32, string, bool) ([]*pb.AgentSkillInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListEmbeddingProviders(context.Context, int32, int32) ([]*pb.EmbeddingProviderInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListEmbeddingModels(context.Context, int32, int32, int64) ([]*pb.EmbeddingModelInfo, int64, error) {
	return nil, 0, nil
}
