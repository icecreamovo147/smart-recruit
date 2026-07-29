package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var ErrAgentSkillEmbeddingNotReady = errors.New("Agent Skill release embeddings are not ready")

func validatePublishedAgentSkillEmbeddingReadiness(
	tx *gorm.DB,
	snapshot PlatformAICapabilitySnapshot,
	packages []PlatformAIAgentSkillReleasePackage,
) error {
	if len(packages) == 0 {
		return nil
	}
	if !tx.Migrator().HasTable((aiEmbeddingRecord{}).TableName()) ||
		!tx.Migrator().HasTable((embeddingModelRecord{}).TableName()) {
		return fmt.Errorf("%w: embedding storage is unavailable", ErrAgentSkillEmbeddingNotReady)
	}
	model, err := resolveReleaseEmbeddingModel(tx, snapshot.ModelPolicy.DefaultEmbeddingModelID)
	if err != nil {
		return err
	}
	versionIDs := make([]int64, 0, len(packages))
	compiledHashes := make(map[int64]string, len(packages))
	for _, pkg := range packages {
		versionIDs = append(versionIDs, pkg.VersionID)
		compiledHashes[pkg.VersionID] = pkg.Package.CompiledHash
	}

	var versionEmbeddings []aiEmbeddingRecord
	if err := tx.Where(
		"object_type = ? AND object_id IN ? AND scope_type = ? AND embedding_model = ? AND status = ?",
		"agent_skill_version", versionIDs, "agent_skill_version", model.ModelName, "ready",
	).Find(&versionEmbeddings).Error; err != nil {
		return err
	}
	readyVersions := make(map[int64]bool, len(versionEmbeddings))
	for _, embedding := range versionEmbeddings {
		hash := compiledHashes[embedding.ObjectID]
		if embedding.ScopeID == embedding.ObjectID && embeddingReadyForPackage(embedding, model, hash) {
			readyVersions[embedding.ObjectID] = true
		}
	}
	for _, versionID := range versionIDs {
		if !readyVersions[versionID] {
			return fmt.Errorf(
				"%w: version %d has no current ready embedding for model %s",
				ErrAgentSkillEmbeddingNotReady,
				versionID,
				model.ModelName,
			)
		}
	}

	var sections []agentSkillSectionRecord
	if err := tx.Where("skill_version_id IN ?", versionIDs).Find(&sections).Error; err != nil {
		return err
	}
	if len(sections) == 0 {
		return nil
	}
	sectionIDs := make([]int64, 0, len(sections))
	sectionsByID := make(map[int64]agentSkillSectionRecord, len(sections))
	for _, section := range sections {
		sectionIDs = append(sectionIDs, section.ID)
		sectionsByID[section.ID] = section
	}
	var sectionEmbeddings []aiEmbeddingRecord
	if err := tx.Where(
		"object_type = ? AND object_id IN ? AND scope_type = ? AND embedding_model = ? AND status = ?",
		"agent_skill_section", sectionIDs, "agent_skill_version", model.ModelName, "ready",
	).Find(&sectionEmbeddings).Error; err != nil {
		return err
	}
	readySections := make(map[int64]bool, len(sectionEmbeddings))
	for _, embedding := range sectionEmbeddings {
		section, ok := sectionsByID[embedding.ObjectID]
		if !ok || embedding.ScopeID != section.SkillVersionID {
			continue
		}
		if embeddingReadyForPackage(embedding, model, compiledHashes[section.SkillVersionID]) {
			readySections[embedding.ObjectID] = true
		}
	}
	for _, section := range sections {
		if !readySections[section.ID] {
			return fmt.Errorf(
				"%w: section %d of version %d has no current ready embedding for model %s",
				ErrAgentSkillEmbeddingNotReady,
				section.ID,
				section.SkillVersionID,
				model.ModelName,
			)
		}
	}
	return nil
}

func resolveReleaseEmbeddingModel(tx *gorm.DB, requestedModelID int64) (embeddingModelRecord, error) {
	var model embeddingModelRecord
	query := tx.Table("embedding_models model").
		Select("model.*").
		Joins("JOIN embedding_providers provider ON provider.id = model.provider_id").
		Where("model.is_enabled = ? AND provider.is_enabled = ?", true, true)
	if requestedModelID > 0 {
		query = query.Where("model.id = ?", requestedModelID)
	} else {
		query = query.Where("model.is_default = ?", true)
	}
	if err := query.Order("model.is_default DESC, model.id ASC").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return embeddingModelRecord{}, fmt.Errorf(
				"%w: no enabled default embedding model is available",
				ErrAgentSkillEmbeddingNotReady,
			)
		}
		return embeddingModelRecord{}, err
	}
	return model, nil
}

func embeddingReadyForPackage(embedding aiEmbeddingRecord, model embeddingModelRecord, compiledHash string) bool {
	if embedding.EmbeddingDim <= 0 ||
		(model.EmbeddingDim > 0 && embedding.EmbeddingDim != model.EmbeddingDim) ||
		!embedding.VectorJSON.Valid ||
		strings.TrimSpace(embedding.VectorJSON.String) == "" {
		return false
	}
	var vector []float64
	if err := json.Unmarshal([]byte(embedding.VectorJSON.String), &vector); err != nil ||
		len(vector) != embedding.EmbeddingDim {
		return false
	}
	if strings.TrimSpace(compiledHash) == "" || !embedding.MetadataJSON.Valid {
		return false
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(embedding.MetadataJSON.String), &metadata); err != nil {
		return false
	}
	storedHash, _ := metadata["compiled_hash"].(string)
	return strings.EqualFold(strings.TrimSpace(storedHash), strings.TrimSpace(compiledHash))
}
