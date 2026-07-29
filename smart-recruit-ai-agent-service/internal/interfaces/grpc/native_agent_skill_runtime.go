package grpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-ai-agent-service/internal/application/contextbudget"
	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	"smart-recruit-proto/recruitment/pb"
)

const (
	defaultAgentSkillMaxTokens = 3000
	defaultAgentSkillMaxRatio  = 0.15
	defaultAgentSkillMaxCount  = 2
)

type hrRuntimeAgentSkillPackageStore interface {
	LoadAgentSkillRuntimePackages(context.Context, []int64) ([]embeddinginfra.AgentSkillRuntimePackage, error)
}

type hrRuntimeAgentSkillSection struct {
	ID              int64
	Key             string
	Title           string
	ContentMarkdown string
	ContentHash     string
	EstimatedTokens int
	FinalRankScore  float64
	Included        bool
	DecisionReason  string
}

type hrRuntimeAgentSkill struct {
	ID                   int64
	VersionID            int64
	Version              string
	CompiledHash         string
	Name                 string
	DisplayName          string
	CoreMarkdown         string
	CoreEstimatedTokens  int
	LoadedTokens         int
	Manual               bool
	Reason               string
	RelevanceMode        string
	CompositionRole      domainagentskill.CompositionRole
	RiskLevel            domainagentskill.RiskLevel
	ActivationPolicy     domainagentskill.ActivationPolicy
	RequiredCapabilities []string
	EvaluationCriteria   []string
	OutputContract       domainagentskill.OutputContract
	AdvisoryInstruction  string
	Sections             []hrRuntimeAgentSkillSection
	AvailableSections    []embeddinginfra.AgentSkillSectionEmbeddingDocument
	Included             bool
	DecisionReason       string
}

type hrRankedAgentSkillVersion struct {
	Document embeddinginfra.AgentSkillVersionEmbeddingDocument
	Ranking  domainagentskill.RankedDocument
	Manual   bool
}

func validateHRRuntimeAgentSkillPackage(runtimePackage embeddinginfra.AgentSkillRuntimePackage) (
	embeddinginfra.AgentSkillVersionEmbeddingDocument,
	[]embeddinginfra.AgentSkillSectionEmbeddingDocument,
	error,
) {
	manifest, storedManifestCanonical, err := domainagentskill.DecodeManifestJSON([]byte(runtimePackage.ManifestJSON))
	if err != nil {
		return embeddinginfra.AgentSkillVersionEmbeddingDocument{}, nil, fmt.Errorf("decode manifest: %w", err)
	}
	draft := domainagentskill.PackageDraft{
		Manifest: manifest,
		Core:     domainagentskill.Core{ContentMarkdown: runtimePackage.CoreMarkdown},
		Sections: make([]domainagentskill.ReferenceSection, 0, len(runtimePackage.Sections)),
	}
	for _, section := range runtimePackage.Sections {
		draft.Sections = append(draft.Sections, domainagentskill.ReferenceSection{
			SectionKey:      section.SectionKey,
			Title:           section.Title,
			Description:     section.Description,
			ContentMarkdown: section.ContentMarkdown,
			TriggerTerms:    append([]string(nil), section.TriggerTerms...),
			SemanticTags:    append([]string(nil), section.SemanticTags...),
			PlannerIntents:  append([]string(nil), section.PlannerIntents...),
			Priority:        section.Priority,
			Ordinal:         section.Ordinal,
		})
	}
	compiled, err := domainagentskill.Compile(draft)
	if err != nil {
		return embeddinginfra.AgentSkillVersionEmbeddingDocument{}, nil, fmt.Errorf("compile package: %w", err)
	}
	_, compiledManifestCanonical, err := domainagentskill.DecodeManifestJSON([]byte(compiled.ManifestJSON))
	if err != nil {
		return embeddinginfra.AgentSkillVersionEmbeddingDocument{}, nil, fmt.Errorf("canonicalize compiled manifest: %w", err)
	}
	if runtimePackage.ID <= 0 ||
		runtimePackage.SkillID <= 0 ||
		!bytes.Equal(storedManifestCanonical, compiledManifestCanonical) ||
		runtimePackage.CoreMarkdown != compiled.Core.ContentMarkdown ||
		runtimePackage.CompiledMarkdown != compiled.CompiledMarkdown ||
		runtimePackage.CompiledHash != compiled.CompiledHash ||
		runtimePackage.CoreEstimatedTokens != compiled.Core.EstimatedTokens ||
		len(runtimePackage.Sections) != len(compiled.Sections) {
		return embeddinginfra.AgentSkillVersionEmbeddingDocument{}, nil, fmt.Errorf("package canonical fields do not match compiled content")
	}
	sections := make([]embeddinginfra.AgentSkillSectionEmbeddingDocument, 0, len(compiled.Sections))
	for i, expected := range compiled.Sections {
		stored := runtimePackage.Sections[i]
		if stored.ID <= 0 ||
			stored.SectionKey != expected.SectionKey ||
			stored.Title != expected.Title ||
			stored.Description != expected.Description ||
			stored.ContentMarkdown != expected.ContentMarkdown ||
			!slices.Equal(stored.TriggerTerms, expected.TriggerTerms) ||
			!slices.Equal(stored.SemanticTags, expected.SemanticTags) ||
			!slices.Equal(stored.PlannerIntents, expected.PlannerIntents) ||
			stored.Priority != expected.Priority ||
			stored.Ordinal != expected.Ordinal ||
			stored.EstimatedTokens != expected.EstimatedTokens ||
			stored.ContentHash != expected.ContentHash {
			return embeddinginfra.AgentSkillVersionEmbeddingDocument{}, nil, fmt.Errorf("section %d does not match compiled content", stored.ID)
		}
		sections = append(sections, embeddinginfra.AgentSkillSectionEmbeddingDocument{
			ID:              stored.ID,
			SkillID:         runtimePackage.SkillID,
			VersionID:       runtimePackage.ID,
			Version:         runtimePackage.Version,
			CompiledHash:    runtimePackage.CompiledHash,
			SectionKey:      stored.SectionKey,
			Title:           stored.Title,
			Description:     stored.Description,
			ContentMarkdown: stored.ContentMarkdown,
			TriggerTerms:    append([]string(nil), stored.TriggerTerms...),
			SemanticTags:    append([]string(nil), stored.SemanticTags...),
			PlannerIntents:  append([]string(nil), stored.PlannerIntents...),
			Priority:        stored.Priority,
			EstimatedTokens: stored.EstimatedTokens,
			ContentHash:     stored.ContentHash,
		})
	}
	return embeddinginfra.AgentSkillVersionEmbeddingDocument{
		ID:              runtimePackage.ID,
		SkillID:         runtimePackage.SkillID,
		Version:         runtimePackage.Version,
		CompiledHash:    runtimePackage.CompiledHash,
		Manifest:        compiled.Manifest,
		CoreMarkdown:    compiled.Core.ContentMarkdown,
		Enabled:         runtimePackage.Enabled,
		ManualInvocable: runtimePackage.ManualInvocable,
	}, sections, nil
}

func canonicalizeRuntimeJSON(raw string) (string, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return "", fmt.Errorf("unexpected trailing JSON value")
		}
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

func (s *nativeAIService) selectHRRuntimeAgentSkillPackages(
	ctx context.Context,
	req *pb.ChatRequest,
	capabilityKeys []string,
	model RuntimeModelInfo,
	enforceRelease bool,
) ([]hrRuntimeAgentSkill, []*pb.AgentSkillRuntimeEvidence, bool, []hrRuntimeGovernanceError) {
	requestedVersionIDs := req.GetAgentSkillVersionIds()
	manualVersionIDs := positiveUniqueRuntimeIDs(requestedVersionIDs)
	if len(requestedVersionIDs) != len(manualVersionIDs) {
		return nil, nil, false, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "manual_selection_invalid"}}
	}
	allowedVersionIDs := positiveUniqueRuntimeIDs(model.ConfigurationRefs.AgentSkillVersionIDs)
	if !enforceRelease {
		if len(manualVersionIDs) > 0 {
			return nil, nil, false, outsideReleaseSkillErrors(manualVersionIDs)
		}
		return nil, nil, false, nil
	}
	if len(allowedVersionIDs) == 0 {
		return nil, nil, false, outsideReleaseSkillErrors(manualVersionIDs)
	}
	store, ok := s.store.(hrRuntimeAgentSkillPackageStore)
	if !ok {
		return nil, nil, false, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "package_store_unavailable"}}
	}

	allowedSet := int64RuntimeSet(allowedVersionIDs)
	if len(manualVersionIDs) > 0 {
		for _, id := range manualVersionIDs {
			if !allowedSet[id] {
				return nil, nil, false, outsideReleaseSkillErrors(manualVersionIDs)
			}
		}
	}

	packages, err := store.LoadAgentSkillRuntimePackages(ctx, allowedVersionIDs)
	if err != nil {
		return nil, nil, false, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "package_lookup_failed"}}
	}
	documents := make([]embeddinginfra.AgentSkillVersionEmbeddingDocument, 0, len(packages))
	documentByID := make(map[int64]embeddinginfra.AgentSkillVersionEmbeddingDocument, len(documents))
	sectionsByVersion := make(map[int64][]embeddinginfra.AgentSkillSectionEmbeddingDocument, len(packages))
	loadedVersionIDs := make(map[int64]bool, len(packages))
	for _, runtimePackage := range packages {
		if !allowedSet[runtimePackage.ID] {
			continue
		}
		loadedVersionIDs[runtimePackage.ID] = true
		document, sections, validationErr := validateHRRuntimeAgentSkillPackage(runtimePackage)
		if validationErr != nil {
			evidence := []*pb.AgentSkillRuntimeEvidence{{
				SkillId:        runtimePackage.SkillID,
				VersionId:      runtimePackage.ID,
				Version:        runtimePackage.Version,
				CompiledHash:   runtimePackage.CompiledHash,
				Included:       false,
				DecisionReason: "package_integrity_failed",
			}}
			return nil, evidence, false, []hrRuntimeGovernanceError{{
				Source:     "agent_skill",
				Code:       "package_integrity_failed",
				ResourceID: runtimePackage.ID,
			}}
		}
		documents = append(documents, document)
		documentByID[document.ID] = document
		sectionsByVersion[document.ID] = sections
	}
	for _, versionID := range allowedVersionIDs {
		if !loadedVersionIDs[versionID] {
			return nil, []*pb.AgentSkillRuntimeEvidence{{
					VersionId:      versionID,
					Included:       false,
					DecisionReason: "version_unavailable",
				}}, false, []hrRuntimeGovernanceError{{
					Source:     "agent_skill",
					Code:       "version_unavailable",
					ResourceID: versionID,
				}}
		}
	}

	availableCapabilities := hrRuntimeAvailableCapabilities(capabilityKeys, req.GetCapabilityKeys())
	maxSkills := effectiveAgentSkillMaxCount(model.SkillRuntimePolicy)
	var ranked []hrRankedAgentSkillVersion
	if len(manualVersionIDs) > 0 {
		if len(manualVersionIDs) > maxSkills {
			return nil, nil, false, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "manual_selection_limit_exceeded"}}
		}
		for _, versionID := range manualVersionIDs {
			document, exists := documentByID[versionID]
			if !exists {
				return nil, nil, false, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "version_unavailable", ResourceID: versionID}}
			}
			ranked = append(ranked, hrRankedAgentSkillVersion{Document: document, Manual: true})
		}
	} else {
		ranked = s.searchHRRuntimeAgentSkillVersions(ctx, req.GetMessage(), documents, allowedVersionIDs)
	}

	eligible := make([]hrRankedAgentSkillVersion, 0, len(ranked))
	var evidence []*pb.AgentSkillRuntimeEvidence
	for _, candidate := range ranked {
		reason := hrRuntimeAgentSkillEligibilityReason(candidate.Document, candidate.Manual, availableCapabilities)
		if reason == "" && !candidate.Manual && candidate.Document.Manifest.RiskLevel == domainagentskill.RiskLevelCritical {
			reason = "critical_auto_excluded"
		}
		if reason != "" {
			evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(candidate, false, reason))
			if candidate.Manual {
				return nil, evidence, false, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: reason, ResourceID: candidate.Document.ID}}
			}
			continue
		}
		eligible = append(eligible, candidate)
	}

	selected, selectionEvidence, selectionErrors := composeHRRuntimeAgentSkills(eligible, maxSkills, len(manualVersionIDs) > 0)
	evidence = append(evidence, selectionEvidence...)
	if len(selectionErrors) > 0 {
		return nil, evidence, false, selectionErrors
	}
	if len(selected) == 0 {
		return nil, evidence, false, nil
	}
	if primary, ok := primaryHRRuntimeAgentSkill(selected); ok {
		switch primary.OutputContract.Mode {
		case domainagentskill.OutputModeStrict:
			primary.DecisionReason = "strict_output_contract_unsupported"
			evidence = append(evidence, hrRuntimeAgentSkillEvidenceToPB(*primary))
			return nil, evidence, false, []hrRuntimeGovernanceError{{
				Source:     "agent_skill",
				Code:       "strict_output_contract_unsupported",
				ResourceID: primary.VersionID,
			}}
		case domainagentskill.OutputModeAdvisory:
			instruction, instructionErr := compileHRRuntimeAdvisoryInstruction(primary.OutputContract)
			if instructionErr != nil {
				primary.DecisionReason = "advisory_output_contract_invalid"
				evidence = append(evidence, hrRuntimeAgentSkillEvidenceToPB(*primary))
				return nil, evidence, false, []hrRuntimeGovernanceError{{
					Source:     "agent_skill",
					Code:       "advisory_output_contract_invalid",
					ResourceID: primary.VersionID,
				}}
			}
			primary.AdvisoryInstruction = instruction
		}
	}
	for i := range selected {
		selected[i].AvailableSections = append(
			[]embeddinginfra.AgentSkillSectionEmbeddingDocument(nil),
			sectionsByVersion[selected[i].VersionID]...,
		)
	}

	budget := effectiveAgentSkillBudget(model)
	remaining := budget
	confirmationRequired := false
	for i := range selected {
		if selected[i].RiskLevel == domainagentskill.RiskLevelHigh || selected[i].RiskLevel == domainagentskill.RiskLevelCritical {
			confirmationRequired = true
			break
		}
	}
	if confirmationRequired && !agentSkillApprovalAllows(ctx, selected, model, req.GetMessage()) {
		for i := range selected {
			selected[i].DecisionReason = "blocked_by_confirmation"
			if selected[i].RiskLevel == domainagentskill.RiskLevelHigh || selected[i].RiskLevel == domainagentskill.RiskLevelCritical {
				selected[i].DecisionReason = "confirmation_required"
			}
			evidence = append(evidence, hrRuntimeAgentSkillEvidenceToPB(selected[i]))
		}
		return nil, evidence, true, nil
	}
	included := make([]hrRuntimeAgentSkill, 0, len(selected))
	for i := range selected {
		skill := &selected[i]
		instructionTokens := contextbudget.EstimateTokensConservative(skill.AdvisoryInstruction)
		requiredTokens := skill.CoreEstimatedTokens + instructionTokens
		if requiredTokens > remaining {
			skill.DecisionReason = "core_budget_exceeded"
			evidence = append(evidence, hrRuntimeAgentSkillEvidenceToPB(*skill))
			if skill.CompositionRole == domainagentskill.CompositionRolePrimary {
				for j := i + 1; j < len(selected); j++ {
					selected[j].Included = false
					selected[j].DecisionReason = "blocked_by_primary_budget"
					evidence = append(evidence, hrRuntimeAgentSkillEvidenceToPB(selected[j]))
				}
				return nil, evidence, false, nil
			}
			continue
		}
		skill.Included = true
		skill.DecisionReason = "core_included"
		skill.LoadedTokens = requiredTokens
		remaining -= requiredTokens
		included = append(included, *skill)
	}
	if len(included) == 0 {
		return nil, evidence, false, nil
	}

	included = s.loadHRRuntimeAgentSkillSections(ctx, req.GetMessage(), included, remaining)
	includedEvidence := make(map[int64]*pb.AgentSkillRuntimeEvidence, len(included))
	for _, skill := range included {
		item := hrRuntimeAgentSkillEvidenceToPB(skill)
		includedEvidence[skill.VersionID] = item
	}
	for _, item := range evidence {
		if item != nil && includedEvidence[item.GetVersionId()] != nil {
			delete(includedEvidence, item.GetVersionId())
		}
	}
	for _, skill := range included {
		if item := includedEvidence[skill.VersionID]; item != nil {
			evidence = append(evidence, item)
		}
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		return evidence[i].GetVersionId() < evidence[j].GetVersionId()
	})
	return included, evidence, false, nil
}

func (s *nativeAIService) searchHRRuntimeAgentSkillVersions(
	ctx context.Context,
	query string,
	documents []embeddinginfra.AgentSkillVersionEmbeddingDocument,
	allowedVersionIDs []int64,
) []hrRankedAgentSkillVersion {
	allowed := int64RuntimeSet(allowedVersionIDs)
	documentByID := make(map[int64]embeddinginfra.AgentSkillVersionEmbeddingDocument, len(documents))
	for _, document := range documents {
		if allowed[document.ID] {
			documentByID[document.ID] = document
		}
	}
	if s.embedding != nil {
		result, err := s.embedding.SearchAgentSkillVersions(ctx, query, allowedVersionIDs, 500)
		if err == nil && result != nil {
			out := make([]hrRankedAgentSkillVersion, 0, len(result.Items))
			for _, item := range result.Items {
				if document, exists := documentByID[item.Document.ID]; exists {
					out = append(out, hrRankedAgentSkillVersion{Document: document, Ranking: item.Ranking})
				}
			}
			return out
		}
	}
	rankingDocuments := make([]domainagentskill.RankingDocument, 0, len(documents))
	for _, document := range documents {
		if !allowed[document.ID] {
			continue
		}
		documentByID[document.ID] = document
		rankingDocuments = append(rankingDocuments, hrRuntimeVersionRankingDocument(document))
	}
	ranked := domainagentskill.RankDocuments(query, rankingDocuments, false)
	out := make([]hrRankedAgentSkillVersion, 0, len(ranked))
	for _, item := range ranked {
		if document, exists := documentByID[item.Document.VersionID]; exists {
			out = append(out, hrRankedAgentSkillVersion{Document: document, Ranking: item})
		}
	}
	return out
}

func hrRuntimeVersionRankingDocument(document embeddinginfra.AgentSkillVersionEmbeddingDocument) domainagentskill.RankingDocument {
	manifest := document.Manifest
	return domainagentskill.RankingDocument{
		ObjectID:  document.ID,
		SkillID:   document.SkillID,
		VersionID: document.ID,
		LexicalText: []string{
			manifest.SkillName, manifest.DisplayName, manifest.Description,
			manifest.Category, manifest.Scenario, document.CoreMarkdown,
		},
		MetadataTerms: append(append([]string{}, manifest.TriggerKeywords...), manifest.SemanticTags...),
		Priority:      manifest.Priority,
	}
}

func hrRuntimeAgentSkillEligibilityReason(document embeddinginfra.AgentSkillVersionEmbeddingDocument, manual bool, capabilities map[string]bool) string {
	if document.ID <= 0 || document.SkillID <= 0 || !validSHA256Hex(document.CompiledHash) || strings.TrimSpace(document.CoreMarkdown) == "" {
		return "package_invalid"
	}
	if !document.Enabled {
		return "skill_disabled"
	}
	if manual && !document.ManualInvocable {
		return "manual_invocation_disabled"
	}
	if !strings.EqualFold(strings.TrimSpace(document.Manifest.AgentType), hrRecruitingAgentType) {
		return "agent_type_mismatch"
	}
	if !validHRRuntimeAgentSkillManifest(document.Manifest) {
		return "package_invalid"
	}
	for _, required := range document.Manifest.RequiredCapabilities {
		if !capabilities[strings.TrimSpace(required)] {
			return "required_capability_missing"
		}
	}
	return ""
}

func validHRRuntimeAgentSkillManifest(manifest domainagentskill.Manifest) bool {
	switch manifest.Composition.Role {
	case domainagentskill.CompositionRolePrimary, domainagentskill.CompositionRoleSupporting:
	default:
		return false
	}
	switch manifest.RiskLevel {
	case domainagentskill.RiskLevelLow, domainagentskill.RiskLevelMedium:
		return manifest.ActivationPolicy == domainagentskill.ActivationPolicyAuto
	case domainagentskill.RiskLevelHigh:
		return manifest.ActivationPolicy == domainagentskill.ActivationPolicyConfirm
	case domainagentskill.RiskLevelCritical:
		return manifest.ActivationPolicy == domainagentskill.ActivationPolicyManualOnly
	default:
		return false
	}
}

func composeHRRuntimeAgentSkills(candidates []hrRankedAgentSkillVersion, maxSkills int, manual bool) ([]hrRuntimeAgentSkill, []*pb.AgentSkillRuntimeEvidence, []hrRuntimeGovernanceError) {
	if maxSkills < 1 {
		return nil, nil, nil
	}
	ordered := append([]hrRankedAgentSkillVersion(nil), candidates...)
	if manual {
		sort.SliceStable(ordered, func(i, j int) bool {
			leftRole := hrRuntimeCompositionRoleOrder(ordered[i].Document.Manifest.Composition.Role)
			rightRole := hrRuntimeCompositionRoleOrder(ordered[j].Document.Manifest.Composition.Role)
			if leftRole != rightRole {
				return leftRole < rightRole
			}
			return ordered[i].Document.ID < ordered[j].Document.ID
		})
	}

	var primary *hrRankedAgentSkillVersion
	supportingCandidates := make([]hrRankedAgentSkillVersion, 0, len(ordered))
	var evidence []*pb.AgentSkillRuntimeEvidence
	for i := range ordered {
		candidate := ordered[i]
		switch candidate.Document.Manifest.Composition.Role {
		case domainagentskill.CompositionRolePrimary:
			if primary == nil {
				primary = &candidate
			} else if manual {
				evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(candidate, false, "composition_conflict"))
				return nil, evidence, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "composition_conflict", ResourceID: candidate.Document.ID}}
			} else {
				evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(candidate, false, "primary_limit_exceeded"))
			}
		case domainagentskill.CompositionRoleSupporting:
			if manual && len(supportingCandidates) > 0 {
				evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(candidate, false, "composition_conflict"))
				return nil, evidence, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "composition_conflict", ResourceID: candidate.Document.ID}}
			}
			supportingCandidates = append(supportingCandidates, candidate)
		default:
			if manual {
				evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(candidate, false, "composition_role_invalid"))
				return nil, evidence, []hrRuntimeGovernanceError{{Source: "agent_skill", Code: "composition_role_invalid", ResourceID: candidate.Document.ID}}
			}
		}
	}
	if primary == nil {
		for _, supporting := range supportingCandidates {
			evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(supporting, false, "supporting_requires_primary"))
		}
		if manual && len(supportingCandidates) > 0 {
			return nil, evidence, []hrRuntimeGovernanceError{{
				Source:     "agent_skill",
				Code:       "supporting_requires_primary",
				ResourceID: supportingCandidates[0].Document.ID,
			}}
		}
		return nil, evidence, nil
	}

	var supporting *hrRankedAgentSkillVersion
	for i := range supportingCandidates {
		candidate := supportingCandidates[i]
		if reason := hrRuntimeAgentSkillCompositionMismatchReason(primary.Document.Manifest, candidate.Document.Manifest); reason != "" {
			evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(candidate, false, reason))
			if manual {
				return nil, evidence, []hrRuntimeGovernanceError{{
					Source:     "agent_skill",
					Code:       reason,
					ResourceID: candidate.Document.ID,
				}}
			}
			continue
		}
		if supporting == nil {
			supporting = &candidate
			continue
		}
		evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(candidate, false, "supporting_limit_exceeded"))
	}

	selected := []hrRuntimeAgentSkill{hrRuntimeAgentSkillFromCandidate(*primary)}
	if supporting != nil && maxSkills > 1 {
		selected = append(selected, hrRuntimeAgentSkillFromCandidate(*supporting))
	} else if supporting != nil {
		evidence = append(evidence, hrRuntimeAgentSkillEvidenceFromCandidate(*supporting, false, "skill_limit_exceeded"))
	}
	return selected, evidence, nil
}

func hrRuntimeCompositionRoleOrder(role domainagentskill.CompositionRole) int {
	switch role {
	case domainagentskill.CompositionRolePrimary:
		return 0
	case domainagentskill.CompositionRoleSupporting:
		return 1
	default:
		return 2
	}
}

func hrRuntimeAgentSkillCompositionMismatchReason(primary, supporting domainagentskill.Manifest) string {
	if normalizeHRRuntimeCompositionValue(primary.AgentType) != normalizeHRRuntimeCompositionValue(supporting.AgentType) {
		return "composition_agent_type_mismatch"
	}
	if normalizeHRRuntimeCompositionValue(primary.Scenario) != normalizeHRRuntimeCompositionValue(supporting.Scenario) {
		return "composition_scenario_mismatch"
	}
	return ""
}

func normalizeHRRuntimeCompositionValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func hrRuntimeAgentSkillFromCandidate(candidate hrRankedAgentSkillVersion) hrRuntimeAgentSkill {
	manifest := candidate.Document.Manifest
	mode := string(candidate.Ranking.Signals.Mode)
	reason := candidate.Ranking.Signals.Reason
	if candidate.Manual {
		mode = "manual"
		reason = "manual exact-version selection"
	}
	return hrRuntimeAgentSkill{
		ID:                   candidate.Document.SkillID,
		VersionID:            candidate.Document.ID,
		Version:              candidate.Document.Version,
		CompiledHash:         candidate.Document.CompiledHash,
		Name:                 manifest.SkillName,
		DisplayName:          manifest.DisplayName,
		CoreMarkdown:         strings.TrimSpace(candidate.Document.CoreMarkdown),
		CoreEstimatedTokens:  contextbudget.EstimateTokensConservative(candidate.Document.CoreMarkdown),
		Manual:               candidate.Manual,
		Reason:               reason,
		RelevanceMode:        mode,
		CompositionRole:      manifest.Composition.Role,
		RiskLevel:            manifest.RiskLevel,
		ActivationPolicy:     manifest.ActivationPolicy,
		RequiredCapabilities: append([]string(nil), manifest.RequiredCapabilities...),
		EvaluationCriteria:   append([]string(nil), manifest.EvaluationCriteria...),
		OutputContract: domainagentskill.OutputContract{
			Mode:     manifest.OutputContract.Mode,
			SchemaID: manifest.OutputContract.SchemaID,
			Schema:   append(json.RawMessage(nil), manifest.OutputContract.Schema...),
		},
	}
}

func primaryHRRuntimeAgentSkill(skills []hrRuntimeAgentSkill) (*hrRuntimeAgentSkill, bool) {
	for i := range skills {
		if skills[i].CompositionRole == domainagentskill.CompositionRolePrimary {
			return &skills[i], true
		}
	}
	return nil, false
}

func governanceHasAgentSkillError(governance hrRuntimeGovernanceContext, code string) bool {
	for _, item := range governance.GovernanceErrors {
		if item.Source == "agent_skill" && item.Code == code {
			return true
		}
	}
	return false
}

const (
	hrRuntimeAdvisoryHeading   = "### Advisory response structure"
	hrRuntimeAdvisoryGuidance  = "Use the following immutable schema only as organization guidance for this answer."
	hrRuntimeAdvisoryFreeText  = "Natural-language output remains allowed. Do not return raw JSON unless the user explicitly asks for JSON."
	hrRuntimeAdvisoryNoEnforce = "The runtime does not validate, repair, retry, or reject this advisory response contract."
	// encoding/json can expand one schema ID source byte to at most a six-byte
	// JSON escape (for example, an ASCII control byte becomes `\u00xx`).
	hrRuntimeMaxJSONEscapedByteExpansion  = 6
	hrRuntimeAdvisoryPayloadOverheadBytes = len(`{"schema_id":"","schema":}`)
	hrRuntimeMaxAdvisoryInstructionBytes  = len(hrRuntimeAdvisoryHeading) +
		len(hrRuntimeAdvisoryGuidance) +
		len(hrRuntimeAdvisoryFreeText) +
		len(hrRuntimeAdvisoryNoEnforce) +
		4 + // strings.Join separators
		hrRuntimeAdvisoryPayloadOverheadBytes +
		domainagentskill.MaxOutputSchemaIDBytes*hrRuntimeMaxJSONEscapedByteExpansion +
		domainagentskill.MaxOutputSchemaBytes
)

func compileHRRuntimeAdvisoryInstruction(contract domainagentskill.OutputContract) (string, error) {
	if contract.Mode != domainagentskill.OutputModeAdvisory {
		return "", nil
	}
	schemaID := strings.TrimSpace(contract.SchemaID)
	if len(schemaID) > domainagentskill.MaxOutputSchemaIDBytes {
		return "", fmt.Errorf(
			"advisory schema ID size %d exceeds limit %d",
			len(schemaID),
			domainagentskill.MaxOutputSchemaIDBytes,
		)
	}
	canonicalSchema, err := canonicalizeRuntimeJSON(string(contract.Schema))
	if err != nil {
		return "", fmt.Errorf("canonicalize advisory schema: %w", err)
	}
	if canonicalSchema == "" {
		return "", fmt.Errorf("canonicalize advisory schema: empty schema")
	}
	if len(canonicalSchema) > domainagentskill.MaxOutputSchemaBytes {
		return "", fmt.Errorf(
			"advisory schema size %d exceeds limit %d",
			len(canonicalSchema),
			domainagentskill.MaxOutputSchemaBytes,
		)
	}
	payload, err := json.Marshal(struct {
		SchemaID string          `json:"schema_id,omitempty"`
		Schema   json.RawMessage `json:"schema"`
	}{
		SchemaID: schemaID,
		Schema:   json.RawMessage(canonicalSchema),
	})
	if err != nil {
		return "", fmt.Errorf("encode advisory schema: %w", err)
	}
	instruction := strings.Join([]string{
		hrRuntimeAdvisoryHeading,
		hrRuntimeAdvisoryGuidance,
		hrRuntimeAdvisoryFreeText,
		hrRuntimeAdvisoryNoEnforce,
		string(payload),
	}, "\n")
	if len(instruction) > hrRuntimeMaxAdvisoryInstructionBytes {
		return "", fmt.Errorf("advisory instruction exceeds bounded runtime limit")
	}
	return instruction, nil
}

func (s *nativeAIService) loadHRRuntimeAgentSkillSections(
	ctx context.Context,
	query string,
	skills []hrRuntimeAgentSkill,
	remaining int,
) []hrRuntimeAgentSkill {
	versionIDs := make([]int64, 0, len(skills))
	skillIndex := make(map[int64]int, len(skills))
	for i := range skills {
		versionIDs = append(versionIDs, skills[i].VersionID)
		skillIndex[skills[i].VersionID] = i
	}
	documents := make([]embeddinginfra.AgentSkillSectionEmbeddingDocument, 0)
	for _, skill := range skills {
		documents = append(documents, skill.AvailableSections...)
	}
	documentByID := make(map[int64]embeddinginfra.AgentSkillSectionEmbeddingDocument, len(documents))
	for _, document := range documents {
		if _, scoped := skillIndex[document.VersionID]; scoped {
			documentByID[document.ID] = document
		}
	}
	var ranked []embeddinginfra.RankedAgentSkillSection
	if s.embedding != nil {
		result, searchErr := s.embedding.SearchAgentSkillSections(ctx, query, versionIDs, 500)
		if searchErr == nil && result != nil {
			ranked = make([]embeddinginfra.RankedAgentSkillSection, 0, len(result.Items))
			for _, item := range result.Items {
				document, exists := documentByID[item.Document.ID]
				if !exists || !reflect.DeepEqual(item.Document, document) {
					continue
				}
				item.Document = document
				ranked = append(ranked, item)
			}
		}
	}
	if ranked == nil {
		rankingDocuments := make([]domainagentskill.RankingDocument, 0, len(documents))
		for _, document := range documents {
			if _, scoped := skillIndex[document.VersionID]; scoped {
				rankingDocuments = append(rankingDocuments, hrRuntimeSectionRankingDocument(document))
			}
		}
		for _, item := range domainagentskill.RankDocuments(query, rankingDocuments, false) {
			if document, exists := documentByID[item.Document.SectionID]; exists {
				ranked = append(ranked, embeddinginfra.RankedAgentSkillSection{Document: document, Ranking: item})
			}
		}
	}
	rankedIDs := make(map[int64]bool, len(ranked))
	for _, item := range ranked {
		document, scoped := documentByID[item.Document.ID]
		index, selected := skillIndex[item.Document.VersionID]
		if !scoped || !selected || document.VersionID != item.Document.VersionID {
			continue
		}
		rankedIDs[document.ID] = true
		estimated := contextbudget.EstimateTokensConservative(document.ContentMarkdown)
		section := hrRuntimeAgentSkillSection{
			ID:              document.ID,
			Key:             document.SectionKey,
			Title:           document.Title,
			ContentHash:     document.ContentHash,
			EstimatedTokens: estimated,
			FinalRankScore:  item.Ranking.Signals.FinalRankScore,
		}
		if !validSHA256Hex(document.ContentHash) ||
			!strings.EqualFold(document.ContentHash, sha256Hex(document.ContentMarkdown)) ||
			document.EstimatedTokens != estimated {
			section.DecisionReason = "section_integrity_failed"
			skills[index].Sections = append(skills[index].Sections, section)
			continue
		}
		if estimated <= remaining {
			section.ContentMarkdown = strings.TrimSpace(document.ContentMarkdown)
			section.Included = true
			section.DecisionReason = "section_included"
			remaining -= estimated
			skills[index].LoadedTokens += estimated
		} else {
			section.DecisionReason = "section_budget_exceeded"
		}
		skills[index].Sections = append(skills[index].Sections, section)
	}
	for _, document := range documents {
		index, selected := skillIndex[document.VersionID]
		if !selected || rankedIDs[document.ID] {
			continue
		}
		skills[index].Sections = append(skills[index].Sections, hrRuntimeAgentSkillSection{
			ID:              document.ID,
			Key:             document.SectionKey,
			Title:           document.Title,
			ContentHash:     document.ContentHash,
			EstimatedTokens: document.EstimatedTokens,
			DecisionReason:  "section_not_relevant",
		})
	}
	for i := range skills {
		sort.SliceStable(skills[i].Sections, func(a, b int) bool {
			left, right := skills[i].Sections[a], skills[i].Sections[b]
			if left.Included != right.Included {
				return left.Included
			}
			if left.FinalRankScore != right.FinalRankScore {
				return left.FinalRankScore > right.FinalRankScore
			}
			return left.ID < right.ID
		})
	}
	return skills
}

func hrRuntimeSectionRankingDocument(document embeddinginfra.AgentSkillSectionEmbeddingDocument) domainagentskill.RankingDocument {
	return domainagentskill.RankingDocument{
		ObjectID:      document.ID,
		SkillID:       document.SkillID,
		VersionID:     document.VersionID,
		SectionID:     document.ID,
		LexicalText:   []string{document.Title, document.Description, document.ContentMarkdown},
		MetadataTerms: append(append(append([]string{}, document.TriggerTerms...), document.SemanticTags...), document.PlannerIntents...),
		Priority:      document.Priority,
	}
}

func effectiveAgentSkillBudget(model RuntimeModelInfo) int {
	maxTokens := defaultAgentSkillMaxTokens
	ratio := defaultAgentSkillMaxRatio
	if model.SkillRuntimePolicy.MaxSkillTokens > 0 && model.SkillRuntimePolicy.MaxSkillTokens < maxTokens {
		maxTokens = model.SkillRuntimePolicy.MaxSkillTokens
	}
	if model.SkillRuntimePolicy.MaxInputRatio > 0 && model.SkillRuntimePolicy.MaxInputRatio < ratio {
		ratio = model.SkillRuntimePolicy.MaxInputRatio
	}
	inputBudget := contextbudget.InputBudgetTokens(contextbudget.ModelInfo{
		ContextWindowTokens: model.ContextWindowTokens,
		MaxOutputTokens:     model.MaxOutputTokens,
	})
	if inputBudget <= 0 {
		return maxTokens
	}
	ratioBudget := int(math.Floor(float64(inputBudget) * ratio))
	if ratioBudget < maxTokens {
		return maxInt(ratioBudget, 0)
	}
	return maxTokens
}

func effectiveAgentSkillMaxCount(policy CapabilitySkillRuntimePolicy) int {
	maxSkills := defaultAgentSkillMaxCount
	if policy.MaxSkills > 0 && policy.MaxSkills < maxSkills {
		maxSkills = policy.MaxSkills
	}
	return maxSkills
}

func hrRuntimeAvailableCapabilities(agentCapabilities, selectedCapabilities []string) map[string]bool {
	available := make(map[string]bool, len(agentCapabilities)*2)
	for _, raw := range agentCapabilities {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		available[value] = true
		normalized := strings.TrimSpace(strings.TrimPrefix(value, "builtin:"))
		available[normalized] = true
		available["builtin:"+normalized] = true
	}
	selected := normalizedStringSet(selectedCapabilities)
	if len(selected) == 0 {
		return available
	}
	for value := range available {
		normalized := strings.TrimPrefix(value, "builtin:")
		if !selected[value] && !selected[normalized] && !selected["builtin:"+normalized] {
			delete(available, value)
		}
	}
	return available
}

func positiveUniqueRuntimeIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func hrRuntimeAgentSkillEvidenceFromCandidate(candidate hrRankedAgentSkillVersion, included bool, reason string) *pb.AgentSkillRuntimeEvidence {
	skill := hrRuntimeAgentSkillFromCandidate(candidate)
	skill.Included = included
	skill.DecisionReason = reason
	return hrRuntimeAgentSkillEvidenceToPB(skill)
}

func hrRuntimeAgentSkillEvidenceToPB(skill hrRuntimeAgentSkill) *pb.AgentSkillRuntimeEvidence {
	sections := make([]*pb.AgentSkillSectionRuntimeEvidence, 0, len(skill.Sections))
	for _, section := range skill.Sections {
		sections = append(sections, &pb.AgentSkillSectionRuntimeEvidence{
			SectionId:       section.ID,
			SectionKey:      section.Key,
			ContentHash:     section.ContentHash,
			EstimatedTokens: int32(section.EstimatedTokens),
			FinalRankScore:  section.FinalRankScore,
			Included:        section.Included,
			DecisionReason:  section.DecisionReason,
		})
	}
	return &pb.AgentSkillRuntimeEvidence{
		SkillId:             skill.ID,
		VersionId:           skill.VersionID,
		Version:             skill.Version,
		CompiledHash:        skill.CompiledHash,
		SkillName:           skill.Name,
		DisplayName:         skill.DisplayName,
		CompositionRole:     agentSkillCompositionRoleToPB(skill.CompositionRole),
		Risk:                agentSkillRiskToPB(skill.RiskLevel),
		ActivationPolicy:    agentSkillActivationPolicyToPB(skill.ActivationPolicy),
		SelectionMode:       map[bool]string{true: "manual", false: "auto"}[skill.Manual],
		RelevanceMode:       skill.RelevanceMode,
		CoreEstimatedTokens: int32(skill.CoreEstimatedTokens),
		LoadedTokens:        int32(skill.LoadedTokens),
		Sections:            sections,
		Included:            skill.Included,
		DecisionReason:      skill.DecisionReason,
	}
}

func agentSkillCompositionRoleToPB(role domainagentskill.CompositionRole) pb.AgentSkillCompositionRole {
	switch role {
	case domainagentskill.CompositionRolePrimary:
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY
	case domainagentskill.CompositionRoleSupporting:
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_SUPPORTING
	default:
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_UNSPECIFIED
	}
}

func agentSkillRiskToPB(risk domainagentskill.RiskLevel) pb.AgentSkillRiskLevel {
	switch risk {
	case domainagentskill.RiskLevelLow:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW
	case domainagentskill.RiskLevelMedium:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM
	case domainagentskill.RiskLevelHigh:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH
	case domainagentskill.RiskLevelCritical:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL
	default:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_UNSPECIFIED
	}
}

func agentSkillActivationPolicyToPB(policy domainagentskill.ActivationPolicy) pb.AgentSkillActivationPolicy {
	switch policy {
	case domainagentskill.ActivationPolicyAuto:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO
	case domainagentskill.ActivationPolicyConfirm:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM
	case domainagentskill.ActivationPolicyManualOnly:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_MANUAL_ONLY
	default:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_UNSPECIFIED
	}
}

func statusErrorFailedPrecondition(code string) error {
	return status.Error(codes.FailedPrecondition, code)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validSHA256Hex(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
