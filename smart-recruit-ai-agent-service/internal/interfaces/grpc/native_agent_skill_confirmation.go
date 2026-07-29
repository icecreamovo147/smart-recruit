package grpc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"smart-recruit-platform-go/businessclock"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

const (
	agentSkillConfirmationTTL      = 10 * time.Minute
	agentSkillDispatchLeaseGrace   = 30 * time.Second
	agentSkillDispatchPendingEvent = "accepted_pending_event"
	agentSkillDispatchReady        = "ready"
	agentSkillDispatchClaimed      = "claimed"
)

var (
	errInvalidAgentSkillConfirmation = errors.New("agent skill confirmation is invalid")
	errExpiredAgentSkillConfirmation = errors.New("agent skill confirmation is expired")
	errAgentRunExecutionLeaseLost    = errors.New("agent skill execution lease is no longer owned")
)

type agentRunSkillConfirmation struct {
	ID                     string   `json:"id"`
	TenantID               int64    `json:"tenant_id,omitempty"`
	UserID                 int64    `json:"user_id"`
	HRID                   int64    `json:"hr_id"`
	RunID                  int64    `json:"run_id"`
	CapabilityVersionID    int64    `json:"capability_version_id"`
	CapabilitySnapshotHash string   `json:"capability_snapshot_hash"`
	VersionIDs             []int64  `json:"version_ids"`
	CompiledHashes         []string `json:"compiled_hashes"`
	MessageDigest          string   `json:"message_digest"`
	SelectionMode          string   `json:"selection_mode"`
	Role                   string   `json:"role"`
	Risk                   string   `json:"risk"`
	UserMessageID          int64    `json:"user_message_id,omitempty"`
	ExpiresAt              string   `json:"expires_at"`
}

type agentRunSkillApproval struct {
	ConfirmationID          string   `json:"confirmation_id"`
	TenantID                int64    `json:"tenant_id,omitempty"`
	UserID                  int64    `json:"user_id"`
	HRID                    int64    `json:"hr_id"`
	RunID                   int64    `json:"run_id"`
	CapabilityVersionID     int64    `json:"capability_version_id"`
	CapabilitySnapshotHash  string   `json:"capability_snapshot_hash"`
	VersionIDs              []int64  `json:"version_ids"`
	CompiledHashes          []string `json:"compiled_hashes"`
	MessageDigest           string   `json:"message_digest"`
	SelectionMode           string   `json:"selection_mode"`
	Role                    string   `json:"role"`
	Risk                    string   `json:"risk"`
	UserMessageID           int64    `json:"user_message_id"`
	ConfirmationExpiresAt   string   `json:"confirmation_expires_at"`
	ApprovedAt              string   `json:"approved_at"`
	ApprovalClientRequestID string   `json:"approval_client_request_id,omitempty"`
	DispatchState           string   `json:"dispatch_state"`
	DispatchLeaseID         string   `json:"dispatch_lease_id,omitempty"`
	DispatchLeaseExpiresAt  string   `json:"dispatch_lease_expires_at,omitempty"`
}

type agentSkillApprovalContextKey struct{}

type agentSkillApprovalContext struct {
	RunID    int64
	HRID     int64
	Approval *agentRunSkillApproval
}

type agentRunConfirmationTransitionStore interface {
	TransitionAgentRunConfirmation(
		ctx context.Context,
		ownerID int64,
		runID int64,
		fromStatus string,
		toStatus string,
		expectedPlanJSON string,
		planJSON string,
		optionContextJSON string,
		errorType string,
		errorMessage string,
	) (AgentRunRow, bool, error)
}

type agentRunSkillApprovalFinalizeStore interface {
	FinalizeAgentRunSkillApproval(
		ctx context.Context,
		ownerID int64,
		runID int64,
		expectedPlanJSON string,
		readyPlanJSON string,
		optionContextJSON string,
		eventPayloadJSON string,
	) (AgentRunRow, AgentRunEventRow, bool, error)
}

type agentRunUserMessageStore interface {
	EnsureAgentRunUserMessage(
		ctx context.Context,
		ownerID int64,
		runID int64,
		message ChatMessageRow,
	) (ChatMessageRow, AgentRunRow, bool, error)
}

type agentRunSkillExecutionFenceStore interface {
	VerifyAgentRunSkillLease(
		ctx context.Context,
		ownerID int64,
		runID int64,
		leaseID string,
	) (bool, error)
	AppendAgentRunToolTraceForSkillLease(
		ctx context.Context,
		ownerID int64,
		runID int64,
		leaseID string,
		step AgentRunStepRow,
		trace ToolTraceRow,
	) (AgentRunStepRow, ToolTraceRow, bool, error)
	UpdateAgentRunPlanForSkillLease(
		ctx context.Context,
		ownerID int64,
		runID int64,
		leaseID string,
		planJSON string,
		optionContextJSON string,
	) (AgentRunRow, bool, error)
	UpdateAgentRunRuntimeGovernanceForSkillLease(
		ctx context.Context,
		ownerID int64,
		runID int64,
		leaseID string,
		model RuntimeModelInfo,
	) (bool, error)
	AppendAgentRunEventForSkillLease(
		ctx context.Context,
		ownerID int64,
		runID int64,
		leaseID string,
		eventType string,
		payload string,
	) (AgentRunEventRow, bool, error)
	AppendAgentRunAssistantMessageForSkillLease(
		ctx context.Context,
		ownerID int64,
		runID int64,
		leaseID string,
		message ChatMessageRow,
	) (ChatMessageRow, bool, error)
	CompleteAgentRunForSkillLease(
		ctx context.Context,
		ownerID int64,
		runID int64,
		leaseID string,
		assistantText string,
		status string,
		errorType string,
		errorMessage string,
		resultMetadataJSON string,
	) (AgentRunRow, bool, error)
}

func withAgentSkillApproval(ctx context.Context, runID, hrID int64, approval *agentRunSkillApproval) context.Context {
	if approval == nil {
		return ctx
	}
	return context.WithValue(ctx, agentSkillApprovalContextKey{}, agentSkillApprovalContext{
		RunID:    runID,
		HRID:     hrID,
		Approval: approval,
	})
}

func agentRunSkillApprovalLeaseFromContext(ctx context.Context) string {
	value, ok := ctx.Value(agentSkillApprovalContextKey{}).(agentSkillApprovalContext)
	if !ok || value.Approval == nil || value.Approval.DispatchState != agentSkillDispatchClaimed {
		return ""
	}
	return strings.TrimSpace(value.Approval.DispatchLeaseID)
}

func hasAgentRunSkillApprovalContext(ctx context.Context) bool {
	value, ok := ctx.Value(agentSkillApprovalContextKey{}).(agentSkillApprovalContext)
	return ok && value.Approval != nil
}

func agentRunSkillApprovalExecutionFromContext(ctx context.Context) (agentSkillApprovalContext, bool) {
	value, ok := ctx.Value(agentSkillApprovalContextKey{}).(agentSkillApprovalContext)
	if !ok || value.Approval == nil ||
		value.RunID <= 0 || value.HRID <= 0 ||
		value.Approval.DispatchState != agentSkillDispatchClaimed ||
		strings.TrimSpace(value.Approval.DispatchLeaseID) == "" {
		return agentSkillApprovalContext{}, false
	}
	return value, true
}

func (s *nativeAIService) verifyAgentRunSkillExecutionLease(ctx context.Context) error {
	if !hasAgentRunSkillApprovalContext(ctx) {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("%w: execution context canceled", errAgentRunExecutionLeaseLost)
	}
	execution, ok := agentRunSkillApprovalExecutionFromContext(ctx)
	if !ok {
		return errAgentRunExecutionLeaseLost
	}
	store, ok := s.store.(agentRunSkillExecutionFenceStore)
	if !ok {
		return errAgentRunExecutionLeaseLost
	}
	owned, err := store.VerifyAgentRunSkillLease(
		ctx,
		execution.HRID,
		execution.RunID,
		execution.Approval.DispatchLeaseID,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: execution context canceled", errAgentRunExecutionLeaseLost)
		}
		return err
	}
	if !owned {
		return errAgentRunExecutionLeaseLost
	}
	return nil
}

func agentSkillApprovalAllows(
	ctx context.Context,
	selected []hrRuntimeAgentSkill,
	model RuntimeModelInfo,
	message string,
) bool {
	value, ok := ctx.Value(agentSkillApprovalContextKey{}).(agentSkillApprovalContext)
	if !ok || value.Approval == nil || value.RunID <= 0 || value.HRID <= 0 || len(selected) == 0 {
		return false
	}
	approval := value.Approval
	versionIDs, hashes := agentSkillCompositionIdentity(selected)
	if strings.TrimSpace(approval.ConfirmationID) == "" ||
		approval.DispatchState != agentSkillDispatchClaimed ||
		approval.RunID != value.RunID ||
		approval.HRID != value.HRID ||
		approval.UserID <= 0 ||
		approval.UserMessageID <= 0 ||
		approval.CapabilityVersionID != model.CapabilityVersionID ||
		approval.CapabilitySnapshotHash != model.CapabilitySnapshotHash ||
		approval.MessageDigest != agentSkillMessageDigest(message) ||
		!equalInt64Order(approval.VersionIDs, versionIDs) ||
		!equalStringOrder(approval.CompiledHashes, hashes) {
		return false
	}
	actor := platformmetadata.GetTenantContext(ctx)
	if approval.TenantID > 0 && approval.TenantID != actor.TenantID {
		return false
	}
	authUserID := platformmetadata.GetAuthUserID(ctx)
	if approval.UserID != authUserID {
		return false
	}
	return true
}

func agentSkillCompositionIdentity(skills []hrRuntimeAgentSkill) ([]int64, []string) {
	versionIDs := make([]int64, 0, len(skills))
	hashes := make([]string, 0, len(skills))
	for _, skill := range skills {
		versionIDs = append(versionIDs, skill.VersionID)
		hashes = append(hashes, skill.CompiledHash)
	}
	return versionIDs, hashes
}

func agentSkillMessageDigest(message string) string {
	sum := sha256.Sum256([]byte(message))
	return hex.EncodeToString(sum[:])
}

func agentSkillSelectionPayload(governance hrRuntimeGovernanceContext, userMessageID int64) *pb.AgentSkillSelection {
	candidates, versionIDs := agentSkillConfirmationCandidates(governance.AgentSkillRuntimeEvidence)
	return &pb.AgentSkillSelection{
		Required:                        true,
		Reason:                          "agent_skill_confirmation_required",
		Candidates:                      candidates,
		UserMessageId:                   userMessageID,
		RecommendedAgentSkillVersionIds: versionIDs,
	}
}

func agentSkillConfirmationCandidates(evidence []*pb.AgentSkillRuntimeEvidence) ([]*pb.AgentSkillSelectionCandidate, []int64) {
	candidates := make([]*pb.AgentSkillSelectionCandidate, 0, len(evidence))
	versionIDs := make([]int64, 0, len(evidence))
	for _, item := range evidence {
		if item == nil {
			continue
		}
		switch item.GetDecisionReason() {
		case "confirmation_required", "blocked_by_confirmation":
		default:
			continue
		}
		candidates = append(candidates, &pb.AgentSkillSelectionCandidate{
			Name:                item.GetSkillName(),
			DisplayName:         item.GetDisplayName(),
			Reason:              item.GetDecisionReason(),
			Recommended:         true,
			SkillId:             item.GetSkillId(),
			VersionId:           item.GetVersionId(),
			Version:             item.GetVersion(),
			CompiledHash:        item.GetCompiledHash(),
			CompositionRole:     item.GetCompositionRole(),
			CoreEstimatedTokens: item.GetCoreEstimatedTokens(),
			Risk:                item.GetRisk(),
			ActivationPolicy:    item.GetActivationPolicy(),
			RelevanceMode:       item.GetRelevanceMode(),
		})
		versionIDs = append(versionIDs, item.GetVersionId())
	}
	return candidates, versionIDs
}

func (s *nativeAIService) ensureAgentRunSkillConfirmationUserMessage(
	ctx context.Context,
	req *pb.ChatRequest,
	governance hrRuntimeGovernanceContext,
	runID int64,
) (ChatMessageRow, error) {
	if s == nil || s.store == nil || req == nil {
		return ChatMessageRow{}, errAIStoreRequired
	}
	store, ok := s.store.(agentRunUserMessageStore)
	if !ok || runID <= 0 {
		return ChatMessageRow{}, errInvalidAgentSkillConfirmation
	}
	_, versionIDs := agentSkillConfirmationCandidates(governance.AgentSkillRuntimeEvidence)
	message, _, _, err := store.EnsureAgentRunUserMessage(ctx, req.GetHrId(), runID, ChatMessageRow{
		OwnerRole:            ownerRoleHR,
		OwnerID:              req.GetHrId(),
		SessionID:            req.GetSessionId(),
		Role:                 "user",
		Content:              req.GetMessage(),
		ModelID:              req.GetModelId(),
		AgentSkillVersionIDs: versionIDs,
	})
	return message, err
}

func newAgentRunSkillConfirmation(
	run AgentRunRow,
	payload agentRunDurablePayload,
	required *agentSkillConfirmationRequiredError,
) (agentRunSkillConfirmation, error) {
	if required == nil {
		return agentRunSkillConfirmation{}, errInvalidAgentSkillConfirmation
	}
	if required.RuntimeModel.CapabilityVersionID <= 0 ||
		!validSHA256Hex(required.RuntimeModel.CapabilitySnapshotHash) {
		return agentRunSkillConfirmation{}, errInvalidAgentSkillConfirmation
	}
	candidates, versionIDs := agentSkillConfirmationCandidates(required.Governance.AgentSkillRuntimeEvidence)
	if len(candidates) == 0 || len(versionIDs) == 0 {
		return agentRunSkillConfirmation{}, errInvalidAgentSkillConfirmation
	}
	hashes := make([]string, 0, len(candidates))
	role, risk := "none", "unknown"
	for _, candidate := range candidates {
		if candidate.GetVersionId() <= 0 || !validSHA256Hex(candidate.GetCompiledHash()) {
			return agentRunSkillConfirmation{}, errInvalidAgentSkillConfirmation
		}
		hashes = append(hashes, candidate.GetCompiledHash())
		if candidate.GetCompositionRole() == pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY {
			role = "primary"
		}
		switch candidate.GetRisk() {
		case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL:
			risk = "critical"
		case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH:
			if risk != "critical" {
				risk = "high"
			}
		}
	}
	selectionMode := required.Governance.AgentSkillSelectionMode
	if selectionMode != "manual" {
		selectionMode = "auto"
	}
	userID := payload.AuthUserID
	if userID <= 0 {
		userID = run.OwnerID
	}
	return agentRunSkillConfirmation{
		ID:                     newAgentSkillConfirmationID(run.ID, versionIDs),
		TenantID:               firstNonZeroInt64(payload.AuthTenantID, run.TenantID),
		UserID:                 userID,
		HRID:                   run.OwnerID,
		RunID:                  run.ID,
		CapabilityVersionID:    required.RuntimeModel.CapabilityVersionID,
		CapabilitySnapshotHash: required.RuntimeModel.CapabilitySnapshotHash,
		VersionIDs:             versionIDs,
		CompiledHashes:         hashes,
		MessageDigest:          agentSkillMessageDigest(payload.Message),
		SelectionMode:          selectionMode,
		Role:                   role,
		Risk:                   risk,
		UserMessageID:          required.UserMessageID,
		ExpiresAt:              businessclock.FormatRFC3339(time.Now().Add(agentSkillConfirmationTTL)),
	}, nil
}

func newAgentSkillConfirmationID(runID int64, versionIDs []int64) string {
	random := make([]byte, 24)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	fallback := sha256.Sum256([]byte(fmt.Sprintf("%d:%v:%d", runID, versionIDs, time.Now().UnixNano())))
	return hex.EncodeToString(fallback[:24])
}

func agentRunSkillConfirmationPB(
	confirmation agentRunSkillConfirmation,
	evidence []*pb.AgentSkillRuntimeEvidence,
) *pb.AgentRunConfirmationPayload {
	candidates, _ := agentSkillConfirmationCandidates(evidence)
	return &pb.AgentRunConfirmationPayload{
		Required:                        true,
		Reason:                          "agent_skill_confirmation_required",
		Candidates:                      candidates,
		AgentSkillConfirmationId:        confirmation.ID,
		RecommendedAgentSkillVersionIds: append([]int64(nil), confirmation.VersionIDs...),
		AgentSkillUserMessageId:         confirmation.UserMessageID,
		AgentSkillConfirmationExpiresAt: confirmation.ExpiresAt,
	}
}

func agentRunSkillConfirmationContextJSON(confirmation *pb.AgentRunConfirmationPayload) string {
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(confirmation)
	if err != nil {
		return "{}"
	}
	return marshalJSONString(map[string]any{"confirmation_request": json.RawMessage(raw)})
}

func agentRunSkillResolutionContextJSON(confirmation agentRunSkillConfirmation, outcome string) string {
	return marshalJSONString(map[string]any{
		"confirmation": map[string]any{
			"required":                            false,
			"reason":                              outcome,
			"agent_skill_confirmation_id":         confirmation.ID,
			"recommended_agent_skill_version_ids": append([]int64(nil), confirmation.VersionIDs...),
			"agent_skill_user_message_id":         confirmation.UserMessageID,
			"agent_skill_confirmation_expires_at": confirmation.ExpiresAt,
		},
	})
}

func (s *nativeAIService) finishAgentRunWaitingForSkillConfirmation(
	ctx context.Context,
	run AgentRunRow,
	required *agentSkillConfirmationRequiredError,
) error {
	storeCtx := agentRunStoreContext(ctx)
	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()

	current, found, err := s.getRun(storeCtx, run.OwnerID, run.ID)
	if err != nil || !found {
		return err
	}
	if current.Status == agentRunStatusCancelRequested || current.Status == agentRunStatusCanceled {
		return s.completeAgentRunCanceledLocked(storeCtx, current)
	}
	if isTerminalAgentRunStatus(current.Status) || current.Status == agentRunStatusWaitingConfirmation {
		return nil
	}
	payload := agentRunPayloadFromRow(current)
	confirmation, err := newAgentRunSkillConfirmation(current, payload, required)
	if err != nil {
		return err
	}
	payload.PendingAgentSkillConfirmation = &confirmation
	payload.AgentSkillApproval = nil
	payload.PendingMCPConfirmation = nil
	payload.MCPApproval = nil
	confirmationPB := agentRunSkillConfirmationPB(confirmation, required.Governance.AgentSkillRuntimeEvidence)
	optionContextJSON := agentRunSkillConfirmationContextJSON(confirmationPB)
	waiting, updated, updateErr := s.transitionAgentRunState(
		storeCtx,
		current,
		current.Status,
		agentRunStatusWaitingConfirmation,
		agentRunPlanJSON(payload),
		optionContextJSON,
		"",
		"",
	)
	if updateErr != nil {
		return updateErr
	}
	if !updated {
		return fmt.Errorf("agent run not found")
	}
	eventPayload := marshalJSONString(map[string]any{
		"status":                       agentRunStatusWaitingConfirmation,
		"confirmation_request":         json.RawMessage(extractAgentRunConfirmationJSON(optionContextJSON)),
		"agent_skill_runtime_evidence": required.Governance.AgentSkillRuntimeEvidence,
	})
	if _, eventErr := s.appendAgentRunEvent(storeCtx, waiting.ID, "confirmation.required", eventPayload); eventErr != nil {
		return eventErr
	}
	s.recordAgentSkillConfirmation(
		withAgentSkillExecutionMode(storeCtx, true),
		confirmation.SelectionMode,
		confirmation.Role,
		confirmation.Risk,
		"confirmation_required",
		"confirmation_required",
	)
	return nil
}

func extractAgentRunConfirmationJSON(optionContextJSON string) string {
	var payload struct {
		ConfirmationRequest json.RawMessage `json:"confirmation_request"`
	}
	if json.Unmarshal([]byte(optionContextJSON), &payload) != nil || len(payload.ConfirmationRequest) == 0 {
		return "{}"
	}
	return string(payload.ConfirmationRequest)
}

func validateAgentRunSkillConfirmation(
	ctx context.Context,
	run AgentRunRow,
	payload agentRunDurablePayload,
	req *pb.ConfirmAgentRunRequest,
	now time.Time,
) error {
	pending := payload.PendingAgentSkillConfirmation
	if pending == nil || req == nil ||
		strings.TrimSpace(req.GetConfirmationPayloadJson()) != "" ||
		strings.TrimSpace(req.GetAgentSkillConfirmationId()) == "" ||
		req.GetAgentSkillConfirmationId() != pending.ID ||
		pending.RunID != run.ID ||
		pending.HRID != run.OwnerID ||
		pending.HRID <= 0 ||
		pending.UserID <= 0 ||
		pending.CapabilityVersionID <= 0 ||
		!validSHA256Hex(pending.CapabilitySnapshotHash) ||
		!validSHA256Hex(pending.MessageDigest) ||
		pending.MessageDigest != agentSkillMessageDigest(payload.Message) ||
		pending.UserMessageID <= 0 ||
		pending.UserMessageID != run.MessageID ||
		len(pending.VersionIDs) == 0 ||
		len(pending.VersionIDs) != len(pending.CompiledHashes) {
		return errInvalidAgentSkillConfirmation
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, pending.ExpiresAt)
	if err != nil || !now.Before(expiresAt) {
		return errExpiredAgentSkillConfirmation
	}
	actor := platformmetadata.GetTenantContext(ctx)
	if pending.TenantID > 0 && pending.TenantID != actor.TenantID {
		return errInvalidAgentSkillConfirmation
	}
	if run.TenantID > 0 && pending.TenantID != run.TenantID {
		return errInvalidAgentSkillConfirmation
	}
	authUserID := platformmetadata.GetAuthUserID(ctx)
	if authUserID <= 0 {
		authUserID = req.GetHrId()
	}
	if pending.UserID != authUserID || pending.UserID != firstNonZeroInt64(payload.AuthUserID, run.OwnerID) {
		return errInvalidAgentSkillConfirmation
	}
	seenVersionIDs := make(map[int64]struct{}, len(pending.VersionIDs))
	for index, versionID := range pending.VersionIDs {
		if versionID <= 0 || !validSHA256Hex(pending.CompiledHashes[index]) {
			return errInvalidAgentSkillConfirmation
		}
		if _, exists := seenVersionIDs[versionID]; exists {
			return errInvalidAgentSkillConfirmation
		}
		seenVersionIDs[versionID] = struct{}{}
	}
	if req.GetAgentSkillConfirmationDecision() == pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE {
		selected := req.GetSelectedAgentSkillVersionIds()
		if len(selected) == 0 || !equalInt64Order(selected, pending.VersionIDs) {
			return errInvalidAgentSkillConfirmation
		}
	} else if req.GetAgentSkillConfirmationDecision() != pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_REJECT {
		return errInvalidAgentSkillConfirmation
	}
	return nil
}

func (s *nativeAIService) revalidateAgentRunSkillBinding(
	ctx context.Context,
	run AgentRunRow,
	payload agentRunDurablePayload,
) (RuntimeModelInfo, error) {
	pending := payload.PendingAgentSkillConfirmation
	if pending == nil {
		return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
	}
	if validateErr := s.validateAgentRunSkillMessageBinding(ctx, run, pending.UserMessageID, pending.MessageDigest); validateErr != nil {
		return RuntimeModelInfo{}, validateErr
	}
	model, err := s.resolveCapabilityRuntimeModel(
		ctx,
		billingOwnerTenant,
		run.OwnerID,
		"ai.agent_run",
		platformAIAudienceTenantHR,
		payload.ModelID,
	)
	if err != nil {
		return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
	}
	if model.CapabilityVersionID != pending.CapabilityVersionID ||
		model.CapabilitySnapshotHash != pending.CapabilitySnapshotHash {
		return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
	}
	allowed := int64RuntimeSet(model.ConfigurationRefs.AgentSkillVersionIDs)
	for _, versionID := range pending.VersionIDs {
		if !allowed[versionID] {
			return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
		}
	}
	store, ok := s.store.(hrRuntimeAgentSkillPackageStore)
	if !ok {
		return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
	}
	packages, err := store.LoadAgentSkillRuntimePackages(ctx, pending.VersionIDs)
	if err != nil || len(packages) != len(pending.VersionIDs) {
		return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
	}
	hashByVersion := make(map[int64]string, len(packages))
	for _, runtimePackage := range packages {
		document, _, validateErr := validateHRRuntimeAgentSkillPackage(runtimePackage)
		if validateErr != nil {
			return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
		}
		hashByVersion[document.ID] = document.CompiledHash
	}
	for index, versionID := range pending.VersionIDs {
		if hashByVersion[versionID] != pending.CompiledHashes[index] {
			return RuntimeModelInfo{}, errInvalidAgentSkillConfirmation
		}
	}
	return model, nil
}

func (s *nativeAIService) validateAgentRunSkillMessageBinding(
	ctx context.Context,
	run AgentRunRow,
	messageID int64,
	messageDigest string,
) error {
	if run.MessageID <= 0 || run.MessageID != messageID || !validSHA256Hex(messageDigest) {
		return errInvalidAgentSkillConfirmation
	}
	history, err := s.hrContextMessages(ctx, run.OwnerID, run.SessionID)
	if err != nil {
		return errInvalidAgentSkillConfirmation
	}
	message := findHRUserMessageByID(history, messageID)
	if message.ID == 0 || agentSkillMessageDigest(message.Content) != messageDigest {
		return errInvalidAgentSkillConfirmation
	}
	return nil
}

func approvedAgentRunSkillPayload(
	payload agentRunDurablePayload,
	pending agentRunSkillConfirmation,
	selectedVersionIDs []int64,
	clientRequestID string,
	now time.Time,
) agentRunDurablePayload {
	payload.AgentSkillVersionIDs = append([]int64(nil), selectedVersionIDs...)
	payload.AgentSkillApproval = &agentRunSkillApproval{
		ConfirmationID:          pending.ID,
		TenantID:                pending.TenantID,
		UserID:                  pending.UserID,
		HRID:                    pending.HRID,
		RunID:                   pending.RunID,
		CapabilityVersionID:     pending.CapabilityVersionID,
		CapabilitySnapshotHash:  pending.CapabilitySnapshotHash,
		VersionIDs:              append([]int64(nil), pending.VersionIDs...),
		CompiledHashes:          append([]string(nil), pending.CompiledHashes...),
		MessageDigest:           pending.MessageDigest,
		SelectionMode:           pending.SelectionMode,
		Role:                    pending.Role,
		Risk:                    pending.Risk,
		UserMessageID:           pending.UserMessageID,
		ConfirmationExpiresAt:   pending.ExpiresAt,
		ApprovedAt:              businessclock.FormatRFC3339(now),
		ApprovalClientRequestID: strings.TrimSpace(clientRequestID),
		DispatchState:           agentSkillDispatchPendingEvent,
	}
	payload.PendingAgentSkillConfirmation = nil
	payload.ConfirmationPayloadJSON = ""
	return payload
}

func validateApprovedAgentSkillRetry(
	ctx context.Context,
	run AgentRunRow,
	payload agentRunDurablePayload,
	req *pb.ConfirmAgentRunRequest,
) error {
	approval := payload.AgentSkillApproval
	if approval == nil || req == nil ||
		req.GetAgentSkillConfirmationDecision() != pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE ||
		req.GetAgentSkillConfirmationId() != approval.ConfirmationID ||
		strings.TrimSpace(req.GetClientRequestId()) != approval.ApprovalClientRequestID ||
		!equalInt64Order(req.GetSelectedAgentSkillVersionIds(), approval.VersionIDs) ||
		approval.RunID != run.ID ||
		approval.HRID != run.OwnerID ||
		approval.UserMessageID <= 0 ||
		approval.UserMessageID != run.MessageID ||
		approval.MessageDigest != agentSkillMessageDigest(payload.Message) {
		return errInvalidAgentSkillConfirmation
	}
	actor := platformmetadata.GetTenantContext(ctx)
	if approval.TenantID > 0 && approval.TenantID != actor.TenantID {
		return errInvalidAgentSkillConfirmation
	}
	authUserID := platformmetadata.GetAuthUserID(ctx)
	if authUserID <= 0 {
		authUserID = req.GetHrId()
	}
	if approval.UserID != authUserID {
		return errInvalidAgentSkillConfirmation
	}
	return nil
}

func agentRunSkillApprovalAcceptedPayload(approval agentRunSkillApproval) string {
	return marshalJSONString(map[string]any{
		"status":                           agentRunStatusQueued,
		"confirmation_type":                "agent_skill",
		"agent_skill_confirmation_id":      approval.ConfirmationID,
		"selected_agent_skill_version_ids": append([]int64(nil), approval.VersionIDs...),
		"agent_skill_user_message_id":      approval.UserMessageID,
	})
}

func readyAgentRunSkillPayload(payload agentRunDurablePayload) agentRunDurablePayload {
	if payload.AgentSkillApproval == nil {
		return payload
	}
	approval := *payload.AgentSkillApproval
	approval.DispatchState = agentSkillDispatchReady
	approval.DispatchLeaseID = ""
	approval.DispatchLeaseExpiresAt = ""
	payload.AgentSkillApproval = &approval
	return payload
}

func claimedAgentRunSkillPayload(
	payload agentRunDurablePayload,
	leaseID string,
	expiresAt time.Time,
) agentRunDurablePayload {
	if payload.AgentSkillApproval == nil {
		return payload
	}
	approval := *payload.AgentSkillApproval
	approval.DispatchState = agentSkillDispatchClaimed
	approval.DispatchLeaseID = leaseID
	approval.DispatchLeaseExpiresAt = businessclock.FormatRFC3339(expiresAt)
	payload.AgentSkillApproval = &approval
	return payload
}

func agentSkillDispatchLeaseExpired(approval *agentRunSkillApproval, now time.Time) bool {
	if approval == nil || approval.DispatchState != agentSkillDispatchClaimed {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, approval.DispatchLeaseExpiresAt)
	return err != nil || !now.Before(expiresAt)
}

func validAgentRunSkillApprovalMarker(run AgentRunRow, approval *agentRunSkillApproval) bool {
	if approval == nil ||
		strings.TrimSpace(approval.ConfirmationID) == "" ||
		approval.RunID != run.ID ||
		approval.HRID != run.OwnerID ||
		approval.UserID <= 0 ||
		approval.UserMessageID <= 0 ||
		approval.UserMessageID != run.MessageID ||
		approval.CapabilityVersionID <= 0 ||
		!validSHA256Hex(approval.CapabilitySnapshotHash) ||
		!validSHA256Hex(approval.MessageDigest) ||
		!validRFC3339Timestamp(approval.ConfirmationExpiresAt) ||
		!validRFC3339Timestamp(approval.ApprovedAt) ||
		len(approval.VersionIDs) == 0 ||
		len(approval.VersionIDs) != len(approval.CompiledHashes) {
		return false
	}
	switch approval.Risk {
	case "high", "critical":
	default:
		return false
	}
	switch approval.DispatchState {
	case agentSkillDispatchPendingEvent, agentSkillDispatchReady:
		if strings.TrimSpace(approval.DispatchLeaseID) != "" ||
			strings.TrimSpace(approval.DispatchLeaseExpiresAt) != "" {
			return false
		}
	case agentSkillDispatchClaimed:
		if strings.TrimSpace(approval.DispatchLeaseID) == "" ||
			!validRFC3339Timestamp(approval.DispatchLeaseExpiresAt) {
			return false
		}
	default:
		return false
	}
	seen := make(map[int64]struct{}, len(approval.VersionIDs))
	for index, versionID := range approval.VersionIDs {
		if versionID <= 0 || !validSHA256Hex(approval.CompiledHashes[index]) {
			return false
		}
		if _, exists := seen[versionID]; exists {
			return false
		}
		seen[versionID] = struct{}{}
	}
	return true
}

func validRFC3339Timestamp(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	return err == nil
}

func agentRunSkillExecutionLease(run AgentRunRow) string {
	payload := agentRunPayloadFromRow(run)
	approval := payload.AgentSkillApproval
	if !validAgentRunSkillApprovalMarker(run, approval) ||
		(run.Status != agentRunStatusRunning && run.Status != agentRunStatusCancelRequested) ||
		approval.DispatchState != agentSkillDispatchClaimed {
		return ""
	}
	return strings.TrimSpace(approval.DispatchLeaseID)
}

func (s *nativeAIService) appendAgentRunExecutionEvent(
	ctx context.Context,
	run AgentRunRow,
	eventType string,
	payload string,
) (AgentRunEventRow, error) {
	runPayload := agentRunPayloadFromRow(run)
	leaseID := agentRunSkillExecutionLease(run)
	if leaseID == "" {
		if runPayload.AgentSkillApproval != nil {
			return AgentRunEventRow{}, errAgentRunExecutionLeaseLost
		}
		return s.appendAgentRunEvent(ctx, run.ID, eventType, payload)
	}
	store, ok := s.store.(agentRunSkillExecutionFenceStore)
	if !ok {
		return AgentRunEventRow{}, errAgentRunExecutionLeaseLost
	}
	event, owned, err := store.AppendAgentRunEventForSkillLease(
		ctx,
		run.OwnerID,
		run.ID,
		leaseID,
		eventType,
		payload,
	)
	if err != nil {
		return AgentRunEventRow{}, err
	}
	if !owned {
		return AgentRunEventRow{}, errAgentRunExecutionLeaseLost
	}
	s.agentRunEvents().publish(event)
	return event, nil
}

func (s *nativeAIService) completeAgentRunExecution(
	ctx context.Context,
	run AgentRunRow,
	assistantText string,
	status string,
	errorType string,
	errorMessage string,
	resultMetadataJSON string,
) (AgentRunRow, bool, error) {
	runPayload := agentRunPayloadFromRow(run)
	leaseID := agentRunSkillExecutionLease(run)
	if leaseID == "" {
		if runPayload.AgentSkillApproval != nil {
			return AgentRunRow{}, false, errAgentRunExecutionLeaseLost
		}
		return s.store.CompleteAgentRun(ctx, run.OwnerID, run.ID, assistantText, status, errorType, errorMessage, resultMetadataJSON)
	}
	store, ok := s.store.(agentRunSkillExecutionFenceStore)
	if !ok {
		return AgentRunRow{}, false, errAgentRunExecutionLeaseLost
	}
	completed, owned, err := store.CompleteAgentRunForSkillLease(
		ctx,
		run.OwnerID,
		run.ID,
		leaseID,
		assistantText,
		status,
		errorType,
		errorMessage,
		resultMetadataJSON,
	)
	if err != nil {
		return AgentRunRow{}, false, err
	}
	if !owned {
		return AgentRunRow{}, false, errAgentRunExecutionLeaseLost
	}
	return completed, true, nil
}

func newAgentSkillDispatchLeaseID(runID int64) string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	fallback := sha256.Sum256([]byte(fmt.Sprintf("dispatch:%d:%d", runID, time.Now().UnixNano())))
	return hex.EncodeToString(fallback[:16])
}

func (s *nativeAIService) finalizeApprovedAgentSkillHandoff(
	ctx context.Context,
	run AgentRunRow,
) (AgentRunRow, error) {
	payload := agentRunPayloadFromRow(run)
	approval := payload.AgentSkillApproval
	if !validAgentRunSkillApprovalMarker(run, approval) {
		return AgentRunRow{}, errInvalidAgentSkillConfirmation
	}
	switch approval.DispatchState {
	case agentSkillDispatchPendingEvent:
		store, ok := s.store.(agentRunSkillApprovalFinalizeStore)
		if !ok || run.Status != agentRunStatusQueued {
			return AgentRunRow{}, errInvalidAgentSkillConfirmation
		}
		readyPayload := readyAgentRunSkillPayload(payload)
		finalized, acceptedEvent, inserted, err := store.FinalizeAgentRunSkillApproval(
			ctx,
			run.OwnerID,
			run.ID,
			run.PlanJSON,
			agentRunPlanJSON(readyPayload),
			agentRunSkillResolutionContextJSON(agentRunSkillConfirmationFromApproval(*approval), "approved"),
			agentRunSkillApprovalAcceptedPayload(*approval),
		)
		if err != nil {
			return AgentRunRow{}, err
		}
		if inserted {
			s.agentRunEvents().publish(acceptedEvent)
		}
		if finalized.ID == 0 {
			return AgentRunRow{}, errInvalidAgentSkillConfirmation
		}
		finalizedPayload := agentRunPayloadFromRow(finalized)
		if finalizedPayload.AgentSkillApproval == nil ||
			finalizedPayload.AgentSkillApproval.ConfirmationID != approval.ConfirmationID ||
			(finalizedPayload.AgentSkillApproval.DispatchState != agentSkillDispatchReady &&
				finalizedPayload.AgentSkillApproval.DispatchState != agentSkillDispatchClaimed) {
			return AgentRunRow{}, errInvalidAgentSkillConfirmation
		}
		s.recordApprovedAgentSkillConfirmationOnce(ctx, finalized)
		return finalized, nil
	case agentSkillDispatchReady:
		if run.Status != agentRunStatusQueued {
			return AgentRunRow{}, errInvalidAgentSkillConfirmation
		}
		s.recordApprovedAgentSkillConfirmationOnce(ctx, run)
		return run, nil
	case agentSkillDispatchClaimed:
		if run.Status != agentRunStatusRunning {
			return AgentRunRow{}, errInvalidAgentSkillConfirmation
		}
		s.recordApprovedAgentSkillConfirmationOnce(ctx, run)
		return run, nil
	default:
		return AgentRunRow{}, errInvalidAgentSkillConfirmation
	}
}

func (s *nativeAIService) recoverStaleAgentSkillDispatchLocked(
	ctx context.Context,
	run AgentRunRow,
	now time.Time,
) (AgentRunRow, error) {
	payload := agentRunPayloadFromRow(run)
	if run.Status != agentRunStatusRunning ||
		payload.AgentSkillApproval == nil ||
		!agentSkillDispatchLeaseExpired(payload.AgentSkillApproval, now) {
		return run, nil
	}
	readyPayload := readyAgentRunSkillPayload(payload)
	recovered, transitioned, err := s.transitionAgentRunState(
		ctx,
		run,
		agentRunStatusRunning,
		agentRunStatusQueued,
		agentRunPlanJSON(readyPayload),
		run.OptionContextJSON,
		"",
		"",
	)
	if err != nil {
		return AgentRunRow{}, err
	}
	if !transitioned {
		latest, found, loadErr := s.getRun(ctx, run.OwnerID, run.ID)
		if loadErr != nil {
			return AgentRunRow{}, loadErr
		}
		if found {
			return latest, nil
		}
		return AgentRunRow{}, errInvalidAgentSkillConfirmation
	}
	return recovered, nil
}

func (s *nativeAIService) recoverApprovedAgentSkillHandoff(
	ctx context.Context,
	run AgentRunRow,
	now time.Time,
) (AgentRunRow, error) {
	payload := agentRunPayloadFromRow(run)
	approval := payload.AgentSkillApproval
	if !validAgentRunSkillApprovalMarker(run, approval) {
		return run, nil
	}
	switch {
	case run.Status == agentRunStatusQueued &&
		(approval.DispatchState == agentSkillDispatchPendingEvent ||
			approval.DispatchState == agentSkillDispatchReady):
	case run.Status == agentRunStatusRunning &&
		approval.DispatchState == agentSkillDispatchClaimed &&
		agentSkillDispatchLeaseExpired(approval, now):
	case run.Status == agentRunStatusRunning &&
		approval.DispatchState == agentSkillDispatchClaimed:
		s.recordApprovedAgentSkillConfirmationOnce(ctx, run)
		return run, nil
	default:
		return run, nil
	}
	s.runTransitionMu.Lock()
	recovered, err := s.recoverStaleAgentSkillDispatchLocked(ctx, run, now)
	if err == nil {
		payload = agentRunPayloadFromRow(recovered)
		if recovered.Status == agentRunStatusQueued && payload.AgentSkillApproval != nil {
			recovered, err = s.finalizeApprovedAgentSkillHandoff(ctx, recovered)
		}
	}
	s.runTransitionMu.Unlock()
	if err == nil && recovered.Status == agentRunStatusQueued {
		if run.Status == agentRunStatusRunning {
			// A lease is deliberately longer than twice the execution timeout. A
			// local worker should therefore already be canceled; cancel again before
			// redispatch so a provider that honors context cannot overlap recovery.
			s.cancelAgentRunExecution(run.ID)
		}
		s.dispatchAgentRun(recovered)
	}
	return recovered, err
}

func (s *nativeAIService) recordApprovedAgentSkillConfirmationOnce(ctx context.Context, run AgentRunRow) {
	if s == nil || s.store == nil {
		return
	}
	payload := agentRunPayloadFromRow(run)
	approval := payload.AgentSkillApproval
	if !validAgentRunSkillApprovalMarker(run, approval) {
		return
	}
	events, err := s.store.ListAgentRunEvents(ctx, run.OwnerID, run.ID, 0)
	if err != nil {
		return
	}
	for _, event := range events {
		if event.EventType != "confirmation.accepted" {
			continue
		}
		var accepted struct {
			ConfirmationType         string `json:"confirmation_type"`
			AgentSkillConfirmationID string `json:"agent_skill_confirmation_id"`
		}
		if json.Unmarshal([]byte(event.PayloadJSON), &accepted) != nil ||
			accepted.ConfirmationType != "agent_skill" ||
			accepted.AgentSkillConfirmationID != approval.ConfirmationID {
			continue
		}
		key := fmt.Sprintf("%d:%d:%s", event.RunID, event.Seq, accepted.AgentSkillConfirmationID)
		s.agentSkillMetricMu.Lock()
		if s.agentSkillMetricKeys == nil {
			s.agentSkillMetricKeys = make(map[string]struct{})
		}
		_, recorded := s.agentSkillMetricKeys[key]
		if !recorded {
			s.agentSkillMetricKeys[key] = struct{}{}
		}
		s.agentSkillMetricMu.Unlock()
		if recorded {
			return
		}
		s.recordAgentSkillConfirmation(
			withAgentSkillExecutionMode(ctx, true),
			approval.SelectionMode,
			approval.Role,
			approval.Risk,
			"passed",
			"approved",
		)
		return
	}
}

func agentRunSkillConfirmationFromApproval(approval agentRunSkillApproval) agentRunSkillConfirmation {
	return agentRunSkillConfirmation{
		ID:            approval.ConfirmationID,
		VersionIDs:    append([]int64(nil), approval.VersionIDs...),
		UserMessageID: approval.UserMessageID,
		ExpiresAt:     approval.ConfirmationExpiresAt,
	}
}

func (s *nativeAIService) transitionAgentRunConfirmation(
	ctx context.Context,
	run AgentRunRow,
	toStatus string,
	planJSON string,
	optionContextJSON string,
	errorType string,
	errorMessage string,
) (AgentRunRow, bool, error) {
	return s.transitionAgentRunState(
		ctx,
		run,
		agentRunStatusWaitingConfirmation,
		toStatus,
		planJSON,
		optionContextJSON,
		errorType,
		errorMessage,
	)
}

func (s *nativeAIService) transitionAgentRunState(
	ctx context.Context,
	run AgentRunRow,
	fromStatus string,
	toStatus string,
	planJSON string,
	optionContextJSON string,
	errorType string,
	errorMessage string,
) (AgentRunRow, bool, error) {
	store, ok := s.store.(agentRunConfirmationTransitionStore)
	if !ok {
		return AgentRunRow{}, false, errInvalidAgentSkillConfirmation
	}
	return store.TransitionAgentRunConfirmation(
		ctx,
		run.OwnerID,
		run.ID,
		fromStatus,
		toStatus,
		run.PlanJSON,
		planJSON,
		optionContextJSON,
		errorType,
		errorMessage,
	)
}

func (s *nativeAIService) expirePendingAgentRunConfirmation(
	ctx context.Context,
	run AgentRunRow,
	now time.Time,
) (AgentRunRow, bool, error) {
	if run.Status != agentRunStatusWaitingConfirmation {
		return run, false, nil
	}
	payload := agentRunPayloadFromRow(run)
	expiresAtRaw := ""
	errorType := "AGENT_CONFIRMATION_EXPIRED"
	errorMessage := "ai.confirmation_expired"
	var pendingSkill *agentRunSkillConfirmation
	switch {
	case payload.PendingAgentSkillConfirmation != nil:
		pendingSkill = payload.PendingAgentSkillConfirmation
		expiresAtRaw = pendingSkill.ExpiresAt
		errorType = "AGENT_SKILL_CONFIRMATION_EXPIRED"
		errorMessage = "ai.agent_skill_confirmation_expired"
	case payload.PendingMCPConfirmation != nil:
		expiresAtRaw = payload.PendingMCPConfirmation.ExpiresAt
		errorType = "MCP_CONFIRMATION_EXPIRED"
		errorMessage = "ai.mcp_confirmation_expired"
	default:
		return run, false, nil
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtRaw)
	if err == nil && now.Before(expiresAt) {
		return run, false, nil
	}

	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()
	current, found, err := s.getRun(ctx, run.OwnerID, run.ID)
	if err != nil || !found || current.Status != agentRunStatusWaitingConfirmation {
		return current, false, err
	}
	currentPayload := agentRunPayloadFromRow(current)
	optionContextJSON := current.OptionContextJSON
	if currentPayload.PendingAgentSkillConfirmation != nil {
		pendingSkill = currentPayload.PendingAgentSkillConfirmation
		expiresAtRaw = pendingSkill.ExpiresAt
		optionContextJSON = agentRunSkillResolutionContextJSON(*pendingSkill, "expired")
	} else if currentPayload.PendingMCPConfirmation != nil {
		pendingSkill = nil
		expiresAtRaw = currentPayload.PendingMCPConfirmation.ExpiresAt
	} else {
		return current, false, nil
	}
	expiresAt, parseErr := time.Parse(time.RFC3339Nano, expiresAtRaw)
	if parseErr == nil && now.Before(expiresAt) {
		return current, false, nil
	}
	expired, transitioned, transitionErr := s.transitionAgentRunConfirmation(
		ctx,
		current,
		agentRunStatusFailed,
		current.PlanJSON,
		optionContextJSON,
		errorType,
		errorMessage,
	)
	if transitionErr != nil || !transitioned {
		return expired, transitioned, transitionErr
	}
	_, _ = s.appendAgentRunEvent(ctx, expired.ID, "run.status_changed", marshalJSONString(map[string]any{
		"status":        agentRunStatusFailed,
		"error_type":    errorType,
		"error_message": errorMessage,
	}))
	if pendingSkill != nil {
		s.recordAgentSkillConfirmation(
			withAgentSkillExecutionMode(ctx, true),
			pendingSkill.SelectionMode,
			pendingSkill.Role,
			pendingSkill.Risk,
			"failed",
			"expired",
		)
	}
	return expired, true, nil
}

func equalInt64Order(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func equalStringOrder(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
