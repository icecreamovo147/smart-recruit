package hr

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/rpc"
	pb "smart-recruit-proto/recruitment/pb"
)

type AgentSkillHandler struct {
	clients *rpc.Clients
}

type agentSkillCompositionRequest struct {
	Role string `json:"role" binding:"required"`
}

type agentSkillOutputContractRequest struct {
	Mode       string `json:"mode"`
	SchemaID   string `json:"schema_id"`
	SchemaJSON string `json:"schema_json"`
}

type agentSkillManifestRequest struct {
	SchemaVersion        int32                           `json:"schema_version"`
	SkillName            string                          `json:"skill_name" binding:"required"`
	DisplayName          string                          `json:"display_name" binding:"required"`
	Description          string                          `json:"description"`
	AgentType            string                          `json:"agent_type" binding:"required"`
	Category             string                          `json:"category"`
	Scenario             string                          `json:"scenario"`
	Priority             int32                           `json:"priority"`
	Risk                 string                          `json:"risk" binding:"required"`
	ActivationPolicy     string                          `json:"activation_policy"`
	RequiredCapabilities []string                        `json:"required_capabilities"`
	TriggerKeywords      []string                        `json:"trigger_keywords"`
	SemanticTags         []string                        `json:"semantic_tags"`
	Composition          agentSkillCompositionRequest    `json:"composition" binding:"required"`
	OutputContract       agentSkillOutputContractRequest `json:"output_contract"`
	EvaluationCriteria   []string                        `json:"evaluation_criteria"`
}

type agentSkillSectionRequest struct {
	SectionKey      string   `json:"section_key" binding:"required"`
	Title           string   `json:"title" binding:"required"`
	Description     string   `json:"description"`
	ContentMarkdown string   `json:"content_markdown" binding:"required"`
	TriggerTerms    []string `json:"trigger_terms"`
	SemanticTags    []string `json:"semantic_tags"`
	PlannerIntents  []string `json:"planner_intents"`
	Priority        int32    `json:"priority"`
	Ordinal         int32    `json:"ordinal"`
}

type agentSkillPackageRequest struct {
	Manifest      agentSkillManifestRequest  `json:"manifest" binding:"required"`
	CoreMarkdown  string                     `json:"core_markdown" binding:"required"`
	Sections      []agentSkillSectionRequest `json:"sections"`
	AuthoringJSON string                     `json:"authoring_json"`
}

type agentSkillCreateRequest struct {
	Version              string                    `json:"version" binding:"required"`
	IsEnabled            bool                      `json:"is_enabled"`
	IsEnabledSet         bool                      `json:"is_enabled_set"`
	IsManualInvocable    bool                      `json:"is_manual_invocable"`
	IsManualInvocableSet bool                      `json:"is_manual_invocable_set"`
	ChangeNote           string                    `json:"change_note"`
	Activate             *bool                     `json:"activate"`
	Package              *agentSkillPackageRequest `json:"package" binding:"required"`
}

type agentSkillUpdateRequest struct {
	DisplayName          *string `json:"display_name"`
	DisplayNameSet       bool    `json:"display_name_set"`
	Description          *string `json:"description"`
	DescriptionSet       bool    `json:"description_set"`
	IsEnabled            *bool   `json:"is_enabled"`
	IsEnabledSet         bool    `json:"is_enabled_set"`
	IsManualInvocable    *bool   `json:"is_manual_invocable"`
	IsManualInvocableSet bool    `json:"is_manual_invocable_set"`
}

type agentSkillCreateVersionRequest struct {
	Version    string                    `json:"version" binding:"required"`
	ChangeNote string                    `json:"change_note"`
	Activate   bool                      `json:"activate"`
	Package    *agentSkillPackageRequest `json:"package" binding:"required"`
}

type agentSkillPreviewRequest struct {
	Package *agentSkillPackageRequest `json:"package" binding:"required"`
}

type agentSkillCompositionPayload struct {
	Role string `json:"role"`
}

type agentSkillOutputContractPayload struct {
	Mode       string `json:"mode"`
	SchemaID   string `json:"schema_id"`
	SchemaJSON string `json:"schema_json"`
}

type agentSkillManifestPayload struct {
	SchemaVersion        int32                           `json:"schema_version"`
	SkillName            string                          `json:"skill_name"`
	DisplayName          string                          `json:"display_name"`
	Description          string                          `json:"description"`
	AgentType            string                          `json:"agent_type"`
	Category             string                          `json:"category"`
	Scenario             string                          `json:"scenario"`
	Priority             int32                           `json:"priority"`
	Risk                 string                          `json:"risk"`
	ActivationPolicy     string                          `json:"activation_policy"`
	RequiredCapabilities []string                        `json:"required_capabilities"`
	TriggerKeywords      []string                        `json:"trigger_keywords"`
	SemanticTags         []string                        `json:"semantic_tags"`
	Composition          agentSkillCompositionPayload    `json:"composition"`
	OutputContract       agentSkillOutputContractPayload `json:"output_contract"`
	EvaluationCriteria   []string                        `json:"evaluation_criteria"`
}

type agentSkillSectionPayload struct {
	ID              int64    `json:"id"`
	SectionKey      string   `json:"section_key"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	ContentMarkdown string   `json:"content_markdown"`
	TriggerTerms    []string `json:"trigger_terms"`
	SemanticTags    []string `json:"semantic_tags"`
	PlannerIntents  []string `json:"planner_intents"`
	Priority        int32    `json:"priority"`
	Ordinal         int32    `json:"ordinal"`
	EstimatedTokens int32    `json:"estimated_tokens"`
	ContentHash     string   `json:"content_hash"`
}

type agentSkillPackagePayload struct {
	Manifest               *agentSkillManifestPayload `json:"manifest"`
	CoreMarkdown           string                     `json:"core_markdown"`
	Sections               []agentSkillSectionPayload `json:"sections"`
	CompiledMarkdown       string                     `json:"compiled_markdown"`
	AuthoringJSON          string                     `json:"authoring_json"`
	CompiledHash           string                     `json:"compiled_hash"`
	CoreEstimatedTokens    int32                      `json:"core_estimated_tokens"`
	PackageEstimatedTokens int32                      `json:"package_estimated_tokens"`
}

type agentSkillVersionSummaryPayload struct {
	VersionID              int64  `json:"version_id"`
	Version                string `json:"version"`
	CompiledHash           string `json:"compiled_hash"`
	AgentType              string `json:"agent_type"`
	Category               string `json:"category"`
	Scenario               string `json:"scenario"`
	Priority               int32  `json:"priority"`
	Risk                   string `json:"risk"`
	ActivationPolicy       string `json:"activation_policy"`
	CompositionRole        string `json:"composition_role"`
	CoreEstimatedTokens    int32  `json:"core_estimated_tokens"`
	PackageEstimatedTokens int32  `json:"package_estimated_tokens"`
}

type agentSkillInfoPayload struct {
	ID                int64                            `json:"id"`
	Name              string                           `json:"name"`
	DisplayName       string                           `json:"display_name"`
	Description       string                           `json:"description"`
	CurrentVersionID  int64                            `json:"current_version_id"`
	IsEnabled         bool                             `json:"is_enabled"`
	IsManualInvocable bool                             `json:"is_manual_invocable"`
	CreatedAt         string                           `json:"created_at"`
	UpdatedAt         string                           `json:"updated_at"`
	CurrentVersion    *agentSkillVersionSummaryPayload `json:"current_version"`
}

type agentSkillVersionPayload struct {
	ID         int64                     `json:"id"`
	SkillID    int64                     `json:"skill_id"`
	Version    string                    `json:"version"`
	ChangeNote string                    `json:"change_note"`
	CreatedAt  string                    `json:"created_at"`
	Package    *agentSkillPackagePayload `json:"package"`
}

func NewAgentSkillHandler(clients *rpc.Clients) *AgentSkillHandler {
	return &AgentSkillHandler{clients: clients}
}

func (h *AgentSkillHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resp, err := h.clients.AgentSkill.ListAgentSkills(c.Request.Context(), &pb.ListAgentSkillsRequest{
		Page:        int32(page),
		PageSize:    int32(pageSize),
		Keyword:     strings.TrimSpace(c.Query("keyword")),
		EnabledOnly: c.Query("enabled_only") == "true" || c.Query("enabled_only") == "1",
	})
	if err != nil {
		logger.L().Error("ListAgentSkills failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": agentSkillInfoListPayload(resp.List)})
}

func (h *AgentSkillHandler) ListAvailable(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))
	resp, err := h.clients.AgentSkill.ListAvailableAgentSkills(c.Request.Context(), &pb.ListAvailableAgentSkillsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListAvailableAgentSkills failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": agentSkillInfoListPayload(resp.List)})
}

func (h *AgentSkillHandler) DebugSemanticRetrieval(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	jobID, _ := strconv.ParseInt(c.DefaultQuery("job_id", "0"), 10, 64)
	applicationID, _ := strconv.ParseInt(c.DefaultQuery("application_id", "0"), 10, 64)
	ownerRole, _ := strconv.Atoi(strings.TrimSpace(c.Query("owner_role")))
	ownerID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("owner_id")), 10, 64)
	hrID := currentUserID(c)
	if ownerRole <= 0 || ownerID == 0 {
		ownerRole = int(memoryOwnerRoleHR)
		ownerID = uint64(hrID)
	}
	resp, err := h.clients.AgentSkill.DebugSemanticRetrieval(c.Request.Context(), &pb.DebugSemanticRetrievalRequest{
		TenantId:      middleware.TenantID(c),
		HrId:          hrID,
		Query:         strings.TrimSpace(c.Query("query")),
		AgentType:     strings.TrimSpace(c.DefaultQuery("agent_type", "hr_recruiting_agent")),
		JobId:         jobID,
		ApplicationId: applicationID,
		Limit:         int32(limit),
		OwnerRole:     int32(ownerRole),
		OwnerId:       ownerID,
	})
	if err != nil {
		logger.L().Error("DebugSemanticRetrieval failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	skills := make([]gin.H, 0, len(resp.GetSkills()))
	for _, item := range resp.GetSkills() {
		if item == nil {
			continue
		}
		skills = append(skills, gin.H{
			"skill_id":         item.GetSkillId(),
			"version_id":       item.GetVersionId(),
			"version":          item.GetVersion(),
			"compiled_hash":    item.GetCompiledHash(),
			"name":             item.GetName(),
			"display_name":     item.GetDisplayName(),
			"category":         item.GetCategory(),
			"scenario":         item.GetScenario(),
			"priority":         item.GetPriority(),
			"risk":             agentSkillRiskValue(item.GetRisk()),
			"composition_role": agentSkillCompositionRoleValue(item.GetCompositionRole()),
			"score":            item.GetScore(),
			"reason":           item.GetReason(),
			"semantic_tags":    nonNilStringList(item.GetSemanticTags()),
			"vector_score":     item.GetVectorScore(),
			"lexical_score":    item.GetLexicalScore(),
			"metadata_score":   item.GetMetadataScore(),
			"relevance_score":  item.GetRelevanceScore(),
			"business_boost":   item.GetBusinessBoost(),
			"final_rank_score": item.GetFinalRankScore(),
			"relevance_mode":   item.GetRelevanceMode(),
			"pool_rank":        item.GetPoolRank(),
		})
	}
	memories := make([]gin.H, 0, len(resp.GetMemories()))
	for _, item := range resp.GetMemories() {
		if item == nil {
			continue
		}
		memories = append(memories, gin.H{
			"id":               item.GetId(),
			"scope_type":       item.GetScopeType(),
			"scope_id":         item.GetScopeId(),
			"memory_type":      item.GetMemoryType(),
			"content":          item.GetContent(),
			"source":           item.GetSource(),
			"confidence":       item.GetConfidence(),
			"importance":       item.GetImportance(),
			"score":            item.GetScore(),
			"reason":           item.GetReason(),
			"created_at":       item.GetCreatedAt(),
			"vector_score":     item.GetVectorScore(),
			"lexical_score":    item.GetLexicalScore(),
			"metadata_score":   item.GetMetadataScore(),
			"relevance_score":  item.GetRelevanceScore(),
			"business_boost":   item.GetBusinessBoost(),
			"final_rank_score": item.GetFinalRankScore(),
			"relevance_mode":   item.GetRelevanceMode(),
			"pool_rank":        item.GetPoolRank(),
		})
	}
	base.From(c, resp.GetCode(), resp.GetMsg(), gin.H{
		"embedding_available":        resp.GetEmbeddingAvailable(),
		"fallback_reason":            resp.GetFallbackReason(),
		"skills":                     skills,
		"memories":                   memories,
		"skill_pool_confidence":      resp.GetSkillPoolConfidence(),
		"memory_pool_confidence":     resp.GetMemoryPoolConfidence(),
		"embedding_provider":         resp.GetEmbeddingProvider(),
		"embedding_model":            resp.GetEmbeddingModel(),
		"embedding_dim":              resp.GetEmbeddingDim(),
		"candidate_count":            resp.GetCandidateCount(),
		"query_embedding_latency_ms": resp.GetQueryEmbeddingLatencyMs(),
	})
}

func (h *AgentSkillHandler) RegenerateEmbedding(c *gin.Context) {
	versionID, err := parseIDParam(c, "id")
	if err != nil || versionID <= 0 {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.EmbeddingConfig.BackfillEmbeddings(c.Request.Context(), &pb.BackfillEmbeddingsRequest{
		ObjectType: "agent_skill_version",
		ObjectId:   versionID,
		Force:      true,
		BatchSize:  1,
	})
	if err != nil {
		logger.L().Error("RegenerateAgentSkillVersionEmbedding failed", zap.Int64("version_id", versionID), zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"success_count": resp.SuccessCount,
		"failed_count":  resp.FailedCount,
		"skipped_count": resp.SkippedCount,
	})
}

func (h *AgentSkillHandler) Get(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil || id <= 0 {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.AgentSkill.GetAgentSkill(c.Request.Context(), &pb.GetAgentSkillRequest{Id: id})
	if err != nil {
		logger.L().Error("GetAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": agentSkillInfoToPayload(resp.Skill)})
}

func (h *AgentSkillHandler) Create(c *gin.Context) {
	var body agentSkillCreateRequest
	if err := bindStrictJSON(c, &body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Version) == "" {
		base.BadRequest(c, "version is required")
		return
	}
	pkg, err := body.Package.toProto()
	if err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	req := pb.CreateAgentSkillRequest{
		Version:              strings.TrimSpace(body.Version),
		ChangeNote:           strings.TrimSpace(body.ChangeNote),
		IsEnabled:            body.IsEnabled,
		IsEnabledSet:         true,
		IsManualInvocable:    true,
		IsManualInvocableSet: true,
		Package:              pkg,
		ActorUserId:          currentUserID(c),
		Activate:             true,
	}
	if body.Activate != nil {
		req.Activate = *body.Activate
	}
	if body.IsEnabledSet {
		req.IsEnabled = body.IsEnabled
	}
	if body.IsManualInvocableSet {
		req.IsManualInvocable = body.IsManualInvocable
	}
	resp, err := h.clients.AgentSkill.CreateAgentSkill(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": agentSkillInfoToPayload(resp.Skill)})
}

func (h *AgentSkillHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil || id <= 0 {
		base.BadRequest(c, "invalid id")
		return
	}
	var body agentSkillUpdateRequest
	if err := bindStrictJSON(c, &body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req := pb.UpdateAgentSkillRequest{
		Id:                   id,
		ActorUserId:          currentUserID(c),
		IsEnabledSet:         body.IsEnabledSet,
		IsManualInvocableSet: body.IsManualInvocableSet,
	}
	if body.DisplayName != nil || body.DisplayNameSet {
		if body.DisplayName != nil {
			req.DisplayName = strings.TrimSpace(*body.DisplayName)
		}
		req.DisplayNameSet = true
	}
	if body.Description != nil || body.DescriptionSet {
		if body.Description != nil {
			req.Description = strings.TrimSpace(*body.Description)
		}
		req.DescriptionSet = true
	}
	if body.IsEnabled != nil {
		req.IsEnabled = *body.IsEnabled
		req.IsEnabledSet = true
	}
	if body.IsManualInvocable != nil {
		req.IsManualInvocable = *body.IsManualInvocable
		req.IsManualInvocableSet = true
	}
	resp, err := h.clients.AgentSkill.UpdateAgentSkill(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": agentSkillInfoToPayload(resp.Skill)})
}

func (h *AgentSkillHandler) CreateVersion(c *gin.Context) {
	skillID, err := parseIDParam(c, "id")
	if err != nil || skillID <= 0 {
		base.BadRequest(c, "invalid id")
		return
	}
	var body agentSkillCreateVersionRequest
	if err := bindStrictJSON(c, &body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Version) == "" {
		base.BadRequest(c, "version is required")
		return
	}
	pkg, err := body.Package.toProto()
	if err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	req := pb.CreateAgentSkillVersionRequest{
		SkillId:     skillID,
		Version:     strings.TrimSpace(body.Version),
		ChangeNote:  strings.TrimSpace(body.ChangeNote),
		Activate:    body.Activate,
		ActorUserId: currentUserID(c),
		Package:     pkg,
	}
	resp, err := h.clients.AgentSkill.CreateAgentSkillVersion(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateAgentSkillVersion failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"version": agentSkillVersionToPayload(resp.Version)})
}

func (h *AgentSkillHandler) ListVersions(c *gin.Context) {
	skillID, err := parseIDParam(c, "id")
	if err != nil || skillID <= 0 {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.AgentSkill.ListAgentSkillVersions(c.Request.Context(), &pb.ListAgentSkillVersionsRequest{SkillId: skillID})
	if err != nil {
		logger.L().Error("ListAgentSkillVersions failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	list := make([]*agentSkillVersionPayload, 0, len(resp.List))
	for _, version := range resp.List {
		if version == nil {
			continue
		}
		list = append(list, agentSkillVersionToPayload(version))
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": list})
}

func (h *AgentSkillHandler) ActivateVersion(c *gin.Context) {
	skillID, err := parseIDParam(c, "id")
	if err != nil || skillID <= 0 {
		base.BadRequest(c, "invalid id")
		return
	}
	versionID, err := parseIDParam(c, "version_id")
	if err != nil || versionID <= 0 {
		base.BadRequest(c, "invalid version_id")
		return
	}
	resp, err := h.clients.AgentSkill.ActivateAgentSkillVersion(c.Request.Context(), &pb.ActivateAgentSkillVersionRequest{
		SkillId:     skillID,
		VersionId:   versionID,
		ActorUserId: currentUserID(c),
	})
	if err != nil {
		logger.L().Error("ActivateAgentSkillVersion failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": agentSkillInfoToPayload(resp.Skill)})
}

func (h *AgentSkillHandler) UpdateStatus(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil || id <= 0 {
		base.BadRequest(c, "invalid id")
		return
	}
	var body struct {
		IsEnabled *bool `json:"is_enabled"`
	}
	if err := bindStrictJSON(c, &body); err != nil || body.IsEnabled == nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	resp, err := h.clients.AgentSkill.UpdateAgentSkillStatus(c.Request.Context(), &pb.UpdateAgentSkillStatusRequest{
		Id:          id,
		IsEnabled:   *body.IsEnabled,
		ActorUserId: currentUserID(c),
	})
	if err != nil {
		logger.L().Error("UpdateAgentSkillStatus failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": agentSkillInfoToPayload(resp.Skill)})
}

func (h *AgentSkillHandler) Preview(c *gin.Context) {
	var body agentSkillPreviewRequest
	if err := bindStrictJSON(c, &body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	pkg, err := body.Package.toProto()
	if err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	resp, err := h.clients.AgentSkill.PreviewAgentSkill(c.Request.Context(), &pb.PreviewAgentSkillRequest{Package: pkg})
	if err != nil {
		logger.L().Error("PreviewAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"package": agentSkillPackageToPayload(resp.Package)})
}

func (r *agentSkillPackageRequest) toProto() (*pb.AgentSkillPackageDraft, error) {
	if r == nil {
		return nil, fmt.Errorf("package is required")
	}
	manifest, err := r.Manifest.toProto()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(r.CoreMarkdown) == "" {
		return nil, fmt.Errorf("package.core_markdown is required")
	}
	if authoringJSON := strings.TrimSpace(r.AuthoringJSON); authoringJSON != "" && !json.Valid([]byte(authoringJSON)) {
		return nil, fmt.Errorf("package.authoring_json must be valid JSON")
	}
	sections := make([]*pb.AgentSkillSectionDraft, 0, len(r.Sections))
	for index, section := range r.Sections {
		if strings.TrimSpace(section.SectionKey) == "" {
			return nil, fmt.Errorf("package.sections[%d].section_key is required", index)
		}
		if strings.TrimSpace(section.Title) == "" {
			return nil, fmt.Errorf("package.sections[%d].title is required", index)
		}
		if strings.TrimSpace(section.ContentMarkdown) == "" {
			return nil, fmt.Errorf("package.sections[%d].content_markdown is required", index)
		}
		sections = append(sections, &pb.AgentSkillSectionDraft{
			SectionKey:      strings.TrimSpace(section.SectionKey),
			Title:           strings.TrimSpace(section.Title),
			Description:     strings.TrimSpace(section.Description),
			ContentMarkdown: section.ContentMarkdown,
			TriggerTerms:    trimStringList(section.TriggerTerms),
			SemanticTags:    trimStringList(section.SemanticTags),
			PlannerIntents:  trimStringList(section.PlannerIntents),
			Priority:        section.Priority,
			Ordinal:         section.Ordinal,
		})
	}
	return &pb.AgentSkillPackageDraft{
		Manifest:      manifest,
		CoreMarkdown:  r.CoreMarkdown,
		Sections:      sections,
		AuthoringJson: strings.TrimSpace(r.AuthoringJSON),
	}, nil
}

func (r agentSkillManifestRequest) toProto() (*pb.AgentSkillManifest, error) {
	if r.SchemaVersion != 2 {
		return nil, fmt.Errorf("package.manifest.schema_version must be 2")
	}
	if strings.TrimSpace(r.SkillName) == "" || strings.TrimSpace(r.DisplayName) == "" || strings.TrimSpace(r.AgentType) == "" {
		return nil, fmt.Errorf("package.manifest skill_name, display_name and agent_type are required")
	}
	risk, policy, err := parseAgentSkillRisk(r.Risk)
	if err != nil {
		return nil, err
	}
	if requested := strings.TrimSpace(strings.ToLower(r.ActivationPolicy)); requested != "" && requested != agentSkillActivationPolicyValue(policy) {
		return nil, fmt.Errorf("package.manifest.activation_policy must match risk")
	}
	role, err := parseAgentSkillCompositionRole(r.Composition.Role)
	if err != nil {
		return nil, err
	}
	outputMode, err := parseAgentSkillOutputMode(r.OutputContract.Mode)
	if err != nil {
		return nil, err
	}
	if role == pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_SUPPORTING &&
		outputMode != pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_NONE {
		return nil, fmt.Errorf("supporting skills must use output mode none")
	}
	if schemaJSON := strings.TrimSpace(r.OutputContract.SchemaJSON); schemaJSON != "" && !json.Valid([]byte(schemaJSON)) {
		return nil, fmt.Errorf("package.manifest.output_contract.schema_json must be valid JSON")
	}
	return &pb.AgentSkillManifest{
		SchemaVersion:        2,
		SkillName:            strings.TrimSpace(r.SkillName),
		DisplayName:          strings.TrimSpace(r.DisplayName),
		Description:          strings.TrimSpace(r.Description),
		AgentType:            strings.TrimSpace(r.AgentType),
		Category:             strings.TrimSpace(r.Category),
		Scenario:             strings.TrimSpace(r.Scenario),
		Priority:             r.Priority,
		Risk:                 risk,
		ActivationPolicy:     policy,
		RequiredCapabilities: trimStringList(r.RequiredCapabilities),
		TriggerKeywords:      trimStringList(r.TriggerKeywords),
		SemanticTags:         trimStringList(r.SemanticTags),
		Composition:          &pb.AgentSkillComposition{Role: role},
		OutputContract: &pb.AgentSkillOutputContract{
			Mode:       outputMode,
			SchemaId:   strings.TrimSpace(r.OutputContract.SchemaID),
			SchemaJson: strings.TrimSpace(r.OutputContract.SchemaJSON),
		},
		EvaluationCriteria: trimStringList(r.EvaluationCriteria),
	}, nil
}

func parseAgentSkillRisk(value string) (pb.AgentSkillRiskLevel, pb.AgentSkillActivationPolicy, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "low":
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW, pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO, nil
	case "medium":
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM, pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO, nil
	case "high":
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH, pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM, nil
	case "critical":
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL, pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_MANUAL_ONLY, nil
	default:
		return 0, 0, fmt.Errorf("package.manifest.risk must be low, medium, high or critical")
	}
}

func parseAgentSkillCompositionRole(value string) (pb.AgentSkillCompositionRole, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "primary":
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY, nil
	case "supporting":
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_SUPPORTING, nil
	default:
		return 0, fmt.Errorf("package.manifest.composition.role must be primary or supporting")
	}
}

func parseAgentSkillOutputMode(value string) (pb.AgentSkillOutputMode, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "none":
		return pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_NONE, nil
	case "advisory":
		return pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_ADVISORY, nil
	case "strict":
		return pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_STRICT, nil
	default:
		return 0, fmt.Errorf("package.manifest.output_contract.mode must be none, advisory or strict")
	}
}

func trimStringList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func agentSkillInfoListPayload(items []*pb.AgentSkillInfo) []*agentSkillInfoPayload {
	result := make([]*agentSkillInfoPayload, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, agentSkillInfoToPayload(item))
	}
	return result
}

func agentSkillInfoToPayload(skill *pb.AgentSkillInfo) *agentSkillInfoPayload {
	if skill == nil {
		return nil
	}
	return &agentSkillInfoPayload{
		ID:                skill.GetId(),
		Name:              skill.GetName(),
		DisplayName:       skill.GetDisplayName(),
		Description:       skill.GetDescription(),
		CurrentVersionID:  skill.GetCurrentVersionId(),
		IsEnabled:         skill.GetIsEnabled(),
		IsManualInvocable: skill.GetIsManualInvocable(),
		CreatedAt:         skill.GetCreatedAt(),
		UpdatedAt:         skill.GetUpdatedAt(),
		CurrentVersion:    agentSkillVersionSummaryToPayload(skill.GetCurrentVersion()),
	}
}

func agentSkillVersionSummaryToPayload(version *pb.AgentSkillVersionSummary) *agentSkillVersionSummaryPayload {
	if version == nil {
		return nil
	}
	return &agentSkillVersionSummaryPayload{
		VersionID:              version.GetVersionId(),
		Version:                version.GetVersion(),
		CompiledHash:           version.GetCompiledHash(),
		AgentType:              version.GetAgentType(),
		Category:               version.GetCategory(),
		Scenario:               version.GetScenario(),
		Priority:               version.GetPriority(),
		Risk:                   agentSkillRiskValue(version.GetRisk()),
		ActivationPolicy:       agentSkillActivationPolicyValue(version.GetActivationPolicy()),
		CompositionRole:        agentSkillCompositionRoleValue(version.GetCompositionRole()),
		CoreEstimatedTokens:    version.GetCoreEstimatedTokens(),
		PackageEstimatedTokens: version.GetPackageEstimatedTokens(),
	}
}

func agentSkillVersionToPayload(version *pb.AgentSkillVersionInfo) *agentSkillVersionPayload {
	if version == nil {
		return nil
	}
	return &agentSkillVersionPayload{
		ID:         version.GetId(),
		SkillID:    version.GetSkillId(),
		Version:    version.GetVersion(),
		ChangeNote: version.GetChangeNote(),
		CreatedAt:  version.GetCreatedAt(),
		Package:    agentSkillPackageToPayload(version.GetPackage()),
	}
}

func agentSkillPackageToPayload(pkg *pb.AgentSkillPackageInfo) *agentSkillPackagePayload {
	if pkg == nil {
		return nil
	}
	sections := make([]agentSkillSectionPayload, 0, len(pkg.GetSections()))
	for _, section := range pkg.GetSections() {
		if section == nil {
			continue
		}
		sections = append(sections, agentSkillSectionPayload{
			ID:              section.GetId(),
			SectionKey:      section.GetSectionKey(),
			Title:           section.GetTitle(),
			Description:     section.GetDescription(),
			ContentMarkdown: section.GetContentMarkdown(),
			TriggerTerms:    nonNilStringList(section.GetTriggerTerms()),
			SemanticTags:    nonNilStringList(section.GetSemanticTags()),
			PlannerIntents:  nonNilStringList(section.GetPlannerIntents()),
			Priority:        section.GetPriority(),
			Ordinal:         section.GetOrdinal(),
			EstimatedTokens: section.GetEstimatedTokens(),
			ContentHash:     section.GetContentHash(),
		})
	}
	return &agentSkillPackagePayload{
		Manifest:               agentSkillManifestToPayload(pkg.GetManifest()),
		CoreMarkdown:           pkg.GetCoreMarkdown(),
		Sections:               sections,
		CompiledMarkdown:       pkg.GetCompiledMarkdown(),
		AuthoringJSON:          pkg.GetAuthoringJson(),
		CompiledHash:           pkg.GetCompiledHash(),
		CoreEstimatedTokens:    pkg.GetCoreEstimatedTokens(),
		PackageEstimatedTokens: pkg.GetPackageEstimatedTokens(),
	}
}

func agentSkillManifestToPayload(manifest *pb.AgentSkillManifest) *agentSkillManifestPayload {
	if manifest == nil {
		return nil
	}
	composition := agentSkillCompositionPayload{}
	if manifest.GetComposition() != nil {
		composition.Role = agentSkillCompositionRoleValue(manifest.GetComposition().GetRole())
	}
	output := agentSkillOutputContractPayload{}
	if manifest.GetOutputContract() != nil {
		output = agentSkillOutputContractPayload{
			Mode:       agentSkillOutputModeValue(manifest.GetOutputContract().GetMode()),
			SchemaID:   manifest.GetOutputContract().GetSchemaId(),
			SchemaJSON: manifest.GetOutputContract().GetSchemaJson(),
		}
	}
	return &agentSkillManifestPayload{
		SchemaVersion:        manifest.GetSchemaVersion(),
		SkillName:            manifest.GetSkillName(),
		DisplayName:          manifest.GetDisplayName(),
		Description:          manifest.GetDescription(),
		AgentType:            manifest.GetAgentType(),
		Category:             manifest.GetCategory(),
		Scenario:             manifest.GetScenario(),
		Priority:             manifest.GetPriority(),
		Risk:                 agentSkillRiskValue(manifest.GetRisk()),
		ActivationPolicy:     agentSkillActivationPolicyValue(manifest.GetActivationPolicy()),
		RequiredCapabilities: nonNilStringList(manifest.GetRequiredCapabilities()),
		TriggerKeywords:      nonNilStringList(manifest.GetTriggerKeywords()),
		SemanticTags:         nonNilStringList(manifest.GetSemanticTags()),
		Composition:          composition,
		OutputContract:       output,
		EvaluationCriteria:   nonNilStringList(manifest.GetEvaluationCriteria()),
	}
}

func agentSkillRiskValue(value pb.AgentSkillRiskLevel) string {
	switch value {
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW:
		return "low"
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM:
		return "medium"
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH:
		return "high"
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL:
		return "critical"
	default:
		return ""
	}
}

func agentSkillActivationPolicyValue(value pb.AgentSkillActivationPolicy) string {
	switch value {
	case pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO:
		return "auto"
	case pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM:
		return "confirm"
	case pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_MANUAL_ONLY:
		return "manual_only"
	default:
		return ""
	}
}

func agentSkillCompositionRoleValue(value pb.AgentSkillCompositionRole) string {
	switch value {
	case pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY:
		return "primary"
	case pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_SUPPORTING:
		return "supporting"
	default:
		return ""
	}
}

func agentSkillOutputModeValue(value pb.AgentSkillOutputMode) string {
	switch value {
	case pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_NONE:
		return "none"
	case pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_ADVISORY:
		return "advisory"
	case pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_STRICT:
		return "strict"
	default:
		return ""
	}
}

func parseIDParam(c *gin.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

func currentUserID(c *gin.Context) int64 {
	if userID, ok := c.Get("user_id"); ok {
		if v, ok := userID.(int64); ok {
			return v
		}
	}
	return 0
}
