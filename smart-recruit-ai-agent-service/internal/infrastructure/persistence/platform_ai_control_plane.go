package persistence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
)

const (
	PlatformAIAudienceTenantHR  = "tenant_hr"
	PlatformAIAudienceCandidate = "candidate"

	PlatformAIReleaseDraft     = "draft"
	PlatformAIReleasePublished = "published"
	PlatformAIReleaseRetired   = "retired"

	ModelFallbackUnavailable = "model_unavailable"

	PlatformAISnapshotSchemaVersion     = 2
	PlatformAISkillPolicyVersion        = "skill-package-v2.1"
	PlatformAIDefaultMaxSkillTokens     = 3000
	PlatformAIDefaultMaxSkillInputRatio = 0.15
	PlatformAIDefaultMaxSkills          = 2
	platformAIMaxSkillTokens            = 3000
	platformAIMaxSkillInputRatio        = 0.15
	platformAIFixedMaxSkills            = 2
)

var platformAICapabilitySnapshotRequiredFields = [...]string{
	"schema_version",
	"capability_key",
	"audience",
	"model_policy",
	"configuration_refs",
	"skill_runtime_policy",
}

var (
	ErrCapabilityNotFound       = errors.New("platform AI capability not found")
	ErrCapabilityUnavailable    = errors.New("platform AI capability is unavailable")
	ErrCapabilityVersionChanged = errors.New("published platform AI capability versions are immutable")
	ErrCapabilitySnapshotHash   = errors.New("platform AI capability snapshot hash does not match its canonical content")
	ErrModelNotAllowed          = errors.New("requested model is not allowed by the capability release")
)

type PlatformAIModelPolicy struct {
	AllowedLLMModelIDs       []int64 `json:"allowed_llm_model_ids"`
	DefaultLLMModelID        int64   `json:"default_llm_model_id,omitempty"`
	AllowedEmbeddingModelIDs []int64 `json:"allowed_embedding_model_ids,omitempty"`
	DefaultEmbeddingModelID  int64   `json:"default_embedding_model_id,omitempty"`
}

type PlatformAICapabilitySnapshot struct {
	SchemaVersion      int                          `json:"schema_version"`
	CapabilityKey      string                       `json:"capability_key"`
	Audience           string                       `json:"audience"`
	ModelPolicy        PlatformAIModelPolicy        `json:"model_policy"`
	ConfigurationRef   PlatformAIConfigurationRefs  `json:"configuration_refs"`
	SkillRuntimePolicy PlatformAISkillRuntimePolicy `json:"skill_runtime_policy"`
}

type PlatformAIConfigurationRefs struct {
	AgentIDs             []int64 `json:"agent_ids"`
	PromptTemplateIDs    []int64 `json:"prompt_template_ids"`
	AgentSkillVersionIDs []int64 `json:"agent_skill_version_ids"`
	MCPPolicyIDs         []int64 `json:"mcp_policy_ids"`
}

type PlatformAISkillRuntimePolicy struct {
	PolicyVersion        string  `json:"policy_version"`
	MaxSkillTokens       int     `json:"max_skill_tokens"`
	MaxInputRatio        float64 `json:"max_input_ratio"`
	MaxSkills            int     `json:"max_skills"`
	EvaluationSuiteHash  string  `json:"evaluation_suite_hash"`
	EvaluationResultHash string  `json:"evaluation_result_hash"`
}

type PlatformAICapability struct {
	ID                        int64
	CapabilityKey             string
	Audience                  string
	Name                      string
	Description               string
	Status                    string
	CurrentPublishedVersionID int64
}

type PlatformAICapabilityVersion struct {
	ID           int64
	CapabilityID int64
	Version      int
	Status       string
	SnapshotJSON string
	SnapshotHash string
	ChangeNote   string
	PublishedAt  *time.Time
}

type RuntimeModelResolution struct {
	CapabilityKey       string
	Audience            string
	CapabilityVersionID int64
	SnapshotHash        string
	RequestedModelID    int64
	EffectiveModelID    int64
	FallbackReason      string
	ModelName           string
	DisplayName         string
	ProviderID          int64
	ProviderName        string
}

type AllowedRuntimeModel struct {
	ID                  int64
	ModelName           string
	DisplayName         string
	ProviderID          int64
	ProviderName        string
	IsDefault           bool
	MaxTokens           int32
	ContextWindowTokens int32
}

type PlatformAIConfigAuditLog struct {
	ID                  int64
	ActorUserID         int64
	Action              string
	ResourceType        string
	ResourceID          int64
	CapabilityID        int64
	CapabilityVersionID int64
	BeforeSnapshot      string
	AfterSnapshot       string
	RequestID           string
	CreatedAt           time.Time
}

type platformAICapabilityRecord struct {
	ID                        int64          `gorm:"primaryKey"`
	CapabilityKey             string         `gorm:"column:capability_key"`
	Audience                  string         `gorm:"column:audience"`
	Name                      string         `gorm:"column:name"`
	Description               sql.NullString `gorm:"column:description"`
	Status                    string         `gorm:"column:status"`
	CurrentPublishedVersionID sql.NullInt64  `gorm:"column:current_published_version_id"`
	CreatedBy                 sql.NullInt64  `gorm:"column:created_by"`
	UpdatedBy                 sql.NullInt64  `gorm:"column:updated_by"`
	CreatedAt                 time.Time      `gorm:"column:created_at"`
	UpdatedAt                 time.Time      `gorm:"column:updated_at"`
}

func (platformAICapabilityRecord) TableName() string { return "platform_ai_capabilities" }

type platformAICapabilityVersionRecord struct {
	ID           int64          `gorm:"primaryKey"`
	CapabilityID int64          `gorm:"column:capability_id"`
	Version      int            `gorm:"column:version"`
	Status       string         `gorm:"column:status"`
	SnapshotJSON string         `gorm:"column:snapshot_json"`
	SnapshotHash string         `gorm:"column:snapshot_hash"`
	ChangeNote   sql.NullString `gorm:"column:change_note"`
	CreatedBy    sql.NullInt64  `gorm:"column:created_by"`
	PublishedBy  sql.NullInt64  `gorm:"column:published_by"`
	PublishedAt  *time.Time     `gorm:"column:published_at"`
	RetiredAt    *time.Time     `gorm:"column:retired_at"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
}

func (platformAICapabilityVersionRecord) TableName() string {
	return "platform_ai_capability_versions"
}

type platformAIConfigAuditRecord struct {
	ID                  int64          `gorm:"primaryKey"`
	ActorUserID         sql.NullInt64  `gorm:"column:actor_user_id"`
	Action              string         `gorm:"column:action"`
	ResourceType        string         `gorm:"column:resource_type"`
	ResourceID          sql.NullInt64  `gorm:"column:resource_id"`
	CapabilityID        sql.NullInt64  `gorm:"column:capability_id"`
	CapabilityVersionID sql.NullInt64  `gorm:"column:capability_version_id"`
	BeforeSnapshot      sql.NullString `gorm:"column:before_snapshot"`
	AfterSnapshot       sql.NullString `gorm:"column:after_snapshot"`
	RequestID           sql.NullString `gorm:"column:request_id"`
	CreatedAt           time.Time      `gorm:"column:created_at"`
}

func (platformAIConfigAuditRecord) TableName() string { return "platform_ai_config_audit_logs" }

type runtimeModelRow struct {
	ID                  int64  `gorm:"column:id"`
	ProviderID          int64  `gorm:"column:provider_id"`
	ModelName           string `gorm:"column:model_name"`
	DisplayName         string `gorm:"column:display_name"`
	ProviderName        string `gorm:"column:provider_name"`
	MaxTokens           int32  `gorm:"column:max_tokens"`
	ContextWindowTokens int32  `gorm:"column:context_window_tokens"`
}

func (s *NativeStore) ListPlatformAICapabilities(ctx context.Context) ([]PlatformAICapability, error) {
	var rows []platformAICapabilityRecord
	if err := s.db.WithContext(ctx).Order("audience ASC, capability_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]PlatformAICapability, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapPlatformAICapability(row))
	}
	return result, nil
}

func (s *NativeStore) ListPlatformAICapabilityVersions(ctx context.Context, capabilityID int64) ([]PlatformAICapabilityVersion, error) {
	var rows []platformAICapabilityVersionRecord
	query := s.db.WithContext(ctx).Order("version DESC")
	if capabilityID > 0 {
		query = query.Where("capability_id = ?", capabilityID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]PlatformAICapabilityVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapPlatformAICapabilityVersion(row))
	}
	return result, nil
}

func (s *NativeStore) QueryPlatformAIConfigAuditLogs(ctx context.Context, page, pageSize int, resourceType string, capabilityID int64) ([]PlatformAIConfigAuditLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := s.db.WithContext(ctx).Model(&platformAIConfigAuditRecord{})
	if resourceType = strings.TrimSpace(resourceType); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if capabilityID > 0 {
		query = query.Where("capability_id = ?", capabilityID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []platformAIConfigAuditRecord
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]PlatformAIConfigAuditLog, 0, len(rows))
	for _, row := range rows {
		result = append(result, PlatformAIConfigAuditLog{
			ID:                  row.ID,
			ActorUserID:         row.ActorUserID.Int64,
			Action:              row.Action,
			ResourceType:        row.ResourceType,
			ResourceID:          row.ResourceID.Int64,
			CapabilityID:        row.CapabilityID.Int64,
			CapabilityVersionID: row.CapabilityVersionID.Int64,
			BeforeSnapshot:      row.BeforeSnapshot.String,
			AfterSnapshot:       row.AfterSnapshot.String,
			RequestID:           row.RequestID.String,
			CreatedAt:           row.CreatedAt,
		})
	}
	return result, total, nil
}

func (s *NativeStore) CreatePlatformAICapabilityDraft(ctx context.Context, capabilityID, actorID int64, snapshotJSON []byte, changeNote, requestID string) (PlatformAICapabilityVersion, error) {
	var result PlatformAICapabilityVersion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var capability platformAICapabilityRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&capability, capabilityID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCapabilityNotFound
			}
			return err
		}
		normalized, hash, err := s.prepareCapabilityDraftSnapshot(ctx, tx, snapshotJSON, capability)
		if err != nil {
			return err
		}
		var latest int
		if err := tx.Model(&platformAICapabilityVersionRecord{}).Where("capability_id = ?", capabilityID).Select("COALESCE(MAX(version), 0)").Scan(&latest).Error; err != nil {
			return err
		}
		row := platformAICapabilityVersionRecord{
			CapabilityID: capabilityID,
			Version:      latest + 1,
			Status:       PlatformAIReleaseDraft,
			SnapshotJSON: normalized,
			SnapshotHash: hash,
			ChangeNote:   nullStringFrom(changeNote, true),
			CreatedBy:    nullInt64From(actorID),
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := createPlatformAIConfigAudit(tx, actorID, "capability.release.draft.create", "platform_ai_capability_version", row.ID, capabilityID, row.ID, "", normalized, requestID); err != nil {
			return err
		}
		result = mapPlatformAICapabilityVersion(row)
		return nil
	})
	return result, err
}

func (s *NativeStore) UpdatePlatformAICapabilityDraft(ctx context.Context, versionID, actorID int64, snapshotJSON []byte, changeNote, requestID string) (PlatformAICapabilityVersion, error) {
	var result PlatformAICapabilityVersion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version platformAICapabilityVersionRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&version, versionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCapabilityNotFound
			}
			return err
		}
		if version.Status != PlatformAIReleaseDraft {
			return ErrCapabilityVersionChanged
		}
		var capability platformAICapabilityRecord
		if err := tx.First(&capability, version.CapabilityID).Error; err != nil {
			return err
		}
		normalized, hash, err := s.prepareCapabilityDraftSnapshot(ctx, tx, snapshotJSON, capability)
		if err != nil {
			return err
		}
		before := version.SnapshotJSON
		version.SnapshotJSON = normalized
		version.SnapshotHash = hash
		version.ChangeNote = nullStringFrom(changeNote, true)
		if err := tx.Save(&version).Error; err != nil {
			return err
		}
		if err := createPlatformAIConfigAudit(tx, actorID, "capability.release.draft.update", "platform_ai_capability_version", version.ID, version.CapabilityID, version.ID, before, normalized, requestID); err != nil {
			return err
		}
		result = mapPlatformAICapabilityVersion(version)
		return nil
	})
	return result, err
}

func (s *NativeStore) DeletePlatformAICapabilityDraft(ctx context.Context, versionID, actorID int64, requestID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version platformAICapabilityVersionRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&version, versionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCapabilityNotFound
			}
			return err
		}
		if version.Status != PlatformAIReleaseDraft {
			return ErrCapabilityVersionChanged
		}

		// Keep the audit evidence, but remove the foreign-key reference before
		// deleting the draft version. resource_id and capability_id retain the
		// deleted version's identity for audit queries.
		if err := tx.Model(&platformAIConfigAuditRecord{}).
			Where("capability_version_id = ?", version.ID).
			Update("capability_version_id", nil).Error; err != nil {
			return err
		}
		if err := createPlatformAIConfigAudit(tx, actorID, "capability.release.draft.delete", "platform_ai_capability_version", version.ID, version.CapabilityID, 0, version.SnapshotJSON, "", requestID); err != nil {
			return err
		}
		return tx.Delete(&version).Error
	})
}

func (s *NativeStore) PublishPlatformAICapabilityVersion(ctx context.Context, versionID, actorID int64, requestID string) (PlatformAICapabilityVersion, error) {
	var result PlatformAICapabilityVersion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version platformAICapabilityVersionRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&version, versionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCapabilityNotFound
			}
			return err
		}
		if version.Status != PlatformAIReleaseDraft {
			return ErrCapabilityVersionChanged
		}
		var capability platformAICapabilityRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&capability, version.CapabilityID).Error; err != nil {
			return err
		}
		_, hash, snapshot, err := normalizeCapabilitySnapshot([]byte(version.SnapshotJSON), capability)
		if err != nil {
			return err
		}
		// MySQL JSON columns may reorder object keys and change insignificant
		// whitespace. The hash is calculated from normalized semantic content,
		// so it remains the authoritative integrity check.
		if hash != version.SnapshotHash {
			return ErrCapabilitySnapshotHash
		}
		if err := validatePublishedModelPolicy(tx, snapshot.ModelPolicy); err != nil {
			return err
		}
		if err := validatePublishedConfigurationRefs(tx, snapshot); err != nil {
			return err
		}
		packages, err := validatePublishedAgentSkillPackages(tx, snapshot)
		if err != nil {
			return err
		}
		if err := s.validateAgentSkillReleaseEvaluation(ctx, snapshot, packages); err != nil {
			return err
		}
		now := time.Now()
		version.Status = PlatformAIReleasePublished
		version.PublishedBy = nullInt64From(actorID)
		version.PublishedAt = &now
		if err := tx.Save(&version).Error; err != nil {
			return err
		}
		if err := tx.Model(&platformAICapabilityRecord{}).Where("id = ?", capability.ID).Updates(map[string]interface{}{
			"current_published_version_id": version.ID,
			"updated_by":                   nullableActor(actorID),
		}).Error; err != nil {
			return err
		}
		if err := createPlatformAIConfigAudit(tx, actorID, "capability.release.publish", "platform_ai_capability_version", version.ID, capability.ID, version.ID, "", version.SnapshotJSON, requestID); err != nil {
			return err
		}
		result = mapPlatformAICapabilityVersion(version)
		return nil
	})
	return result, err
}

func (s *NativeStore) ResolveRuntimeModel(ctx context.Context, capabilityKey, audience string, releaseVersionID, requestedModelID int64) (RuntimeModelResolution, error) {
	capability, version, snapshot, err := s.loadPublishedCapability(ctx, capabilityKey, audience, releaseVersionID)
	if err != nil {
		return RuntimeModelResolution{}, err
	}
	allowed := int64Set(snapshot.ModelPolicy.AllowedLLMModelIDs)
	if requestedModelID > 0 {
		if _, ok := allowed[requestedModelID]; !ok {
			return RuntimeModelResolution{}, ErrModelNotAllowed
		}
		if model, found, err := s.loadAvailableRuntimeModel(ctx, requestedModelID); err != nil {
			return RuntimeModelResolution{}, err
		} else if found {
			return runtimeModelResolution(capability, version, requestedModelID, requestedModelID, "", model), nil
		}
	}

	defaultID := snapshot.ModelPolicy.DefaultLLMModelID
	if defaultID <= 0 {
		return RuntimeModelResolution{}, ErrCapabilityUnavailable
	}
	if _, ok := allowed[defaultID]; !ok {
		return RuntimeModelResolution{}, ErrCapabilityUnavailable
	}
	model, found, err := s.loadAvailableRuntimeModel(ctx, defaultID)
	if err != nil {
		return RuntimeModelResolution{}, err
	}
	if !found {
		return RuntimeModelResolution{}, ErrCapabilityUnavailable
	}
	fallbackReason := ""
	if requestedModelID > 0 && requestedModelID != defaultID {
		fallbackReason = ModelFallbackUnavailable
	}
	return runtimeModelResolution(capability, version, requestedModelID, defaultID, fallbackReason, model), nil
}

func (s *NativeStore) ResolveCapabilityRuntimeModel(ctx context.Context, capabilityKey, audience string, releaseVersionID, requestedModelID int64) (aiagentgrpc.CapabilityRuntimeModelResolution, error) {
	resolution, err := s.ResolveRuntimeModel(ctx, capabilityKey, audience, releaseVersionID, requestedModelID)
	if err != nil {
		return aiagentgrpc.CapabilityRuntimeModelResolution{}, err
	}
	model, found, err := s.loadAvailableRuntimeModel(ctx, resolution.EffectiveModelID)
	if err != nil {
		return aiagentgrpc.CapabilityRuntimeModelResolution{}, err
	}
	if !found {
		return aiagentgrpc.CapabilityRuntimeModelResolution{}, ErrCapabilityUnavailable
	}
	_, _, snapshot, err := s.loadPublishedCapability(ctx, capabilityKey, audience, resolution.CapabilityVersionID)
	if err != nil {
		return aiagentgrpc.CapabilityRuntimeModelResolution{}, err
	}
	return aiagentgrpc.CapabilityRuntimeModelResolution{
		EffectiveModelID:       resolution.EffectiveModelID,
		ModelName:              resolution.ModelName,
		ProviderName:           resolution.ProviderName,
		RequestedModelID:       resolution.RequestedModelID,
		FallbackReason:         resolution.FallbackReason,
		CapabilityVersionID:    resolution.CapabilityVersionID,
		CapabilitySnapshotHash: resolution.SnapshotHash,
		ContextWindowTokens:    model.ContextWindowTokens,
		MaxOutputTokens:        model.MaxTokens,
		ConfigurationRefs: aiagentgrpc.CapabilityConfigurationRefs{
			AgentIDs:             append([]int64(nil), snapshot.ConfigurationRef.AgentIDs...),
			PromptTemplateIDs:    append([]int64(nil), snapshot.ConfigurationRef.PromptTemplateIDs...),
			AgentSkillVersionIDs: append([]int64(nil), snapshot.ConfigurationRef.AgentSkillVersionIDs...),
			MCPPolicyIDs:         append([]int64(nil), snapshot.ConfigurationRef.MCPPolicyIDs...),
		},
		SkillRuntimePolicy: aiagentgrpc.CapabilitySkillRuntimePolicy{
			PolicyVersion:  snapshot.SkillRuntimePolicy.PolicyVersion,
			MaxSkillTokens: snapshot.SkillRuntimePolicy.MaxSkillTokens,
			MaxInputRatio:  snapshot.SkillRuntimePolicy.MaxInputRatio,
			MaxSkills:      snapshot.SkillRuntimePolicy.MaxSkills,
		},
	}, nil
}

func (s *NativeStore) ListAllowedRuntimeModels(ctx context.Context, capabilityKey, audience string, releaseVersionID int64) ([]AllowedRuntimeModel, PlatformAICapabilityVersion, error) {
	_, version, snapshot, err := s.loadPublishedCapability(ctx, capabilityKey, audience, releaseVersionID)
	if err != nil {
		return nil, PlatformAICapabilityVersion{}, err
	}
	ids := uniquePositiveIDs(snapshot.ModelPolicy.AllowedLLMModelIDs)
	if len(ids) == 0 {
		return []AllowedRuntimeModel{}, mapPlatformAICapabilityVersion(version), nil
	}
	var rows []runtimeModelRow
	if err := s.db.WithContext(ctx).Table("llm_models model").
		Select("model.id, model.provider_id, model.model_name, model.display_name, model.max_tokens, model.context_window_tokens, provider.name AS provider_name").
		Joins("JOIN llm_providers provider ON provider.id = model.provider_id AND provider.is_enabled = ?", true).
		Where("model.id IN ? AND model.is_enabled = ?", ids, true).
		Order("model.id ASC").Scan(&rows).Error; err != nil {
		return nil, PlatformAICapabilityVersion{}, err
	}
	models := make([]AllowedRuntimeModel, 0, len(rows))
	for _, row := range rows {
		models = append(models, AllowedRuntimeModel{ID: row.ID, ModelName: row.ModelName, DisplayName: row.DisplayName, ProviderID: row.ProviderID, ProviderName: row.ProviderName, IsDefault: row.ID == snapshot.ModelPolicy.DefaultLLMModelID, MaxTokens: row.MaxTokens, ContextWindowTokens: row.ContextWindowTokens})
	}
	return models, mapPlatformAICapabilityVersion(version), nil
}

func (s *NativeStore) ResolveReleasedMCPToolKeys(ctx context.Context, policyIDs []int64) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(policyIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		ServerID int64  `gorm:"column:server_id"`
		ToolName string `gorm:"column:tool_name"`
	}
	if err := s.db.WithContext(ctx).Table("mcp_tool_policies p").
		Select("p.server_id, p.tool_name").Joins("JOIN mcp_servers s ON s.id = p.server_id AND s.is_enabled = ?", true).
		Where("p.id IN ? AND p.is_enabled = ? AND p.effect = ?", policyIDs, true, "allow").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[fmt.Sprintf("mcp:%d:%s", row.ServerID, strings.TrimSpace(row.ToolName))] = true
	}
	return result, nil
}

func (s *NativeStore) loadPublishedCapability(ctx context.Context, capabilityKey, audience string, releaseVersionID int64) (platformAICapabilityRecord, platformAICapabilityVersionRecord, PlatformAICapabilitySnapshot, error) {
	var capability platformAICapabilityRecord
	if err := s.db.WithContext(ctx).Where("capability_key = ? AND audience = ? AND status = ?", strings.TrimSpace(capabilityKey), strings.TrimSpace(audience), "active").First(&capability).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return capability, platformAICapabilityVersionRecord{}, PlatformAICapabilitySnapshot{}, ErrCapabilityNotFound
		}
		return capability, platformAICapabilityVersionRecord{}, PlatformAICapabilitySnapshot{}, err
	}
	versionID := releaseVersionID
	if versionID <= 0 && capability.CurrentPublishedVersionID.Valid {
		versionID = capability.CurrentPublishedVersionID.Int64
	}
	if versionID <= 0 {
		return capability, platformAICapabilityVersionRecord{}, PlatformAICapabilitySnapshot{}, ErrCapabilityUnavailable
	}
	var version platformAICapabilityVersionRecord
	if err := s.db.WithContext(ctx).Where("id = ? AND capability_id = ? AND status = ?", versionID, capability.ID, PlatformAIReleasePublished).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return capability, version, PlatformAICapabilitySnapshot{}, ErrCapabilityUnavailable
		}
		return capability, version, PlatformAICapabilitySnapshot{}, err
	}
	_, hash, snapshot, err := normalizeCapabilitySnapshot([]byte(version.SnapshotJSON), capability)
	if err != nil {
		return capability, version, snapshot, err
	}
	// MySQL may serialize JSON differently from the canonical encoder. Compare
	// the canonical hash so formatting-only changes remain valid while semantic
	// tampering is still rejected.
	if hash != version.SnapshotHash {
		return capability, version, PlatformAICapabilitySnapshot{}, ErrCapabilitySnapshotHash
	}
	return capability, version, snapshot, nil
}

func normalizeCapabilitySnapshot(raw []byte, capability platformAICapabilityRecord) (string, string, PlatformAICapabilitySnapshot, error) {
	var snapshot PlatformAICapabilitySnapshot
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return "", "", snapshot, fmt.Errorf("invalid capability snapshot: %w", err)
	}
	for _, field := range platformAICapabilitySnapshotRequiredFields {
		if _, ok := object[field]; !ok {
			return "", "", snapshot, fmt.Errorf("capability snapshot %s is required", field)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return "", "", snapshot, fmt.Errorf("invalid capability snapshot: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", "", snapshot, errors.New("invalid capability snapshot: trailing JSON value")
	}
	if snapshot.SchemaVersion != PlatformAISnapshotSchemaVersion {
		return "", "", snapshot, fmt.Errorf("unsupported capability snapshot schema_version %d", snapshot.SchemaVersion)
	}
	if snapshot.CapabilityKey != capability.CapabilityKey || snapshot.Audience != capability.Audience {
		return "", "", snapshot, errors.New("capability snapshot identity does not match catalogue")
	}
	if !validAudience(snapshot.Audience) {
		return "", "", snapshot, errors.New("invalid capability audience")
	}
	var err error
	snapshot.ModelPolicy.AllowedLLMModelIDs, err = normalizeIDs(snapshot.ModelPolicy.AllowedLLMModelIDs)
	if err != nil {
		return "", "", snapshot, fmt.Errorf("allowed LLM model IDs: %w", err)
	}
	snapshot.ModelPolicy.AllowedEmbeddingModelIDs, err = normalizeIDs(snapshot.ModelPolicy.AllowedEmbeddingModelIDs)
	if err != nil {
		return "", "", snapshot, fmt.Errorf("allowed embedding model IDs: %w", err)
	}
	if snapshot.ModelPolicy.DefaultLLMModelID > 0 && !containsID(snapshot.ModelPolicy.AllowedLLMModelIDs, snapshot.ModelPolicy.DefaultLLMModelID) {
		return "", "", snapshot, errors.New("default LLM model must belong to the allowed model pool")
	}
	if snapshot.ModelPolicy.DefaultEmbeddingModelID > 0 && !containsID(snapshot.ModelPolicy.AllowedEmbeddingModelIDs, snapshot.ModelPolicy.DefaultEmbeddingModelID) {
		return "", "", snapshot, errors.New("default embedding model must belong to the allowed model pool")
	}
	refs := &snapshot.ConfigurationRef
	for label, ids := range map[string]*[]int64{
		"agent IDs":               &refs.AgentIDs,
		"prompt template IDs":     &refs.PromptTemplateIDs,
		"agent skill version IDs": &refs.AgentSkillVersionIDs,
		"MCP policy IDs":          &refs.MCPPolicyIDs,
	} {
		normalized, normalizeErr := normalizeIDs(*ids)
		if normalizeErr != nil {
			return "", "", snapshot, fmt.Errorf("%s: %w", label, normalizeErr)
		}
		*ids = normalized
	}
	if err := normalizeSkillRuntimePolicy(&snapshot.SkillRuntimePolicy); err != nil {
		return "", "", snapshot, err
	}
	normalized, err := json.Marshal(snapshot)
	if err != nil {
		return "", "", snapshot, err
	}
	sum := sha256.Sum256(normalized)
	return string(normalized), hex.EncodeToString(sum[:]), snapshot, nil
}

func normalizeSkillRuntimePolicy(policy *PlatformAISkillRuntimePolicy) error {
	policy.PolicyVersion = strings.TrimSpace(policy.PolicyVersion)
	if policy.PolicyVersion != PlatformAISkillPolicyVersion {
		return fmt.Errorf("unsupported skill runtime policy_version %q", policy.PolicyVersion)
	}
	if policy.MaxSkillTokens == 0 {
		policy.MaxSkillTokens = PlatformAIDefaultMaxSkillTokens
	}
	if policy.MaxSkillTokens < 1 || policy.MaxSkillTokens > platformAIMaxSkillTokens {
		return fmt.Errorf("skill runtime max_skill_tokens must be between 1 and %d", platformAIMaxSkillTokens)
	}
	if policy.MaxInputRatio == 0 {
		policy.MaxInputRatio = PlatformAIDefaultMaxSkillInputRatio
	}
	if policy.MaxInputRatio < 0 || policy.MaxInputRatio > platformAIMaxSkillInputRatio {
		return fmt.Errorf("skill runtime max_input_ratio must be greater than 0 and at most %.2f", platformAIMaxSkillInputRatio)
	}
	if policy.MaxSkills == 0 {
		policy.MaxSkills = PlatformAIDefaultMaxSkills
	}
	if policy.MaxSkills != platformAIFixedMaxSkills {
		return fmt.Errorf("skill runtime max_skills must be %d", platformAIFixedMaxSkills)
	}
	var err error
	policy.EvaluationSuiteHash, err = normalizeOptionalSHA256(policy.EvaluationSuiteHash)
	if err != nil {
		return fmt.Errorf("skill runtime evaluation_suite_hash: %w", err)
	}
	policy.EvaluationResultHash, err = normalizeOptionalSHA256(policy.EvaluationResultHash)
	if err != nil {
		return fmt.Errorf("skill runtime evaluation_result_hash: %w", err)
	}
	return nil
}

func normalizeOptionalSHA256(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if len(value) != sha256.Size*2 {
		return "", errors.New("must be a SHA-256 hex digest")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", errors.New("must be a SHA-256 hex digest")
	}
	return value, nil
}

func validatePublishedModelPolicy(tx *gorm.DB, policy PlatformAIModelPolicy) error {
	if len(policy.AllowedLLMModelIDs) == 0 || policy.DefaultLLMModelID <= 0 {
		return ErrCapabilityUnavailable
	}
	var count int64
	if err := tx.Table("llm_models model").Joins("JOIN llm_providers provider ON provider.id = model.provider_id AND provider.is_enabled = ?", true).
		Where("model.id IN ? AND model.is_enabled = ?", policy.AllowedLLMModelIDs, true).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(policy.AllowedLLMModelIDs)) {
		return errors.New("all released LLM models must be enabled and have an enabled provider")
	}
	if len(policy.AllowedEmbeddingModelIDs) > 0 {
		if policy.DefaultEmbeddingModelID <= 0 {
			return errors.New("an embedding default is required when an embedding pool is configured")
		}
		if err := tx.Table("embedding_models model").Joins("JOIN embedding_providers provider ON provider.id = model.provider_id AND provider.is_enabled = ?", true).
			Where("model.id IN ? AND model.is_enabled = ?", policy.AllowedEmbeddingModelIDs, true).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(policy.AllowedEmbeddingModelIDs)) {
			return errors.New("all released embedding models must be enabled and have an enabled provider")
		}
	}
	return nil
}

func validatePublishedConfigurationRefs(tx *gorm.DB, snapshot PlatformAICapabilitySnapshot) error {
	refs := snapshot.ConfigurationRef
	if capabilityRequiresAgentPrompt(snapshot.CapabilityKey) {
		if len(refs.AgentIDs) == 0 || len(refs.PromptTemplateIDs) == 0 {
			return errors.New("Agent-backed releases require at least one Agent and Prompt template")
		}
	}
	if snapshot.CapabilityKey == "ai.resume_parse" || snapshot.CapabilityKey == "ai.match_evaluation" {
		if len(refs.PromptTemplateIDs) == 0 {
			return errors.New("structured AI releases require at least one Prompt template")
		}
	}
	if len(refs.AgentSkillVersionIDs) > 0 && tx.Migrator().HasTable("agent_skill_versions") {
		var count int64
		if err := tx.Table("agent_skill_versions v").
			Joins("JOIN agent_skills s ON s.id = v.skill_id").
			Where("v.id IN ?", refs.AgentSkillVersionIDs).
			Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(refs.AgentSkillVersionIDs)) {
			return errors.New("all released agent skill versions must exist and belong to an Agent Skill registry")
		}
	}
	checks := []struct {
		label string
		table string
		base  string
		ids   []int64
		where string
	}{
		{"agents", "agent_configs", "agent_configs", refs.AgentIDs, "is_enabled = 1"},
		{"prompt templates", "prompt_templates", "prompt_templates", refs.PromptTemplateIDs, "is_active = 1"},
		{"MCP policies", "mcp_tool_policies p JOIN mcp_servers s ON s.id = p.server_id", "mcp_tool_policies", refs.MCPPolicyIDs, "p.is_enabled = 1 AND s.is_enabled = 1"},
	}
	for _, check := range checks {
		if len(check.ids) == 0 || !tx.Migrator().HasTable(check.base) {
			continue
		}
		var count int64
		query := tx.Table(check.table)
		switch check.label {
		case "MCP policies":
			query = query.Where("p.id IN ?", check.ids)
		default:
			query = query.Where("id IN ?", check.ids)
		}
		if err := query.Where(check.where).Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(check.ids)) {
			return fmt.Errorf("all released %s must exist and be enabled", check.label)
		}
	}
	if capabilityRequiresAgentPrompt(snapshot.CapabilityKey) && tx.Migrator().HasTable("agent_configs") && tx.Migrator().HasTable("prompt_templates") {
		var invalidBindings int64
		if err := tx.Table("agent_configs agent").
			Joins("LEFT JOIN prompt_templates prompt ON prompt.id = agent.prompt_template_id").
			Where("agent.id IN ?", refs.AgentIDs).
			Where(`agent.prompt_template_id IS NULL
				OR agent.prompt_template_id NOT IN ?
				OR prompt.id IS NULL
				OR prompt.is_active <> 1
				OR LOWER(TRIM(prompt.prompt_role)) <> 'system'
				OR NOT (
					LOWER(TRIM(prompt.agent_type)) = LOWER(TRIM(agent.agent_type))
					OR (LOWER(TRIM(agent.agent_type)) = 'hr_recruiting_agent' AND LOWER(TRIM(prompt.agent_type)) = 'hr_agent')
				)`, refs.PromptTemplateIDs).
			Count(&invalidBindings).Error; err != nil {
			return err
		}
		if invalidBindings > 0 {
			return errors.New("every released Agent must reference an active compatible Prompt included in the same capability release")
		}
	}
	return nil
}

func capabilityRequiresAgentPrompt(capabilityKey string) bool {
	switch strings.TrimSpace(capabilityKey) {
	case "ai.chat", "ai.agent_run", "ai.application_analysis":
		return true
	default:
		return false
	}
}

func (s *NativeStore) loadAvailableRuntimeModel(ctx context.Context, modelID int64) (runtimeModelRow, bool, error) {
	var row runtimeModelRow
	err := s.db.WithContext(ctx).Table("llm_models model").
		Select("model.id, model.provider_id, model.model_name, model.display_name, model.max_tokens, model.context_window_tokens, provider.name AS provider_name").
		Joins("JOIN llm_providers provider ON provider.id = model.provider_id AND provider.is_enabled = ?", true).
		Where("model.id = ? AND model.is_enabled = ?", modelID, true).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return runtimeModelRow{}, false, nil
	}
	return row, err == nil, err
}

func createPlatformAIConfigAudit(tx *gorm.DB, actorID int64, action, resourceType string, resourceID, capabilityID, versionID int64, before, after, requestID string) error {
	row := platformAIConfigAuditRecord{
		ActorUserID:         nullInt64From(actorID),
		Action:              action,
		ResourceType:        resourceType,
		ResourceID:          nullInt64From(resourceID),
		CapabilityID:        nullInt64From(capabilityID),
		CapabilityVersionID: nullInt64From(versionID),
		// JSON columns must receive SQL NULL when a snapshot is absent. An empty
		// string is not valid JSON in MySQL and causes Error 3140 on draft create
		// and publish audit records.
		BeforeSnapshot: nullableJSONText(before),
		AfterSnapshot:  nullableJSONText(after),
		RequestID:      nullStringFrom(requestID, true),
	}
	return tx.Create(&row).Error
}

func runtimeModelResolution(capability platformAICapabilityRecord, version platformAICapabilityVersionRecord, requestedID, effectiveID int64, fallbackReason string, model runtimeModelRow) RuntimeModelResolution {
	return RuntimeModelResolution{
		CapabilityKey:       capability.CapabilityKey,
		Audience:            capability.Audience,
		CapabilityVersionID: version.ID,
		SnapshotHash:        version.SnapshotHash,
		RequestedModelID:    requestedID,
		EffectiveModelID:    effectiveID,
		FallbackReason:      fallbackReason,
		ModelName:           model.ModelName,
		DisplayName:         model.DisplayName,
		ProviderID:          model.ProviderID,
		ProviderName:        model.ProviderName,
	}
}

func mapPlatformAICapability(row platformAICapabilityRecord) PlatformAICapability {
	return PlatformAICapability{ID: row.ID, CapabilityKey: row.CapabilityKey, Audience: row.Audience, Name: row.Name, Description: row.Description.String, Status: row.Status, CurrentPublishedVersionID: row.CurrentPublishedVersionID.Int64}
}

func mapPlatformAICapabilityVersion(row platformAICapabilityVersionRecord) PlatformAICapabilityVersion {
	return PlatformAICapabilityVersion{ID: row.ID, CapabilityID: row.CapabilityID, Version: row.Version, Status: row.Status, SnapshotJSON: row.SnapshotJSON, SnapshotHash: row.SnapshotHash, ChangeNote: row.ChangeNote.String, PublishedAt: row.PublishedAt}
}

func validAudience(audience string) bool {
	return audience == PlatformAIAudienceTenantHR || audience == PlatformAIAudienceCandidate
}

func normalizeIDs(ids []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("IDs must be positive")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func uniquePositiveIDs(ids []int64) []int64 {
	normalized, _ := normalizeIDs(ids)
	return normalized
}

func int64Set(ids []int64) map[int64]struct{} {
	result := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result
}

func containsID(ids []int64, target int64) bool {
	_, ok := int64Set(ids)[target]
	return ok
}

func nullableActor(value int64) interface{} {
	if value <= 0 {
		return nil
	}
	return value
}
