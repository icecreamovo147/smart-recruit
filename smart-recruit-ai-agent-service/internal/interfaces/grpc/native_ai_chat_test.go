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

func TestCandidateChatStreamIncludesScopedRuntimeContextAndToolTraces(t *testing.T) {
	store := newFakeAIStore()
	store.candidateContext = CandidateRuntimeContext{
		Applications: []CandidateApplicationContext{{ApplicationID: 7001, JobID: 9001, JobTitle: "Backend Engineer", StatusKey: "interview", StatusText: "面试中", AppliedAt: "2026-07-01T10:00:00Z"}},
		Resume:       CandidateResumeContext{Available: true, ResumeID: 8001, FileName: "resume.pdf", TextLength: 42, Summary: "Go backend candidate"},
		Jobs:         []CandidateJobContext{{JobID: 9001, Title: "Backend Engineer", Status: 1, StatusText: "招募中", HasApplied: true}},
		Interviews:   []CandidateInterviewContext{{InterviewID: 6001, ApplicationID: 7001, JobTitle: "Backend Engineer", RoundNo: 1, Title: "初试", Status: "scheduled"}},
		Offers:       []CandidateOfferContext{{OfferID: 5001, ApplicationID: 7001, JobID: 9001, Title: "Backend Engineer", Status: "sent"}},
	}
	provider := &fakeChatProvider{
		reply: "assistant reply",
		onComplete: func(prompt string) {
			assertPromptContains(t, prompt, "Candidate runtime context (current candidate only")
			assertPromptContains(t, prompt, `"application_id": 7001`)
			assertPromptContains(t, prompt, `"file_name": "resume.pdf"`)
			assertPromptContains(t, prompt, `"has_applied": true`)
			assertPromptContains(t, prompt, `"interview_id": 6001`)
			assertPromptContains(t, prompt, `"offer_id": 5001`)
		},
	}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, Message: "show my progress"}, stream); err != nil {
		t.Fatalf("CandidateChatStream returned error: %v", err)
	}

	if len(store.toolTraces) != 5 {
		t.Fatalf("tool traces = %d, want 5", len(store.toolTraces))
	}
	if store.toolTraces[0].ToolName != "list_my_applications" || !strings.Contains(store.toolTraces[0].ResultContent, "Backend Engineer") {
		t.Fatalf("application trace = %#v", store.toolTraces[0])
	}
	for _, trace := range store.toolTraces {
		if trace.ToolName == "get_my_resume_text" && strings.Contains(trace.ResultContent, "Go backend candidate") {
			t.Fatalf("resume trace leaked summary text: %s", trace.ResultContent)
		}
	}
}

func TestCandidateChatStreamStripsSuggestedQuestionsPersistsCleanReplyAndAudits(t *testing.T) {
	store := newFakeAIStore()
	provider := &fakeChatProvider{reply: "clean assistant reply\n" + candidateSuggestedQuestionsStartMarker + "\n[\"Q1\",\"Q2\",\"Q3\"]\n" + candidateSuggestedQuestionsEndMarker}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, Message: "candidate asks"}, stream); err != nil {
		t.Fatalf("CandidateChatStream returned error: %v", err)
	}

	if got := store.messages[len(store.messages)-1].Content; got != "clean assistant reply" {
		t.Fatalf("assistant content = %q, want clean reply", got)
	}
	if len(stream.responses) != 1 {
		t.Fatalf("stream responses = %d, want 1", len(stream.responses))
	}
	if got := stream.responses[0].GetSuggestedQuestions(); len(got) != 3 || got[0] != "Q1" || got[2] != "Q3" {
		t.Fatalf("suggested questions = %#v", got)
	}
	if len(store.usageAudits) != 1 {
		t.Fatalf("usage audits = %d, want 1", len(store.usageAudits))
	}
	if audit := store.usageAudits[0]; audit.UserID != 55 || audit.PermissionKey != "ai.candidate.use" || audit.Status != "ok" || audit.ResponseChars != len([]rune("clean assistant reply")) {
		t.Fatalf("usage audit = %#v", audit)
	}
}

func TestCandidateChatStreamFallsBackFromCandidateContextWhenProviderFails(t *testing.T) {
	store := newFakeAIStore()
	store.candidateContext = CandidateRuntimeContext{
		Applications: []CandidateApplicationContext{{ApplicationID: 7001, JobID: 9001, JobTitle: "Backend Engineer", StatusText: "面试中", AppliedAt: "2026-07-01T10:00:00Z"}},
	}
	provider := &fakeChatProvider{err: errors.New("provider unavailable")}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.CandidateChatStream(&pb.CandidateChatRequest{UserId: 55, Message: "my applications"}, stream); err != nil {
		t.Fatalf("CandidateChatStream returned error: %v", err)
	}

	if len(stream.responses) != 2 {
		t.Fatalf("stream responses = %d, want partial_done and done", len(stream.responses))
	}
	if stream.responses[0].EventType != "partial_done" || !stream.responses[1].Done {
		t.Fatalf("stream responses = %#v", stream.responses)
	}
	if got := store.messages[len(store.messages)-1].Content; !strings.Contains(got, "Backend Engineer") {
		t.Fatalf("fallback assistant content = %q, want application summary", got)
	}
	if audit := store.usageAudits[0]; audit.Status != "error" || audit.ErrorCode != "provider_error" {
		t.Fatalf("usage audit = %#v", audit)
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

func TestHRChatRuntimeUsesApplicationToolContextAndPersistsTrace(t *testing.T) {
	store := newFakeAIStore()
	apps := &fakeApplicationSnapshotClient{response: &pb.GetApplicationSnapshotResponse{
		Code:            0,
		ApplicationId:   99,
		CandidateName:   "Ada",
		JobTitle:        "Backend Engineer",
		LegacyStatus:    2,
		StatusKey:       "screening",
		CandidateUserId: 55,
		JobId:           88,
		ResumeId:        66,
		RoundNo:         1,
		IsCurrent:       true,
	}}
	provider := &fakeChatProvider{
		reply: "Ada is screening for Backend Engineer.",
		onComplete: func(prompt string) {
			assertPromptContains(t, prompt, "Tool results:")
			assertPromptContains(t, prompt, "get_application_snapshot")
			assertPromptContains(t, prompt, "Ada")
			assertPromptContains(t, prompt, "Backend Engineer")
			if strings.TrimSpace(prompt) == "summarize application" {
				t.Fatalf("provider received raw prompt without HR runtime context")
			}
		},
	}
	service := newNativeAIService(store, provider, apps)

	resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "summarize application", ApplicationId: 99, ModelId: 123})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.GetCode() != 0 || resp.GetReply() != "Ada is screening for Backend Engineer." || resp.GetCandidateName() != "Ada" || resp.GetJobTitle() != "Backend Engineer" || resp.GetStatus() != 2 {
		t.Fatalf("chat response = %#v", resp)
	}
	if resp.GetContextUsage() == nil || !resp.GetContextUsage().GetEstimated() || resp.GetContextUsage().GetPromptTokensEstimated() == 0 {
		t.Fatalf("context usage = %#v, want estimated prompt usage", resp.GetContextUsage())
	}
	if len(store.toolTraces) != 1 {
		t.Fatalf("tool traces = %d, want 1", len(store.toolTraces))
	}
	trace := store.toolTraces[0]
	if trace.ToolName != "get_application_snapshot" || trace.SessionID != resp.GetSessionId() || !strings.Contains(trace.ResultContent, "Ada") || trace.ErrorMsg != "" {
		t.Fatalf("tool trace = %#v", trace)
	}
	if len(store.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(store.messages))
	}
	if got := store.messages[1]; got.Role != "assistant" || got.ProcessContent == "" || !strings.Contains(got.ProcessContent, "native-hr-runtime") {
		t.Fatalf("assistant message = %#v, want process content", got)
	}
}

func TestHRChatRuntimeAppliesAgentPromptAndManualAgentSkills(t *testing.T) {
	store := newFakeAIStore()
	store.agentConfigs = []*pb.AgentConfigInfo{
		{
			Id:                  501,
			Name:                "default-hr",
			DisplayName:         "Default HR Agent",
			AgentType:           hrRecruitingAgentType,
			PromptTemplateId:    901,
			Instruction:         "Prioritize recruiting governance instruction.",
			MaxIterations:       4,
			TemperatureOverride: 0.33,
			IsDefault:           true,
			IsEnabled:           true,
			ToolBindings: []*pb.AgentToolBindingInfo{
				{ToolName: hrApplicationSnapshotTool, IsEnabled: true},
			},
			CapabilityBindings: []*pb.AgentCapabilityBindingInfo{
				{CapabilitySource: "builtin", CapabilityKey: hrCandidateSearchCapability, IsEnabled: true, Priority: 10},
			},
		},
	}
	store.promptByID[901] = &pb.PromptTemplateInfo{Id: 901, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem, IsActive: true, Content: "Active HR prompt from governance."}
	store.agentSkills = []*pb.AgentSkillInfo{
		{Id: 7001, Name: "candidate_screen", DisplayName: "Candidate Screen", Description: "Screen the candidate", AgentType: hrRecruitingAgentType, IsEnabled: true, IsManualInvocable: true, Priority: 9, RiskLevel: "medium", RequiredCapabilities: []string{hrCandidateSearchCapability}},
		{Id: 7002, Name: "auto_should_not_fill", DisplayName: "Auto", AgentType: hrRecruitingAgentType, IsEnabled: true, IsManualInvocable: true, Priority: 99, RequiredCapabilities: []string{hrCandidateSearchCapability}},
	}
	apps := &fakeApplicationSnapshotClient{response: &pb.GetApplicationSnapshotResponse{Code: 0, ApplicationId: 99, CandidateName: "Ada", JobTitle: "Backend Engineer"}}
	provider := &fakeChatProvider{
		reply: "governed reply",
		onComplete: func(prompt string) {
			assertPromptContains(t, prompt, "Active HR prompt from governance.")
			assertPromptContains(t, prompt, "Prioritize recruiting governance instruction.")
			assertPromptContains(t, prompt, `"capability_keys":["candidate_search"]`)
			assertPromptContains(t, prompt, `"mode":"manual"`)
			assertPromptContains(t, prompt, "candidate_screen")
			assertPromptNotContains(t, prompt, "auto_should_not_fill")
		},
	}
	service := newNativeAIService(store, provider, apps)

	resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "screen this candidate", ApplicationId: 99, AgentSkillIds: []int64{7001}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.GetCode() != 0 {
		t.Fatalf("chat response = %#v, want success", resp)
	}
	if len(apps.calls) != 1 {
		t.Fatalf("application snapshot calls = %d, want 1", len(apps.calls))
	}
	if len(provider.optionCalls) != 1 || provider.optionCalls[0].TemperatureOverride == nil || *provider.optionCalls[0].TemperatureOverride != 0.33 {
		t.Fatalf("provider option calls = %#v, want temperature override 0.33", provider.optionCalls)
	}
	if len(store.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(store.messages))
	}
	for _, message := range store.messages {
		if len(message.AgentSkillIDs) != 1 || message.AgentSkillIDs[0] != 7001 || len(message.AgentSkillNames) != 1 || message.AgentSkillNames[0] != "candidate_screen" {
			t.Fatalf("message skill metadata = %#v, want selected candidate_screen", message)
		}
	}
	if !strings.Contains(store.messages[1].ProcessContent, `"agent_skill_selection_mode":"manual"`) || !strings.Contains(store.messages[1].ProcessContent, `"prompt_template_id":901`) {
		t.Fatalf("assistant process content = %s, want governance metadata", store.messages[1].ProcessContent)
	}
}

func TestHRChatRuntimeCapabilityBindingsRestrictApplicationTool(t *testing.T) {
	tests := []struct {
		name        string
		toolEnabled bool
		capability  string
		selected    []string
	}{
		{name: "capability missing", toolEnabled: true, capability: "resume_intelligence", selected: []string{"resume_intelligence"}},
		{name: "tool disabled", toolEnabled: false, capability: hrCandidateSearchCapability, selected: []string{hrCandidateSearchCapability}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAIStore()
			store.agentConfigs = []*pb.AgentConfigInfo{
				{
					Id:        502,
					Name:      "restricted",
					AgentType: hrRecruitingAgentType,
					IsDefault: true,
					IsEnabled: true,
					ToolBindings: []*pb.AgentToolBindingInfo{
						{ToolName: hrApplicationSnapshotTool, IsEnabled: tt.toolEnabled},
					},
					CapabilityBindings: []*pb.AgentCapabilityBindingInfo{
						{CapabilitySource: "builtin", CapabilityKey: tt.capability, IsEnabled: true, Priority: 10},
					},
				},
			}
			apps := &fakeApplicationSnapshotClient{response: &pb.GetApplicationSnapshotResponse{Code: 0, ApplicationId: 99, CandidateName: "Ada"}}
			provider := &fakeChatProvider{
				reply: "tool restricted reply",
				onComplete: func(prompt string) {
					assertPromptContains(t, prompt, "application snapshot tool is not enabled")
					assertPromptNotContains(t, prompt, `"candidate_name":"Ada"`)
				},
			}
			service := newNativeAIService(store, provider, apps)

			resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "summarize application", ApplicationId: 99, SkillCapabilityKeys: tt.selected})
			if err != nil {
				t.Fatalf("Chat returned error: %v", err)
			}
			if resp.GetCode() != 0 || resp.GetReply() != "tool restricted reply" {
				t.Fatalf("chat response = %#v, want provider reply with restricted tool trace", resp)
			}
			if len(apps.calls) != 0 {
				t.Fatalf("application snapshot calls = %d, want 0 when capability/tool is unavailable", len(apps.calls))
			}
			if len(store.toolTraces) != 1 || store.toolTraces[0].ErrorMsg == "" {
				t.Fatalf("tool traces = %#v, want restricted tool trace", store.toolTraces)
			}
		})
	}
}

func TestHRChatRuntimeAutoSelectsEligibleAgentSkill(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkills = []*pb.AgentSkillInfo{
		{Id: 7101, Name: "resume_match", DisplayName: "Resume Match", AgentType: hrRecruitingAgentType, IsEnabled: true, IsManualInvocable: true, Priority: 5, Category: "candidate", RequiredCapabilities: []string{hrCandidateSearchCapability}},
	}
	provider := &fakeChatProvider{
		reply: "auto selected reply",
		onComplete: func(prompt string) {
			assertPromptContains(t, prompt, `"mode":"auto"`)
			assertPromptContains(t, prompt, "resume_match")
		},
	}
	service := newNativeAIService(store, provider, nil)

	resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "analyze candidate resume match"})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.GetCode() != 0 {
		t.Fatalf("chat response = %#v, want success", resp)
	}
	if len(store.messages) != 2 || len(store.messages[0].AgentSkillIDs) != 1 || store.messages[0].AgentSkillIDs[0] != 7101 {
		t.Fatalf("messages = %#v, want auto selected skill metadata on user message", store.messages)
	}
}

func TestHRChatRuntimeSelectedCapabilitiesRestrictAgentSkillSelection(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkills = []*pb.AgentSkillInfo{
		{Id: 7101, Name: "candidate_search_skill", DisplayName: "Candidate Search", AgentType: hrRecruitingAgentType, IsEnabled: true, IsManualInvocable: true, Priority: 9, Category: "candidate", RequiredCapabilities: []string{hrCandidateSearchCapability}},
		{Id: 7102, Name: "resume_skill", DisplayName: "Resume", AgentType: hrRecruitingAgentType, IsEnabled: true, IsManualInvocable: true, Priority: 5, Category: "resume", RequiredCapabilities: []string{"resume_intelligence"}},
	}
	provider := &fakeChatProvider{
		reply: "selected capability reply",
		onComplete: func(prompt string) {
			assertPromptContains(t, prompt, "resume_skill")
			assertPromptNotContains(t, prompt, "candidate_search_skill")
		},
	}
	service := newNativeAIService(store, provider, nil)

	resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "analyze candidate resume match", SkillCapabilityKeys: []string{"resume_intelligence"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.GetCode() != 0 {
		t.Fatalf("chat response = %#v, want success", resp)
	}
	if len(store.messages) != 2 || len(store.messages[0].AgentSkillIDs) != 1 || store.messages[0].AgentSkillIDs[0] != 7102 {
		t.Fatalf("messages = %#v, want only resume skill selected", store.messages)
	}
}

func TestHRChatRuntimeFallsBackFromToolTraceWhenProviderFails(t *testing.T) {
	store := newFakeAIStore()
	apps := &fakeApplicationSnapshotClient{response: &pb.GetApplicationSnapshotResponse{Code: 0, ApplicationId: 99, CandidateName: "Ada", JobTitle: "Backend Engineer", LegacyStatus: 2}}
	provider := &fakeChatProvider{err: errors.New("model timeout")}
	service := newNativeAIService(store, provider, apps)

	resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "summarize application", ApplicationId: 99})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.GetCode() != 0 || !strings.Contains(resp.GetReply(), "AI 模型回答失败") {
		t.Fatalf("chat response = %#v, want deterministic fallback", resp)
	}
	if len(store.toolTraces) != 1 {
		t.Fatalf("tool traces = %d, want 1", len(store.toolTraces))
	}
	if len(store.messages) != 2 || !strings.Contains(store.messages[1].ProcessContent, `"fallback_used":true`) {
		t.Fatalf("messages = %#v, want fallback assistant process content", store.messages)
	}
}

func TestHRChatRuntimeDoesNotFallbackWhenToolFailed(t *testing.T) {
	store := newFakeAIStore()
	provider := &fakeChatProvider{err: errors.New("model timeout")}
	service := newNativeAIService(store, provider, nil)

	_, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "summarize application", ApplicationId: 99})
	if err == nil || !strings.Contains(err.Error(), "model timeout") {
		t.Fatalf("Chat error = %v, want provider error without fallback", err)
	}
	if len(store.toolTraces) != 1 || store.toolTraces[0].ErrorMsg == "" {
		t.Fatalf("tool traces = %#v, want failed missing-client trace", store.toolTraces)
	}
	if len(store.messages) != 1 {
		t.Fatalf("messages = %#v, want only user message when no useful tool fallback exists", store.messages)
	}
}

func TestHRChatStreamEmitsRuntimeEvents(t *testing.T) {
	store := newFakeAIStore()
	apps := &fakeApplicationSnapshotClient{response: &pb.GetApplicationSnapshotResponse{Code: 0, ApplicationId: 99, CandidateName: "Ada", JobTitle: "Backend Engineer"}}
	provider := &fakeChatProvider{reply: "streamed hr reply"}
	service := newNativeAIService(store, provider, apps)
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.ChatStream(&pb.ChatRequest{HrId: 77, Message: "summarize application", ApplicationId: 99}, stream); err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	eventTypes := make([]string, 0, len(stream.responses))
	for _, resp := range stream.responses {
		eventTypes = append(eventTypes, resp.GetEventType())
	}
	want := []string{"thinking", "tool_calling", "tool_done", "context_usage", "generating", "done"}
	for _, eventType := range want {
		if !containsString(eventTypes, eventType) {
			t.Fatalf("event types = %#v, missing %q", eventTypes, eventType)
		}
	}
	last := stream.responses[len(stream.responses)-1]
	if !last.GetDone() || last.GetDelta() != "streamed hr reply" || last.GetContextUsage() == nil {
		t.Fatalf("last stream response = %#v", last)
	}
}

func TestHRChatStreamEmitsToolErrorWhenSnapshotFails(t *testing.T) {
	store := newFakeAIStore()
	apps := &fakeApplicationSnapshotClient{response: &pb.GetApplicationSnapshotResponse{Code: 503, Msg: "snapshot unavailable"}}
	provider := &fakeChatProvider{err: errors.New("model timeout")}
	service := newNativeAIService(store, provider, apps)
	stream := &captureChatStream{ctx: context.Background()}

	err := service.ChatStream(&pb.ChatRequest{HrId: 77, Message: "summarize application", ApplicationId: 99}, stream)
	if err == nil || !strings.Contains(err.Error(), "model timeout") {
		t.Fatalf("ChatStream error = %v, want provider error without fallback", err)
	}

	foundToolError := false
	for _, resp := range stream.responses {
		if resp.GetEventType() == "error" && resp.GetErrorType() == "TOOL_ERROR" && strings.Contains(resp.GetEventMessage(), "snapshot unavailable") {
			foundToolError = true
		}
		if resp.GetEventType() == "fallback" {
			t.Fatalf("stream responses = %#v, did not expect fallback for failed tool", stream.responses)
		}
	}
	if !foundToolError {
		t.Fatalf("stream responses = %#v, want tool error event", stream.responses)
	}
}

func TestHRChatStreamEmitsFallbackEventWhenProviderFailsAfterTool(t *testing.T) {
	store := newFakeAIStore()
	apps := &fakeApplicationSnapshotClient{response: &pb.GetApplicationSnapshotResponse{Code: 0, ApplicationId: 99, CandidateName: "Ada", JobTitle: "Backend Engineer"}}
	provider := &fakeChatProvider{err: errors.New("model timeout")}
	service := newNativeAIService(store, provider, apps)
	stream := &captureChatStream{ctx: context.Background()}

	if err := service.ChatStream(&pb.ChatRequest{HrId: 77, Message: "summarize application", ApplicationId: 99}, stream); err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}

	eventTypes := make([]string, 0, len(stream.responses))
	for _, resp := range stream.responses {
		eventTypes = append(eventTypes, resp.GetEventType())
	}
	if !containsString(eventTypes, "fallback") {
		t.Fatalf("event types = %#v, want fallback", eventTypes)
	}
	last := stream.responses[len(stream.responses)-1]
	if !last.GetDone() || !strings.Contains(last.GetDelta(), "AI 模型回答失败") {
		t.Fatalf("last stream response = %#v, want fallback done", last)
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
	reply       string
	err         error
	onComplete  func(prompt string)
	calls       int
	prompts     []string
	optionCalls []ChatCompletionOptions
}

func (p *fakeChatProvider) Complete(_ context.Context, prompt string) (string, error) {
	p.calls++
	p.prompts = append(p.prompts, prompt)
	if p.onComplete != nil {
		p.onComplete(prompt)
	}
	if p.err != nil {
		return "", p.err
	}
	return p.reply, nil
}

func (p *fakeChatProvider) CompleteWithOptions(ctx context.Context, prompt string, _ int64, opts ChatCompletionOptions) (string, error) {
	p.optionCalls = append(p.optionCalls, opts)
	return p.Complete(ctx, prompt)
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
	promptByID        map[int64]*pb.PromptTemplateInfo
	agentConfigs      []*pb.AgentConfigInfo
	agentSkills       []*pb.AgentSkillInfo
	toolTraces        []ToolTraceRow
	candidateContext  CandidateRuntimeContext
	usageAudits       []CandidateUsageAuditRow
}

func newFakeAIStore() *fakeAIStore {
	return &fakeAIStore{nextSessionID: 100, nextMessageID: 200, sessionOwners: make(map[int64]fakeChatSessionOwner), promptByID: make(map[int64]*pb.PromptTemplateInfo)}
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

func (s *fakeAIStore) ListToolTraces(_ context.Context, _ int64, sessionID int64) ([]ToolTraceRow, error) {
	if sessionID == 0 {
		return s.toolTraces, nil
	}
	filtered := make([]ToolTraceRow, 0, len(s.toolTraces))
	for _, trace := range s.toolTraces {
		if trace.SessionID == sessionID {
			filtered = append(filtered, trace)
		}
	}
	return filtered, nil
}

func (s *fakeAIStore) AppendToolTrace(_ context.Context, _ int64, trace ToolTraceRow) (ToolTraceRow, error) {
	if trace.ID == 0 {
		trace.ID = int64(len(s.toolTraces) + 1)
	}
	if trace.CreatedAt.IsZero() {
		trace.CreatedAt = time.Now()
	}
	s.toolTraces = append(s.toolTraces, trace)
	return trace, nil
}

func (s *fakeAIStore) LoadCandidateRuntimeContext(context.Context, int64, int32) (CandidateRuntimeContext, error) {
	return s.candidateContext, nil
}

func (s *fakeAIStore) RecordCandidateUsageAudit(_ context.Context, row CandidateUsageAuditRow) (int64, error) {
	s.usageAudits = append(s.usageAudits, row)
	return int64(len(s.usageAudits)), nil
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

func (s *fakeAIStore) UpdateAgentRunPlan(context.Context, int64, int64, string, string) (AgentRunRow, bool, error) {
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

func (s *fakeAIStore) GetRuntimePromptTemplateByID(_ context.Context, id int64) (*pb.PromptTemplateInfo, bool, error) {
	template, ok := s.promptByID[id]
	return template, ok, nil
}

func (s *fakeAIStore) ListAgentConfigs(_ context.Context, _ int32, _ int32, agentType string) ([]*pb.AgentConfigInfo, int64, error) {
	items := make([]*pb.AgentConfigInfo, 0, len(s.agentConfigs))
	for _, config := range s.agentConfigs {
		if config == nil {
			continue
		}
		if strings.TrimSpace(agentType) != "" && config.GetAgentType() != strings.TrimSpace(agentType) {
			continue
		}
		items = append(items, config)
	}
	return items, int64(len(items)), nil
}

func (s *fakeAIStore) ListMCPServers(context.Context, int32, int32) ([]*pb.MCPServerInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListAgentSkills(_ context.Context, _ int32, _ int32, keyword string, enabledOnly bool) ([]*pb.AgentSkillInfo, int64, error) {
	items := make([]*pb.AgentSkillInfo, 0, len(s.agentSkills))
	for _, skill := range s.agentSkills {
		if skill == nil {
			continue
		}
		if enabledOnly && !skill.GetIsEnabled() {
			continue
		}
		if strings.TrimSpace(keyword) != "" && !strings.Contains(skill.GetName(), strings.TrimSpace(keyword)) && !strings.Contains(skill.GetDisplayName(), strings.TrimSpace(keyword)) {
			continue
		}
		items = append(items, skill)
	}
	return items, int64(len(items)), nil
}

func (s *fakeAIStore) ListEmbeddingProviders(context.Context, int32, int32) ([]*pb.EmbeddingProviderInfo, int64, error) {
	return nil, 0, nil
}

func (s *fakeAIStore) ListEmbeddingModels(context.Context, int32, int32, int64) ([]*pb.EmbeddingModelInfo, int64, error) {
	return nil, 0, nil
}

type fakeApplicationSnapshotClient struct {
	response *pb.GetApplicationSnapshotResponse
	err      error
	calls    []*pb.GetApplicationSnapshotRequest
}

func (c *fakeApplicationSnapshotClient) GetApplicationSnapshot(_ context.Context, req *pb.GetApplicationSnapshotRequest, _ ...gogrpc.CallOption) (*pb.GetApplicationSnapshotResponse, error) {
	c.calls = append(c.calls, req)
	if c.err != nil {
		return nil, c.err
	}
	if c.response != nil {
		return c.response, nil
	}
	return &pb.GetApplicationSnapshotResponse{Code: 404, Msg: "not found"}, nil
}
