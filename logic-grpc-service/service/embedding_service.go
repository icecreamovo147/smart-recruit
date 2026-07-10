package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/crypto"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

const (
	EmbeddingStatusReady       = "ready"
	EmbeddingStatusUnavailable = "unavailable"
	EmbeddingStatusInactive    = "inactive"

	DefaultEmbeddingModel = "unavailable"
)

var (
	ErrEmbeddingProviderUnavailable = errors.New("embedding provider unavailable")
	ErrEmbeddingVectorInvalid       = errors.New("embedding vector invalid")
)

type EmbeddingProvider interface {
	EmbedText(ctx context.Context, text string) (EmbeddingVector, error)
	// Name returns a stable, human-readable identifier for the provider
	// (e.g. "bailian", "openai", "unavailable"). Used by debug endpoints to
	// surface which provider actually served the request.
	Name() string
}

type EmbeddingVector struct {
	Model  string
	Vector []float64
}

type UnavailableEmbeddingProvider struct{}

func (UnavailableEmbeddingProvider) EmbedText(context.Context, string) (EmbeddingVector, error) {
	return EmbeddingVector{Model: DefaultEmbeddingModel}, ErrEmbeddingProviderUnavailable
}

// Name implements EmbeddingProvider.
func (UnavailableEmbeddingProvider) Name() string { return "unavailable" }

type EmbeddingService struct {
	repo     *repository.AIEmbeddingRepo
	provider atomic.Pointer[EmbeddingProvider]
	policy   AgentRuntimePolicy
	factory  *EmbeddingProviderFactory
	encKey   crypto.EncryptionKey

	// lastSearchMeta 记录最近一次 Search / SearchObjects 调用的元数据。
	// 由 Search / SearchObjects 在执行末尾写入；由 LastSearchMeta() 读取。
	// 用于 debug 端点展示当前生效的 provider / model / dim / candidate count / latency。
	// 字段在并发场景下不严格 thread-safe（同一 service 多 goroutine 调用）；
	// debug 端点单 goroutine 调用足够。
	lastSearchMeta atomic.Pointer[SearchMeta]
}

// SearchMeta 记录一次 embedding 搜索的元数据。
// 由 EmbeddingService 在 Search / SearchObjects 末尾写入。
type SearchMeta struct {
	ProviderName   string
	ModelName      string
	VectorDim      int
	CandidateCount int
	LatencyMs      int64
}

func NewEmbeddingService(repo *repository.AIEmbeddingRepo, factory *EmbeddingProviderFactory, encKey crypto.EncryptionKey) *EmbeddingService {
	s := &EmbeddingService{
		repo:    repo,
		factory: factory,
		encKey:  encKey,
		policy:  DefaultAgentRuntimePolicy(),
	}
	initial := s.buildProvider(context.Background())
	s.provider.Store(&initial)
	return s
}

func (s *EmbeddingService) WithRuntimePolicy(policy AgentRuntimePolicy) *EmbeddingService {
	if s != nil {
		s.policy = policy.withDefaults()
	}
	return s
}

func (s *EmbeddingService) currentProvider() EmbeddingProvider {
	if s == nil {
		return UnavailableEmbeddingProvider{}
	}
	p := s.provider.Load()
	if p == nil {
		return UnavailableEmbeddingProvider{}
	}
	return *p
}

func (s *EmbeddingService) buildProvider(ctx context.Context) EmbeddingProvider {
	if s == nil || s.factory == nil {
		return UnavailableEmbeddingProvider{}
	}
	return s.factory.Build(ctx, s.encKey)
}

func (s *EmbeddingService) RebuildProvider(ctx context.Context) {
	if s == nil {
		return
	}
	next := s.buildProvider(ctx)
	s.provider.Store(&next)
	if _, ok := next.(UnavailableEmbeddingProvider); ok {
		logger.L().Warn("[logic][embedding] provider rebuilt: unavailable (check provider/model config)")
		return
	}
	logger.L().Info("[logic][embedding] provider rebuilt successfully")
}

// InvalidateObjectEmbeddings marks all embeddings for an object inactive.
func (s *EmbeddingService) InvalidateObjectEmbeddings(ctx context.Context, objectType string, objectID uint64) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	count, err := s.repo.MarkStatusByObject(ctx, objectType, objectID, EmbeddingStatusInactive)
	if err != nil {
		return 0, err
	}
	logger.L().Info("embedding object invalidated",
		zap.String("object_type", objectType),
		zap.Uint64("object_id", objectID),
		zap.Int64("rows_affected", count),
	)
	return count, nil
}

// SetProviderForTest injects a custom provider, bypassing the factory.
// Intended for unit tests that need a deterministic fake provider.
func (s *EmbeddingService) SetProviderForTest(p EmbeddingProvider) {
	if s == nil {
		return
	}
	if p == nil {
		p = UnavailableEmbeddingProvider{}
	}
	s.provider.Store(&p)
}

type EmbedObjectInput struct {
	ObjectType   string
	ObjectID     uint64
	ScopeType    string
	ScopeID      uint64
	Text         string
	MetadataJSON string
	Vector       []float64
	Model        string
}

type EmbeddingSearchInput struct {
	QueryText   string
	QueryVector []float64
	Model       string
	ObjectTypes []string
	ScopeType   string
	ScopeID     uint64
	Limit       int
}

type EmbeddingObjectSearchInput struct {
	QueryText   string
	QueryVector []float64
	Model       string
	ObjectType  string
	ObjectIDs   []uint64
	Limit       int
}

type EmbeddingSearchResult struct {
	Embedding model.AIEmbedding
	Score     float64
}

func (s *EmbeddingService) EmbedObject(ctx context.Context, input EmbedObjectInput) (*model.AIEmbedding, error) {
	started := time.Now()
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][embedding] EmbedObject started",
		zap.String("object_type", input.ObjectType),
		zap.Uint64("object_id", input.ObjectID),
		zap.String("scope_type", input.ScopeType),
		zap.Uint64("scope_id", input.ScopeID))
	if s == nil || s.repo == nil {
		log.Error("[logic][embedding] repository not configured")
		return nil, fmt.Errorf("embedding repository is not configured")
	}
	objectType := strings.TrimSpace(input.ObjectType)
	if objectType == "" || input.ObjectID == 0 {
		log.Warn("[logic][embedding] missing object type or id")
		return nil, fmt.Errorf("embedding object type and id are required")
	}
	textHash := hashEmbeddingText(input.Text)
	row := &model.AIEmbedding{
		ObjectType:   objectType,
		ObjectID:     input.ObjectID,
		ScopeType:    strings.TrimSpace(input.ScopeType),
		ScopeID:      input.ScopeID,
		TextHash:     textHash,
		MetadataJSON: jsonStringPtr(normalizeEmbeddingMetadata(input.MetadataJSON)),
	}

	vector := append([]float64(nil), input.Vector...)
	modelName := strings.TrimSpace(input.Model)
	if len(vector) == 0 {
		result, err := s.currentProvider().EmbedText(ctx, input.Text)
		if err != nil {
			log.Warn("[logic][embedding] provider call failed, falling back to unavailable",
				zap.String("object_type", objectType),
				zap.Uint64("object_id", input.ObjectID),
				zap.String("text_hash", textHash),
				zap.Error(err))
			row.EmbeddingModel = normalizeEmbeddingModel(result.Model)
			row.Status = EmbeddingStatusUnavailable
			row.LastError = err.Error()
			if upsertErr := s.repo.Upsert(ctx, row); upsertErr != nil {
				return nil, upsertErr
			}
			return row, nil
		}
		vector = append([]float64(nil), result.Vector...)
		modelName = normalizeEmbeddingModel(result.Model)
	}
	if err := validateEmbeddingVector(vector); err != nil {
		log.Error("[logic][embedding] invalid embedding vector", zap.Error(err))
		return nil, err
	}
	vectorJSON, err := marshalEmbeddingVector(vector)
	if err != nil {
		return nil, err
	}
	row.EmbeddingModel = normalizeEmbeddingModel(modelName)
	row.EmbeddingDim = len(vector)
	row.VectorJSON = jsonStringPtr(vectorJSON)
	row.Status = EmbeddingStatusReady
	row.LastError = ""
	if err := s.repo.Upsert(ctx, row); err != nil {
		return nil, err
	}
	log.Info("[logic][embedding] EmbedObject succeeded",
		zap.String("object_type", objectType),
		zap.Uint64("object_id", input.ObjectID),
		zap.String("text_hash", textHash),
		zap.String("model", row.EmbeddingModel),
		zap.Int("dim", row.EmbeddingDim),
		zap.String("status", row.Status),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	return row, nil
}

func (s *EmbeddingService) Search(ctx context.Context, input EmbeddingSearchInput) ([]EmbeddingSearchResult, error) {
	results, meta, err := s.searchWithMeta(ctx, input)
	s.persistSearchMeta(meta)
	return results, err
}

// SearchWithMeta runs Search and returns request-local metadata for the same call.
func (s *EmbeddingService) SearchWithMeta(ctx context.Context, input EmbeddingSearchInput) ([]EmbeddingSearchResult, SearchMeta, error) {
	results, meta, err := s.searchWithMeta(ctx, input)
	s.persistSearchMeta(meta)
	return results, meta, err
}

func (s *EmbeddingService) searchWithMeta(ctx context.Context, input EmbeddingSearchInput) ([]EmbeddingSearchResult, SearchMeta, error) {
	log := logger.GetRequestLogger(ctx)
	if s == nil || s.repo == nil {
		return nil, SearchMeta{ProviderName: "unavailable"}, fmt.Errorf("embedding repository is not configured")
	}
	policy := s.policy.withDefaults()
	if !policy.SemanticRetrieval {
		log.Info("[logic][embedding] semantic retrieval disabled",
			zap.String("event", "agent.semantic_retrieval.search"),
			zap.String("status", "disabled"),
			zap.Strings("object_types", input.ObjectTypes))
		return nil, s.buildSearchMeta(s.currentProvider().Name(), "", 0, 0, 0), ErrAgentCapabilityDisabled
	}
	started := time.Now()
	ctx, cancel := contextWithPolicyTimeout(ctx, policy.SemanticRetrievalTimeout)
	defer cancel()

	querySource := "text"
	if len(input.QueryVector) > 0 {
		querySource = "vector"
	}
	log.Info("[logic][embedding] Search started",
		zap.String("query_source", querySource),
		zap.String("model", input.Model),
		zap.Strings("object_types", input.ObjectTypes),
		zap.Int("limit", input.Limit))

	queryVector, modelName, providerName, err := s.resolveQueryVector(ctx, input.QueryText, input.QueryVector, input.Model)
	if err != nil {
		meta := s.buildSearchMeta(providerName, modelName, 0, 0, time.Since(started).Milliseconds())
		s.logSearchFinished("agent.semantic_retrieval.search", started, "fallback", input.ObjectTypes, 0, err)
		return nil, meta, err
	}
	log.Info("[logic][embedding] query vector resolved",
		zap.String("model", modelName),
		zap.Int("vector_dim", len(queryVector)))

	rows, err := s.repo.ListCandidates(ctx, repository.AIEmbeddingQuery{
		ObjectTypes:    input.ObjectTypes,
		ScopeType:      input.ScopeType,
		ScopeID:        input.ScopeID,
		EmbeddingModel: modelName,
		Status:         EmbeddingStatusReady,
		Limit:          candidateLimit(input.Limit),
	})
	if err != nil {
		meta := s.buildSearchMeta(providerName, modelName, len(queryVector), 0, time.Since(started).Milliseconds())
		s.logSearchFinished("agent.semantic_retrieval.search", started, "failed", input.ObjectTypes, 0, err)
		return nil, meta, err
	}
	log.Info("[logic][embedding] candidates loaded",
		zap.Int("candidate_count", len(rows)))

	results := rankEmbeddingRows(queryVector, rows)
	limit := input.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	finalResults := results[:limit]
	log.Info("[logic][embedding] ranking finished",
		zap.Int("candidate_count", len(rows)),
		zap.Int("returned_count", limit))
	if len(finalResults) > 0 && len(finalResults) <= 10 {
		for i, r := range finalResults {
			log.Debug("[logic][embedding] ranked result",
				zap.Int("rank", i+1),
				zap.Uint64("object_id", r.Embedding.ObjectID),
				zap.String("object_type", r.Embedding.ObjectType),
				zap.Float64("score", r.Score))
		}
	}
	meta := s.buildSearchMeta(providerName, modelName, len(queryVector), len(finalResults), time.Since(started).Milliseconds())
	s.logSearchFinished("agent.semantic_retrieval.search", started, "succeeded", input.ObjectTypes, limit, nil)
	return finalResults, meta, nil
}

// buildSearchMeta constructs request-local metadata for one search call.
func (s *EmbeddingService) buildSearchMeta(providerName, modelName string, vectorDim, candidateCount int, latencyMs int64) SearchMeta {
	if providerName == "" {
		providerName = "unavailable"
	}
	return SearchMeta{
		ProviderName:   providerName,
		ModelName:      modelName,
		VectorDim:      vectorDim,
		CandidateCount: candidateCount,
		LatencyMs:      latencyMs,
	}
}

func (s *EmbeddingService) persistSearchMeta(meta SearchMeta) {
	if s == nil {
		return
	}
	stored := meta
	s.lastSearchMeta.Store(&stored)
}

// recordSearchMeta 写入最近一次搜索的元数据。
// modelName / vectorDim / candidateCount / latencyMs 由 Search 末尾填入；
// 即便 err 也会写入（带 candidateCount=0 / latencyMs 部分时长），
// 让 debug 端点能区分"未搜索"与"搜索失败"两种状态。
func (s *EmbeddingService) recordSearchMeta(providerName, modelName string, vectorDim, candidateCount int, latencyMs int64) {
	s.persistSearchMeta(s.buildSearchMeta(providerName, modelName, vectorDim, candidateCount, latencyMs))
}

// LastSearchMeta 返回最近一次 Search / SearchObjects 调用的元数据。
// 未调用过任何 search 时返回 zero value（ProviderName="unavailable"）。
// 用于 debug 端点展示当前生效的 provider / model / dim / candidate count / latency。
func (s *EmbeddingService) LastSearchMeta() SearchMeta {
	if s == nil {
		return SearchMeta{ProviderName: "unavailable"}
	}
	if m := s.lastSearchMeta.Load(); m != nil {
		return *m
	}
	return SearchMeta{ProviderName: s.currentProvider().Name()}
}

func (s *EmbeddingService) SearchObjects(ctx context.Context, input EmbeddingObjectSearchInput) ([]EmbeddingSearchResult, error) {
	log := logger.GetRequestLogger(ctx)
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("embedding repository is not configured")
	}
	policy := s.policy.withDefaults()
	if !policy.SemanticRetrieval {
		log.Info("[logic][embedding] semantic retrieval disabled",
			zap.String("event", "agent.semantic_retrieval.search_objects"),
			zap.String("status", "disabled"),
			zap.String("object_type", input.ObjectType))
		return nil, ErrAgentCapabilityDisabled
	}
	objectType := strings.TrimSpace(input.ObjectType)
	if objectType == "" || len(input.ObjectIDs) == 0 {
		return nil, nil
	}
	started := time.Now()
	ctx, cancel := contextWithPolicyTimeout(ctx, policy.SemanticRetrievalTimeout)
	defer cancel()

	querySource := "text"
	if len(input.QueryVector) > 0 {
		querySource = "vector"
	}
	log.Info("[logic][embedding] SearchObjects started",
		zap.String("object_type", objectType),
		zap.Int("object_id_count", len(input.ObjectIDs)),
		zap.String("query_source", querySource),
		zap.Int("limit", input.Limit))

	queryVector, modelName, providerName, err := s.resolveQueryVector(ctx, input.QueryText, input.QueryVector, input.Model)
	if err != nil {
		s.recordSearchMeta(providerName, modelName, 0, 0, time.Since(started).Milliseconds())
		s.logSearchFinished("agent.semantic_retrieval.search_objects", started, "fallback", []string{objectType}, 0, err)
		return nil, err
	}
	rows, err := s.repo.ListByObjectIDs(ctx, objectType, input.ObjectIDs, modelName, EmbeddingStatusReady)
	if err != nil {
		s.recordSearchMeta(providerName, modelName, len(queryVector), 0, time.Since(started).Milliseconds())
		s.logSearchFinished("agent.semantic_retrieval.search_objects", started, "failed", []string{objectType}, 0, err)
		return nil, err
	}
	results := rankEmbeddingRows(queryVector, rows)
	limit := input.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	finalResults := results[:limit]
	log.Info("[logic][embedding] SearchObjects finished",
		zap.Int("candidate_count", len(rows)),
		zap.Int("returned_count", len(finalResults)),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	s.recordSearchMeta(providerName, modelName, len(queryVector), len(finalResults), time.Since(started).Milliseconds())
	s.logSearchFinished("agent.semantic_retrieval.search_objects", started, "succeeded", []string{objectType}, limit, nil)
	return finalResults, nil
}

func (s *EmbeddingService) logSearchFinished(event string, started time.Time, status string, objectTypes []string, resultCount int, err error) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.String("status", status),
		zap.Strings("object_types", objectTypes),
		zap.Int("result_count", resultCount),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.L().Warn("semantic retrieval finished", fields...)
		return
	}
	logger.L().Info("semantic retrieval finished", fields...)
}

func hashEmbeddingText(text string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

func normalizeEmbeddingModel(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return DefaultEmbeddingModel
	}
	return modelName
}

func normalizeEmbeddingMetadata(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	if json.Valid([]byte(raw)) {
		return raw
	}
	return "{}"
}

func validateEmbeddingVector(vector []float64) error {
	if len(vector) == 0 {
		return ErrEmbeddingVectorInvalid
	}
	for _, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ErrEmbeddingVectorInvalid
		}
	}
	return nil
}

func (s *EmbeddingService) resolveQueryVector(ctx context.Context, text string, vector []float64, modelName string) ([]float64, string, string, error) {
	queryVector := append([]float64(nil), vector...)
	normalizedModel := normalizeEmbeddingModel(modelName)
	provider := s.currentProvider()
	providerName := provider.Name()
	if len(queryVector) == 0 {
		result, err := provider.EmbedText(ctx, text)
		if err != nil {
			return nil, "", providerName, err
		}
		queryVector = append([]float64(nil), result.Vector...)
		normalizedModel = normalizeEmbeddingModel(result.Model)
	}
	if err := validateEmbeddingVector(queryVector); err != nil {
		return nil, "", providerName, err
	}
	return queryVector, normalizedModel, providerName, nil
}

func rankEmbeddingRows(queryVector []float64, rows []model.AIEmbedding) []EmbeddingSearchResult {
	results := make([]EmbeddingSearchResult, 0, len(rows))
	resultIndexes := make(map[string]int, len(rows))
	for _, row := range rows {
		vector, err := unmarshalEmbeddingVector(row.VectorJSON)
		if err != nil || len(vector) != len(queryVector) {
			continue
		}
		score := cosineSimilarity(queryVector, vector)
		if math.IsNaN(score) || math.IsInf(score, 0) {
			continue
		}
		key := embeddingObjectResultKey(row)
		if existingIndex, ok := resultIndexes[key]; ok {
			existing := results[existingIndex]
			if score > existing.Score || (score == existing.Score && row.UpdatedAt.After(existing.Embedding.UpdatedAt)) {
				results[existingIndex] = EmbeddingSearchResult{Embedding: row, Score: score}
			}
			continue
		}
		resultIndexes[key] = len(results)
		results = append(results, EmbeddingSearchResult{Embedding: row, Score: score})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Embedding.ID < results[j].Embedding.ID
		}
		return results[i].Score > results[j].Score
	})
	return results
}

func embeddingObjectResultKey(row model.AIEmbedding) string {
	return fmt.Sprintf("%s:%d:%s", row.ObjectType, row.ObjectID, row.EmbeddingModel)
}

func marshalEmbeddingVector(vector []float64) (string, error) {
	data, err := json.Marshal(vector)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalEmbeddingVector(raw *string) ([]float64, error) {
	if raw == nil {
		return nil, ErrEmbeddingVectorInvalid
	}
	var vector []float64
	if err := json.Unmarshal([]byte(*raw), &vector); err != nil {
		return nil, err
	}
	return vector, validateEmbeddingVector(vector)
}

func jsonStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return math.NaN()
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return math.NaN()
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func candidateLimit(limit int) int {
	if limit <= 0 {
		return 1000
	}
	expanded := limit * 20
	if expanded < 200 {
		return 200
	}
	return expanded
}
