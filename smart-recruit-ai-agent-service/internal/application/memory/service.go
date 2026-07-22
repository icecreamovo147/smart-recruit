package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
	"smart-recruit-platform-go/logger"
)

type Service struct {
	repo      Repository
	extractor *Extractor
	embedder  EmbeddingUpserter
	cfg       Config
}

type RecallRequest struct {
	OwnerRole       domainmemory.OwnerRole
	OwnerID         uint64
	Scopes          []domainmemory.Scope
	Query           string
	TargetScopeType string
	TargetScopeID   uint64
	MemoryTypes     []string
	VectorScores    map[uint64]float64
}

type RecallResult struct {
	Items      []domainmemory.RankedMemory
	InjectText string
	Evidence   RecallEvidence
}

type RecallEvidence struct {
	MemoryIDs      []uint64
	Count          int
	Chars          int
	RelevanceModes []string
}

type WriteRequest struct {
	OwnerRole       domainmemory.OwnerRole
	OwnerID         uint64
	Scope           domainmemory.Scope
	MemoryType      string
	Content         string
	Source          string
	Confidence      float64
	Importance      float64
	SourceSessionID *uint64
	SourceMessageID *uint64
	SourceRunID     *uint64
	CreatedBy       *uint64
	ConfirmHighPII  bool
}

func NewService(repo Repository, extractor *Extractor, embedder EmbeddingUpserter, cfg Config) *Service {
	return &Service{repo: repo, extractor: extractor, embedder: embedder, cfg: cfg}
}

func (s *Service) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

func (s *Service) WriteEnabled() bool {
	return s.Enabled() && s.cfg.WriteEnabled
}

func (s *Service) InjectEnabled() bool {
	return s.Enabled() && s.cfg.InjectEnabled
}

func (s *Service) Recall(ctx context.Context, req RecallRequest) (RecallResult, error) {
	if s == nil || s.repo == nil || !s.Enabled() {
		return RecallResult{}, nil
	}
	if req.OwnerID == 0 {
		return RecallResult{}, nil
	}
	for _, scope := range req.Scopes {
		if err := domainmemory.ValidateScope(req.OwnerRole, scope); err != nil {
			return RecallResult{}, err
		}
	}
	limit := s.cfg.MaxMemories * 3
	if limit <= 0 {
		limit = 30
	}
	memories, err := s.repo.ListActiveForRecall(ctx, RecallFilter{
		OwnerRole:   req.OwnerRole,
		OwnerID:     req.OwnerID,
		Scopes:      req.Scopes,
		Query:       req.Query,
		Limit:       limit,
		MemoryTypes: req.MemoryTypes,
	})
	if err != nil {
		return RecallResult{}, err
	}
	query := strings.TrimSpace(req.Query)
	tokenCount := len(strings.Fields(strings.ToLower(query)))
	input := domainmemory.RecallQueryContext{
		Query:              query,
		TargetScopeType:    req.TargetScopeType,
		TargetScopeID:      req.TargetScopeID,
		QueryTokenCount:    tokenCount,
		EmbeddingAvailable: len(req.VectorScores) > 0,
	}
	ranked := domainmemory.RankMemories(s.cfg.Ranking, memories, input, req.VectorScores)
	ranked = domainmemory.FilterInjectables(ranked)
	ranked = domainmemory.TruncateRankedMemories(ranked, s.cfg.MaxMemories, s.cfg.MaxMemoryChars)
	injectText := ""
	if s.InjectEnabled() {
		injectText = FormatForPrompt(ranked)
	}
	return RecallResult{Items: ranked, InjectText: injectText, Evidence: buildRecallEvidence(ranked)}, nil
}

func (s *Service) Write(ctx context.Context, req WriteRequest) (domainmemory.Memory, error) {
	if s == nil || s.repo == nil || !s.Enabled() {
		return domainmemory.Memory{}, nil
	}
	if err := domainmemory.ValidateScope(req.OwnerRole, req.Scope); err != nil {
		return domainmemory.Memory{}, err
	}
	piiLevel := domainmemory.ClassifyPIILevel(req.Content)
	if !domainmemory.WriteAllowedPIILevel(piiLevel, req.ConfirmHighPII) {
		return domainmemory.Memory{}, fmt.Errorf("memory write skipped: high PII content")
	}
	if !s.WriteEnabled() {
		logger.L().Debug("memory write dry-run", zap.Uint64("owner_id", req.OwnerID), zap.String("scope_type", req.Scope.Type))
		return domainmemory.Memory{}, nil
	}
	memory := domainmemory.Memory{
		OwnerRole:       req.OwnerRole,
		OwnerID:         req.OwnerID,
		Scope:           req.Scope,
		MemoryType:      req.MemoryType,
		Content:         req.Content,
		Source:          req.Source,
		Confidence:      req.Confidence,
		Importance:      req.Importance,
		Status:          domainmemory.StatusActive,
		PIILevel:        piiLevel,
		SourceSessionID: req.SourceSessionID,
		SourceMessageID: req.SourceMessageID,
		SourceRunID:     req.SourceRunID,
		CreatedBy:       req.CreatedBy,
		AllowHighPII:    req.ConfirmHighPII,
	}
	if memory.Source == "" {
		memory.Source = "agent"
	}
	if memory.Confidence <= 0 {
		memory.Confidence = 0.8
	}
	if memory.Importance <= 0 {
		memory.Importance = 0.7
	}
	saved, err := s.repo.CreateMemory(ctx, memory)
	if err != nil {
		return domainmemory.Memory{}, err
	}
	if s.embedder != nil {
		if embedErr := s.embedder.UpsertMemoryEmbedding(ctx, saved); embedErr != nil {
			logger.L().Warn("memory embedding upsert failed", zap.Uint64("memory_id", saved.ID), zap.Error(embedErr))
		}
	}
	return saved, nil
}

func (s *Service) WriteFromExtractor(ctx context.Context, input ExtractInput, scopes []domainmemory.Scope, provenance WriteRequest) error {
	if s == nil || !s.Enabled() {
		return nil
	}
	candidates, err := s.extractor.Extract(ctx, input)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		scope := candidate.Scope
		if !scopeMatchesAny(scope, scopes) {
			if len(scopes) > 0 {
				scope = scopes[0]
			}
		}
		_, writeErr := s.Write(ctx, WriteRequest{
			OwnerRole:       input.OwnerRole,
			OwnerID:         input.OwnerID,
			Scope:           scope,
			MemoryType:      candidate.MemoryType,
			Content:         candidate.Content,
			Source:          candidate.Source,
			Confidence:      candidate.Confidence,
			Importance:      candidate.Importance,
			SourceSessionID: provenance.SourceSessionID,
			SourceMessageID: provenance.SourceMessageID,
			SourceRunID:     provenance.SourceRunID,
			CreatedBy:       provenance.CreatedBy,
		})
		if writeErr != nil {
			logger.L().Debug("memory extract write skipped", zap.Error(writeErr))
		}
	}
	return nil
}

func (s *Service) Revoke(ctx context.Context, ownerRole domainmemory.OwnerRole, ownerID, id, revokedBy uint64, reason string) error {
	if s == nil || s.repo == nil || !s.Enabled() {
		return nil
	}
	if err := s.repo.RevokeMemory(ctx, ownerRole, ownerID, id, revokedBy, reason); err != nil {
		return err
	}
	if invalidator, ok := s.embedder.(MemoryEmbeddingInvalidator); ok {
		if invErr := invalidator.InvalidateMemoryEmbedding(ctx, id); invErr != nil {
			logger.L().Warn("memory embedding invalidate failed", zap.Uint64("memory_id", id), zap.Error(invErr))
		}
	}
	return nil
}

func (s *Service) Get(ctx context.Context, ownerRole domainmemory.OwnerRole, ownerID, id uint64) (domainmemory.Memory, bool, error) {
	if s == nil || s.repo == nil || !s.Enabled() || ownerID == 0 {
		return domainmemory.Memory{}, false, nil
	}
	return s.repo.GetMemory(ctx, ownerRole, ownerID, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]domainmemory.Memory, int64, error) {
	if s == nil || s.repo == nil || !s.Enabled() || filter.OwnerID == 0 {
		return nil, 0, nil
	}
	return s.repo.ListMemories(ctx, filter)
}

func (s *Service) Update(ctx context.Context, memory domainmemory.Memory) (domainmemory.Memory, error) {
	if s == nil || s.repo == nil || !s.Enabled() {
		return domainmemory.Memory{}, nil
	}
	updated, err := s.repo.UpdateMemory(ctx, memory)
	if err != nil {
		return domainmemory.Memory{}, err
	}
	if s.embedder != nil && strings.TrimSpace(updated.Content) != "" {
		if embedErr := s.embedder.UpsertMemoryEmbedding(ctx, updated); embedErr != nil {
			logger.L().Warn("memory embedding upsert failed", zap.Uint64("memory_id", updated.ID), zap.Error(embedErr))
		}
	}
	return updated, nil
}

func (s *Service) ExpireAndCleanup(ctx context.Context, revokedRetention time.Duration) (CleanupResult, error) {
	if s == nil || s.repo == nil || !s.Enabled() {
		return CleanupResult{}, nil
	}
	if revokedRetention <= 0 {
		revokedRetention = 30 * 24 * time.Hour
	}
	result, err := s.repo.ExpireAndCleanup(ctx, revokedRetention)
	if err != nil {
		return CleanupResult{}, err
	}
	if invalidator, ok := s.embedder.(MemoryEmbeddingInvalidator); ok {
		for _, memoryID := range result.InvalidatedMemoryIDs {
			if invErr := invalidator.InvalidateMemoryEmbedding(ctx, memoryID); invErr != nil {
				logger.L().Warn("memory embedding invalidate failed", zap.Uint64("memory_id", memoryID), zap.Error(invErr))
				continue
			}
			result.EmbeddingsInvalidated++
		}
	}
	return result, nil
}

func FormatForPrompt(items []domainmemory.RankedMemory) string {
	if len(items) == 0 {
		return "当前没有额外注入的长期记忆。"
	}
	var b strings.Builder
	b.WriteString("以下是与当前对话相关的长期记忆：\n")
	for i, item := range items {
		b.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, item.Memory.MemoryType, strings.TrimSpace(item.Memory.Content)))
	}
	return strings.TrimSpace(b.String())
}

func scopeMatchesAny(scope domainmemory.Scope, scopes []domainmemory.Scope) bool {
	for _, candidate := range scopes {
		if candidate.Type == scope.Type && candidate.ID == scope.ID {
			return true
		}
	}
	return false
}

func buildRecallEvidence(items []domainmemory.RankedMemory) RecallEvidence {
	if len(items) == 0 {
		return RecallEvidence{}
	}
	evidence := RecallEvidence{
		MemoryIDs: make([]uint64, 0, len(items)),
		Count:     len(items),
	}
	modeSet := make(map[string]struct{}, len(items))
	for _, item := range items {
		evidence.MemoryIDs = append(evidence.MemoryIDs, item.Memory.ID)
		evidence.Chars += len([]rune(strings.TrimSpace(item.Memory.Content)))
		mode := strings.TrimSpace(item.Reason)
		if mode == "" {
			mode = strings.TrimSpace(item.Signals.RelevanceMode)
		}
		if mode != "" {
			modeSet[mode] = struct{}{}
		}
	}
	evidence.RelevanceModes = make([]string, 0, len(modeSet))
	for mode := range modeSet {
		evidence.RelevanceModes = append(evidence.RelevanceModes, mode)
	}
	sort.Strings(evidence.RelevanceModes)
	return evidence
}

func BuildHRRecallScopes(hrID, applicationID, jobID, candidateUserID uint64) []domainmemory.Scope {
	scopes := []domainmemory.Scope{{Type: domainmemory.ScopeHR, ID: 0}}
	if applicationID > 0 {
		scopes = append(scopes, domainmemory.Scope{Type: domainmemory.ScopeApplication, ID: applicationID})
	}
	if jobID > 0 {
		scopes = append(scopes, domainmemory.Scope{Type: domainmemory.ScopeJob, ID: jobID})
	}
	if candidateUserID > 0 {
		scopes = append(scopes, domainmemory.Scope{Type: domainmemory.ScopeCandidate, ID: candidateUserID})
	}
	_ = hrID
	return scopes
}

func BuildCandidateRecallScopes(userID, applicationID, jobID uint64) []domainmemory.Scope {
	scopes := []domainmemory.Scope{{Type: domainmemory.ScopeUser, ID: userID}}
	if applicationID > 0 {
		scopes = append(scopes, domainmemory.Scope{Type: domainmemory.ScopeApplication, ID: applicationID})
	}
	if jobID > 0 {
		scopes = append(scopes, domainmemory.Scope{Type: domainmemory.ScopeJob, ID: jobID})
	}
	return scopes
}
