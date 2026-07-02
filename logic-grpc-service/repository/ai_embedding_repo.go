package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"logic-grpc-service/model"
)

type AIEmbeddingRepo struct {
	db *gorm.DB
}

func NewAIEmbeddingRepo(db *gorm.DB) *AIEmbeddingRepo {
	return &AIEmbeddingRepo{db: db}
}

type AIEmbeddingQuery struct {
	ObjectTypes    []string
	ScopeType      string
	ScopeID        uint64
	EmbeddingModel string
	Status         string
	Limit          int
}

func (r *AIEmbeddingRepo) Upsert(ctx context.Context, row *model.AIEmbedding) error {
	if row == nil {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "object_type"},
			{Name: "object_id"},
			{Name: "embedding_model"},
			{Name: "text_hash"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"scope_type",
			"scope_id",
			"embedding_dim",
			"vector_json",
			"metadata_json",
			"status",
			"last_error",
			"updated_at",
		}),
	}).Create(row).Error
}

func (r *AIEmbeddingRepo) GetByObject(ctx context.Context, objectType string, objectID uint64, modelName string) (*model.AIEmbedding, error) {
	var row model.AIEmbedding
	err := r.db.WithContext(ctx).
		Where("object_type = ? AND object_id = ? AND embedding_model = ?", objectType, objectID, modelName).
		Order("updated_at DESC, id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *AIEmbeddingRepo) ListCandidates(ctx context.Context, query AIEmbeddingQuery) ([]model.AIEmbedding, error) {
	db := r.db.WithContext(ctx).Model(&model.AIEmbedding{})
	if len(query.ObjectTypes) > 0 {
		db = db.Where("object_type IN ?", query.ObjectTypes)
	}
	if query.ScopeType != "" {
		db = db.Where("scope_type = ?", query.ScopeType)
	}
	if query.ScopeID > 0 {
		db = db.Where("scope_id = ?", query.ScopeID)
	}
	if query.EmbeddingModel != "" {
		db = db.Where("embedding_model = ?", query.EmbeddingModel)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	var rows []model.AIEmbedding
	err := db.Order("updated_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
