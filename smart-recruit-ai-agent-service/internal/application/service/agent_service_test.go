package service

import (
	"context"
	"testing"
	"time"

	"smart-recruit-ai-agent-service/internal/application/command"
	"smart-recruit-ai-agent-service/internal/domain/model"
)

func TestAgentRunServiceCreateIsIdempotentAndDispatches(t *testing.T) {
	repo := newFakeRunRepo()
	dispatcher := &fakeDispatcher{}
	service := newTestAgentRunService(t, repo, dispatcher)

	result, err := service.Create(context.Background(), command.CreateAgentRun{
		ActorID:         7,
		SessionID:       10,
		ClientRequestID: "req-1",
		Message:         "hello",
		ModelName:       "gpt-test",
	})
	if err != nil {
		t.Fatalf("Create returned %v", err)
	}
	if result.Run.Status != model.AgentRunStatusPlanning || !result.Dispatched {
		t.Fatalf("result = %+v", result)
	}
	if len(dispatcher.runIDs) != 1 || dispatcher.runIDs[0] != result.Run.ID {
		t.Fatalf("dispatches = %#v", dispatcher.runIDs)
	}

	again, err := service.Create(context.Background(), command.CreateAgentRun{
		ActorID:         7,
		SessionID:       10,
		ClientRequestID: "req-1",
		Message:         "hello",
	})
	if err != nil {
		t.Fatalf("Create again returned %v", err)
	}
	if !again.Idempotent || again.Run.ID != result.Run.ID || len(dispatcher.runIDs) != 1 {
		t.Fatalf("again = %+v dispatches=%#v", again, dispatcher.runIDs)
	}
}

func TestAgentRunServiceCancelAndConfirm(t *testing.T) {
	repo := newFakeRunRepo()
	dispatcher := &fakeDispatcher{}
	service := newTestAgentRunService(t, repo, dispatcher)
	run := repo.seed(model.AgentRun{ActorID: 7, SessionID: 10, Status: model.AgentRunStatusRunning})

	canceled, err := service.Cancel(context.Background(), command.CancelAgentRun{ActorID: 7, RunID: run.ID})
	if err != nil {
		t.Fatalf("Cancel returned %v", err)
	}
	if canceled.Run.Status != model.AgentRunStatusCancelRequested {
		t.Fatalf("canceled = %+v", canceled.Run)
	}
	again, err := service.Cancel(context.Background(), command.CancelAgentRun{ActorID: 7, RunID: run.ID})
	if err != nil {
		t.Fatalf("Cancel again returned %v", err)
	}
	if !again.Idempotent {
		t.Fatalf("cancel again = %+v, want idempotent", again)
	}

	waiting := repo.seed(model.AgentRun{ActorID: 7, SessionID: 10, Status: model.AgentRunStatusWaitingConfirmation})
	confirmed, err := service.Confirm(context.Background(), command.ConfirmAgentRun{ActorID: 7, RunID: waiting.ID})
	if err != nil {
		t.Fatalf("Confirm returned %v", err)
	}
	if confirmed.Run.Status != model.AgentRunStatusRunning || !confirmed.Dispatched {
		t.Fatalf("confirmed = %+v", confirmed)
	}
}

func TestPromptServiceCreatesVersionOnlyWhenContentChanges(t *testing.T) {
	prompts := newFakePromptRepo()
	service, err := NewPromptService(PromptDeps{Prompts: prompts, Audit: fakeAudit{}})
	if err != nil {
		t.Fatalf("NewPromptService returned %v", err)
	}
	created, err := service.CreateTemplate(context.Background(), command.CreatePromptTemplate{ActorID: 7, Name: "hr", Content: "old", AgentType: "hr_recruiting_agent"})
	if err != nil {
		t.Fatalf("CreateTemplate returned %v", err)
	}
	if !created.VersionCreated || created.Template.Version != 1 {
		t.Fatalf("created = %+v", created)
	}
	unchanged, err := service.UpdateTemplate(context.Background(), command.UpdatePromptTemplate{ActorID: 7, ID: created.Template.ID, Name: "hr-renamed"})
	if err != nil {
		t.Fatalf("UpdateTemplate rename returned %v", err)
	}
	if unchanged.VersionCreated || unchanged.Template.Version != 1 {
		t.Fatalf("unchanged = %+v", unchanged)
	}
	changed, err := service.UpdateTemplate(context.Background(), command.UpdatePromptTemplate{ActorID: 7, ID: created.Template.ID, Content: "new"})
	if err != nil {
		t.Fatalf("UpdateTemplate content returned %v", err)
	}
	if !changed.VersionCreated || changed.Template.Version != 2 || len(prompts.versions) != 2 {
		t.Fatalf("changed = %+v versions=%#v", changed, prompts.versions)
	}
}

func newTestAgentRunService(t *testing.T, repo *fakeRunRepo, dispatcher *fakeDispatcher) *AgentRunService {
	t.Helper()
	service, err := NewAgentRunService(AgentRunDeps{
		Sessions:   fakeSessions{},
		Runs:       repo,
		Dispatcher: dispatcher,
		Audit:      fakeAudit{},
		Now:        func() time.Time { return time.Date(2026, 7, 13, 11, 30, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewAgentRunService returned %v", err)
	}
	return service
}

type fakeSessions struct{}

func (fakeSessions) GetSessionForOwner(context.Context, uint64, uint64) (*model.ChatSession, error) {
	return &model.ChatSession{ID: 10, OwnerID: 7}, nil
}

func (fakeSessions) AppendMessage(context.Context, model.ChatMessage) error { return nil }

type fakeRunRepo struct {
	next   uint64
	runs   map[uint64]model.AgentRun
	byReq  map[string]uint64
	events []model.AgentRunEvent
}

func newFakeRunRepo() *fakeRunRepo {
	return &fakeRunRepo{next: 1, runs: make(map[uint64]model.AgentRun), byReq: make(map[string]uint64)}
}

func (f *fakeRunRepo) seed(run model.AgentRun) model.AgentRun {
	run.ID = f.next
	f.next++
	f.runs[run.ID] = run
	return run
}

func (f *fakeRunRepo) FindByClientRequestID(_ context.Context, actorID uint64, clientRequestID string) (*model.AgentRun, error) {
	id, ok := f.byReq[clientRequestID]
	if !ok {
		return nil, nil
	}
	run := f.runs[id]
	if run.ActorID != actorID {
		return nil, nil
	}
	return &run, nil
}

func (f *fakeRunRepo) CreateRun(_ context.Context, run model.AgentRun) (*model.AgentRun, error) {
	run = f.seed(run)
	f.byReq[run.ClientRequestID] = run.ID
	return &run, nil
}

func (f *fakeRunRepo) GetOwnedRun(_ context.Context, runID, actorID uint64) (*model.AgentRun, error) {
	run := f.runs[runID]
	if run.ActorID != actorID {
		return nil, ErrSessionForbidden
	}
	return &run, nil
}

func (f *fakeRunRepo) UpdateRunStatus(_ context.Context, runID uint64, _, to string) (*model.AgentRun, error) {
	run := f.runs[runID]
	run.Status = to
	f.runs[runID] = run
	return &run, nil
}

func (f *fakeRunRepo) AppendEvent(_ context.Context, event model.AgentRunEvent) error {
	f.events = append(f.events, event)
	return nil
}

type fakeDispatcher struct {
	runIDs []uint64
}

func (f *fakeDispatcher) DispatchAgentRun(_ context.Context, runID uint64) error {
	f.runIDs = append(f.runIDs, runID)
	return nil
}

type fakePromptRepo struct {
	next      uint64
	templates map[uint64]model.PromptTemplate
	versions  []model.PromptVersion
}

func newFakePromptRepo() *fakePromptRepo {
	return &fakePromptRepo{next: 1, templates: make(map[uint64]model.PromptTemplate)}
}

func (f *fakePromptRepo) GetTemplate(_ context.Context, templateID uint64) (*model.PromptTemplate, error) {
	template := f.templates[templateID]
	return &template, nil
}

func (f *fakePromptRepo) CreateTemplate(_ context.Context, template model.PromptTemplate, version model.PromptVersion) (*model.PromptTemplate, error) {
	template.ID = f.next
	f.next++
	version.TemplateID = template.ID
	f.templates[template.ID] = template
	f.versions = append(f.versions, version)
	return &template, nil
}

func (f *fakePromptRepo) UpdateTemplate(_ context.Context, template model.PromptTemplate, version *model.PromptVersion) (*model.PromptTemplate, error) {
	f.templates[template.ID] = template
	if version != nil {
		f.versions = append(f.versions, *version)
	}
	return &template, nil
}

type fakeAudit struct{}

func (fakeAudit) RecordAudit(context.Context, model.AuditEvent) error { return nil }
