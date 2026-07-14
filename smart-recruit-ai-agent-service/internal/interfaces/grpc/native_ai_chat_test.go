package grpc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-proto/recruitment/pb"
)

func TestCandidateChatStreamPersistsMessagesWithCandidateOwnerRole(t *testing.T) {
	store := newFakeAIStore()
	provider := &fakeChatProvider{
		reply: "assistant reply",
		onComplete: func(prompt string) {
			assertPromptContains(t, prompt, candidateSystemPrompt)
			assertPromptContains(t, prompt, "user:\ncandidate asks")
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

func TestCandidateChatStreamRejectsEmptyMessage(t *testing.T) {
	store := newFakeAIStore()
	provider := &fakeChatProvider{reply: "must not be called"}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, Message: " \n\t "}, stream); err != nil {
		t.Fatalf("CandidateChatStream returned error: %v", err)
	}

	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
	if len(store.ensureCalls) != 0 || len(store.lookupCalls) != 0 {
		t.Fatalf("session calls ensure=%d lookup=%d, want none", len(store.ensureCalls), len(store.lookupCalls))
	}
	if len(store.messages) != 0 {
		t.Fatalf("messages = %#v, want none", store.messages)
	}
	if len(stream.responses) != 1 {
		t.Fatalf("stream responses = %d, want 1", len(stream.responses))
	}
	if resp := stream.responses[0]; resp.Code != agentRunCodeBadRequest || !resp.Done || resp.EventType != "done" || resp.SessionId != 0 {
		t.Fatalf("stream response = %#v, want bad request done without session", resp)
	}
}

func TestCandidateChatStreamBuildsPromptWithActiveCandidatePromptAndHistory(t *testing.T) {
	store := newFakeAIStore()
	session := store.seedChatSession(ownerRoleCandidate, 55, 1101, "owned candidate session")
	store.activePrompt = &pb.PromptTemplateInfo{
		Content:    "ACTIVE candidate assistant system prompt",
		IsActive:   true,
		AgentType:  candidateAssistantAgentType,
		PromptRole: candidatePromptRoleSystem,
	}
	store.seedChatMessage(ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: 55, SessionID: session.ID, Role: "user", Content: "previous candidate question"})
	store.seedChatMessage(ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: 55, SessionID: session.ID, Role: "assistant", Content: "previous assistant answer"})
	store.seedChatMessage(ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: 55, SessionID: session.ID, Role: "user", Content: "hr secret should not appear"})
	store.seedChatMessage(ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: 66, SessionID: session.ID, Role: "user", Content: "other candidate should not appear"})
	store.seedChatMessage(ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: 55, SessionID: session.ID + 1, Role: "user", Content: "other session should not appear"})

	provider := &fakeChatProvider{
		reply: "assistant reply",
		onComplete: func(prompt string) {
			assertPromptOrder(t, prompt, []string{
				"System:\nACTIVE candidate assistant system prompt",
				"user:\nprevious candidate question",
				"assistant:\nprevious assistant answer",
				"user:\ncurrent candidate question",
				"Assistant:",
			})
			assertPromptNotContains(t, prompt, candidateSystemPrompt)
			assertPromptNotContains(t, prompt, "hr secret should not appear")
			assertPromptNotContains(t, prompt, "other candidate should not appear")
			assertPromptNotContains(t, prompt, "other session should not appear")
		},
	}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, SessionId: session.ID, Message: "current candidate question"}, stream); err != nil {
		t.Fatalf("CandidateChatStream returned error: %v", err)
	}

	if len(store.activePromptCalls) != 1 {
		t.Fatalf("active prompt calls = %d, want 1", len(store.activePromptCalls))
	}
	if call := store.activePromptCalls[0]; call.agentType != candidateAssistantAgentType || call.promptRole != candidatePromptRoleSystem {
		t.Fatalf("active prompt call = %#v, want candidate system prompt", call)
	}
	if len(store.listMessageCalls) != 1 {
		t.Fatalf("list message calls = %d, want 1", len(store.listMessageCalls))
	}
	if call := store.listMessageCalls[0]; call.ownerRole != ownerRoleCandidate || call.ownerID != 55 || call.sessionID != session.ID {
		t.Fatalf("list message call = %#v, want candidate owner/session", call)
	}
}

func TestCandidateChatStreamUsesFallbackCandidatePromptWhenActivePromptMissing(t *testing.T) {
	store := newFakeAIStore()
	provider := &fakeChatProvider{
		reply: "fallback reply",
		onComplete: func(prompt string) {
			assertPromptContains(t, prompt, candidateSystemPrompt)
			assertPromptContains(t, prompt, "user:\nneed candidate help")
		},
	}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, Message: "need candidate help"}, stream); err != nil {
		t.Fatalf("CandidateChatStream returned error: %v", err)
	}
}

func TestCandidateChatStreamMissingProviderReturnsDiagnosticError(t *testing.T) {
	store := newFakeAIStore()
	service := &nativeAIService{store: store}
	stream := &captureChatStream{ctx: context.Background()}

	err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, Message: "candidate asks"}, stream)
	if !errors.Is(err, errAIProviderRequired) {
		t.Fatalf("CandidateChatStream error = %v, want %v", err, errAIProviderRequired)
	}
	if len(store.messages) != 1 {
		t.Fatalf("messages = %d, want persisted user message before provider error", len(store.messages))
	}
	if len(stream.responses) != 0 {
		t.Fatalf("stream responses = %#v, want none on provider error", stream.responses)
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

func TestHRChatExistingSessionRequiresOwner(t *testing.T) {
	tests := []struct {
		name      string
		seed      func(*fakeAIStore) int64
		sessionID int64
	}{
		{
			name: "foreign hr session",
			seed: func(store *fakeAIStore) int64 {
				return store.seedChatSession(ownerRoleHR, 88, 901, "foreign hr").ID
			},
		},
		{
			name: "candidate collision session",
			seed: func(store *fakeAIStore) int64 {
				return store.seedChatSession(ownerRoleCandidate, 77, 902, "candidate collision").ID
			},
		},
		{name: "missing session", sessionID: 903},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAIStore()
			sessionID := tt.sessionID
			if tt.seed != nil {
				sessionID = tt.seed(store)
			}
			provider := &fakeChatProvider{reply: "must not be called"}
			service := &nativeAIService{store: store, provider: provider}

			resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, SessionId: sessionID, Message: "hr asks"})
			if status.Code(err) != codes.NotFound {
				t.Fatalf("Chat status code = %v, want %v (err = %v)", status.Code(err), codes.NotFound, err)
			}
			if resp != nil {
				t.Fatalf("Chat response = %#v, want nil", resp)
			}
			if provider.calls != 0 {
				t.Fatalf("provider calls = %d, want 0", provider.calls)
			}
			if len(store.messages) != 0 {
				t.Fatalf("messages = %#v, want none", store.messages)
			}
			if len(store.lookupCalls) != 1 {
				t.Fatalf("lookup calls = %d, want 1", len(store.lookupCalls))
			}
			if call := store.lookupCalls[0]; call.ownerRole != ownerRoleHR || call.ownerID != 77 || call.sessionID != sessionID {
				t.Fatalf("lookup call = %#v, want hr owner/session", call)
			}
			if len(store.ensureCalls) != 0 {
				t.Fatalf("ensure calls = %d, want 0 for existing session_id", len(store.ensureCalls))
			}
		})
	}
}

func TestHRChatStreamExistingSessionRequiresOwner(t *testing.T) {
	store := newFakeAIStore()
	session := store.seedChatSession(ownerRoleCandidate, 77, 904, "candidate collision")
	provider := &fakeChatProvider{reply: "must not be called"}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	err := service.ChatStream(&pb.ChatRequest{HrId: 77, SessionId: session.ID, Message: "hr stream asks"}, stream)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("ChatStream status code = %v, want %v (err = %v)", status.Code(err), codes.NotFound, err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
	if len(store.messages) != 0 {
		t.Fatalf("messages = %#v, want none", store.messages)
	}
	if len(stream.responses) != 0 {
		t.Fatalf("stream responses = %#v, want none", stream.responses)
	}
}

func TestCandidateChatStreamExistingSessionRequiresOwner(t *testing.T) {
	tests := []struct {
		name      string
		seed      func(*fakeAIStore) int64
		sessionID int64
	}{
		{
			name: "foreign candidate session",
			seed: func(store *fakeAIStore) int64 {
				return store.seedChatSession(ownerRoleCandidate, 66, 1001, "foreign candidate").ID
			},
		},
		{
			name: "hr collision session",
			seed: func(store *fakeAIStore) int64 {
				return store.seedChatSession(ownerRoleHR, 55, 1002, "hr collision").ID
			},
		},
		{name: "missing session", sessionID: 1003},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAIStore()
			sessionID := tt.sessionID
			if tt.seed != nil {
				sessionID = tt.seed(store)
			}
			provider := &fakeChatProvider{reply: "must not be called"}
			service := &nativeAIService{store: store, provider: provider}
			stream := &captureChatStream{ctx: context.Background()}

			err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, SessionId: sessionID, Message: "candidate asks"}, stream)
			if status.Code(err) != codes.NotFound {
				t.Fatalf("CandidateChatStream status code = %v, want %v (err = %v)", status.Code(err), codes.NotFound, err)
			}
			if provider.calls != 0 {
				t.Fatalf("provider calls = %d, want 0", provider.calls)
			}
			if len(store.messages) != 0 {
				t.Fatalf("messages = %#v, want none", store.messages)
			}
			if len(stream.responses) != 0 {
				t.Fatalf("stream responses = %#v, want none", stream.responses)
			}
			if len(store.lookupCalls) != 1 {
				t.Fatalf("lookup calls = %d, want 1", len(store.lookupCalls))
			}
			if call := store.lookupCalls[0]; call.ownerRole != ownerRoleCandidate || call.ownerID != 55 || call.sessionID != sessionID {
				t.Fatalf("lookup call = %#v, want candidate owner/session", call)
			}
			if len(store.ensureCalls) != 0 {
				t.Fatalf("ensure calls = %d, want 0 for existing session_id", len(store.ensureCalls))
			}
		})
	}
}

type fakeChatProvider struct {
	reply      string
	onComplete func(prompt string)
	calls      int
	prompts    []string
}

func (p *fakeChatProvider) Complete(_ context.Context, prompt string) (string, error) {
	p.calls++
	p.prompts = append(p.prompts, prompt)
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

func assertPromptContains(t *testing.T, prompt, want string) {
	t.Helper()
	if !strings.Contains(prompt, want) {
		t.Fatalf("provider prompt = %q, want to contain %q", prompt, want)
	}
}

func assertPromptNotContains(t *testing.T, prompt, forbidden string) {
	t.Helper()
	if strings.Contains(prompt, forbidden) {
		t.Fatalf("provider prompt = %q, want not to contain %q", prompt, forbidden)
	}
}

func assertPromptOrder(t *testing.T, prompt string, parts []string) {
	t.Helper()
	offset := 0
	for _, part := range parts {
		index := strings.Index(prompt[offset:], part)
		if index < 0 {
			t.Fatalf("provider prompt = %q, want %q after byte offset %d", prompt, part, offset)
		}
		offset += index + len(part)
	}
}

type ensureChatSessionCall struct {
	ownerRole     int32
	ownerID       int64
	title         string
	applicationID int64
}

type lookupChatSessionCall struct {
	ownerRole int32
	ownerID   int64
	sessionID int64
}

type listChatMessagesCall struct {
	ownerRole int32
	ownerID   int64
	sessionID int64
	page      int32
	pageSize  int32
}

type activePromptCall struct {
	agentType  string
	promptRole string
}

type fakeChatSessionOwner struct {
	ownerRole int32
	ownerID   int64
}

type fakeAIStore struct {
	nextSessionID     int64
	nextMessageID     int64
	ensureCalls       []ensureChatSessionCall
	lookupCalls       []lookupChatSessionCall
	listMessageCalls  []listChatMessagesCall
	activePromptCalls []activePromptCall
	sessionOwners     map[int64]fakeChatSessionOwner
	sessions          []ChatSessionRow
	messages          []ChatMessageRow
	activePrompt      *pb.PromptTemplateInfo
	activePromptErr   error
	promptTemplates   []*pb.PromptTemplateInfo
}

func newFakeAIStore() *fakeAIStore {
	return &fakeAIStore{nextSessionID: 100, nextMessageID: 200, sessionOwners: make(map[int64]fakeChatSessionOwner)}
}

func (s *fakeAIStore) EnsureChatSession(_ context.Context, ownerRole int32, ownerID int64, title string, applicationID int64) (ChatSessionRow, error) {
	s.nextSessionID++
	now := time.Now()
	row := ChatSessionRow{ID: s.nextSessionID, Title: title, ApplicationID: applicationID, CreatedAt: now, UpdatedAt: now}
	s.ensureCalls = append(s.ensureCalls, ensureChatSessionCall{ownerRole: ownerRole, ownerID: ownerID, title: title, applicationID: applicationID})
	s.sessions = append(s.sessions, row)
	s.sessionOwners[row.ID] = fakeChatSessionOwner{ownerRole: ownerRole, ownerID: ownerID}
	return row, nil
}

func (s *fakeAIStore) GetChatSession(_ context.Context, ownerRole int32, ownerID, sessionID int64) (ChatSessionRow, bool, error) {
	s.lookupCalls = append(s.lookupCalls, lookupChatSessionCall{ownerRole: ownerRole, ownerID: ownerID, sessionID: sessionID})
	for _, row := range s.sessions {
		if row.ID != sessionID {
			continue
		}
		owner, ok := s.sessionOwners[sessionID]
		if !ok || owner.ownerRole != ownerRole || owner.ownerID != ownerID {
			return ChatSessionRow{}, false, nil
		}
		return row, true, nil
	}
	return ChatSessionRow{}, false, nil
}

func (s *fakeAIStore) seedChatSession(ownerRole int32, ownerID, sessionID int64, title string) ChatSessionRow {
	now := time.Now()
	row := ChatSessionRow{ID: sessionID, Title: title, CreatedAt: now, UpdatedAt: now}
	s.sessions = append(s.sessions, row)
	s.sessionOwners[row.ID] = fakeChatSessionOwner{ownerRole: ownerRole, ownerID: ownerID}
	return row
}

func (s *fakeAIStore) seedChatMessage(message ChatMessageRow) ChatMessageRow {
	s.nextMessageID++
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	message.ID = s.nextMessageID
	s.messages = append(s.messages, message)
	return message
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
	return s.seedChatMessage(message), nil
}

func (s *fakeAIStore) ListChatMessages(_ context.Context, ownerRole int32, ownerID, sessionID int64, page, pageSize int32) ([]ChatMessageRow, error) {
	s.listMessageCalls = append(s.listMessageCalls, listChatMessagesCall{ownerRole: ownerRole, ownerID: ownerID, sessionID: sessionID, page: page, pageSize: pageSize})
	filtered := make([]ChatMessageRow, 0, len(s.messages))
	for _, message := range s.messages {
		if message.OwnerRole != ownerRole || message.OwnerID != ownerID {
			continue
		}
		if sessionID > 0 && message.SessionID != sessionID {
			continue
		}
		filtered = append(filtered, message)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = int32(len(filtered))
	}
	start := int((page - 1) * pageSize)
	if start >= len(filtered) {
		return []ChatMessageRow{}, nil
	}
	end := start + int(pageSize)
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], nil
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

func (s *fakeAIStore) ListPromptTemplates(_ context.Context, _ int32, _ int32, agentType string) ([]*pb.PromptTemplateInfo, int64, error) {
	items := make([]*pb.PromptTemplateInfo, 0, len(s.promptTemplates))
	for _, template := range s.promptTemplates {
		if template == nil {
			continue
		}
		if strings.TrimSpace(agentType) != "" && template.GetAgentType() != strings.TrimSpace(agentType) {
			continue
		}
		items = append(items, template)
	}
	return items, int64(len(items)), nil
}

func (s *fakeAIStore) GetActivePromptByAgentType(_ context.Context, req *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error) {
	s.activePromptCalls = append(s.activePromptCalls, activePromptCall{agentType: req.GetAgentType(), promptRole: req.GetPromptRole()})
	if s.activePromptErr != nil {
		return nil, s.activePromptErr
	}
	if s.activePrompt == nil || !s.activePrompt.GetIsActive() {
		return &pb.GetActivePromptByAgentTypeResponse{Code: 404, Msg: "active prompt not found"}, nil
	}
	if strings.TrimSpace(req.GetAgentType()) != "" && s.activePrompt.GetAgentType() != strings.TrimSpace(req.GetAgentType()) {
		return &pb.GetActivePromptByAgentTypeResponse{Code: 404, Msg: "active prompt not found"}, nil
	}
	if strings.TrimSpace(req.GetPromptRole()) != "" && s.activePrompt.GetPromptRole() != strings.TrimSpace(req.GetPromptRole()) {
		return &pb.GetActivePromptByAgentTypeResponse{Code: 404, Msg: "active prompt not found"}, nil
	}
	return &pb.GetActivePromptByAgentTypeResponse{Code: 0, Msg: "success", Template: s.activePrompt}, nil
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
