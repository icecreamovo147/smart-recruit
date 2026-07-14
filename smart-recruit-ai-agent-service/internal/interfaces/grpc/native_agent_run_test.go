package grpc

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-proto/recruitment/pb"
)

func TestCreateAgentRunDispatchesDetachedAndCompletes(t *testing.T) {
	store := newAgentRunTestStore()
	provider := newBlockingAgentRunProvider("assistant reply")
	service := &nativeAIService{store: store, provider: provider}

	respCh := make(chan *pb.CreateAgentRunResponse, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := service.CreateAgentRun(context.Background(), &pb.CreateAgentRunRequest{
			HrId:            77,
			SessionId:       101,
			ClientRequestId: "req-detached",
			Message:         "user asks",
			ModelId:         123,
		})
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	var resp *pb.CreateAgentRunResponse
	select {
	case err := <-errCh:
		t.Fatalf("CreateAgentRun returned error: %v", err)
	case resp = <-respCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("CreateAgentRun did not return before provider was released")
	}
	if resp.GetCode() != 0 || resp.GetIdempotentReplay() || resp.GetRun() == nil {
		t.Fatalf("CreateAgentRun response = %#v", resp)
	}
	if status := resp.GetRun().GetStatus(); status != "queued" && status != "planning" && status != "running" {
		t.Fatalf("run status = %q, want queued/planning/running", status)
	}

	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider was not dispatched in the background")
	}
	select {
	case <-provider.done:
		t.Fatal("provider completed before test released it")
	default:
	}

	close(provider.release)
	waitUntilAgentRunTest(t, time.Second, func() bool {
		run, found := store.runSnapshot(resp.GetRun().GetRunId())
		return found && run.Status == "succeeded" && len(store.messagesSnapshot()) == 1
	})

	if provider.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.callCount())
	}
	message := store.messagesSnapshot()[0]
	if message.Role != "assistant" || message.Content != "assistant reply" || message.OwnerID != 77 || message.SessionID != 101 {
		t.Fatalf("assistant message = %#v", message)
	}
	eventTypes := store.eventTypes(resp.GetRun().GetRunId())
	for _, want := range []string{"run.created", "run.status_changed", "assistant.delta", "run.completed"} {
		if !containsString(eventTypes, want) {
			t.Fatalf("event types = %v, want %s", eventTypes, want)
		}
	}
}

func TestCreateAgentRunIdempotentReplayDoesNotRedispatch(t *testing.T) {
	store := newAgentRunTestStore()
	provider := newBlockingAgentRunProvider("assistant reply")
	service := &nativeAIService{store: store, provider: provider}
	req := &pb.CreateAgentRunRequest{HrId: 77, SessionId: 101, ClientRequestId: "same-request", Message: "user asks", ModelId: 123}

	first, err := service.CreateAgentRun(context.Background(), req)
	if err != nil {
		t.Fatalf("first CreateAgentRun returned error: %v", err)
	}
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider was not dispatched for first run")
	}

	second, err := service.CreateAgentRun(context.Background(), req)
	if err != nil {
		t.Fatalf("second CreateAgentRun returned error: %v", err)
	}
	if !second.GetIdempotentReplay() {
		t.Fatalf("second CreateAgentRun idempotent replay = false")
	}
	if second.GetRun().GetRunId() != first.GetRun().GetRunId() {
		t.Fatalf("second run id = %d, want %d", second.GetRun().GetRunId(), first.GetRun().GetRunId())
	}
	if provider.callCount() != 1 {
		t.Fatalf("provider calls after replay = %d, want 1", provider.callCount())
	}
	if count := store.countEvents(first.GetRun().GetRunId(), "run.created"); count != 1 {
		t.Fatalf("run.created events = %d, want 1", count)
	}

	close(provider.release)
	waitUntilAgentRunTest(t, time.Second, func() bool {
		run, found := store.runSnapshot(first.GetRun().GetRunId())
		return found && run.Status == "succeeded"
	})
	if provider.callCount() != 1 {
		t.Fatalf("provider calls after completion = %d, want 1", provider.callCount())
	}
}

func TestSubscribeAgentRunEventsReplaysThenLiveTailsUntilContextCancel(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}
	run, _, err := store.CreateAgentRun(context.Background(), fallbackAgentRun(77, 101, "sub-request", "user asks", 123))
	if err != nil {
		t.Fatalf("CreateAgentRun seed returned error: %v", err)
	}
	if _, err := store.AppendAgentRunEvent(context.Background(), run.ID, "run.created", `{"status":"queued"}`); err != nil {
		t.Fatalf("AppendAgentRunEvent seed returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream := newCaptureAgentRunEventStream(ctx)
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.SubscribeAgentRunEvents(&pb.SubscribeAgentRunEventsRequest{HrId: 77, RunId: run.ID}, stream)
	}()

	replayed := stream.waitForEvent(t, "run.created", time.Second)
	if replayed.GetStatus() != "queued" {
		t.Fatalf("replayed status = %q, want queued", replayed.GetStatus())
	}
	if _, err := service.appendAgentRunEvent(context.Background(), run.ID, "assistant.delta", `{"status":"running","delta":"live delta"}`); err != nil {
		t.Fatalf("append live event returned error: %v", err)
	}
	live := stream.waitForEvent(t, "assistant.delta", time.Second)
	if live.GetDelta() != "live delta" {
		t.Fatalf("live delta = %q, want live delta", live.GetDelta())
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("SubscribeAgentRunEvents returned error after cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SubscribeAgentRunEvents did not exit after stream context cancel")
	}
	if count := store.countStatusUpdates("cancel_requested"); count != 0 {
		t.Fatalf("cancel_requested updates = %d, want 0", count)
	}
	finalRun, found := store.runSnapshot(run.ID)
	if !found || finalRun.Status != "queued" {
		t.Fatalf("run after stream cancel = %#v, found=%v; want queued and not canceled", finalRun, found)
	}
}

func TestSubscribeAgentRunEventsRejectsWrongHROwnerBeforeLiveSubscription(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}
	run, _, err := store.CreateAgentRun(context.Background(), fallbackAgentRun(77, 101, "wrong-owner-request", "user asks", 123))
	if err != nil {
		t.Fatalf("CreateAgentRun seed returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newCaptureAgentRunEventStream(ctx)
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.SubscribeAgentRunEvents(&pb.SubscribeAgentRunEventsRequest{HrId: 88, RunId: run.ID}, stream)
	}()

	select {
	case err := <-errCh:
		if status.Code(err) != codes.NotFound {
			t.Fatalf("SubscribeAgentRunEvents error = %v, want NotFound", err)
		}
	case <-time.After(200 * time.Millisecond):
		if _, err := service.appendAgentRunEvent(context.Background(), run.ID, "assistant.delta", `{"status":"running","delta":"private live delta"}`); err != nil {
			t.Fatalf("append live event returned error: %v", err)
		}
		select {
		case event := <-stream.sendCh:
			t.Fatalf("wrong HR received live event before owner check completed: %#v", event)
		case <-time.After(200 * time.Millisecond):
		}
		cancel()
		t.Fatal("SubscribeAgentRunEvents did not return NotFound for wrong HR owner")
	}

	if _, err := service.appendAgentRunEvent(context.Background(), run.ID, "assistant.delta", `{"status":"running","delta":"private live delta"}`); err != nil {
		t.Fatalf("append live event returned error: %v", err)
	}
	if events := stream.eventsSnapshot(); len(events) != 0 {
		t.Fatalf("wrong HR received events after rejected subscription: %#v", events)
	}
	select {
	case event := <-stream.sendCh:
		t.Fatalf("wrong HR received live event after rejected subscription: %#v", event)
	default:
	}
}

type blockingAgentRunProvider struct {
	reply     string
	started   chan struct{}
	release   chan struct{}
	done      chan struct{}
	mu        sync.Mutex
	calls     int
	startOnce sync.Once
	doneOnce  sync.Once
}

func newBlockingAgentRunProvider(reply string) *blockingAgentRunProvider {
	return &blockingAgentRunProvider{
		reply:   reply,
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (p *blockingAgentRunProvider) Complete(ctx context.Context, _ string) (string, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	p.startOnce.Do(func() { close(p.started) })
	select {
	case <-ctx.Done():
		p.doneOnce.Do(func() { close(p.done) })
		return "", ctx.Err()
	case <-p.release:
		p.doneOnce.Do(func() { close(p.done) })
		return p.reply, nil
	}
}

func (p *blockingAgentRunProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

type agentRunTestStore struct {
	*fakeAIStore
	mu                sync.Mutex
	nextRunID         int64
	runEventSeq       map[int64]int64
	runs              map[int64]AgentRunRow
	clientRequestRuns map[string]int64
	runEvents         map[int64][]AgentRunEventRow
	statusUpdates     []string
}

func newAgentRunTestStore() *agentRunTestStore {
	return &agentRunTestStore{
		fakeAIStore:       newFakeAIStore(),
		nextRunID:         300,
		runEventSeq:       make(map[int64]int64),
		runs:              make(map[int64]AgentRunRow),
		clientRequestRuns: make(map[string]int64),
		runEvents:         make(map[int64][]AgentRunEventRow),
	}
}

func (s *agentRunTestStore) CreateAgentRun(_ context.Context, run AgentRunRow) (AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key := agentRunTestIdempotencyKey(run); key != "" {
		if runID, ok := s.clientRequestRuns[key]; ok {
			return s.runs[runID], true, nil
		}
	}
	s.nextRunID++
	now := time.Now()
	run.ID = s.nextRunID
	if run.Status == "" {
		run.Status = "queued"
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = now
	}
	s.runs[run.ID] = run
	if key := agentRunTestIdempotencyKey(run); key != "" {
		s.clientRequestRuns[key] = run.ID
	}
	return run, false, nil
}

func (s *agentRunTestStore) ListAgentRuns(_ context.Context, ownerID, sessionID int64) ([]AgentRunRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]AgentRunRow, 0)
	for _, run := range s.runs {
		if run.OwnerID == ownerID && (sessionID == 0 || run.SessionID == sessionID) {
			rows = append(rows, run)
		}
	}
	return rows, nil
}

func (s *agentRunTestStore) GetAgentRun(_ context.Context, ownerID, runID int64) (AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID {
		return AgentRunRow{}, false, nil
	}
	return run, true, nil
}

func (s *agentRunTestStore) GetActiveAgentRun(_ context.Context, ownerID, sessionID int64) (AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, run := range s.runs {
		if run.OwnerID == ownerID && run.SessionID == sessionID && !isTerminalAgentRunStatus(run.Status) {
			return run, true, nil
		}
	}
	return AgentRunRow{}, false, nil
}

func (s *agentRunTestStore) UpdateAgentRunStatus(_ context.Context, ownerID, runID int64, status string) (AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID {
		return AgentRunRow{}, false, nil
	}
	run.Status = status
	run.UpdatedAt = time.Now()
	s.runs[runID] = run
	s.statusUpdates = append(s.statusUpdates, status)
	return run, true, nil
}

func (s *agentRunTestStore) CompleteAgentRun(_ context.Context, ownerID, runID int64, assistantText, status, errorType, errorMessage string) (AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID {
		return AgentRunRow{}, false, nil
	}
	now := time.Now()
	run.Status = status
	run.AssistantText = assistantText
	run.ErrorType = errorType
	run.ErrorMessage = errorMessage
	run.CompletedAt = &now
	run.UpdatedAt = now
	s.runs[runID] = run
	return run, true, nil
}

func (s *agentRunTestStore) AppendAgentRunEvent(_ context.Context, runID int64, eventType, payload string) (AgentRunEventRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runEventSeq[runID]++
	row := AgentRunEventRow{RunID: runID, Seq: s.runEventSeq[runID], EventType: eventType, PayloadJSON: payload, CreatedAt: time.Now()}
	s.runEvents[runID] = append(s.runEvents[runID], row)
	if run, ok := s.runs[runID]; ok {
		run.LastEventSeq = row.Seq
		run.UpdatedAt = row.CreatedAt
		s.runs[runID] = run
	}
	return row, nil
}

func (s *agentRunTestStore) ListAgentRunEvents(_ context.Context, ownerID, runID, afterSeq int64) ([]AgentRunEventRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID {
		return nil, nil
	}
	rows := make([]AgentRunEventRow, 0)
	for _, event := range s.runEvents[runID] {
		if event.Seq > afterSeq {
			rows = append(rows, event)
		}
	}
	return rows, nil
}

func (s *agentRunTestStore) AppendChatMessage(_ context.Context, message ChatMessageRow) (ChatMessageRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextMessageID++
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	message.ID = s.nextMessageID
	s.messages = append(s.messages, message)
	return message, nil
}

func (s *agentRunTestStore) runSnapshot(runID int64) (AgentRunRow, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, found := s.runs[runID]
	return run, found
}

func (s *agentRunTestStore) messagesSnapshot() []ChatMessageRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	messages := make([]ChatMessageRow, len(s.messages))
	copy(messages, s.messages)
	return messages
}

func (s *agentRunTestStore) eventTypes(runID int64) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	types := make([]string, 0, len(s.runEvents[runID]))
	for _, event := range s.runEvents[runID] {
		types = append(types, event.EventType)
	}
	return types
}

func (s *agentRunTestStore) countEvents(runID int64, eventType string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, event := range s.runEvents[runID] {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func (s *agentRunTestStore) countStatusUpdates(status string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, update := range s.statusUpdates {
		if update == status {
			count++
		}
	}
	return count
}

func agentRunTestIdempotencyKey(run AgentRunRow) string {
	if run.ClientRequestID == "" {
		return ""
	}
	return fmt.Sprintf("%d/%d/%s", run.OwnerID, run.SessionID, run.ClientRequestID)
}

type captureAgentRunEventStream struct {
	gogrpc.ServerStream
	ctx    context.Context
	mu     sync.Mutex
	events []*pb.AgentRunEvent
	sendCh chan *pb.AgentRunEvent
}

func newCaptureAgentRunEventStream(ctx context.Context) *captureAgentRunEventStream {
	return &captureAgentRunEventStream{ctx: ctx, sendCh: make(chan *pb.AgentRunEvent, 16)}
}

func (s *captureAgentRunEventStream) Context() context.Context {
	return s.ctx
}

func (s *captureAgentRunEventStream) Send(event *pb.AgentRunEvent) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	s.sendCh <- event
	return nil
}

func (s *captureAgentRunEventStream) eventsSnapshot() []*pb.AgentRunEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := make([]*pb.AgentRunEvent, len(s.events))
	copy(events, s.events)
	return events
}

func (s *captureAgentRunEventStream) waitForEvent(t *testing.T, eventType string, timeout time.Duration) *pb.AgentRunEvent {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case event := <-s.sendCh:
			if event.GetEventType() == eventType {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event %s", eventType)
		}
	}
}

func waitUntilAgentRunTest(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if condition() {
			return
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for condition")
		case <-ticker.C:
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
