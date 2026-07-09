package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

type BackfillInput struct {
	ObjectType string
	ObjectID   int64
	Limit      int
	BatchSize  int
	Force      bool
	DryRun     bool
	ModelID    int64
}

type BackfillResult struct {
	SuccessCount int
	FailedCount  int
	SkippedCount int
	Errors       []string
}

type EmbeddingBackfillService struct {
	db         *gorm.DB
	embedding  *EmbeddingService
	skillRepo  *repository.AgentSkillRepo
	memoryRepo *repository.MemoryRepo
}

func NewEmbeddingBackfillService(db *gorm.DB, embedding *EmbeddingService, skillRepo *repository.AgentSkillRepo, memoryRepo *repository.MemoryRepo) *EmbeddingBackfillService {
	return &EmbeddingBackfillService{
		db:         db,
		embedding:  embedding,
		skillRepo:  skillRepo,
		memoryRepo: memoryRepo,
	}
}

func (s *EmbeddingBackfillService) Run(ctx context.Context, input BackfillInput) (*BackfillResult, error) {
	result := &BackfillResult{}

	batchSize := input.BatchSize
	if batchSize <= 0 {
		batchSize = 20
	}

	switch input.ObjectType {
	case "", "agent_skill":
		skillResult := s.backfillSkills(ctx, input, batchSize)
		result.SuccessCount += skillResult.SuccessCount
		result.FailedCount += skillResult.FailedCount
		result.SkippedCount += skillResult.SkippedCount
		result.Errors = append(result.Errors, skillResult.Errors...)

	case "ai_memory":
		memoryResult := s.backfillMemories(ctx, input, batchSize)
		result.SuccessCount += memoryResult.SuccessCount
		result.FailedCount += memoryResult.FailedCount
		result.SkippedCount += memoryResult.SkippedCount
		result.Errors = append(result.Errors, memoryResult.Errors...)

	default:
		return nil, fmt.Errorf("unsupported object_type: %s", input.ObjectType)
	}

	return result, nil
}

func (s *EmbeddingBackfillService) backfillSkills(ctx context.Context, input BackfillInput, batchSize int) *BackfillResult {
	result := &BackfillResult{}

	var skills []model.AgentSkill
	query := s.db.WithContext(ctx).Model(&model.AgentSkill{}).
		Where("is_enabled = 1 AND current_version_id IS NOT NULL").
		Order("id ASC")

	if input.ObjectID > 0 {
		query = query.Where("id = ?", input.ObjectID)
	}
	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	if err := query.Find(&skills).Error; err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("query skills: %v", err))
		return result
	}

	for _, skill := range skills {
		if ctx.Err() != nil {
			break
		}

		semanticTags := unmarshalStringList(skill.SemanticTags)

		bodyMarkdown := ""
		if skill.CurrentVersionID != nil {
			version, verErr := s.skillRepo.GetVersionByID(ctx, *skill.CurrentVersionID)
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
		if text == "" {
			result.SkippedCount++
			continue
		}

		if input.DryRun {
			logger.L().Info("[backfill] dry-run: would process skill",
				zap.String("name", skill.Name),
				zap.Int64("id", skill.ID))
			result.SuccessCount++
			continue
		}

		_, err := s.embedding.EmbedObject(ctx, EmbedObjectInput{
			ObjectType: "agent_skill",
			ObjectID:   uint64(skill.ID),
			ScopeType:  "agent_skill",
			ScopeID:    uint64(skill.ID),
			Text:       text,
		})
		if err != nil {
			logger.L().Error("[backfill] skill embedding failed",
				zap.Int64("skill_id", skill.ID),
				zap.Error(err))
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("skill %d: %v", skill.ID, err))
		} else {
			result.SuccessCount++
		}

		if (result.SuccessCount+result.FailedCount)%batchSize == 0 {
			logger.L().Info("[backfill] progress",
				zap.Int("success", result.SuccessCount),
				zap.Int("failed", result.FailedCount),
				zap.Int("skipped", result.SkippedCount))
		}
	}

	return result
}

func (s *EmbeddingBackfillService) backfillMemories(ctx context.Context, input BackfillInput, batchSize int) *BackfillResult {
	result := &BackfillResult{}

	var memories []model.AIMemory
	query := s.db.WithContext(ctx).Model(&model.AIMemory{}).
		Order("id ASC")

	if input.ObjectID > 0 {
		query = query.Where("id = ?", input.ObjectID)
	}
	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	if err := query.Find(&memories).Error; err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("query memories: %v", err))
		return result
	}

	for _, memory := range memories {
		if ctx.Err() != nil {
			break
		}

		scopeDescription := fmt.Sprintf("%s:%d", memory.ScopeType, memory.ScopeID)
		text := BuildMemoryEmbeddingText(memory.Content, memory.MemoryType, scopeDescription)
		if text == "" {
			result.SkippedCount++
			continue
		}

		if input.DryRun {
			logger.L().Info("[backfill] dry-run: would process memory",
				zap.Uint64("id", memory.ID),
				zap.String("memory_type", memory.MemoryType))
			result.SuccessCount++
			continue
		}

		_, err := s.embedding.EmbedObject(ctx, EmbedObjectInput{
			ObjectType: "ai_memory",
			ObjectID:   memory.ID,
			ScopeType:  memory.ScopeType,
			ScopeID:    memory.ScopeID,
			Text:       text,
		})
		if err != nil {
			logger.L().Error("[backfill] memory embedding failed",
				zap.Uint64("memory_id", memory.ID),
				zap.Error(err))
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("memory %d: %v", memory.ID, err))
		} else {
			result.SuccessCount++
		}

		if (result.SuccessCount+result.FailedCount)%batchSize == 0 {
			logger.L().Info("[backfill] progress",
				zap.Int("success", result.SuccessCount),
				zap.Int("failed", result.FailedCount),
				zap.Int("skipped", result.SkippedCount))
		}

		rateLimiter := time.Duration(50) * time.Millisecond
		select {
		case <-time.After(rateLimiter):
		case <-ctx.Done():
			break
		}
	}

	return result
}
