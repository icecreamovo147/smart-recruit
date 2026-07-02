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

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

const (
	EmbeddingStatusReady       = "ready"
	EmbeddingStatusUnavailable = "unavailable"

	DefaultEmbeddingModel = "unavailable"
)

var (
	ErrEmbeddingProviderUnavailable = errors.New("embedding provider unavailable")
	ErrEmbeddingVectorInvalid       = errors.New("embedding vector invalid")
)

type EmbeddingProvider interface {
	EmbedText(ctx context.Context, text string) (EmbeddingVector, error)
}

type EmbeddingVector struct {
	Model  string
	Vector []float64
}

type UnavailableEmbeddingProvider struct{}

func (UnavailableEmbeddingProvider) EmbedText(context.Context, string) (EmbeddingVector, error) {
	return EmbeddingVector{Model: DefaultEmbeddingModel}, ErrEmbeddingProviderUnavailable
}

type EmbeddingService struct {
	repo     *repository.AIEmbeddingRepo
	provider EmbeddingProvider
}

func NewEmbeddingService(repo *repository.AIEmbeddingRepo, provider EmbeddingProvider) *EmbeddingService {
	if provider == nil {
		provider = UnavailableEmbeddingProvider{}
	}
	return &EmbeddingService{repo: repo, provider: provider}
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
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("embedding repository is not configured")
	}
	objectType := strings.TrimSpace(input.ObjectType)
	if objectType == "" || input.ObjectID == 0 {
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
		result, err := s.provider.EmbedText(ctx, input.Text)
		if err != nil {
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
	return row, nil
}

func (s *EmbeddingService) Search(ctx context.Context, input EmbeddingSearchInput) ([]EmbeddingSearchResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("embedding repository is not configured")
	}
	queryVector, modelName, err := s.resolveQueryVector(ctx, input.QueryText, input.QueryVector, input.Model)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListCandidates(ctx, repository.AIEmbeddingQuery{
		ObjectTypes:    input.ObjectTypes,
		ScopeType:      input.ScopeType,
		ScopeID:        input.ScopeID,
		EmbeddingModel: modelName,
		Status:         EmbeddingStatusReady,
		Limit:          candidateLimit(input.Limit),
	})
	if err != nil {
		return nil, err
	}
	results := rankEmbeddingRows(queryVector, rows)
	limit := input.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	return results[:limit], nil
}

func (s *EmbeddingService) SearchObjects(ctx context.Context, input EmbeddingObjectSearchInput) ([]EmbeddingSearchResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("embedding repository is not configured")
	}
	objectType := strings.TrimSpace(input.ObjectType)
	if objectType == "" || len(input.ObjectIDs) == 0 {
		return nil, nil
	}
	queryVector, modelName, err := s.resolveQueryVector(ctx, input.QueryText, input.QueryVector, input.Model)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListByObjectIDs(ctx, objectType, input.ObjectIDs, modelName, EmbeddingStatusReady)
	if err != nil {
		return nil, err
	}
	results := rankEmbeddingRows(queryVector, rows)
	limit := input.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	return results[:limit], nil
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

func (s *EmbeddingService) resolveQueryVector(ctx context.Context, text string, vector []float64, modelName string) ([]float64, string, error) {
	queryVector := append([]float64(nil), vector...)
	normalizedModel := normalizeEmbeddingModel(modelName)
	if len(queryVector) == 0 {
		result, err := s.provider.EmbedText(ctx, text)
		if err != nil {
			return nil, "", err
		}
		queryVector = append([]float64(nil), result.Vector...)
		normalizedModel = normalizeEmbeddingModel(result.Model)
	}
	if err := validateEmbeddingVector(queryVector); err != nil {
		return nil, "", err
	}
	return queryVector, normalizedModel, nil
}

func rankEmbeddingRows(queryVector []float64, rows []model.AIEmbedding) []EmbeddingSearchResult {
	results := make([]EmbeddingSearchResult, 0, len(rows))
	for _, row := range rows {
		vector, err := unmarshalEmbeddingVector(row.VectorJSON)
		if err != nil || len(vector) != len(queryVector) {
			continue
		}
		score := cosineSimilarity(queryVector, vector)
		if math.IsNaN(score) || math.IsInf(score, 0) {
			continue
		}
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
		return 100
	}
	if limit < 20 {
		return 20
	}
	return limit * 4
}
