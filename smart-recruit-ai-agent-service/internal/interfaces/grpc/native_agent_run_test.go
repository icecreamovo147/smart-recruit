package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-proto/recruitment/pb"
)

func TestCreateAgentRunRejectsBlankMessageBeforeDispatch(t *testing.T) {
	service := &nativeAIService{}
	_, err := service.CreateAgentRun(context.Background(), &pb.CreateAgentRunRequest{HrId: 77, SessionId: 101, Message: " \n\t "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %v, want %v; err=%v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestApplicationAnalysisRunReusesSeededUserMessage(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}
	analysis, err := service.CreateApplicationAnalysisSession(context.Background(), &pb.CreateApplicationAnalysisSessionRequest{
		HrId:          77,
		ApplicationId: 901,
		ModelId:       123,
	})
	if err != nil {
		t.Fatalf("CreateApplicationAnalysisSession returned error: %v", err)
	}
	if len(analysis.GetMessages()) != 1 {
		t.Fatalf("analysis messages = %d, want 1", len(analysis.GetMessages()))
	}
	created, err := service.CreateAgentRun(context.Background(), &pb.CreateAgentRunRequest{
		HrId:            77,
		SessionId:       analysis.GetSession().GetSessionId(),
		ClientRequestId: "analysis-run",
		Message:         analysis.GetMessages()[0].GetContent(),
		ActionType:      "analyze_application",
		ApplicationId:   901,
		ModelId:         123,
	})
	if err != nil {
		t.Fatalf("CreateAgentRun returned error: %v", err)
	}
	waitUntilAgentRunTest(t, time.Second, func() bool {
		run, found := store.runSnapshot(created.GetRun().GetRunId())
		return found && isTerminalAgentRunStatus(run.Status)
	})
	messages := store.messagesSnapshot()
	if len(messages) != 2 {
		t.Fatalf("messages = %#v, want one seeded user message and one assistant message", messages)
	}
	userCount := 0
	for _, message := range messages {
		if message.Role == "user" {
			userCount++
		}
	}
	if userCount != 1 {
		t.Fatalf("user message count = %d, want 1; messages=%#v", userCount, messages)
	}
}

func TestCreateAgentRunDispatchesDetachedAndCompletes(t *testing.T) {
	store := newAgentRunTestStore()
	store.seedChatSession(ownerRoleHR, 77, 101, "hr run session")
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
	createdRun, found := store.runSnapshot(resp.GetRun().GetRunId())
	if !found {
		t.Fatalf("created run %d not found in store", resp.GetRun().GetRunId())
	}
	var durable struct {
		DurableRequest agentRunDurablePayload `json:"durable_request"`
	}
	if err := json.Unmarshal([]byte(createdRun.PlanJSON), &durable); err != nil {
		t.Fatalf("PlanJSON = %q, want durable request JSON: %v", createdRun.PlanJSON, err)
	}
	if durable.DurableRequest.Message != "user asks" || durable.DurableRequest.ModelID != 123 {
		t.Fatalf("durable request = %#v, want message/model preserved", durable.DurableRequest)
	}

	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider was not dispatched in the background")
	}
	prompts := provider.promptsSnapshot()
	if len(prompts) != 1 || !strings.Contains(prompts[0], "System:\n") || !strings.Contains(prompts[0], "User:\nuser asks") {
		t.Fatalf("provider prompts = %#v, want HR runtime prompt rendered from durable payload", prompts)
	}
	select {
	case <-provider.done:
		t.Fatal("provider completed before test released it")
	default:
	}

	close(provider.release)
	waitUntilAgentRunTest(t, time.Second, func() bool {
		run, found := store.runSnapshot(resp.GetRun().GetRunId())
		return found && run.Status == "succeeded" && len(store.messagesSnapshot()) == 2
	})

	if provider.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.callCount())
	}
	messages := store.messagesSnapshot()
	if messages[0].Role != "user" || messages[0].Content != "user asks" || messages[0].OwnerID != 77 || messages[0].SessionID != 101 {
		t.Fatalf("user message = %#v", messages[0])
	}
	message := messages[1]
	if message.Role != "assistant" || message.Content != "assistant reply" || message.OwnerID != 77 || message.SessionID != 101 {
		t.Fatalf("assistant message = %#v", message)
	}
	eventTypes := store.eventTypes(resp.GetRun().GetRunId())
	for _, want := range []string{"run.created", "run.status_changed", "process.delta", "assistant.delta", "run.result", "run.completed"} {
		if !containsString(eventTypes, want) {
			t.Fatalf("event types = %v, want %s", eventTypes, want)
		}
	}
}

func TestCreateAgentRunIdempotentReplayDoesNotRedispatch(t *testing.T) {
	store := newAgentRunTestStore()
	store.seedChatSession(ownerRoleHR, 77, 101, "hr run session")
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

func TestCancelAgentRunCancelsBackgroundProviderAndFinishesCanceled(t *testing.T) {
	store := newAgentRunTestStore()
	store.seedChatSession(ownerRoleHR, 77, 101, "hr run session")
	provider := newBlockingAgentRunProvider("late assistant reply")
	service := &nativeAIService{store: store, provider: provider}

	created, err := service.CreateAgentRun(context.Background(), &pb.CreateAgentRunRequest{
		HrId:            77,
		SessionId:       101,
		ClientRequestId: "cancel-running",
		Message:         "user asks",
		ModelId:         123,
	})
	if err != nil {
		t.Fatalf("CreateAgentRun returned error: %v", err)
	}
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider was not dispatched")
	}

	canceled, err := service.CancelAgentRun(context.Background(), &pb.CancelAgentRunRequest{HrId: 77, RunId: created.GetRun().GetRunId()})
	if err != nil {
		t.Fatalf("CancelAgentRun returned error: %v", err)
	}
	if canceled.GetCode() != 0 || canceled.GetRun().GetStatus() != "cancel_requested" {
		t.Fatalf("CancelAgentRun response = %#v, want cancel_requested success", canceled)
	}
	select {
	case <-provider.canceled:
	case <-time.After(time.Second):
		t.Fatal("provider context was not canceled")
	}

	waitUntilAgentRunTest(t, time.Second, func() bool {
		run, found := store.runSnapshot(created.GetRun().GetRunId())
		return found && run.Status == "canceled"
	})
	close(provider.release)
	time.Sleep(20 * time.Millisecond)

	finalRun, found := store.runSnapshot(created.GetRun().GetRunId())
	if !found || finalRun.Status != "canceled" {
		t.Fatalf("final run = %#v, found=%v; want canceled", finalRun, found)
	}
	if messages := store.messagesSnapshot(); len(messages) != 1 || messages[0].Role != "user" {
		t.Fatalf("messages = %#v, want only persisted user message after cancel", messages)
	}
	eventTypes := store.eventTypes(created.GetRun().GetRunId())
	if !containsString(eventTypes, "run.canceled") {
		t.Fatalf("event types = %v, want run.canceled", eventTypes)
	}
	if containsString(eventTypes, "run.completed") {
		t.Fatalf("event types = %v, did not expect run.completed after cancel", eventTypes)
	}
}

func TestCancelAgentRunIdempotentStatusesDoNotAppendEvents(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}

	tests := []struct {
		name      string
		status    string
		eventType string
		payload   string
	}{
		{name: "terminal succeeded", status: "succeeded", eventType: "run.completed", payload: `{"status":"succeeded"}`},
		{name: "already cancel requested", status: "cancel_requested", eventType: "run.status_changed", payload: `{"status":"cancel_requested"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := fallbackAgentRun(77, 101, "idempotent-cancel-"+tt.name, agentRunDurablePayload{Message: "user asks", ModelID: 123})
			run.Status = tt.status
			created, _, err := store.CreateAgentRun(context.Background(), run)
			if err != nil {
				t.Fatalf("CreateAgentRun seed returned error: %v", err)
			}
			if _, err := store.AppendAgentRunEvent(context.Background(), created.ID, tt.eventType, tt.payload); err != nil {
				t.Fatalf("AppendAgentRunEvent seed returned error: %v", err)
			}
			beforeEvents := store.totalEventCount(created.ID)

			resp, err := service.CancelAgentRun(context.Background(), &pb.CancelAgentRunRequest{HrId: 77, RunId: created.ID})
			if err != nil {
				t.Fatalf("CancelAgentRun returned error: %v", err)
			}
			if resp.GetCode() != 0 || resp.GetRun().GetStatus() != tt.status {
				t.Fatalf("CancelAgentRun response = %#v, want %s idempotent success", resp, tt.status)
			}
			if afterEvents := store.totalEventCount(created.ID); afterEvents != beforeEvents {
				t.Fatalf("event count = %d, want %d", afterEvents, beforeEvents)
			}
		})
	}
	if count := store.countStatusUpdates("cancel_requested"); count != 0 {
		t.Fatalf("cancel_requested updates = %d, want 0", count)
	}
}

func TestCancelAgentRunQueuedWorkFinishesCanceledWithoutActiveWorker(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}
	run := fallbackAgentRun(77, 101, "cancel-queued", agentRunDurablePayload{Message: "user asks", ModelID: 123})
	run.Status = "queued"
	created, _, err := store.CreateAgentRun(context.Background(), run)
	if err != nil {
		t.Fatalf("CreateAgentRun seed returned error: %v", err)
	}

	resp, err := service.CancelAgentRun(context.Background(), &pb.CancelAgentRunRequest{HrId: 77, RunId: created.ID})
	if err != nil {
		t.Fatalf("CancelAgentRun returned error: %v", err)
	}
	if resp.GetCode() != 0 || resp.GetRun().GetStatus() != "canceled" {
		t.Fatalf("CancelAgentRun response = %#v, want terminal canceled success", resp)
	}
	finalRun, found := store.runSnapshot(created.ID)
	if !found || finalRun.Status != "canceled" || finalRun.CanceledAt == nil || finalRun.CompletedAt == nil {
		t.Fatalf("run after queued cancel = %#v, found=%v; want terminal canceled timestamps", finalRun, found)
	}
	eventTypes := store.eventTypes(created.ID)
	for _, want := range []string{"run.status_changed", "run.canceled"} {
		if !containsString(eventTypes, want) {
			t.Fatalf("event types = %v, want %s", eventTypes, want)
		}
	}
}

func TestConfirmAgentRunWaitingConfirmationRedispatchesAndCompletes(t *testing.T) {
	store := newAgentRunTestStore()
	store.seedChatSession(ownerRoleHR, 77, 101, "hr run session")
	store.seedChatMessage(ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: 77, SessionID: 101, Role: "user", Content: "user asks", ModelID: 123})
	provider := newBlockingAgentRunProvider("assistant after confirmation")
	service := &nativeAIService{store: store, provider: provider}
	run := fallbackAgentRun(77, 101, "confirm-waiting", agentRunDurablePayload{Message: "user asks", ModelID: 123})
	run.Status = "waiting_confirmation"
	created, _, err := store.CreateAgentRun(context.Background(), run)
	if err != nil {
		t.Fatalf("CreateAgentRun seed returned error: %v", err)
	}

	resp, err := service.ConfirmAgentRun(context.Background(), &pb.ConfirmAgentRunRequest{
		HrId:                         77,
		RunId:                        created.ID,
		ClientRequestId:              "confirm-1",
		AgentSkillIds:                []int64{7001, 7002},
		AgentSkillSelectionConfirmed: true,
		AgentSkillSelectionMessageId: 9001,
		ConfirmationPayloadJson:      `{"approved":true}`,
	})
	if err != nil {
		t.Fatalf("ConfirmAgentRun returned error: %v", err)
	}
	if resp.GetCode() != 0 || resp.GetRun().GetStatus() != "running" {
		t.Fatalf("ConfirmAgentRun response = %#v, want running success", resp)
	}
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider was not redispatched after confirmation")
	}
	close(provider.release)
	waitUntilAgentRunTest(t, time.Second, func() bool {
		finalRun, found := store.runSnapshot(created.ID)
		return found && finalRun.Status == "succeeded"
	})
	finalRun, found := store.runSnapshot(created.ID)
	if !found || finalRun.Status != "succeeded" || finalRun.AssistantText != "assistant after confirmation" {
		t.Fatalf("run after confirm = %#v, found=%v; want succeeded with provider reply", finalRun, found)
	}
	messages := store.messagesSnapshot()
	if len(messages) != 2 || messages[0].Role != "user" || messages[0].Content != "user asks" || messages[1].Role != "assistant" {
		t.Fatalf("messages after confirm = %#v, want existing user plus assistant without duplicate user", messages)
	}
	var durable struct {
		DurableRequest agentRunDurablePayload `json:"durable_request"`
	}
	if err := json.Unmarshal([]byte(finalRun.PlanJSON), &durable); err != nil {
		t.Fatalf("PlanJSON = %q, want durable request JSON: %v", finalRun.PlanJSON, err)
	}
	if got := durable.DurableRequest.AgentSkillIDs; len(got) != 2 || got[0] != 7001 || got[1] != 7002 {
		t.Fatalf("durable AgentSkillIDs = %v, want [7001 7002]", got)
	}
	if !durable.DurableRequest.AgentSkillSelectionConfirmed || durable.DurableRequest.AgentSkillSelectionMessageID != 9001 || durable.DurableRequest.ConfirmationPayloadJSON != `{"approved":true}` {
		t.Fatalf("durable confirmation fields = %#v, want confirmed selection metadata", durable.DurableRequest)
	}
	if prompts := provider.promptsSnapshot(); len(prompts) != 1 || !strings.Contains(prompts[0], "agent_skill_ids") || !strings.Contains(prompts[0], "7001") {
		t.Fatalf("provider prompts = %#v, want resumed runtime to use confirmed durable selection", prompts)
	}
	eventTypes := store.eventTypes(created.ID)
	if !containsString(eventTypes, "confirmation.accepted") {
		t.Fatalf("event types = %v, want confirmation.accepted", eventTypes)
	}
	if !containsString(eventTypes, "run.completed") {
		t.Fatalf("event types = %v, want run.completed", eventTypes)
	}
	if count := store.countStatusUpdates("running"); count != 1 {
		t.Fatalf("running status updates = %d, want 1", count)
	}
	confirmationEvent := store.lastEvent(created.ID, "confirmation.accepted")
	mapped := mapAgentRunEvent(confirmationEvent)
	if got := mapped.GetConfirmation().GetRecommendedAgentSkillIds(); len(got) != 2 || got[0] != 7001 || got[1] != 7002 {
		t.Fatalf("confirmation event = %#v, want confirmed skill ids", mapped.GetConfirmation())
	}
}

func TestConfirmAgentRunNonWaitingStatusesDoNotJump(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		wantCode int32
	}{
		{name: "queued", status: "queued", wantCode: agentRunCodeBadRequest},
		{name: "running retry", status: "running", wantCode: 0},
		{name: "succeeded retry", status: "succeeded", wantCode: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newAgentRunTestStore()
			service := &nativeAIService{store: store}
			run := fallbackAgentRun(77, 101, "confirm-"+tt.name, agentRunDurablePayload{Message: "user asks", ModelID: 123})
			run.Status = tt.status
			created, _, err := store.CreateAgentRun(context.Background(), run)
			if err != nil {
				t.Fatalf("CreateAgentRun seed returned error: %v", err)
			}
			beforeEvents := store.totalEventCount(created.ID)

			resp, err := service.ConfirmAgentRun(context.Background(), &pb.ConfirmAgentRunRequest{HrId: 77, RunId: created.ID})
			if err != nil {
				t.Fatalf("ConfirmAgentRun returned error: %v", err)
			}
			if resp.GetCode() != tt.wantCode {
				t.Fatalf("ConfirmAgentRun code = %d, want %d; response=%#v", resp.GetCode(), tt.wantCode, resp)
			}
			finalRun, found := store.runSnapshot(created.ID)
			if !found || finalRun.Status != tt.status {
				t.Fatalf("run after confirm = %#v, found=%v; want status %s", finalRun, found, tt.status)
			}
			if afterEvents := store.totalEventCount(created.ID); afterEvents != beforeEvents {
				t.Fatalf("event count = %d, want %d", afterEvents, beforeEvents)
			}
		})
	}
}

func TestSubscribeAgentRunEventsReplaysThenLiveTailsUntilContextCancel(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}
	run, _, err := store.CreateAgentRun(context.Background(), fallbackAgentRun(77, 101, "sub-request", agentRunDurablePayload{Message: "user asks", ModelID: 123}))
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

func TestSubscribeAgentRunEventsReplaysStructuredMetadata(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}
	run, _, err := store.CreateAgentRun(context.Background(), fallbackAgentRun(77, 101, "sub-metadata", agentRunDurablePayload{Message: "user asks", ModelID: 123}))
	if err != nil {
		t.Fatalf("CreateAgentRun seed returned error: %v", err)
	}
	if _, err := store.AppendAgentRunEvent(context.Background(), run.ID, "run.result", `{"status":"succeeded","result_metadata":{"application_id":42,"candidate_name":"Ada","job_title":"Engineer","status":3,"context_usage":{"model_id":123,"prompt_tokens_estimated":88,"usage_ratio":0.25,"estimated":true,"source":"estimate","stage":"hr_chat"}}}`); err != nil {
		t.Fatalf("AppendAgentRunEvent result seed returned error: %v", err)
	}
	if _, err := store.AppendAgentRunEvent(context.Background(), run.ID, "confirmation.required", `{"status":"waiting_confirmation","confirmation":{"required":true,"reason":"skill confirmation","recommended_agent_skill_ids":[7,8],"user_message_id":2001}}`); err != nil {
		t.Fatalf("AppendAgentRunEvent confirmation seed returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newCaptureAgentRunEventStream(ctx)
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.SubscribeAgentRunEvents(&pb.SubscribeAgentRunEventsRequest{HrId: 77, RunId: run.ID}, stream)
	}()

	resultEvent := stream.waitForEvent(t, "run.result", time.Second)
	if resultEvent.GetResultMetadata().GetApplicationId() != 42 || resultEvent.GetResultMetadata().GetCandidateName() != "Ada" {
		t.Fatalf("result metadata = %#v, want replayed structured result", resultEvent.GetResultMetadata())
	}
	if resultEvent.GetResultMetadata().GetContextUsage().GetPromptTokensEstimated() != 88 {
		t.Fatalf("context usage = %#v, want replayed usage", resultEvent.GetResultMetadata().GetContextUsage())
	}
	confirmationEvent := stream.waitForEvent(t, "confirmation.required", time.Second)
	if !confirmationEvent.GetConfirmation().GetRequired() || confirmationEvent.GetConfirmation().GetReason() != "skill confirmation" {
		t.Fatalf("confirmation = %#v, want replayed structured confirmation", confirmationEvent.GetConfirmation())
	}
	if got := confirmationEvent.GetConfirmation().GetRecommendedAgentSkillIds(); len(got) != 2 || got[0] != 7 || got[1] != 8 {
		t.Fatalf("recommended skill ids = %v, want [7 8]", got)
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
}

func TestSubscribeAgentRunEventsRejectsWrongHROwnerBeforeLiveSubscription(t *testing.T) {
	store := newAgentRunTestStore()
	service := &nativeAIService{store: store}
	run, _, err := store.CreateAgentRun(context.Background(), fallbackAgentRun(77, 101, "wrong-owner-request", agentRunDurablePayload{Message: "user asks", ModelID: 123}))
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
	reply      string
	started    chan struct{}
	release    chan struct{}
	done       chan struct{}
	canceled   chan struct{}
	mu         sync.Mutex
	calls      int
	prompts    []string
	startOnce  sync.Once
	doneOnce   sync.Once
	cancelOnce sync.Once
}

func newBlockingAgentRunProvider(reply string) *blockingAgentRunProvider {
	return &blockingAgentRunProvider{
		reply:    reply,
		started:  make(chan struct{}),
		release:  make(chan struct{}),
		done:     make(chan struct{}),
		canceled: make(chan struct{}),
	}
}

func (p *blockingAgentRunProvider) Complete(ctx context.Context, prompt string) (string, error) {
	p.mu.Lock()
	p.calls++
	p.prompts = append(p.prompts, prompt)
	p.mu.Unlock()
	p.startOnce.Do(func() { close(p.started) })
	select {
	case <-ctx.Done():
		p.cancelOnce.Do(func() { close(p.canceled) })
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

func (p *blockingAgentRunProvider) promptsSnapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.prompts))
	copy(out, p.prompts)
	return out
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
	now := time.Now()
	run.UpdatedAt = now
	if status == "cancel_requested" {
		run.CancelRequestedAt = &now
	}
	if status == "canceled" {
		run.CompletedAt = &now
		run.CanceledAt = &now
	}
	s.runs[runID] = run
	s.statusUpdates = append(s.statusUpdates, status)
	return run, true, nil
}

func (s *agentRunTestStore) UpdateAgentRunPlan(_ context.Context, ownerID, runID int64, planJSON, optionContextJSON string) (AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID {
		return AgentRunRow{}, false, nil
	}
	run.PlanJSON = planJSON
	run.OptionContextJSON = optionContextJSON
	run.UpdatedAt = time.Now()
	s.runs[runID] = run
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
	if status == "canceled" {
		run.CanceledAt = &now
	}
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

func (s *agentRunTestStore) lastEvent(runID int64, eventType string) AgentRunEventRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	var last AgentRunEventRow
	for _, event := range s.runEvents[runID] {
		if event.EventType == eventType {
			last = event
		}
	}
	return last
}

func (s *agentRunTestStore) totalEventCount(runID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.runEvents[runID])
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
