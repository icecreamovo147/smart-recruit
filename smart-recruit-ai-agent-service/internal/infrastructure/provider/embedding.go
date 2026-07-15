package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"smart-recruit-proto/recruitment/pb"
)

const defaultEmbeddingTestText = "Smart Recruit embedding runtime validation"

type EmbeddingConfig struct {
	ProviderID     int64
	ProviderName   string
	ProviderType   string
	Endpoint       string
	APIKey         string
	ExtraHeaders   string
	ModelID        int64
	ModelName      string
	Dimension      int
	BatchSize      int
	TimeoutSeconds int
	MaxRetries     int
}

type EmbedRequest struct {
	Config EmbeddingConfig
	Texts  []string
}

type EmbedResult struct {
	Vectors   [][]float64
	RequestID string
	Latency   time.Duration
}

type EmbeddingRunner interface {
	Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error)
}

type HTTPEmbeddingRunner struct {
	client *http.Client
}

func NewHTTPEmbeddingRunner() *HTTPEmbeddingRunner {
	return &HTTPEmbeddingRunner{client: http.DefaultClient}
}

func (r *HTTPEmbeddingRunner) Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error) {
	cfg := req.Config
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return EmbedResult{}, errors.New("embedding provider endpoint is empty")
	}
	if strings.TrimSpace(cfg.ModelName) == "" {
		return EmbedResult{}, errors.New("embedding model name is empty")
	}
	texts := compactTexts(req.Texts)
	if len(texts) == 0 {
		return EmbedResult{}, errors.New("embedding input text is empty")
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	payload := map[string]any{"model": cfg.ModelName, "input": texts}
	body, err := json.Marshal(payload)
	if err != nil {
		return EmbedResult{}, err
	}
	httpReq, err := http.NewRequestWithContext(callCtx, http.MethodPost, cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return EmbedResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(cfg.APIKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	}
	for key, value := range parseHeaders(cfg.ExtraHeaders) {
		httpReq.Header.Set(key, value)
	}
	start := time.Now()
	client := r.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return EmbedResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return EmbedResult{}, fmt.Errorf("embedding provider returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded struct {
		ID   string `json:"id"`
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return EmbedResult{}, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decoded.Data) != len(texts) {
		return EmbedResult{}, fmt.Errorf("embedding response vector count mismatch: got %d want %d", len(decoded.Data), len(texts))
	}
	vectors := make([][]float64, 0, len(decoded.Data))
	for i, item := range decoded.Data {
		if len(item.Embedding) == 0 {
			return EmbedResult{}, fmt.Errorf("embedding response vector %d is empty", i)
		}
		vectors = append(vectors, item.Embedding)
	}
	return EmbedResult{Vectors: vectors, RequestID: decoded.ID, Latency: time.Since(start)}, nil
}

type EmbeddingStore interface {
	ResolveEmbeddingConfig(ctx context.Context, providerID, modelID int64) (EmbeddingConfig, bool, error)
	UpdateEmbeddingTestStatus(ctx context.Context, modelID int64, status, lastError string, testedAt time.Time) error
	ListAgentSkillEmbeddingDocuments(ctx context.Context, objectID int64, limit int) ([]AgentSkillEmbeddingDocument, error)
	UpsertAIEmbedding(ctx context.Context, row AIEmbeddingRecord) error
	InvalidateAIEmbedding(ctx context.Context, objectType string, objectID int64) error
	ListAIEmbeddings(ctx context.Context, objectType, modelName string, limit int) ([]AIEmbeddingRecord, error)
}

type AgentSkillEmbeddingDocument struct {
	ID                   int64
	Name                 string
	DisplayName          string
	Description          string
	AgentType            string
	Category             string
	Scenario             string
	Priority             int
	RiskLevel            string
	TriggerKeywords      []string
	RequiredCapabilities []string
	EvaluationCriteria   []string
	SemanticTags         []string
	OutputSchema         string
	BodyMarkdown         string
	Enabled              bool
}

type AIEmbeddingRecord struct {
	ObjectType     string
	ObjectID       int64
	ScopeType      string
	ScopeID        int64
	TextHash       string
	EmbeddingModel string
	EmbeddingDim   int
	Vector         []float64
	Metadata       map[string]any
	Status         string
	LastError      string
}

type EmbeddingService struct {
	store  EmbeddingStore
	runner EmbeddingRunner
}

func NewEmbeddingService(store EmbeddingStore, runner EmbeddingRunner) *EmbeddingService {
	if runner == nil {
		runner = NewHTTPEmbeddingRunner()
	}
	return &EmbeddingService{store: store, runner: runner}
}

func (s *EmbeddingService) TestModel(ctx context.Context, req *pb.TestEmbeddingModelRequest) (*pb.TestEmbeddingModelResponse, error) {
	if s == nil || s.store == nil || s.runner == nil {
		return &pb.TestEmbeddingModelResponse{Code: 501, Msg: "embedding runtime validation is not configured", Success: false, Detail: "embedding store or runner is not bound"}, nil
	}
	cfg, ok, err := s.store.ResolveEmbeddingConfig(ctx, req.GetProviderId(), req.GetModelId())
	if err != nil {
		return nil, err
	}
	if !ok {
		return &pb.TestEmbeddingModelResponse{Code: 404, Msg: "embedding model not found", Success: false, Detail: "no enabled provider/model matched request"}, nil
	}
	text := strings.TrimSpace(req.GetTestText())
	if text == "" {
		text = defaultEmbeddingTestText
	}
	result, err := s.runner.Embed(ctx, EmbedRequest{Config: cfg, Texts: []string{text}})
	now := time.Now()
	if err != nil {
		_ = s.store.UpdateEmbeddingTestStatus(ctx, cfg.ModelID, "failed", err.Error(), now)
		return &pb.TestEmbeddingModelResponse{Code: 500, Msg: "embedding model test failed", Success: false, Detail: err.Error()}, nil
	}
	dim := 0
	if len(result.Vectors) > 0 {
		dim = len(result.Vectors[0])
	}
	_ = s.store.UpdateEmbeddingTestStatus(ctx, cfg.ModelID, "success", "", now)
	return &pb.TestEmbeddingModelResponse{Code: 0, Msg: "success", Success: true, Dimension: int32(dim), LatencyMs: result.Latency.Milliseconds(), RequestId: result.RequestID, Detail: fmt.Sprintf("provider=%s model=%s", cfg.ProviderName, cfg.ModelName)}, nil
}

func (s *EmbeddingService) Backfill(ctx context.Context, req *pb.BackfillEmbeddingsRequest) (*pb.BackfillEmbeddingsResponse, error) {
	objectType := strings.TrimSpace(req.GetObjectType())
	if objectType == "" {
		objectType = "agent_skill"
	}
	if objectType != "agent_skill" {
		return &pb.BackfillEmbeddingsResponse{Code: 400, Msg: "only agent_skill embedding backfill is supported in native runtime"}, nil
	}
	if s == nil || s.store == nil || s.runner == nil {
		return &pb.BackfillEmbeddingsResponse{Code: 501, Msg: "embedding backfill runtime is not configured"}, nil
	}
	cfg, ok, err := s.store.ResolveEmbeddingConfig(ctx, 0, req.GetModelId())
	if err != nil {
		return nil, err
	}
	if !ok {
		return &pb.BackfillEmbeddingsResponse{Code: 501, Msg: "embedding provider/model is not configured"}, nil
	}
	limit := int(req.GetLimit())
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	docs, err := s.store.ListAgentSkillEmbeddingDocuments(ctx, req.GetObjectId(), limit)
	if err != nil {
		return nil, err
	}
	var success, failed, skipped int32
	for _, doc := range docs {
		text := AgentSkillEmbeddingText(doc)
		if strings.TrimSpace(text) == "" {
			skipped++
			continue
		}
		if req.GetDryRun() {
			skipped++
			continue
		}
		if err := s.UpsertAgentSkillDocument(ctx, cfg, doc, text); err != nil {
			failed++
			continue
		}
		success++
	}
	return &pb.BackfillEmbeddingsResponse{Code: 0, Msg: "success", SuccessCount: success, FailedCount: failed, SkippedCount: skipped}, nil
}

func (s *EmbeddingService) UpsertAgentSkill(ctx context.Context, id int64) error {
	if s == nil || s.store == nil || s.runner == nil || id <= 0 {
		return nil
	}
	cfg, ok, err := s.store.ResolveEmbeddingConfig(ctx, 0, 0)
	if err != nil || !ok {
		return err
	}
	docs, err := s.store.ListAgentSkillEmbeddingDocuments(ctx, id, 1)
	if err != nil || len(docs) == 0 {
		return err
	}
	return s.UpsertAgentSkillDocument(ctx, cfg, docs[0], AgentSkillEmbeddingText(docs[0]))
}

func (s *EmbeddingService) UpsertAgentSkillDocument(ctx context.Context, cfg EmbeddingConfig, doc AgentSkillEmbeddingDocument, text string) error {
	result, err := s.runner.Embed(ctx, EmbedRequest{Config: cfg, Texts: []string{text}})
	status := "ready"
	lastError := ""
	vector := []float64(nil)
	if err != nil {
		status = "failed"
		lastError = err.Error()
	} else if len(result.Vectors) > 0 {
		vector = result.Vectors[0]
	}
	if err := s.store.InvalidateAIEmbedding(ctx, "agent_skill", doc.ID); err != nil {
		return err
	}
	return s.store.UpsertAIEmbedding(ctx, AIEmbeddingRecord{
		ObjectType:     "agent_skill",
		ObjectID:       doc.ID,
		ScopeType:      "agent_skill",
		ScopeID:        doc.ID,
		TextHash:       hashText(text),
		EmbeddingModel: cfg.ModelName,
		EmbeddingDim:   len(vector),
		Vector:         vector,
		Metadata:       agentSkillMetadata(doc),
		Status:         status,
		LastError:      lastError,
	})
}

func (s *EmbeddingService) InvalidateAgentSkill(ctx context.Context, id int64) error {
	if s == nil || s.store == nil || id <= 0 {
		return nil
	}
	return s.store.InvalidateAIEmbedding(ctx, "agent_skill", id)
}

func (s *EmbeddingService) SemanticScores(ctx context.Context, query string, limit int) (map[uint64]float64, string) {
	result, err := s.SearchAgentSkills(ctx, query, limit)
	if err != nil || !result.EmbeddingAvailable {
		if err != nil {
			return nil, err.Error()
		}
		return nil, result.FallbackReason
	}
	scores := make(map[uint64]float64, len(result.Skills))
	for _, item := range result.Skills {
		if item != nil && item.GetId() > 0 {
			scores[uint64(item.GetId())] = item.GetFinalRankScore()
		}
	}
	return scores, ""
}

func (s *EmbeddingService) SearchAgentSkills(ctx context.Context, query string, limit int) (*pb.DebugSemanticRetrievalResponse, error) {
	if s == nil || s.store == nil || s.runner == nil {
		return &pb.DebugSemanticRetrievalResponse{Code: 501, Msg: "semantic retrieval debug is not configured", EmbeddingAvailable: false, FallbackReason: "embedding runner is not bound"}, nil
	}
	cfg, ok, err := s.store.ResolveEmbeddingConfig(ctx, 0, 0)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &pb.DebugSemanticRetrievalResponse{Code: 501, Msg: "embedding provider/model is not configured", EmbeddingAvailable: false, FallbackReason: "embedding provider/model is not configured"}, nil
	}
	limit = normalizeLimit(limit)
	start := time.Now()
	embed, err := s.runner.Embed(ctx, EmbedRequest{Config: cfg, Texts: []string{query}})
	if err != nil {
		return &pb.DebugSemanticRetrievalResponse{Code: 0, Msg: "success", EmbeddingAvailable: false, FallbackReason: err.Error(), EmbeddingProvider: cfg.ProviderName, EmbeddingModel: cfg.ModelName, EmbeddingDim: int32(cfg.Dimension), QueryEmbeddingLatencyMs: time.Since(start).Milliseconds()}, nil
	}
	queryVector := embed.Vectors[0]
	rows, err := s.store.ListAIEmbeddings(ctx, "agent_skill", cfg.ModelName, 500)
	if err != nil {
		return nil, err
	}
	items := make([]*pb.SemanticSkillDebugItem, 0, len(rows))
	lowerQuery := strings.ToLower(query)
	for _, row := range rows {
		if row.Status != "ready" || len(row.Vector) == 0 {
			continue
		}
		vectorScore := cosine(queryVector, row.Vector)
		lexicalScore := lexicalScore(lowerQuery, row.Metadata)
		metadataScore := metadataScore(lowerQuery, row.Metadata)
		boost := priorityBoost(row.Metadata)
		final := vectorScore*0.7 + lexicalScore*0.15 + metadataScore*0.1 + boost
		items = append(items, semanticSkillItem(row, vectorScore, lexicalScore, metadataScore, boost, final))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].FinalRankScore == items[j].FinalRankScore {
			return items[i].Id < items[j].Id
		}
		return items[i].FinalRankScore > items[j].FinalRankScore
	})
	if len(items) > limit {
		items = items[:limit]
	}
	for i, item := range items {
		item.PoolRank = int32(i + 1)
	}
	fallbackReason := ""
	if len(items) == 0 {
		fallbackReason = "no ready agent_skill embeddings matched current model"
	}
	return &pb.DebugSemanticRetrievalResponse{
		Code:                    0,
		Msg:                     "success",
		EmbeddingAvailable:      true,
		FallbackReason:          fallbackReason,
		Skills:                  items,
		Memories:                []*pb.SemanticMemoryDebugItem{},
		SkillPoolConfidence:     confidenceLabel(items),
		MemoryPoolConfidence:    "none",
		EmbeddingProvider:       cfg.ProviderName,
		EmbeddingModel:          cfg.ModelName,
		EmbeddingDim:            int32(len(queryVector)),
		CandidateCount:          int32(len(rows)),
		QueryEmbeddingLatencyMs: embed.Latency.Milliseconds(),
	}, nil
}

func AgentSkillEmbeddingText(doc AgentSkillEmbeddingDocument) string {
	parts := []string{doc.Name, doc.DisplayName, doc.Description, doc.Category, doc.Scenario, doc.RiskLevel, strings.Join(doc.TriggerKeywords, " "), strings.Join(doc.SemanticTags, " "), strings.Join(doc.EvaluationCriteria, " "), doc.OutputSchema, doc.BodyMarkdown}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func compactTexts(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func parseHeaders(raw string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return out
	}
	for key, value := range decoded {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			out[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return out
}

func hashText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func agentSkillMetadata(doc AgentSkillEmbeddingDocument) map[string]any {
	return map[string]any{
		"id":                    doc.ID,
		"name":                  doc.Name,
		"display_name":          doc.DisplayName,
		"description":           doc.Description,
		"agent_type":            doc.AgentType,
		"category":              doc.Category,
		"scenario":              doc.Scenario,
		"priority":              doc.Priority,
		"risk_level":            doc.RiskLevel,
		"trigger_keywords":      doc.TriggerKeywords,
		"required_capabilities": doc.RequiredCapabilities,
		"evaluation_criteria":   doc.EvaluationCriteria,
		"semantic_tags":         doc.SemanticTags,
	}
}

func semanticSkillItem(row AIEmbeddingRecord, vectorScore, lexical, metadata, boost, final float64) *pb.SemanticSkillDebugItem {
	meta := row.Metadata
	return &pb.SemanticSkillDebugItem{
		Id:             row.ObjectID,
		Name:           stringMeta(meta, "name"),
		DisplayName:    stringMeta(meta, "display_name"),
		Category:       stringMeta(meta, "category"),
		Scenario:       stringMeta(meta, "scenario"),
		Priority:       int32(floatMeta(meta, "priority")),
		Score:          final,
		Reason:         "semantic vector ranking",
		SemanticTags:   stringSliceMeta(meta, "semantic_tags"),
		VectorScore:    vectorScore,
		LexicalScore:   lexical,
		MetadataScore:  metadata,
		RelevanceScore: vectorScore,
		BusinessBoost:  boost,
		FinalRankScore: final,
		RelevanceMode:  "semantic",
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func cosine(a, b []float64) float64 {
	size := len(a)
	if len(b) < size {
		size = len(b)
	}
	var dot, an, bn float64
	for i := 0; i < size; i++ {
		dot += a[i] * b[i]
		an += a[i] * a[i]
		bn += b[i] * b[i]
	}
	if an == 0 || bn == 0 {
		return 0
	}
	score := dot / (math.Sqrt(an) * math.Sqrt(bn))
	if score < 0 {
		return 0
	}
	return score
}

func lexicalScore(query string, meta map[string]any) float64 {
	if strings.TrimSpace(query) == "" {
		return 0
	}
	haystack := strings.ToLower(strings.Join([]string{stringMeta(meta, "name"), stringMeta(meta, "display_name"), stringMeta(meta, "description")}, " "))
	if strings.Contains(haystack, query) {
		return 1
	}
	score := 0.0
	for _, token := range strings.Fields(query) {
		if strings.Contains(haystack, token) {
			score += 0.2
		}
	}
	if score > 1 {
		return 1
	}
	return score
}

func metadataScore(query string, meta map[string]any) float64 {
	values := append(stringSliceMeta(meta, "semantic_tags"), stringSliceMeta(meta, "trigger_keywords")...)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) || strings.Contains(query, strings.ToLower(value)) {
			return 1
		}
	}
	return 0
}

func priorityBoost(meta map[string]any) float64 {
	priority := floatMeta(meta, "priority")
	if priority <= 0 {
		return 0
	}
	if priority > 100 {
		priority = 100
	}
	return priority / 1000
}

func confidenceLabel(items []*pb.SemanticSkillDebugItem) string {
	if len(items) == 0 {
		return "none"
	}
	top := items[0].GetFinalRankScore()
	switch {
	case top >= 0.75:
		return "high"
	case top >= 0.45:
		return "medium"
	default:
		return "low"
	}
}

func stringMeta(meta map[string]any, key string) string {
	value, _ := meta[key].(string)
	return value
}

func floatMeta(meta map[string]any, key string) float64 {
	switch value := meta[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		n, _ := value.Float64()
		return n
	default:
		return 0
	}
}

func stringSliceMeta(meta map[string]any, key string) []string {
	switch values := meta[key].(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
