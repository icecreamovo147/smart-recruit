package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/application/agentskilleval"
	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

var (
	ErrAgentSkillReleaseEvaluationUnavailable = errors.New("Agent Skill release evaluation is unavailable")
	ErrAgentSkillReleaseEvaluationFailed      = errors.New("Agent Skill release evaluation did not pass")
)

type PlatformAIAgentSkillReleasePackage struct {
	SkillID   int64
	VersionID int64
	Package   agentskill.CompiledPackage
}

type PlatformAIAgentSkillReleaseEvaluationInput struct {
	CapabilityKey string
	Audience      string
	Policy        PlatformAISkillRuntimePolicy
	Packages      []PlatformAIAgentSkillReleasePackage
}

type PlatformAIAgentSkillReleaseEvaluationResult struct {
	Passed       bool
	SuiteVersion string
	SuiteHash    string
	ResultHash   string
	Cases        []agentskilleval.CaseResult
}

// PlatformAIAgentSkillReleaseEvaluator is the deterministic publication gate.
// Production publication intentionally fails closed until an evaluator is
// installed and returns hashes matching the immutable capability snapshot.
type PlatformAIAgentSkillReleaseEvaluator interface {
	EvaluateAgentSkillRelease(context.Context, PlatformAIAgentSkillReleaseEvaluationInput) (PlatformAIAgentSkillReleaseEvaluationResult, error)
}

type publishedAgentSkillVersionRow struct {
	ID                  int64          `gorm:"column:id"`
	SkillID             int64          `gorm:"column:skill_id"`
	Version             string         `gorm:"column:version"`
	ManifestJSON        string         `gorm:"column:manifest_json"`
	CoreMarkdown        string         `gorm:"column:core_markdown"`
	CompiledMarkdown    string         `gorm:"column:compiled_markdown"`
	AuthoringJSON       sql.NullString `gorm:"column:authoring_json"`
	CompiledHash        string         `gorm:"column:compiled_hash"`
	CoreEstimatedTokens int            `gorm:"column:core_estimated_tokens"`
	ChangeNote          sql.NullString `gorm:"column:change_note"`
	CreatedBy           sql.NullInt64  `gorm:"column:created_by"`
	CreatedAt           time.Time      `gorm:"column:created_at"`
	RegistryEnabled     bool           `gorm:"column:registry_enabled"`
}

func (row publishedAgentSkillVersionRow) versionRecord() agentSkillVersionRecord {
	return agentSkillVersionRecord{
		ID:                  row.ID,
		SkillID:             row.SkillID,
		Version:             row.Version,
		ManifestJSON:        row.ManifestJSON,
		CoreMarkdown:        row.CoreMarkdown,
		CompiledMarkdown:    row.CompiledMarkdown,
		AuthoringJSON:       row.AuthoringJSON,
		CompiledHash:        row.CompiledHash,
		CoreEstimatedTokens: row.CoreEstimatedTokens,
		ChangeNote:          row.ChangeNote,
		CreatedBy:           row.CreatedBy,
		CreatedAt:           row.CreatedAt,
	}
}

type releaseAgent struct {
	ID        int64  `gorm:"column:id"`
	AgentType string `gorm:"column:agent_type"`
}

type releaseAgentSkillSectionLists struct {
	triggerTerms   []string
	semanticTags   []string
	plannerIntents []string
}

func validatePublishedAgentSkillPackages(tx *gorm.DB, snapshot PlatformAICapabilitySnapshot) ([]PlatformAIAgentSkillReleasePackage, error) {
	versionIDs := snapshot.ConfigurationRef.AgentSkillVersionIDs
	if len(versionIDs) == 0 {
		return []PlatformAIAgentSkillReleasePackage{}, nil
	}

	rows, err := loadPublishedAgentSkillVersionRows(tx, versionIDs)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(versionIDs) {
		return nil, errors.New("all released Agent Skill versions must exist")
	}

	sectionsByVersion, err := loadReleaseAgentSkillSections(tx, versionIDs)
	if err != nil {
		return nil, err
	}
	agents, capabilities, err := loadReleaseAgentsAndCapabilities(tx, snapshot.ConfigurationRef.AgentIDs)
	if err != nil {
		return nil, err
	}

	packages := make([]PlatformAIAgentSkillReleasePackage, 0, len(rows))
	for _, row := range rows {
		if !row.RegistryEnabled {
			return nil, fmt.Errorf("released Agent Skill version %d belongs to a disabled registry", row.ID)
		}
		compiled, err := recompilePublishedAgentSkillPackage(row.versionRecord(), sectionsByVersion[row.ID])
		if err != nil {
			return nil, err
		}
		if err := validateAgentSkillReleaseCompatibility(snapshot, compiled.Manifest, agents, capabilities); err != nil {
			return nil, fmt.Errorf("released Agent Skill version %d: %w", row.ID, err)
		}
		packages = append(packages, PlatformAIAgentSkillReleasePackage{
			SkillID:   row.SkillID,
			VersionID: row.ID,
			Package:   *compiled,
		})
	}
	if err := validateAgentSkillReleaseComposition(packages); err != nil {
		return nil, err
	}
	return packages, nil
}

func loadPublishedAgentSkillVersionRows(tx *gorm.DB, versionIDs []int64) ([]publishedAgentSkillVersionRow, error) {
	var rows []publishedAgentSkillVersionRow
	if err := tx.Table("agent_skill_versions v").
		Select(
			"v.id AS id, v.skill_id AS skill_id, v.version AS version, v.manifest_json AS manifest_json, "+
				"v.core_markdown AS core_markdown, v.compiled_markdown AS compiled_markdown, "+
				"v.authoring_json AS authoring_json, v.compiled_hash AS compiled_hash, "+
				"v.core_estimated_tokens AS core_estimated_tokens, v.change_note AS change_note, "+
				"v.created_by AS created_by, v.created_at AS created_at, "+
				"s.is_enabled AS registry_enabled",
		).
		Joins("JOIN agent_skills s ON s.id = v.skill_id").
		Where("v.id IN ?", versionIDs).
		Order("v.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func loadReleaseAgentSkillSections(tx *gorm.DB, versionIDs []int64) (map[int64][]agentSkillSectionRecord, error) {
	result := make(map[int64][]agentSkillSectionRecord, len(versionIDs))
	var sections []agentSkillSectionRecord
	if err := tx.Where("skill_version_id IN ?", versionIDs).
		Order("skill_version_id ASC, ordinal ASC, section_key ASC, id ASC").
		Find(&sections).Error; err != nil {
		return nil, err
	}
	for _, section := range sections {
		result[section.SkillVersionID] = append(result[section.SkillVersionID], section)
	}
	return result, nil
}

func recompilePublishedAgentSkillPackage(row agentSkillVersionRecord, sections []agentSkillSectionRecord) (*agentskill.CompiledPackage, error) {
	var manifest agentskill.Manifest
	if err := json.Unmarshal([]byte(row.ManifestJSON), &manifest); err != nil {
		return nil, fmt.Errorf("released Agent Skill version %d has invalid manifest: %w", row.ID, err)
	}
	draft := agentskill.PackageDraft{
		Manifest: manifest,
		Core:     agentskill.Core{ContentMarkdown: row.CoreMarkdown},
		Sections: make([]agentskill.ReferenceSection, 0, len(sections)),
	}
	sectionLists := make([]releaseAgentSkillSectionLists, 0, len(sections))
	for _, section := range sections {
		triggerTerms, err := decodeReleaseAgentSkillSectionStringList(section.TriggerTermsJSON)
		if err != nil {
			return nil, fmt.Errorf(
				"released Agent Skill version %d section %q trigger_terms_json must be an array of strings: %w",
				row.ID,
				section.SectionKey,
				err,
			)
		}
		semanticTags, err := decodeReleaseAgentSkillSectionStringList(section.SemanticTagsJSON)
		if err != nil {
			return nil, fmt.Errorf(
				"released Agent Skill version %d section %q semantic_tags_json must be an array of strings: %w",
				row.ID,
				section.SectionKey,
				err,
			)
		}
		plannerIntents, err := decodeReleaseAgentSkillSectionStringList(section.PlannerIntentsJSON)
		if err != nil {
			return nil, fmt.Errorf(
				"released Agent Skill version %d section %q planner_intents_json must be an array of strings: %w",
				row.ID,
				section.SectionKey,
				err,
			)
		}
		sectionLists = append(sectionLists, releaseAgentSkillSectionLists{
			triggerTerms:   triggerTerms,
			semanticTags:   semanticTags,
			plannerIntents: plannerIntents,
		})
		draft.Sections = append(draft.Sections, agentskill.ReferenceSection{
			SectionKey:      section.SectionKey,
			Title:           section.Title,
			Description:     nullString(section.Description),
			ContentMarkdown: section.ContentMarkdown,
			TriggerTerms:    triggerTerms,
			SemanticTags:    semanticTags,
			PlannerIntents:  plannerIntents,
			Priority:        section.Priority,
			Ordinal:         section.Ordinal,
		})
	}
	compiled, err := agentskill.Compile(draft)
	if err != nil {
		return nil, fmt.Errorf("released Agent Skill version %d does not compile: %w", row.ID, err)
	}
	if compiled.ManifestJSON != row.ManifestJSON ||
		compiled.Core.ContentMarkdown != row.CoreMarkdown ||
		compiled.CompiledMarkdown != row.CompiledMarkdown ||
		compiled.CompiledHash != row.CompiledHash ||
		compiled.Core.EstimatedTokens != row.CoreEstimatedTokens {
		return nil, fmt.Errorf("released Agent Skill version %d package hash or canonical content does not match persisted data", row.ID)
	}
	if len(compiled.Sections) != len(sections) {
		return nil, fmt.Errorf("released Agent Skill version %d section set does not match persisted data", row.ID)
	}
	for index, expected := range compiled.Sections {
		persisted := sections[index]
		lists := sectionLists[index]
		if expected.SectionKey != persisted.SectionKey ||
			expected.Title != persisted.Title ||
			expected.Description != nullString(persisted.Description) ||
			expected.ContentMarkdown != persisted.ContentMarkdown ||
			!equalStrings(expected.TriggerTerms, lists.triggerTerms) ||
			!equalStrings(expected.SemanticTags, lists.semanticTags) ||
			!equalStrings(expected.PlannerIntents, lists.plannerIntents) ||
			expected.Priority != persisted.Priority ||
			expected.Ordinal != persisted.Ordinal ||
			expected.EstimatedTokens != persisted.EstimatedTokens ||
			expected.ContentHash != persisted.ContentHash {
			return nil, fmt.Errorf("released Agent Skill version %d section %q does not match its compiled package", row.ID, persisted.SectionKey)
		}
	}
	return compiled, nil
}

func decodeReleaseAgentSkillSectionStringList(value sql.NullString) ([]string, error) {
	if !value.Valid {
		return []string{}, nil
	}
	raw := strings.TrimSpace(value.String)
	if raw == "" || raw == "null" {
		return nil, errors.New("value is null or empty")
	}
	var encodedItems []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &encodedItems); err != nil {
		return nil, err
	}
	if encodedItems == nil {
		return nil, errors.New("value is null")
	}
	items := make([]string, 0, len(encodedItems))
	for index, encodedItem := range encodedItems {
		var item any
		if err := json.Unmarshal(encodedItem, &item); err != nil {
			return nil, fmt.Errorf("item %d: %w", index, err)
		}
		stringItem, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("item %d is not a string", index)
		}
		items = append(items, stringItem)
	}
	return items, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func loadReleaseAgentsAndCapabilities(tx *gorm.DB, agentIDs []int64) ([]releaseAgent, map[int64]map[string]bool, error) {
	if len(agentIDs) == 0 {
		return nil, map[int64]map[string]bool{}, nil
	}
	var agents []releaseAgent
	if err := tx.Table("agent_configs").
		Select("id, agent_type").
		Where("id IN ? AND is_enabled = ?", agentIDs, true).
		Order("id ASC").
		Scan(&agents).Error; err != nil {
		return nil, nil, err
	}
	if len(agents) != len(agentIDs) {
		return nil, nil, errors.New("all released Agents must exist and be enabled")
	}
	capabilities := make(map[int64]map[string]bool, len(agentIDs))
	var bindings []agentCapabilityBindingRecord
	if err := tx.Where("agent_id IN ? AND is_enabled = ?", agentIDs, true).
		Order("agent_id ASC, capability_key ASC").
		Find(&bindings).Error; err != nil {
		return nil, nil, err
	}
	for _, binding := range bindings {
		if capabilities[binding.AgentID] == nil {
			capabilities[binding.AgentID] = make(map[string]bool)
		}
		capabilities[binding.AgentID][strings.TrimSpace(binding.CapabilityKey)] = true
	}
	return agents, capabilities, nil
}

func validateAgentSkillReleaseCompatibility(
	snapshot PlatformAICapabilitySnapshot,
	manifest agentskill.Manifest,
	agents []releaseAgent,
	capabilities map[int64]map[string]bool,
) error {
	if structuredTypes := structuredAgentTypesForCapability(snapshot.CapabilityKey); len(structuredTypes) > 0 {
		if !structuredTypes[manifest.AgentType] {
			return fmt.Errorf("agent type %q is incompatible with capability %q", manifest.AgentType, snapshot.CapabilityKey)
		}
		if len(manifest.RequiredCapabilities) > 0 {
			return errors.New("structured Agent Skills cannot require Agent capability bindings")
		}
		return nil
	}

	for _, agent := range agents {
		if strings.TrimSpace(agent.AgentType) != manifest.AgentType {
			continue
		}
		if containsAllCapabilities(capabilities[agent.ID], manifest.RequiredCapabilities) {
			return nil
		}
	}
	return fmt.Errorf("no released Agent of type %q satisfies required capabilities", manifest.AgentType)
}

func containsAllCapabilities(available map[string]bool, required []string) bool {
	for _, key := range required {
		if !available[strings.TrimSpace(key)] {
			return false
		}
	}
	return true
}

func structuredAgentTypesForCapability(capabilityKey string) map[string]bool {
	switch strings.TrimSpace(capabilityKey) {
	case "ai.resume_parse":
		return map[string]bool{"resume_profile_extractor": true}
	case "ai.match_evaluation":
		return map[string]bool{
			"job_requirement_extractor": true,
			"candidate_match_evaluator": true,
		}
	default:
		return nil
	}
}

func validateAgentSkillReleaseComposition(packages []PlatformAIAgentSkillReleasePackage) error {
	type count struct {
		primary    int
		supporting int
	}
	byScenario := make(map[string]count)
	structuredPrimary := make(map[string]int)
	for _, item := range packages {
		manifest := item.Package.Manifest
		key := manifest.AgentType + "\x00" + manifest.Scenario
		current := byScenario[key]
		switch manifest.Composition.Role {
		case agentskill.CompositionRolePrimary:
			current.primary++
			if structuredAgentType(manifest.AgentType) {
				structuredPrimary[manifest.AgentType]++
			}
		case agentskill.CompositionRoleSupporting:
			current.supporting++
		default:
			return fmt.Errorf("released Agent Skill version %d has invalid composition role", item.VersionID)
		}
		byScenario[key] = current
	}
	for key, current := range byScenario {
		if current.primary > 1 || current.supporting > 1 {
			parts := strings.SplitN(key, "\x00", 2)
			return fmt.Errorf("Agent Skill composition for agent type %q scenario %q exceeds one Primary plus one Supporting", parts[0], parts[1])
		}
		if current.supporting == 1 && current.primary != 1 {
			parts := strings.SplitN(key, "\x00", 2)
			return fmt.Errorf("Agent Skill composition for agent type %q scenario %q requires exactly one Primary when a Supporting Skill is published", parts[0], parts[1])
		}
	}
	for agentType, total := range structuredPrimary {
		if total > 1 {
			return fmt.Errorf("structured agent type %q cannot publish more than one Primary Skill", agentType)
		}
	}
	return nil
}

func structuredAgentType(agentType string) bool {
	switch strings.TrimSpace(agentType) {
	case "resume_profile_extractor", "job_requirement_extractor", "candidate_match_evaluator":
		return true
	default:
		return false
	}
}

func (s *NativeStore) validateAgentSkillReleaseEvaluation(
	ctx context.Context,
	snapshot PlatformAICapabilitySnapshot,
	packages []PlatformAIAgentSkillReleasePackage,
) error {
	policy := snapshot.SkillRuntimePolicy
	if policy.EvaluationSuiteHash == "" || policy.EvaluationResultHash == "" {
		return fmt.Errorf("%w: evaluation hashes are required", ErrAgentSkillReleaseEvaluationUnavailable)
	}
	result, err := s.evaluateAgentSkillRelease(ctx, snapshot, packages)
	if err != nil {
		return err
	}
	if result.SuiteHash != policy.EvaluationSuiteHash || result.ResultHash != policy.EvaluationResultHash {
		return fmt.Errorf("%w: evaluator hashes do not match the immutable snapshot", ErrAgentSkillReleaseEvaluationFailed)
	}
	return nil
}

func (s *NativeStore) evaluateAgentSkillRelease(
	ctx context.Context,
	snapshot PlatformAICapabilitySnapshot,
	packages []PlatformAIAgentSkillReleasePackage,
) (PlatformAIAgentSkillReleaseEvaluationResult, error) {
	if s.agentSkillReleaseEvaluator == nil {
		return PlatformAIAgentSkillReleaseEvaluationResult{}, ErrAgentSkillReleaseEvaluationUnavailable
	}
	result, err := s.agentSkillReleaseEvaluator.EvaluateAgentSkillRelease(ctx, PlatformAIAgentSkillReleaseEvaluationInput{
		CapabilityKey: snapshot.CapabilityKey,
		Audience:      snapshot.Audience,
		Policy:        policy,
		Packages:      packages,
	})
	if err != nil {
		return PlatformAIAgentSkillReleaseEvaluationResult{}, fmt.Errorf("%w: %v", ErrAgentSkillReleaseEvaluationUnavailable, err)
	}
	suiteHash, suiteErr := normalizeOptionalSHA256(result.SuiteHash)
	resultHash, resultErr := normalizeOptionalSHA256(result.ResultHash)
	if suiteErr != nil || resultErr != nil || suiteHash == "" || resultHash == "" {
		return PlatformAIAgentSkillReleaseEvaluationResult{}, fmt.Errorf("%w: evaluator returned invalid hashes", ErrAgentSkillReleaseEvaluationUnavailable)
	}
	if !result.Passed {
		return PlatformAIAgentSkillReleaseEvaluationResult{}, ErrAgentSkillReleaseEvaluationFailed
	}
	result.SuiteHash = suiteHash
	result.ResultHash = resultHash
	return result, nil
}

func (s *NativeStore) prepareCapabilityDraftSnapshot(
	ctx context.Context,
	tx *gorm.DB,
	snapshotJSON []byte,
	capability platformAICapabilityRecord,
) (string, string, error) {
	_, _, snapshot, err := normalizeCapabilitySnapshot(snapshotJSON, capability)
	if err != nil {
		return "", "", err
	}
	if err := validatePublishedModelPolicy(tx, snapshot.ModelPolicy); err != nil {
		return "", "", err
	}
	if err := validatePublishedConfigurationRefs(tx, snapshot); err != nil {
		return "", "", err
	}
	packages, err := validatePublishedAgentSkillPackages(tx, snapshot)
	if err != nil {
		return "", "", err
	}
	result, err := s.evaluateAgentSkillRelease(ctx, snapshot, packages)
	if err != nil {
		return "", "", err
	}
	snapshot.SkillRuntimePolicy.EvaluationSuiteHash = result.SuiteHash
	snapshot.SkillRuntimePolicy.EvaluationResultHash = result.ResultHash
	prepared, err := json.Marshal(snapshot)
	if err != nil {
		return "", "", err
	}
	normalized, hash, _, err := normalizeCapabilitySnapshot(prepared, capability)
	return normalized, hash, err
}
