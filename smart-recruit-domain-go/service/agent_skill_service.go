package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/logger"
	"smart-recruit-proto/recruitment/pb"
)

type embeddingUpsertPublisher interface {
	PublishUpsertBestEffort(ctx context.Context, event EmbeddingUpsertEvent)
}

type AgentSkillService struct {
	pb.UnimplementedAgentSkillServiceServer
	repo            *repository.AgentSkillRepo
	agentConfigRepo *repository.AgentConfigRepo
	memoryRepo      *repository.MemoryRepo
	embeddings      *EmbeddingService
	eventPublisher  embeddingUpsertPublisher
}

const (
	defaultAgentSkillAgentType = "hr_recruiting_agent"
	defaultAgentSkillCategory  = "general"
	defaultAgentSkillRiskLevel = "medium"
	maxAgentSkillListItems     = 50
	maxAgentSkillListItemLen   = 128
	maxAgentSkillJSONLen       = 8192
)

func NewAgentSkillService(repo *repository.AgentSkillRepo) *AgentSkillService {
	return &AgentSkillService{repo: repo}
}

func NewAgentSkillServiceWithAgentConfigRepo(repo *repository.AgentSkillRepo, agentConfigRepo *repository.AgentConfigRepo) *AgentSkillService {
	return &AgentSkillService{repo: repo, agentConfigRepo: agentConfigRepo}
}

func (s *AgentSkillService) WithSemanticDebugDependencies(memoryRepo *repository.MemoryRepo, embeddings *EmbeddingService) *AgentSkillService {
	if s != nil {
		s.memoryRepo = memoryRepo
		s.embeddings = embeddings
	}
	return s
}

func (s *AgentSkillService) WithEmbeddingEventPublisher(publisher embeddingUpsertPublisher) *AgentSkillService {
	if s != nil {
		s.eventPublisher = publisher
	}
	return s
}

func (s *AgentSkillService) ListAgentSkills(ctx context.Context, req *pb.ListAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	page, pageSize := normalizePage(req.GetPage(), req.GetPageSize())
	skills, total, err := s.repo.ListSkills(ctx, page, pageSize, req.GetEnabledOnly(), req.GetKeyword())
	if err != nil {
		return nil, status.Error(codes.Internal, "list agent skills failed")
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: s.agentSkillsToPB(ctx, skills)}, nil
}

func (s *AgentSkillService) ListAvailableAgentSkills(ctx context.Context, req *pb.ListAvailableAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	page, pageSize := normalizePage(req.GetPage(), req.GetPageSize())
	skills, total, err := s.repo.ListAvailable(ctx, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "list available agent skills failed")
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: s.agentSkillsToPB(ctx, skills)}, nil
}

func (s *AgentSkillService) DebugSemanticRetrieval(ctx context.Context, req *pb.DebugSemanticRetrievalRequest) (*pb.DebugSemanticRetrievalResponse, error) {
	log := logger.GetRequestLogger(ctx)
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return &pb.DebugSemanticRetrievalResponse{Code: 400, Msg: "query is required"}, nil
	}
	agentType := strings.TrimSpace(req.GetAgentType())
	if agentType == "" {
		agentType = defaultAgentSkillAgentType
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	skillScores, skillSearchMeta, skillSearchErr := s.semanticDebugSkillScores(ctx, query, limit)
	embeddingAvailable := skillSearchErr == nil
	selected, err := selectAgentSkillsWithSemantic(ctx, s.repo, agentType, query, nil, nil, skillScores)
	if err != nil {
		return nil, status.Error(codes.Internal, "debug semantic skill retrieval failed")
	}
	if len(selected) > limit {
		selected = selected[:limit]
	}
	// TASK-FU-001：memories + 内部 state 写入 ctx 透传（不再使用包级 var）。
	memories, debugCtx := s.semanticDebugMemories(ctx, req, limit, &embeddingAvailable)
	fallbackReason := ""
	if !embeddingAvailable {
		fallbackReason = "embedding retrieval unavailable; showing rule-based Skill matches and scoped memory fallback ordering"
	}

	// TASK-004 → TASK-006 → TASK-FU-001：pool_confidence 暂存到 ctx（per-request）。
	memoryRankings := getDebugMemoryRankingsFromContext(debugCtx)
	poolView := computeDebugPoolConfidence(selected, memoryRankings)
	debugCtx = withDebugPoolConfidence(debugCtx, poolView)
	log.Debug("[debug_semantic_retrieval] pool confidences computed",
		zap.String("skill_pool_confidence", string(poolView.SkillPoolConfidence)),
		zap.String("memory_pool_confidence", string(poolView.MemoryPoolConfidence)),
	)

	// 使用 Skill 检索阶段捕获的元数据，避免后续 Memory 检索覆盖 LastSearchMeta。
	searchMeta := skillSearchMeta

	return &pb.DebugSemanticRetrievalResponse{
		Code:                    0,
		Msg:                     "success",
		EmbeddingAvailable:      embeddingAvailable,
		FallbackReason:          fallbackReason,
		Skills:                  semanticDebugSkillsToPB(selected),
		Memories:                memories,
		SkillPoolConfidence:     string(poolView.SkillPoolConfidence),
		MemoryPoolConfidence:    string(poolView.MemoryPoolConfidence),
		EmbeddingProvider:       searchMeta.ProviderName,
		EmbeddingModel:          searchMeta.ModelName,
		EmbeddingDim:            int32(searchMeta.VectorDim),
		CandidateCount:          int32(searchMeta.CandidateCount),
		QueryEmbeddingLatencyMs: searchMeta.LatencyMs,
	}, nil
}

// embeddingMetaSnapshot 安全地从 EmbeddingService 读取最近一次搜索的元数据。
// service 为 nil 时返回 zero value（ProviderName="unavailable"），
// 与 EmbeddingService.LastSearchMeta 行为一致。
func embeddingMetaSnapshot(svc *EmbeddingService) SearchMeta {
	if svc == nil {
		return SearchMeta{ProviderName: "unavailable"}
	}
	return svc.LastSearchMeta()
}

// debugPoolConfidenceView 是 DebugSemanticRetrieval 内部暂存 pool_confidence 的 struct。
// TASK-004 引入；TASK-006 将这两个字段透传到 proto。
// TASK-FU-001：把状态从包级 var 改为 context.Context 私有 key 传递，消除并发覆盖。
type debugPoolConfidenceView struct {
	SkillPoolConfidence  RankConfidence
	MemoryPoolConfidence RankConfidence
}

// debugContextKey 是 DebugSemanticRetrieval 内部 state 的私有 context key。
// 使用 struct{} 类型避免与其他包的 context 值冲突。
type debugContextKey struct{}

// debugPoolConfidenceKey / debugMemoryRankingsKey 是两个独立的私有 key，
// 避免 DebugSemanticRetrieval 一次请求只能存一个值的限制。
var (
	debugPoolConfidenceKey = debugContextKey{}
	debugMemoryRankingsKey = debugContextKey{}
)

// withDebugPoolConfidence 把 pool confidence 写入 ctx（私有 key）。
func withDebugPoolConfidence(ctx context.Context, v debugPoolConfidenceView) context.Context {
	return context.WithValue(ctx, debugPoolConfidenceKey, v)
}

// getDebugPoolConfidenceFromContext 从 ctx 读取 pool confidence；缺失时返回 zero value。
func getDebugPoolConfidenceFromContext(ctx context.Context) debugPoolConfidenceView {
	if v, ok := ctx.Value(debugPoolConfidenceKey).(debugPoolConfidenceView); ok {
		return v
	}
	return debugPoolConfidenceView{
		SkillPoolConfidence:  RankConfidenceNone,
		MemoryPoolConfidence: RankConfidenceNone,
	}
}

// withDebugMemoryRankings 把 memory 完整 ranking 写入 ctx。
func withDebugMemoryRankings(ctx context.Context, items []RankedMemoryItem) context.Context {
	if items == nil {
		items = []RankedMemoryItem{}
	}
	return context.WithValue(ctx, debugMemoryRankingsKey, items)
}

// getDebugMemoryRankingsFromContext 从 ctx 读取 memory 完整 ranking；缺失时返回 nil。
func getDebugMemoryRankingsFromContext(ctx context.Context) []RankedMemoryItem {
	if v, ok := ctx.Value(debugMemoryRankingsKey).([]RankedMemoryItem); ok {
		return v
	}
	return nil
}

// computeDebugPoolConfidence 从 Skill 选中列表和 Memory ranking 列表推断两个池的 confidence。
// 仅使用 FinalRankScore 字段；空 / 0 / NaN 的信号由 ComputePoolConfidence 内部过滤。
func computeDebugPoolConfidence(selected []selectedAgentSkill, memoryRankings []RankedMemoryItem) debugPoolConfidenceView {
	view := debugPoolConfidenceView{
		SkillPoolConfidence:  RankConfidenceNone,
		MemoryPoolConfidence: RankConfidenceNone,
	}
	skillSignals := make([]RankingSignals, 0, len(selected))
	for _, sk := range selected {
		if sk.Manual {
			continue
		}
		skillSignals = append(skillSignals, RankingSignals{FinalRankScore: sk.FinalRankScore})
	}
	view.SkillPoolConfidence = ComputePoolConfidence(skillSignals)

	memSignals := make([]RankingSignals, 0, len(memoryRankings))
	for _, m := range memoryRankings {
		memSignals = append(memSignals, m.Signals)
	}
	view.MemoryPoolConfidence = ComputePoolConfidence(memSignals)
	return view
}

func (s *AgentSkillService) GetAgentSkill(ctx context.Context, req *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: s.agentSkillToPB(ctx, skill)}, nil
}

func (s *AgentSkillService) semanticDebugSkillScores(ctx context.Context, query string, limit int) (map[int64]float64, SearchMeta, error) {
	if s == nil || s.embeddings == nil {
		return nil, SearchMeta{ProviderName: "unavailable"}, fmt.Errorf("embedding service not configured")
	}
	results, searchMeta, err := s.embeddings.SearchWithMeta(ctx, EmbeddingSearchInput{
		QueryText:   query,
		ObjectTypes: []string{"agent_skill"},
		Limit:       limit * 4,
	})
	if err != nil {
		return nil, searchMeta, err
	}
	scores := make(map[int64]float64, len(results))
	for _, result := range results {
		if result.Embedding.ObjectType == "agent_skill" {
			scores[int64(result.Embedding.ObjectID)] = result.Score
		}
	}
	return scores, searchMeta, nil
}

func (s *AgentSkillService) semanticDebugMemories(ctx context.Context, req *pb.DebugSemanticRetrievalRequest, limit int, embeddingAvailable *bool) ([]*pb.SemanticMemoryDebugItem, context.Context) {
	if s == nil || s.memoryRepo == nil {
		return nil, ctx
	}
	builder := (&AgentContextBuilder{memories: s.memoryRepo}).WithEmbeddingService(s.embeddings)
	input := AgentContextInput{
		HrID:           req.GetHrId(),
		JobID:          req.GetJobId(),
		ApplicationID:  req.GetApplicationId(),
		CurrentMessage: req.GetQuery(),
	}
	scopes := memoryRecallScopes(input)
	rows, err := s.memoryRepo.ListRecallCandidates(ctx, input.HrID, scopes, nil, limit*4)
	if err != nil {
		return nil, ctx
	}
	// TASK-003 重构后，rankMemories 内部走混合打分；semanticScores 仅用于在 proto 旧 reason 字段里保留"semantic vs fallback"语义。
	semanticScores := builder.semanticMemoryScores(ctx, input, rows)
	// embeddingAvailable was determined by semanticDebugSkillScores from the
	// embedding service's health, not from hit count. Don't override it here.
	ranked := builder.rankMemories(ctx, input, rows)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	rankings := builder.lastMemoryRankings
	if len(rankings) > limit {
		rankings = rankings[:limit]
	}
	// TASK-FU-001：把 memory 完整 ranking 写入 ctx 并返回新 ctx。
	// 写入后 DebugSemanticRetrieval 可从 ctx 读取 rankings 计算 pool_confidence。
	ctx = withDebugMemoryRankings(ctx, rankings)

	signalsByID := make(map[uint64]RankingSignals, len(rankings))
	for _, r := range rankings {
		signalsByID[r.Memory.ID] = r.Signals
	}

	items := make([]*pb.SemanticMemoryDebugItem, 0, len(ranked))
	for _, memory := range ranked {
		reason := "scope/importance fallback"
		if _, ok := semanticScores[memory.ID]; ok {
			reason = "semantic similarity and scope/importance"
		}
		// TASK-006: 从 lastDebugMemoryRankings 读取 breakdown signals；
		// 旧 score 字段改为 final_rank_score（SDD §3.10 兼容映射）。
		signals := signalsByID[memory.ID]
		items = append(items, &pb.SemanticMemoryDebugItem{
			Id:             memory.ID,
			ScopeType:      memory.ScopeType,
			ScopeId:        memory.ScopeID,
			MemoryType:     memory.MemoryType,
			Content:        memory.Content,
			Source:         memory.Source,
			Confidence:     memory.Confidence,
			Importance:     memory.Importance,
			Score:          signals.FinalRankScore,
			Reason:         reason,
			CreatedAt:      formatTime(memory.CreatedAt),
			VectorScore:    signals.VectorScore,
			LexicalScore:   signals.LexicalScore,
			MetadataScore:  signals.MetadataScore,
			RelevanceScore: signals.RelevanceScore,
			BusinessBoost:  signals.BusinessBoost,
			FinalRankScore: signals.FinalRankScore,
			RelevanceMode:  signals.RelevanceMode,
			PoolRank:       int32(memoryPoolRankOf(rankings, memory.ID)),
		})
	}
	return items, ctx
}

// memoryPoolRankOf 返回给定 memory ID 在 rankings 中的 1-based 排名。
// 未找到时返回 0（与 selectedAgentSkill 中 manual 池的 PoolRank=0 语义一致）。
func memoryPoolRankOf(rankings []RankedMemoryItem, id uint64) int {
	for i, r := range rankings {
		if r.Memory.ID == id {
			return i + 1
		}
	}
	return 0
}

func semanticDebugSkillsToPB(skills []selectedAgentSkill) []*pb.SemanticSkillDebugItem {
	items := make([]*pb.SemanticSkillDebugItem, 0, len(skills))
	for _, skill := range skills {
		// TASK-006: 填充新 breakdown 字段；旧 Score 字段改为 final_rank_score（SDD §3.10 兼容映射）。
		items = append(items, &pb.SemanticSkillDebugItem{
			Id:             skill.ID,
			Name:           skill.Name,
			DisplayName:    skill.DisplayName,
			Category:       skill.Category,
			Scenario:       skill.Scenario,
			Priority:       skill.Priority,
			Score:          skill.FinalRankScore,
			Reason:         skill.Reason,
			SemanticTags:   append([]string(nil), skill.SemanticTags...),
			VectorScore:    skill.VectorScore,
			LexicalScore:   skill.LexicalScore,
			MetadataScore:  skill.MetadataScore,
			RelevanceScore: skill.RelevanceScore,
			BusinessBoost:  skill.BusinessBoost,
			FinalRankScore: skill.FinalRankScore,
			RelevanceMode:  skill.RelevanceMode,
			PoolRank:       int32(skill.PoolRank),
		})
	}
	return items
}

func (s *AgentSkillService) CreateAgentSkill(ctx context.Context, req *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	log := logger.GetRequestLogger(ctx)
	name := strings.TrimSpace(req.GetName())
	log.Info("[domain][agent_skill] CreateAgentSkill started",
		zap.String("name", name),
		zap.String("display_name", req.GetDisplayName()),
		zap.String("version", req.GetVersion()))
	if !agentSkillNamePattern.MatchString(name) {
		log.Warn("[domain][agent_skill] CreateAgentSkill invalid name", zap.String("name", name))
		return nil, status.Error(codes.InvalidArgument, "invalid agent skill name")
	}
	displayName := strings.TrimSpace(req.GetDisplayName())
	if displayName == "" {
		displayName = name
	}
	triggerKeywords, err := marshalStringList(req.GetTriggerKeywords())
	if err != nil {
		log.Warn("[domain][agent_skill] CreateAgentSkill invalid trigger_keywords", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, "invalid trigger_keywords")
	}
	metadata, err := s.validateAgentSkillMetadata(ctx, agentSkillMetadataInput{
		AgentType:            req.GetAgentType(),
		Category:             req.GetCategory(),
		Scenario:             req.GetScenario(),
		Priority:             req.GetPriority(),
		RiskLevel:            req.GetRiskLevel(),
		RequiredCapabilities: req.GetRequiredCapabilities(),
		OutputSchema:         req.GetOutputSchema(),
		EvaluationCriteria:   req.GetEvaluationCriteria(),
		SemanticTags:         req.GetSemanticTags(),
	}, true)
	if err != nil {
		log.Warn("[domain][agent_skill] CreateAgentSkill validation failed", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	enabled := boolToInt32(true)
	if req.GetIsEnabledSet() {
		enabled = boolToInt32(req.GetIsEnabled())
	}
	manual := boolToInt32(true)
	if req.GetIsManualInvocableSet() {
		manual = boolToInt32(req.GetIsManualInvocable())
	}
	createdBy := optionalPositiveInt64(req.GetActorUserId())
	skill := &model.AgentSkill{
		Name:                 name,
		DisplayName:          displayName,
		Description:          strings.TrimSpace(req.GetDescription()),
		IsEnabled:            enabled,
		IsManualInvocable:    manual,
		TriggerKeywords:      triggerKeywords,
		AgentType:            metadata.AgentType,
		Category:             metadata.Category,
		Scenario:             metadata.Scenario,
		Priority:             metadata.Priority,
		RiskLevel:            metadata.RiskLevel,
		RequiredCapabilities: metadata.RequiredCapabilitiesJSON,
		OutputSchema:         metadata.OutputSchema,
		EvaluationCriteria:   metadata.EvaluationCriteriaJSON,
		SemanticTags:         metadata.SemanticTagsJSON,
		CreatedBy:            createdBy,
		UpdatedBy:            createdBy,
	}
	versionText := strings.TrimSpace(req.GetVersion())
	flowJSON := strings.TrimSpace(req.GetFlowJson())
	skillMDInput := strings.TrimSpace(req.GetSkillMd())
	if flowJSON != "" || skillMDInput != "" || versionText != "" {
		if flowJSON == "" && skillMDInput == "" {
			return nil, status.Error(codes.InvalidArgument, "flow_json or skill_md is required")
		}
		if versionText == "" {
			versionText = time.Now().Format("20060102150405")
		}
		if !agentSkillVersionRegexp.MatchString(versionText) {
			return nil, status.Error(codes.InvalidArgument, "invalid version")
		}
		skillMD, frontmatterJSON, bodyMarkdown, err := buildAgentSkillVersionContent(skill.Name, skill.Description, flowJSON, skillMDInput)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		canonicalFlow := ""
		if strings.TrimSpace(flowJSON) != "" {
			_, canonicalFlow, err = parseAgentSkillFlow(flowJSON)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		version := &model.AgentSkillVersion{
			Version:         versionText,
			FlowJSON:        canonicalFlow,
			SkillMD:         skillMD,
			FrontmatterJSON: frontmatterJSON,
			BodyMarkdown:    bodyMarkdown,
			ChangeNote:      strings.TrimSpace(req.GetChangeNote()),
			CreatedBy:       createdBy,
		}
		if err := s.repo.CreateSkillWithVersion(ctx, skill, version, req.GetActivate()); err != nil {
			if errors.Is(err, repository.ErrAgentSkillDuplicateName) {
				return nil, status.Error(codes.AlreadyExists, "agent skill name already exists")
			}
			if errors.Is(err, repository.ErrAgentSkillDuplicateVersion) {
				return nil, status.Error(codes.AlreadyExists, "agent skill version already exists")
			}
			return nil, status.Error(codes.Internal, "create agent skill failed")
		}
	} else if err := s.repo.CreateSkill(ctx, skill); err != nil {
		if errors.Is(err, repository.ErrAgentSkillDuplicateName) {
			log.Warn("[domain][agent_skill] CreateAgentSkill duplicate name", zap.String("name", name))
			return nil, status.Error(codes.AlreadyExists, "agent skill name already exists")
		}
		log.Error("[domain][agent_skill] CreateAgentSkill failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create agent skill failed")
	}
	log.Info("[domain][agent_skill] CreateAgentSkill succeeded", zap.Int64("skill_id", skill.ID))
	s.publishEmbeddingEvent(ctx, uint64(skill.ID))
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: s.agentSkillToPB(ctx, skill)}, nil
}

func (s *AgentSkillService) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	log := logger.GetRequestLogger(ctx)
	log.Info("[domain][agent_skill] UpdateAgentSkill started", zap.Int64("skill_id", req.GetId()))
	if req.GetId() <= 0 {
		log.Warn("[domain][agent_skill] UpdateAgentSkill invalid id")
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	updates := map[string]any{}
	if req.GetDisplayNameSet() {
		updates["display_name"] = strings.TrimSpace(req.GetDisplayName())
	}
	if req.GetDescriptionSet() {
		updates["description"] = strings.TrimSpace(req.GetDescription())
	}
	if req.GetTriggerKeywordsSet() {
		triggerKeywords, err := marshalStringList(req.GetTriggerKeywords())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid trigger_keywords")
		}
		updates["trigger_keywords"] = triggerKeywords
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = boolToInt32(req.GetIsEnabled())
	}
	if req.GetIsManualInvocableSet() {
		updates["is_manual_invocable"] = boolToInt32(req.GetIsManualInvocable())
	}
	if hasAgentSkillMetadataUpdate(req) {
		existing, err := s.repo.GetSkillByID(ctx, req.GetId())
		if err != nil {
			return nil, status.Error(codes.NotFound, "agent skill not found")
		}
		agentType := req.GetAgentType()
		if !req.GetAgentTypeSet() {
			agentType = existing.AgentType
		}
		requiredCapabilities := req.GetRequiredCapabilities()
		requiredCapabilitiesSet := req.GetRequiredCapabilitiesSet()
		if req.GetAgentTypeSet() && !requiredCapabilitiesSet {
			requiredCapabilities = unmarshalStringList(existing.RequiredCapabilities)
			requiredCapabilitiesSet = true
		}
		metadata, err := s.validateAgentSkillMetadata(ctx, agentSkillMetadataInput{
			AgentType:               agentType,
			AgentTypeSet:            req.GetAgentTypeSet(),
			Category:                req.GetCategory(),
			CategorySet:             req.GetCategorySet(),
			Scenario:                req.GetScenario(),
			ScenarioSet:             req.GetScenarioSet(),
			Priority:                req.GetPriority(),
			PrioritySet:             req.GetPrioritySet(),
			RiskLevel:               req.GetRiskLevel(),
			RiskLevelSet:            req.GetRiskLevelSet(),
			RequiredCapabilities:    requiredCapabilities,
			RequiredCapabilitiesSet: requiredCapabilitiesSet,
			OutputSchema:            req.GetOutputSchema(),
			OutputSchemaSet:         req.GetOutputSchemaSet(),
			EvaluationCriteria:      req.GetEvaluationCriteria(),
			EvaluationCriteriaSet:   req.GetEvaluationCriteriaSet(),
			SemanticTags:            req.GetSemanticTags(),
			SemanticTagsSet:         req.GetSemanticTagsSet(),
		}, false)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if req.GetAgentTypeSet() {
			updates["agent_type"] = metadata.AgentType
		}
		if req.GetCategorySet() {
			updates["category"] = metadata.Category
		}
		if req.GetScenarioSet() {
			updates["scenario"] = metadata.Scenario
		}
		if req.GetPrioritySet() {
			updates["priority"] = metadata.Priority
		}
		if req.GetRiskLevelSet() {
			updates["risk_level"] = metadata.RiskLevel
		}
		if req.GetRequiredCapabilitiesSet() {
			updates["required_capabilities"] = metadata.RequiredCapabilitiesJSON
		}
		if req.GetOutputSchemaSet() {
			updates["output_schema"] = metadata.OutputSchema
		}
		if req.GetEvaluationCriteriaSet() {
			updates["evaluation_criteria"] = metadata.EvaluationCriteriaJSON
		}
		if req.GetSemanticTagsSet() {
			updates["semantic_tags"] = metadata.SemanticTagsJSON
		}
	}
	if actor := optionalPositiveInt64(req.GetActorUserId()); actor != nil {
		updates["updated_by"] = actor
	}
	if len(updates) > 0 {
		if err := s.repo.UpdateSkillPartial(ctx, req.GetId(), updates); err != nil {
			return nil, status.Error(codes.Internal, "update agent skill failed")
		}
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	if len(updates) > 0 {
		if req.GetIsEnabledSet() {
			if err := s.syncSkillEmbeddingsAfterStatusChange(ctx, req.GetId(), req.GetIsEnabled()); err != nil {
				return nil, status.Error(codes.Internal, "sync agent skill embeddings failed")
			}
		} else if skill.IsEnabled == 1 {
			s.publishEmbeddingEvent(ctx, uint64(skill.ID))
		}
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: s.agentSkillToPB(ctx, skill)}, nil
}

func (s *AgentSkillService) syncSkillEmbeddingsAfterStatusChange(ctx context.Context, skillID int64, enabled bool) error {
	if enabled {
		s.publishEmbeddingEvent(ctx, uint64(skillID))
		return nil
	}
	if s.embeddings == nil {
		return nil
	}
	_, err := s.embeddings.InvalidateObjectEmbeddings(ctx, "agent_skill", uint64(skillID))
	return err
}

func (s *AgentSkillService) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error) {
	if req.GetSkillId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id is required")
	}
	if !agentSkillVersionRegexp.MatchString(strings.TrimSpace(req.GetVersion())) {
		return nil, status.Error(codes.InvalidArgument, "invalid version")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	skillMDInput := strings.TrimSpace(req.GetSkillMd())
	flowJSON := strings.TrimSpace(req.GetFlowJson())
	if flowJSON == "" && skillMDInput == "" {
		return nil, status.Error(codes.InvalidArgument, "flow_json or skill_md is required")
	}
	skillMD, frontmatterJSON, bodyMarkdown, err := buildAgentSkillVersionContent(skill.Name, skill.Description, flowJSON, skillMDInput)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	canonicalFlow := ""
	if flowJSON != "" {
		_, canonicalFlow, err = parseAgentSkillFlow(flowJSON)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	version := &model.AgentSkillVersion{
		SkillID:         skill.ID,
		Version:         strings.TrimSpace(req.GetVersion()),
		FlowJSON:        canonicalFlow,
		SkillMD:         skillMD,
		FrontmatterJSON: frontmatterJSON,
		BodyMarkdown:    bodyMarkdown,
		ChangeNote:      strings.TrimSpace(req.GetChangeNote()),
		CreatedBy:       optionalPositiveInt64(req.GetActorUserId()),
	}
	actor := optionalPositiveInt64(req.GetActorUserId())
	if err := s.repo.CreateVersion(ctx, version, req.GetActivate(), actor); err != nil {
		if errors.Is(err, repository.ErrAgentSkillDuplicateVersion) {
			return nil, status.Error(codes.AlreadyExists, "agent skill version already exists")
		}
		return nil, status.Error(codes.Internal, "create agent skill version failed")
	}
	return &pb.AgentSkillVersionResponse{Code: 0, Msg: "success", Version: agentSkillVersionToPB(version)}, nil
}

func (s *AgentSkillService) ListAgentSkillVersions(ctx context.Context, req *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error) {
	if req.GetSkillId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id is required")
	}
	versions, err := s.repo.ListVersions(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.Internal, "list agent skill versions failed")
	}
	list := make([]*pb.AgentSkillVersionInfo, 0, len(versions))
	for i := range versions {
		list = append(list, agentSkillVersionToPB(&versions[i]))
	}
	return &pb.ListAgentSkillVersionsResponse{Code: 0, Msg: "success", List: list}, nil
}

func (s *AgentSkillService) ActivateAgentSkillVersion(ctx context.Context, req *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error) {
	log := logger.GetRequestLogger(ctx)
	log.Info("[domain][agent_skill] ActivateAgentSkillVersion started",
		zap.Int64("skill_id", req.GetSkillId()),
		zap.Int64("version_id", req.GetVersionId()))
	if req.GetSkillId() <= 0 || req.GetVersionId() <= 0 {
		log.Warn("[domain][agent_skill] ActivateAgentSkillVersion invalid params")
		return nil, status.Error(codes.InvalidArgument, "skill_id and version_id are required")
	}
	if err := s.repo.ActivateVersion(ctx, req.GetSkillId(), req.GetVersionId(), optionalPositiveInt64(req.GetActorUserId())); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("[domain][agent_skill] ActivateAgentSkillVersion version not found")
			return nil, status.Error(codes.NotFound, "agent skill version not found")
		}
		log.Error("[domain][agent_skill] ActivateAgentSkillVersion failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "activate agent skill version failed")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetSkillId())
	if err != nil {
		log.Error("[domain][agent_skill] ActivateAgentSkillVersion get skill failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get agent skill failed")
	}
	log.Info("[domain][agent_skill] ActivateAgentSkillVersion succeeded",
		zap.Int64("skill_id", req.GetSkillId()),
		zap.Int64("version_id", req.GetVersionId()))
	s.publishEmbeddingEvent(ctx, uint64(skill.ID))
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: s.agentSkillToPB(ctx, skill)}, nil
}

func (s *AgentSkillService) UpdateAgentSkillStatus(ctx context.Context, req *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error) {
	log := logger.GetRequestLogger(ctx)
	log.Info("[domain][agent_skill] UpdateAgentSkillStatus started",
		zap.Int64("skill_id", req.GetId()),
		zap.Bool("is_enabled", req.GetIsEnabled()))
	if req.GetId() <= 0 {
		log.Warn("[domain][agent_skill] UpdateAgentSkillStatus invalid id")
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	updates := map[string]any{"is_enabled": boolToInt32(req.GetIsEnabled())}
	if actor := optionalPositiveInt64(req.GetActorUserId()); actor != nil {
		updates["updated_by"] = actor
	}
	if err := s.repo.UpdateSkillPartial(ctx, req.GetId(), updates); err != nil {
		log.Error("[domain][agent_skill] UpdateAgentSkillStatus repo failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "update agent skill status failed")
	}
	if err := s.syncSkillEmbeddingsAfterStatusChange(ctx, req.GetId(), req.GetIsEnabled()); err != nil {
		log.Error("[domain][agent_skill] UpdateAgentSkillStatus embedding sync failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "sync agent skill embeddings failed")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetId())
	if err != nil {
		log.Error("[domain][agent_skill] UpdateAgentSkillStatus get skill failed", zap.Error(err))
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	log.Info("[domain][agent_skill] UpdateAgentSkillStatus succeeded", zap.Int64("skill_id", req.GetId()))
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: s.agentSkillToPB(ctx, skill)}, nil
}

func (s *AgentSkillService) publishEmbeddingEvent(ctx context.Context, skillID uint64) {
	if s.eventPublisher == nil {
		return
	}

	skill, err := s.repo.GetSkillByID(ctx, int64(skillID))
	if err != nil || skill == nil || skill.IsEnabled == 0 {
		return
	}

	semanticTags := unmarshalStringList(skill.SemanticTags)

	bodyMarkdown := ""
	if skill.CurrentVersionID != nil {
		version, verErr := s.repo.GetVersionByID(ctx, *skill.CurrentVersionID)
		if verErr == nil && version != nil {
			bodyMarkdown = version.BodyMarkdown
		}
	}

	text := BuildAgentSkillEmbeddingTextWithMetadata(AgentSkillEmbeddingTextInput{
		SkillName:          skill.Name,
		Description:        skill.Description,
		BodyMarkdown:       bodyMarkdown,
		Category:           skill.Category,
		Scenario:           skill.Scenario,
		RiskLevel:          skill.RiskLevel,
		TriggerKeywords:    unmarshalStringList(skill.TriggerKeywords),
		SemanticTags:       semanticTags,
		EvaluationCriteria: skill.EvaluationCriteria,
		OutputSchema:       skill.OutputSchema,
	})
	if strings.TrimSpace(text) == "" {
		return
	}

	hash := sha256.Sum256([]byte(text))
	textHash := hex.EncodeToString(hash[:])

	defaultModelID := int64(0)

	s.eventPublisher.PublishUpsertBestEffort(ctx, EmbeddingUpsertEvent{
		ObjectType: "agent_skill",
		ObjectID:   skillID,
		ModelID:    defaultModelID,
		Text:       text,
		TextHash:   textHash,
	})
}

func (s *AgentSkillService) PreviewAgentSkill(ctx context.Context, req *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error) {
	log := logger.GetRequestLogger(ctx)
	name := strings.TrimSpace(req.GetName())
	log.Info("[domain][agent_skill] PreviewAgentSkill started", zap.String("name", name))
	if !agentSkillNamePattern.MatchString(name) {
		log.Warn("[domain][agent_skill] PreviewAgentSkill invalid name")
		return nil, status.Error(codes.InvalidArgument, "invalid agent skill name")
	}
	skillMD, frontmatterJSON, bodyMarkdown, err := GenerateAgentSkillMarkdown(name, strings.TrimSpace(req.GetDescription()), req.GetFlowJson())
	if err != nil {
		log.Warn("[domain][agent_skill] PreviewAgentSkill generation failed", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	log.Info("[domain][agent_skill] PreviewAgentSkill succeeded",
		zap.String("name", name),
		zap.Int("skill_md_length", len(skillMD)))
	return &pb.PreviewAgentSkillResponse{Code: 0, Msg: "success", SkillMd: skillMD, FrontmatterJson: frontmatterJSON, BodyMarkdown: bodyMarkdown}, nil
}

func normalizePage(page, pageSize int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func agentSkillsToPB(skills []model.AgentSkill) []*pb.AgentSkillInfo {
	list := make([]*pb.AgentSkillInfo, 0, len(skills))
	for i := range skills {
		list = append(list, agentSkillToPB(&skills[i]))
	}
	return list
}

func (s *AgentSkillService) agentSkillsToPB(ctx context.Context, skills []model.AgentSkill) []*pb.AgentSkillInfo {
	list := make([]*pb.AgentSkillInfo, 0, len(skills))
	for i := range skills {
		list = append(list, s.agentSkillToPB(ctx, &skills[i]))
	}
	return list
}

func agentSkillToPB(skill *model.AgentSkill) *pb.AgentSkillInfo {
	if skill == nil {
		return nil
	}
	requiredCapabilities := unmarshalStringList(skill.RequiredCapabilities)
	unavailable, warnings := evaluateAgentSkillCapabilityAvailability(skill.AgentType, requiredCapabilities)
	return buildAgentSkillPB(skill, requiredCapabilities, unavailable, warnings)
}

func (s *AgentSkillService) agentSkillToPB(ctx context.Context, skill *model.AgentSkill) *pb.AgentSkillInfo {
	if skill == nil {
		return nil
	}
	requiredCapabilities := unmarshalStringList(skill.RequiredCapabilities)
	unavailable, warnings := s.evaluateAgentSkillCapabilityAvailability(ctx, skill.AgentType, requiredCapabilities)
	return buildAgentSkillPB(skill, requiredCapabilities, unavailable, warnings)
}

func buildAgentSkillPB(skill *model.AgentSkill, requiredCapabilities, unavailable, warnings []string) *pb.AgentSkillInfo {
	var currentVersionID int64
	if skill.CurrentVersionID != nil {
		currentVersionID = *skill.CurrentVersionID
	}
	keywords := []string{}
	_ = json.Unmarshal([]byte(skill.TriggerKeywords), &keywords)
	evaluationCriteria := unmarshalStringList(skill.EvaluationCriteria)
	semanticTags := unmarshalStringList(skill.SemanticTags)
	return &pb.AgentSkillInfo{
		Id:                      skill.ID,
		Name:                    skill.Name,
		DisplayName:             skill.DisplayName,
		Description:             skill.Description,
		CurrentVersionId:        currentVersionID,
		IsEnabled:               skill.IsEnabled == 1,
		IsManualInvocable:       skill.IsManualInvocable == 1,
		TriggerKeywords:         keywords,
		CreatedAt:               formatTime(skill.CreatedAt),
		UpdatedAt:               formatTime(skill.UpdatedAt),
		AgentType:               skill.AgentType,
		Category:                skill.Category,
		Scenario:                skill.Scenario,
		Priority:                skill.Priority,
		RiskLevel:               skill.RiskLevel,
		RequiredCapabilities:    requiredCapabilities,
		OutputSchema:            skill.OutputSchema,
		EvaluationCriteria:      evaluationCriteria,
		SemanticTags:            semanticTags,
		UnavailableCapabilities: unavailable,
		ValidationWarnings:      warnings,
	}
}

func agentSkillVersionToPB(version *model.AgentSkillVersion) *pb.AgentSkillVersionInfo {
	if version == nil {
		return nil
	}
	return &pb.AgentSkillVersionInfo{
		Id:              version.ID,
		SkillId:         version.SkillID,
		Version:         version.Version,
		FlowJson:        version.FlowJSON,
		SkillMd:         version.SkillMD,
		FrontmatterJson: version.FrontmatterJSON,
		BodyMarkdown:    version.BodyMarkdown,
		ChangeNote:      version.ChangeNote,
		CreatedAt:       formatTime(version.CreatedAt),
	}
}

func buildAgentSkillVersionContent(name, description, flowJSON, skillMDInput string) (skillMD, frontmatterJSON, bodyMarkdown string, err error) {
	if strings.TrimSpace(skillMDInput) == "" {
		return GenerateAgentSkillMarkdown(name, description, flowJSON)
	}
	doc, err := ParseAgentSkillMarkdown(skillMDInput)
	if err != nil {
		return "", "", "", err
	}
	if frontmatterString(doc.Frontmatter, "name") != strings.TrimSpace(name) {
		return "", "", "", fmt.Errorf("skill_md frontmatter name must match agent skill name")
	}
	if frontmatterString(doc.Frontmatter, "description") != strings.TrimSpace(description) {
		return "", "", "", fmt.Errorf("skill_md frontmatter description must match agent skill description")
	}
	frontmatterBytes, err := json.Marshal(doc.Frontmatter)
	if err != nil {
		return "", "", "", err
	}
	return strings.TrimSpace(skillMDInput) + "\n", string(frontmatterBytes), doc.Body, nil
}

func marshalStringList(values []string) (string, error) {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			clean = append(clean, value)
		}
	}
	b, err := json.Marshal(clean)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalStringList(raw string) []string {
	values := []string{}
	if strings.TrimSpace(raw) == "" {
		return values
	}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

type agentSkillMetadataInput struct {
	AgentType               string
	AgentTypeSet            bool
	Category                string
	CategorySet             bool
	Scenario                string
	ScenarioSet             bool
	Priority                int32
	PrioritySet             bool
	RiskLevel               string
	RiskLevelSet            bool
	RequiredCapabilities    []string
	RequiredCapabilitiesSet bool
	OutputSchema            string
	OutputSchemaSet         bool
	EvaluationCriteria      []string
	EvaluationCriteriaSet   bool
	SemanticTags            []string
	SemanticTagsSet         bool
}

type agentSkillMetadataValues struct {
	AgentType                string
	Category                 string
	Scenario                 string
	Priority                 int32
	RiskLevel                string
	RequiredCapabilitiesJSON string
	OutputSchema             string
	EvaluationCriteriaJSON   string
	SemanticTagsJSON         string
}

func (s *AgentSkillService) validateAgentSkillMetadata(ctx context.Context, input agentSkillMetadataInput, create bool) (agentSkillMetadataValues, error) {
	values := agentSkillMetadataValues{}
	effectiveAgentType := normalizeDefaultString(input.AgentType, defaultAgentSkillAgentType)
	if create || input.AgentTypeSet {
		values.AgentType = effectiveAgentType
		if err := validateAgentSkillToken("agent_type", values.AgentType, 64, false); err != nil {
			return values, err
		}
	}
	if create || input.CategorySet {
		values.Category = normalizeDefaultString(input.Category, defaultAgentSkillCategory)
		if err := validateAgentSkillToken("category", values.Category, 64, false); err != nil {
			return values, err
		}
	}
	if create || input.ScenarioSet {
		values.Scenario = strings.TrimSpace(input.Scenario)
		if err := validateAgentSkillText("scenario", values.Scenario, 128); err != nil {
			return values, err
		}
	}
	if create || input.PrioritySet {
		values.Priority = input.Priority
		if values.Priority < -1000 || values.Priority > 1000 {
			return values, fmt.Errorf("priority must be between -1000 and 1000")
		}
	}
	if create || input.RiskLevelSet {
		values.RiskLevel = strings.ToLower(normalizeDefaultString(input.RiskLevel, defaultAgentSkillRiskLevel))
		switch values.RiskLevel {
		case "low", "medium", "high", "critical":
		default:
			return values, fmt.Errorf("risk_level must be one of low, medium, high, critical")
		}
	}
	if create || input.RequiredCapabilitiesSet {
		capabilities, err := normalizeAgentSkillRequiredCapabilities(input.RequiredCapabilities, effectiveAgentType)
		if err != nil {
			return values, err
		}
		if unavailable, _ := s.evaluateAgentSkillCapabilityAvailability(ctx, effectiveAgentType, capabilities); len(unavailable) > 0 {
			return values, fmt.Errorf("unavailable required capabilities: %s", strings.Join(unavailable, ", "))
		}
		values.RequiredCapabilitiesJSON, err = marshalStringList(capabilities)
		if err != nil {
			return values, fmt.Errorf("invalid required_capabilities")
		}
	}
	if create || input.OutputSchemaSet {
		outputSchema, err := normalizeAgentSkillJSON("output_schema", input.OutputSchema)
		if err != nil {
			return values, err
		}
		values.OutputSchema = outputSchema
	}
	if create || input.EvaluationCriteriaSet {
		criteria, err := normalizeAgentSkillStringList("evaluation_criteria", input.EvaluationCriteria, maxAgentSkillListItems, 512)
		if err != nil {
			return values, err
		}
		values.EvaluationCriteriaJSON, err = marshalStringList(criteria)
		if err != nil {
			return values, fmt.Errorf("invalid evaluation_criteria")
		}
	}
	if create || input.SemanticTagsSet {
		tags, err := normalizeAgentSkillStringList("semantic_tags", input.SemanticTags, maxAgentSkillListItems, maxAgentSkillListItemLen)
		if err != nil {
			return values, err
		}
		values.SemanticTagsJSON, err = marshalStringList(tags)
		if err != nil {
			return values, fmt.Errorf("invalid semantic_tags")
		}
	}
	return values, nil
}

func hasAgentSkillMetadataUpdate(req *pb.UpdateAgentSkillRequest) bool {
	return req.GetAgentTypeSet() || req.GetCategorySet() || req.GetScenarioSet() ||
		req.GetPrioritySet() || req.GetRiskLevelSet() || req.GetRequiredCapabilitiesSet() ||
		req.GetOutputSchemaSet() || req.GetEvaluationCriteriaSet() || req.GetSemanticTagsSet()
}

func normalizeDefaultString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func validateAgentSkillToken(field, value string, maxLen int, allowEmpty bool) error {
	if strings.TrimSpace(value) == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("%s is required", field)
	}
	if len(value) > maxLen {
		return fmt.Errorf("%s is too long", field)
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return fmt.Errorf("%s contains invalid characters", field)
	}
	return nil
}

func validateAgentSkillText(field, value string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s is too long", field)
	}
	return nil
}

func normalizeAgentSkillStringList(field string, values []string, maxItems, maxLen int) ([]string, error) {
	clean := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > maxLen {
			return nil, fmt.Errorf("%s contains an item that is too long", field)
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		clean = append(clean, value)
		if len(clean) > maxItems {
			return nil, fmt.Errorf("%s has too many items", field)
		}
	}
	return clean, nil
}

func normalizeAgentSkillJSON(field, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if len(raw) > maxAgentSkillJSONLen {
		return "", fmt.Errorf("%s is too large", field)
	}
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("%s must be valid JSON", field)
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("%s must be valid JSON", field)
	}
	return string(canonical), nil
}

func normalizeAgentSkillRequiredCapabilities(values []string, agentType string) ([]string, error) {
	clean, err := normalizeAgentSkillStringList("required_capabilities", values, maxAgentSkillListItems, 256)
	if err != nil {
		return nil, err
	}
	normalized := make([]string, 0, len(clean))
	seen := map[string]bool{}
	for _, item := range clean {
		source, key, err := parseAgentSkillCapabilityRef(item)
		if err != nil {
			return nil, err
		}
		if source == "" {
			source = "builtin"
		}
		ref := source + ":" + key
		if seen[ref] {
			continue
		}
		seen[ref] = true
		normalized = append(normalized, ref)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func parseAgentSkillCapabilityRef(value string) (source, key string, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", fmt.Errorf("required_capabilities contains empty item")
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 2 {
		source = strings.ToLower(strings.TrimSpace(parts[0]))
		key = strings.TrimSpace(parts[1])
	} else {
		key = value
	}
	if source != "" && source != "builtin" && source != "mcp" && source != "skill" {
		return "", "", fmt.Errorf("required_capabilities contains unsupported source %q", source)
	}
	if err := validateAgentSkillToken("required_capabilities", key, 256, false); err != nil {
		return "", "", err
	}
	return source, key, nil
}

func evaluateAgentSkillCapabilityAvailability(agentType string, refs []string) ([]string, []string) {
	if strings.TrimSpace(agentType) == "" {
		agentType = defaultAgentSkillAgentType
	}
	builtin := builtinAgentCapabilitySet(agentType)
	unavailable := []string{}
	warnings := []string{}
	for _, ref := range refs {
		source, key, err := parseAgentSkillCapabilityRef(ref)
		if err != nil {
			unavailable = append(unavailable, ref)
			continue
		}
		if source == "" {
			source = "builtin"
		}
		switch source {
		case "builtin":
			if !builtin[key] {
				unavailable = append(unavailable, source+":"+key)
			}
		case "mcp", "skill":
			warnings = append(warnings, source+":"+key+" requires runtime registry validation")
		}
	}
	sort.Strings(unavailable)
	sort.Strings(warnings)
	return unavailable, warnings
}

func (s *AgentSkillService) evaluateAgentSkillCapabilityAvailability(ctx context.Context, agentType string, refs []string) ([]string, []string) {
	if strings.TrimSpace(agentType) == "" {
		agentType = defaultAgentSkillAgentType
	}
	configured := s.configuredAgentCapabilities(ctx, agentType)
	if len(configured) == 0 {
		return evaluateAgentSkillCapabilityAvailability(agentType, refs)
	}
	unavailable := []string{}
	for _, ref := range refs {
		source, key, err := parseAgentSkillCapabilityRef(ref)
		if err != nil {
			unavailable = append(unavailable, ref)
			continue
		}
		if source == "" {
			source = "builtin"
		}
		normalized := source + ":" + key
		if !configured[normalized] {
			unavailable = append(unavailable, normalized)
		}
	}
	sort.Strings(unavailable)
	return unavailable, nil
}

func (s *AgentSkillService) configuredAgentCapabilities(ctx context.Context, agentType string) map[string]bool {
	if s == nil || s.agentConfigRepo == nil {
		return nil
	}
	bindings, err := s.agentConfigRepo.ListEnabledCapabilityBindingsByAgentType(ctx, agentType)
	if err != nil || len(bindings) == 0 {
		return nil
	}
	configured := make(map[string]bool, len(bindings))
	for _, binding := range bindings {
		source := strings.ToLower(strings.TrimSpace(binding.CapabilitySource))
		key := strings.TrimSpace(binding.CapabilityKey)
		if source == "" || key == "" {
			continue
		}
		configured[source+":"+key] = true
	}
	return configured
}

func builtinAgentCapabilitySet(agentType string) map[string]bool {
	builtin := map[string]bool{}
	for _, cap := range builtinCapabilities(agentType) {
		if cap != nil && cap.GetSource() == "builtin" {
			builtin[cap.GetKey()] = true
		}
	}
	return builtin
}

func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func optionalPositiveInt64(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}
