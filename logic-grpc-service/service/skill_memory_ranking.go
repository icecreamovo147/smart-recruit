package service

import (
	"math"
	"sort"
	"strings"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

// ------------------------------------------------------------------
// 权重与阈值默认值（与 SDD §3.3 严格对齐）
// TASK-FU-004：原 const 改为 var；启动时由 LoadRankingConfig 从 config.Ranking 段覆盖。
// 测试可通过 ResetRankingConfigForTest() 恢复 hardcode 默认。
// ------------------------------------------------------------------

// rankingDefaults 是 11 个权重 / 阈值的 hardcode 默认值。
// 与 const 旧实现的数值完全一致；保持向后兼容。
var rankingDefaults = struct {
	WeightVector     float64
	WeightLexical    float64
	WeightMetadata   float64
	BusinessBoostMax float64
	PriorityNorm     float64
	BoostAlpha       float64
	BoostBeta        float64
	BoostGamma       float64
	RelevanceGate    float64
	GapHigh          float64
	GapMedium        float64
}{
	WeightVector:     0.6,
	WeightLexical:    0.3,
	WeightMetadata:   0.1,
	BusinessBoostMax: 1.5,
	PriorityNorm:     50.0,
	BoostAlpha:       0.2,
	BoostBeta:        0.2,
	BoostGamma:       0.1,
	RelevanceGate:    0.15,
	GapHigh:          0.10,
	GapMedium:        0.03,
}

var (
	// rankingConfig TASK-FU-007 收敛：11 个权重 / 阈值集中到一个 struct，
	// 替代 TASK-FU-004 引入的 11 个包级 var。
	// 字段语义与 SDD §3.3 完全对齐。
	// 测试通过 ResetRankingConfigForTest() 恢复 hardcode 默认。
	rankingConfig = struct {
		WeightVector     float64
		WeightLexical    float64
		WeightMetadata   float64
		BusinessBoostMax float64
		PriorityNorm     float64
		BoostAlpha       float64
		BoostBeta        float64
		BoostGamma       float64
		RelevanceGate    float64
		GapHigh          float64
		GapMedium        float64
	}{
		WeightVector:     rankingDefaults.WeightVector,
		WeightLexical:    rankingDefaults.WeightLexical,
		WeightMetadata:   rankingDefaults.WeightMetadata,
		BusinessBoostMax: rankingDefaults.BusinessBoostMax,
		PriorityNorm:     rankingDefaults.PriorityNorm,
		BoostAlpha:       rankingDefaults.BoostAlpha,
		BoostBeta:        rankingDefaults.BoostBeta,
		BoostGamma:       rankingDefaults.BoostGamma,
		RelevanceGate:    rankingDefaults.RelevanceGate,
		GapHigh:          rankingDefaults.GapHigh,
		GapMedium:        rankingDefaults.GapMedium,
	}
)

const rankingFloatEpsilon = 1e-9

// rankingFloatAlmostEqual 在稳定排序 tie-breaking 时使用的浮点近似比较。
func rankingFloatAlmostEqual(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return false
	}
	return math.Abs(a-b) <= rankingFloatEpsilon
}

// RankingConfig TASK-FU-004 引入：LoadRankingConfig 的入参类型。
// 形参而非 config.Ranking 直传，避免 service 包依赖 config 包。
// 启动入口：main.go 负责把 config.Ranking 转换为 RankingConfig 后传入。
type RankingConfig struct {
	WeightVector     float64
	WeightLexical    float64
	WeightMetadata   float64
	BusinessBoostMax float64
	PriorityNorm     float64
	BoostAlpha       float64
	BoostBeta        float64
	BoostGamma       float64
	RelevanceGate    float64
	GapHigh          float64
	GapMedium        float64
}

// LoadRankingConfig TASK-FU-004 引入：在启动时把 RankingConfig 段的非空值覆盖到 package-level var。
// 空值 / 0 字段保持 hardcode 默认；越界值 clamp 到合法范围。
//
// 启动入口：main.go 在 config.Load() 之后调用本函数（需先转 config.Ranking → RankingConfig）。
// 测试入口：LoadRankingConfig(nil) 或 LoadRankingConfig(RankingConfig{}) 走默认；测试也可用 LoadRankingConfig(cfg) 设置特定值。
func LoadRankingConfig(cfg RankingConfig) {
	applyRankingField(&rankingConfig.WeightVector, cfg.WeightVector, 0, 1, "weight_vector")
	applyRankingField(&rankingConfig.WeightLexical, cfg.WeightLexical, 0, 1, "weight_lexical")
	applyRankingField(&rankingConfig.WeightMetadata, cfg.WeightMetadata, 0, 1, "weight_metadata")
	applyRankingField(&rankingConfig.BusinessBoostMax, cfg.BusinessBoostMax, 1.0, rankingDefaults.BusinessBoostMax, "business_boost_max")
	applyRankingField(&rankingConfig.PriorityNorm, cfg.PriorityNorm, 0.0001, 10000, "priority_norm")
	applyRankingField(&rankingConfig.BoostAlpha, cfg.BoostAlpha, 0, 1, "boost_alpha")
	applyRankingField(&rankingConfig.BoostBeta, cfg.BoostBeta, 0, 1, "boost_beta")
	applyRankingField(&rankingConfig.BoostGamma, cfg.BoostGamma, 0, 1, "boost_gamma")
	applyRankingField(&rankingConfig.RelevanceGate, cfg.RelevanceGate, 0, 1, "relevance_gate")
	applyRankingField(&rankingConfig.GapHigh, cfg.GapHigh, 0, 1, "gap_high")
	applyRankingField(&rankingConfig.GapMedium, cfg.GapMedium, 0, 1, "gap_medium")
}

// applyRankingField 把 value（非 0）写入 target；越界 clamp 到 [min, max]。
// 0 表示"未设置"，保持 target 原值（hardcode 默认）。
// TASK-FU-005：越界时输出 zap.Warn 日志（含 key / value / clamped 字段），
// 便于运维 / 开发者定位 env 配置错误。
func applyRankingField(target *float64, value, min, max float64, key string) {
	if value == 0 {
		return
	}
	if value < min {
		logger.L().Warn("[ranking] env value out of range, clamped",
			zap.String("key", key),
			zap.Float64("value", value),
			zap.Float64("clamped", min),
			zap.Float64("min", min),
			zap.Float64("max", max),
		)
		*target = min
		return
	}
	if value > max {
		logger.L().Warn("[ranking] env value out of range, clamped",
			zap.String("key", key),
			zap.Float64("value", value),
			zap.Float64("clamped", max),
			zap.Float64("min", min),
			zap.Float64("max", max),
		)
		*target = max
		return
	}
	*target = value
}

// ResetRankingConfigForTest TASK-FU-004 引入：测试 helper，恢复所有 var 到 hardcode 默认。
// 测试应在 setUp / tearDown 中调用，避免 var 跨测试污染。
func ResetRankingConfigForTest() {
	rankingConfig.WeightVector = rankingDefaults.WeightVector
	rankingConfig.WeightLexical = rankingDefaults.WeightLexical
	rankingConfig.WeightMetadata = rankingDefaults.WeightMetadata
	rankingConfig.BusinessBoostMax = rankingDefaults.BusinessBoostMax
	rankingConfig.PriorityNorm = rankingDefaults.PriorityNorm
	rankingConfig.BoostAlpha = rankingDefaults.BoostAlpha
	rankingConfig.BoostBeta = rankingDefaults.BoostBeta
	rankingConfig.BoostGamma = rankingDefaults.BoostGamma
	rankingConfig.RelevanceGate = rankingDefaults.RelevanceGate
	rankingConfig.GapHigh = rankingDefaults.GapHigh
	rankingConfig.GapMedium = rankingDefaults.GapMedium
}

// SnapshotRankingConfig TASK-FU-008 引入：返回当前生效的 rankingConfig struct 副本。
// 用于 cmd/ranking-show 等只读场景；不暴露可变引用。
func SnapshotRankingConfig() RankingConfig {
	return RankingConfig{
		WeightVector:     rankingConfig.WeightVector,
		WeightLexical:    rankingConfig.WeightLexical,
		WeightMetadata:   rankingConfig.WeightMetadata,
		BusinessBoostMax: rankingConfig.BusinessBoostMax,
		PriorityNorm:     rankingConfig.PriorityNorm,
		BoostAlpha:       rankingConfig.BoostAlpha,
		BoostBeta:        rankingConfig.BoostBeta,
		BoostGamma:       rankingConfig.BoostGamma,
		RelevanceGate:    rankingConfig.RelevanceGate,
		GapHigh:          rankingConfig.GapHigh,
		GapMedium:        rankingConfig.GapMedium,
	}
}

// ------------------------------------------------------------------
// 数据结构
// ------------------------------------------------------------------

// RankingSignals 描述一次召回的完整打分分解。
//
// 所有信号都归一化到 [0, 1]；final_rank_score ∈ [0, rankingConfig.BusinessBoostMax]。
type RankingSignals struct {
	VectorScore    float64  `json:"vector_score"`
	LexicalScore   float64  `json:"lexical_score"`
	MetadataScore  float64  `json:"metadata_score"`
	RelevanceScore float64  `json:"relevance_score"`
	BusinessBoost  float64  `json:"business_boost"`
	FinalRankScore float64  `json:"final_rank_score"`
	RelevanceMode  string   `json:"relevance_mode"` // "vector_lexical_metadata" | "lexical_metadata"
	Reasons        []string `json:"reasons,omitempty"`
}

// RankConfidence 表示一个召回池（Skill 池 / Memory 池）整体的可信度。
type RankConfidence string

const (
	RankConfidenceHigh   RankConfidence = "high"
	RankConfidenceMedium RankConfidence = "medium"
	RankConfidenceLow    RankConfidence = "low"
	RankConfidenceNone   RankConfidence = "none"
)

// ------------------------------------------------------------------
// 纯函数工具：相关性与 boost
// ------------------------------------------------------------------

// clamp01 将数值限制在 [0, 1] 区间；NaN / Inf 一律返回 0。
func clamp01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// computeRelevanceScore 根据三种信号加权得到相关性分数，输出 ∈ [0, 1]。
//
// 公式：relevance = w_v * vector + w_l * lexical + w_m * metadata
// 权重和为 1.0；任何 NaN / Inf 信号被视作 0。
func computeRelevanceScore(vector, lexical, metadata float64) float64 {
	v := safeSignal(vector)
	l := safeSignal(lexical)
	m := safeSignal(metadata)
	score := rankingConfig.WeightVector*v + rankingConfig.WeightLexical*l + rankingConfig.WeightMetadata*m
	return clamp01(score)
}

// safeSignal 把 NaN / Inf 归零，把越界值裁剪到 [0, 1]。
func safeSignal(v float64) float64 {
	return clamp01(v)
}

// computeBusinessBoost 根据 priority / importance / confidence 计算业务加权系数。
//
// isSkill=true 时只使用 priority_normalized；isSkill=false 时使用 importance / confidence（已在调用方 gating）。
// 输出 ∈ [1.0, rankingConfig.BusinessBoostMax]；priority / importance / confidence 的 NaN / Inf 一律视作 0。
func computeBusinessBoost(priority, importanceSignal, confidenceSignal float64, isSkill bool) float64 {
	priorityNorm := 0.0
	if isSkill {
		priorityNorm = clamp01(priority / rankingConfig.PriorityNorm)
	}
	imp := clamp01(importanceSignal)
	conf := clamp01(confidenceSignal)

	boost := 1.0 +
		rankingConfig.BoostAlpha*priorityNorm +
		rankingConfig.BoostBeta*imp +
		rankingConfig.BoostGamma*conf

	if boost < 1.0 {
		boost = 1.0
	}
	if boost > rankingConfig.BusinessBoostMax {
		boost = rankingConfig.BusinessBoostMax
	}
	return boost
}

// computeFinalRankScore 终值 = relevance * boost。
// relevance ∈ [0,1]，boost ∈ [1.0, rankingConfig.BusinessBoostMax]；输出 ∈ [0, rankingConfig.BusinessBoostMax]。
func computeFinalRankScore(relevance, boost float64) float64 {
	r := clamp01(relevance)
	b := boost
	if math.IsNaN(b) || math.IsInf(b, 0) {
		b = 1.0
	}
	if b < 1.0 {
		b = 1.0
	}
	if b > rankingConfig.BusinessBoostMax {
		b = rankingConfig.BusinessBoostMax
	}
	score := r * b
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0
	}
	if score < 0 {
		return 0
	}
	return score
}

// memoryImportanceSignal 给出 Memory 在相关性 gating 之下的 importance 信号。
// 当 relevance < rankingConfig.RelevanceGate 时返回 0；否则返回 clamp01(importance)。
func memoryImportanceSignal(relevance, importance float64) float64 {
	if relevance < rankingConfig.RelevanceGate {
		return 0
	}
	return clamp01(importance)
}

// memoryConfidenceSignal 给出 Memory 在相关性 gating 之下的 confidence 信号。
// 当 relevance < rankingConfig.RelevanceGate 时返回 0；否则返回 clamp01(confidence)。
func memoryConfidenceSignal(relevance, confidence float64) float64 {
	if relevance < rankingConfig.RelevanceGate {
		return 0
	}
	return clamp01(confidence)
}

// ------------------------------------------------------------------
// 池置信度
// ------------------------------------------------------------------

// ------------------------------------------------------------------
// Skill 池打分：scoreSkillRankingSignals / RankSkillCandidates
// ------------------------------------------------------------------

// skillLexicalScoreMax 用于把 `scoreAgentSkillMatch` 返回的整数归一化到 [0, 1] 的分母。
//
// scoreAgentSkillMatch 的 token 权重为 meta×2 + content×1，每个 token 最多 3 分；
// 归一化策略：分母 = max(1, nTokens * 3)，再 clip 到 1.0。
const skillLexicalScoreMax = 3.0

// scoreSkillRankingSignals 给出单个 Skill 候选的混合打分 breakdown。
//
// vector_score 来自 semanticScores[id]（cosine ∈ [0, 1]），embedding 不可用 / 越界时取 0；
// lexical_score 来自 `scoreAgentSkillMatch(question, skill)` 归一化（meta×2 + content×1）；
// metadata_score 来自 category / scenario / risk_level / semantic_tags / trigger_keywords 的 token 命中率；
// relevance_score / business_boost / final_rank_score 由本文件顶部纯函数计算。
func scoreSkillRankingSignals(
	question string,
	skill repository.AgentSkillRuntimeRecord,
	semanticScore float64,
	embeddingAvailable bool,
) RankingSignals {
	vectorScore := 0.0
	if embeddingAvailable {
		vectorScore = clamp01(semanticScore)
	}

	rawRuleScore := scoreAgentSkillMatch(question, skill)
	if rawRuleScore < 0 {
		rawRuleScore = 0
	}
	nTokens := len(tokenizeAgentSkillText(question))
	denom := float64(nTokens) * skillLexicalScoreMax
	if denom < 1.0 {
		denom = 1.0
	}
	lexicalScore := float64(rawRuleScore) / denom
	if lexicalScore > 1.0 {
		lexicalScore = 1.0
	}

	metadataScore := normalizeSkillMetadataScore(question, skill)

	mode := "lexical_metadata"
	if vectorScore > 0 {
		mode = "vector_lexical_metadata"
	}

	relevance := computeRelevanceScore(vectorScore, lexicalScore, metadataScore)
	boost := computeBusinessBoost(float64(skill.Priority), 0, 0, true)
	final := computeFinalRankScore(relevance, boost)

	return RankingSignals{
		VectorScore:    vectorScore,
		LexicalScore:   lexicalScore,
		MetadataScore:  metadataScore,
		RelevanceScore: relevance,
		BusinessBoost:  boost,
		FinalRankScore: final,
		RelevanceMode:  mode,
	}
}

// normalizeSkillMetadataScore 把 category / scenario / risk_level / semantic_tags / trigger_keywords
// 的 token 命中数量除以非空字段数量，得到 ∈ [0, 1] 的 metadata_score。
func normalizeSkillMetadataScore(question string, skill repository.AgentSkillRuntimeRecord) float64 {
	tokens := tokenizeAgentSkillText(question)
	if len(tokens) == 0 {
		return 0
	}

	candidates := []string{
		skill.Category,
		skill.Scenario,
		skill.RiskLevel,
		strings.Join(agentSkillSemanticTags(skill.SemanticTags), " "),
		strings.Join(agentSkillTriggerKeywords(skill.TriggerKeywords), " "),
	}

	matchedFields := 0
	totalFields := 0
	for _, raw := range candidates {
		field := strings.TrimSpace(raw)
		if field == "" {
			continue
		}
		totalFields++
		fieldTokens := tokenizeAgentSkillText(field)
		for token := range tokens {
			if _, ok := fieldTokens[token]; ok {
				matchedFields++
				break
			}
		}
	}
	if totalFields == 0 {
		return 0
	}
	score := float64(matchedFields) / float64(totalFields)
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// rankSkillCandidatesWithReason 返回候选的 reason 字符串，供日志 / trace 使用。
func rankSkillCandidatesWithReason(signals RankingSignals, hasVector bool) string {
	if hasVector {
		return "semantic + lexical + metadata hybrid score, priority boost"
	}
	return "lexical + metadata hybrid score (embedding unavailable), priority boost"
}

// RankSkillCandidates 对一组 Skill 候选进行混合打分、自动排序、过滤 zero-score，
// 返回按 `final_rank_score desc, priority desc, id asc` 排序的 `selectedAgentSkill` 列表。
//
// 过滤规则：rawRuleScore <= 0 的候选直接丢弃（保留旧实现中"语义单独命中不入选"的行为）。
// selectedAgentSkill.Score（int 旧字段）= int(round(FinalRankScore * 100))，仅用于兼容旧调试 API。
func RankSkillCandidates(
	question string,
	candidates []repository.AgentSkillRuntimeRecord,
	semanticScores map[int64]float64,
	embeddingAvailable bool,
) []selectedAgentSkill {
	result := make([]selectedAgentSkill, 0, len(candidates))
	for _, skill := range candidates {
		rawRuleScore := scoreAgentSkillMatch(question, skill)
		if rawRuleScore <= 0 {
			continue
		}

		var semScore float64
		if semanticScores != nil {
			semScore = semanticScores[skill.ID]
		}
		signals := scoreSkillRankingSignals(question, skill, semScore, embeddingAvailable)
		if signals.FinalRankScore <= 0 {
			continue
		}

		hasVector := signals.VectorScore > 0
		reason := rankSkillCandidatesWithReason(signals, hasVector)
		s := toSelectedAgentSkill(skill, false, int(math.Round(signals.FinalRankScore*100)), reason)
		s.VectorScore = signals.VectorScore
		s.LexicalScore = signals.LexicalScore
		s.MetadataScore = signals.MetadataScore
		s.RelevanceScore = signals.RelevanceScore
		s.BusinessBoost = signals.BusinessBoost
		s.FinalRankScore = signals.FinalRankScore
		s.RelevanceMode = signals.RelevanceMode
		result = append(result, s)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if rankingFloatAlmostEqual(result[i].FinalRankScore, result[j].FinalRankScore) {
			if result[i].Priority == result[j].Priority {
				return result[i].ID < result[j].ID
			}
			return result[i].Priority > result[j].Priority
		}
		return result[i].FinalRankScore > result[j].FinalRankScore
	})
	return result
}

// ------------------------------------------------------------------
// Memory 池打分：scoreMemoryRankingSignals / RankMemoryCandidates
// ------------------------------------------------------------------

// RankedMemoryItem 携带 RankingSignals 的 Memory 候选，供 debug API 与内部排序复用。
type RankedMemoryItem struct {
	Memory  model.AIMemory
	Signals RankingSignals
}

// scoreMemoryRankingSignals 给出单个 Memory 候选的混合打分 breakdown。
//
// vector_score：embedding cosine ∈ [0, 1]；embedding 不可用 / 越界 → 0；
// lexical_score：keywordMemoryScore(query, content) / nTokens，再 clip 到 1.0；
// metadata_score：scope 命中（application=1.0 / job=0.7 / hr=0.4 / miss=0）；
// importance / confidence 在相关性 gating 阈值之上才参与 boost。
func scoreMemoryRankingSignals(
	memory model.AIMemory,
	input AgentContextInput,
	semanticScore float64,
	embeddingAvailable bool,
	queryTokenCount int,
) RankingSignals {
	vectorScore := 0.0
	if embeddingAvailable {
		vectorScore = clamp01(semanticScore)
	}

	rawLexScore := keywordMemoryScore(input.CurrentMessage, memory.Content)
	if rawLexScore < 0 {
		rawLexScore = 0
	}
	denom := float64(queryTokenCount)
	if denom < 1.0 {
		denom = 1.0
	}
	lexicalScore := rawLexScore / denom
	if lexicalScore > 1.0 {
		lexicalScore = 1.0
	}

	metadataScore := normalizeMemoryMetadataScore(memory, input)
	relevance := computeRelevanceScore(vectorScore, lexicalScore, metadataScore)
	importanceSignal := memoryImportanceSignal(relevance, memory.Importance)
	confidenceSignal := memoryConfidenceSignal(relevance, memory.Confidence)
	boost := computeBusinessBoost(0, importanceSignal, confidenceSignal, false)
	final := computeFinalRankScore(relevance, boost)

	mode := "lexical_metadata"
	if vectorScore > 0 {
		mode = "vector_lexical_metadata"
	}

	return RankingSignals{
		VectorScore:    vectorScore,
		LexicalScore:   lexicalScore,
		MetadataScore:  metadataScore,
		RelevanceScore: relevance,
		BusinessBoost:  boost,
		FinalRankScore: final,
		RelevanceMode:  mode,
	}
}

// normalizeMemoryMetadataScore 把 scope 命中映射到 ∈ [0, 1]：
//
//	application 精确命中 → 1.0
//	job 精确命中         → 0.7
//	hr（永远参与）       → 0.4
//	miss                → 0.0
func normalizeMemoryMetadataScore(memory model.AIMemory, input AgentContextInput) float64 {
	switch {
	case memory.ScopeType == "application" && input.ApplicationID > 0 && memory.ScopeID == uint64(input.ApplicationID):
		return 1.0
	case memory.ScopeType == "job" && input.JobID > 0 && memory.ScopeID == uint64(input.JobID):
		return 0.7
	case memory.ScopeType == "hr":
		return 0.4
	}
	return 0.0
}

// RankMemoryCandidates 对一组 Memory 候选进行混合打分、自动排序、过滤 zero-score，
// 返回按 `final_rank_score desc, importance desc, id asc` 排序的 `RankedMemoryItem` 列表。
//
// 过滤规则：final_rank_score == 0 的候选直接丢弃。
//
// 复用既有 `keywordMemoryScore` / `memoryBaseRecallScore` / `semanticMemoryScores` 的 helper 行为：
// keywordMemoryScore 提供 raw lexical 命中数；semanticScores 提供 vector cosine。
// 旧 `memoryBaseRecallScore` 的 Importance*20 + Confidence*10 + scope_bonus 行为由
// `normalizeMemoryMetadataScore` + gating 后的 `importanceSignal` / `confidenceSignal` 完整覆盖。
func RankMemoryCandidates(
	query string,
	memories []model.AIMemory,
	input AgentContextInput,
	semanticScores map[uint64]float64,
	embeddingAvailable bool,
) []RankedMemoryItem {
	queryTokens := tokenizeAgentSkillText(query)
	items := make([]RankedMemoryItem, 0, len(memories))
	for _, m := range memories {
		var semScore float64
		if semanticScores != nil {
			semScore = semanticScores[m.ID]
		}
		signals := scoreMemoryRankingSignals(m, input, semScore, embeddingAvailable, len(queryTokens))
		if signals.FinalRankScore <= 0 {
			continue
		}
		items = append(items, RankedMemoryItem{Memory: m, Signals: signals})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if rankingFloatAlmostEqual(items[i].Signals.FinalRankScore, items[j].Signals.FinalRankScore) {
			if items[i].Memory.Importance == items[j].Memory.Importance {
				return items[i].Memory.ID < items[j].Memory.ID
			}
			return items[i].Memory.Importance > items[j].Memory.Importance
		}
		return items[i].Signals.FinalRankScore > items[j].Signals.FinalRankScore
	})
	return items
}

// memoryRankScoreCompatMap 返回旧 debug API 兼容用的整数 score（= round(final * 100)）。
func memoryRankScoreCompatMap(items []RankedMemoryItem) map[uint64]int {
	m := make(map[uint64]int, len(items))
	for _, it := range items {
		m[it.Memory.ID] = int(math.Round(it.Signals.FinalRankScore * 100))
	}
	return m
}

// 规则：
//   - 空池：none
//   - 1 个候选：top1 >= rankingConfig.RelevanceGate → low（gap 无法定义）；否则 none
//   - ≥ 2 个候选：
//     gap = top1 - top2
//     gap >= rankingConfig.GapHigh             → high
//     rankingConfig.GapMedium <= gap < rankingConfig.GapHigh → medium
//     gap < rankingConfig.GapMedium            → low
//     若 top1 < rankingConfig.RelevanceGate    → 强制 low
func ComputePoolConfidence(ranked []RankingSignals) RankConfidence {
	if len(ranked) == 0 {
		return RankConfidenceNone
	}

	sorted := make([]RankingSignals, 0, len(ranked))
	for _, r := range ranked {
		if math.IsNaN(r.FinalRankScore) || math.IsInf(r.FinalRankScore, 0) {
			continue
		}
		sorted = append(sorted, r)
	}
	if len(sorted) == 0 {
		return RankConfidenceNone
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].FinalRankScore > sorted[j].FinalRankScore
	})

	top1 := sorted[0].FinalRankScore

	if len(sorted) == 1 {
		if top1 <= 0 {
			return RankConfidenceNone
		}
		if top1 < rankingConfig.RelevanceGate {
			return RankConfidenceNone
		}
		return RankConfidenceLow
	}

	top2 := sorted[1].FinalRankScore
	if top2 < 0 {
		top2 = 0
	}
	if top2 > top1 {
		top2 = top1
	}

	if top1 <= 0 {
		return RankConfidenceNone
	}
	if top1 < rankingConfig.RelevanceGate {
		return RankConfidenceLow
	}

	gap := top1 - top2
	if gap < 0 {
		gap = 0
	}
	switch {
	case gap >= rankingConfig.GapHigh:
		return RankConfidenceHigh
	case gap >= rankingConfig.GapMedium:
		return RankConfidenceMedium
	default:
		return RankConfidenceLow
	}
}
