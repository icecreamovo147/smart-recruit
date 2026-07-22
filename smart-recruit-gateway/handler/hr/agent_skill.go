package hr

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/rpc"
	pb "smart-recruit-proto/recruitment/pb"
)

type AgentSkillHandler struct {
	clients *rpc.Clients
}

type agentSkillNodeRequest struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int32  `json:"order"`
}

type agentSkillCreateRequest struct {
	Name                 string                  `json:"name"`
	DisplayName          string                  `json:"display_name"`
	Description          string                  `json:"description"`
	Version              string                  `json:"version"`
	FlowJSON             string                  `json:"flow_json"`
	Nodes                []agentSkillNodeRequest `json:"nodes"`
	IsEnabled            bool                    `json:"is_enabled"`
	IsEnabledSet         bool                    `json:"is_enabled_set"`
	IsManualInvocable    bool                    `json:"is_manual_invocable"`
	IsManualInvocableSet bool                    `json:"is_manual_invocable_set"`
	TriggerKeywords      []string                `json:"trigger_keywords"`
	ChangeNote           string                  `json:"change_note"`
	Activate             *bool                   `json:"activate"`
	AgentType            string                  `json:"agent_type"`
	Category             string                  `json:"category"`
	Scenario             string                  `json:"scenario"`
	Priority             int32                   `json:"priority"`
	RiskLevel            string                  `json:"risk_level"`
	RequiredCapabilities []string                `json:"required_capabilities"`
	OutputSchema         string                  `json:"output_schema"`
	EvaluationCriteria   []string                `json:"evaluation_criteria"`
	SemanticTags         []string                `json:"semantic_tags"`
}

type agentSkillUpdateRequest struct {
	DisplayName             *string  `json:"display_name"`
	DisplayNameSet          bool     `json:"display_name_set"`
	Description             *string  `json:"description"`
	DescriptionSet          bool     `json:"description_set"`
	IsEnabled               *bool    `json:"is_enabled"`
	IsEnabledSet            bool     `json:"is_enabled_set"`
	IsManualInvocable       *bool    `json:"is_manual_invocable"`
	IsManualInvocableSet    bool     `json:"is_manual_invocable_set"`
	TriggerKeywords         []string `json:"trigger_keywords"`
	TriggerKeywordsSet      bool     `json:"trigger_keywords_set"`
	AgentType               *string  `json:"agent_type"`
	AgentTypeSet            bool     `json:"agent_type_set"`
	Category                *string  `json:"category"`
	CategorySet             bool     `json:"category_set"`
	Scenario                *string  `json:"scenario"`
	ScenarioSet             bool     `json:"scenario_set"`
	Priority                *int32   `json:"priority"`
	PrioritySet             bool     `json:"priority_set"`
	RiskLevel               *string  `json:"risk_level"`
	RiskLevelSet            bool     `json:"risk_level_set"`
	RequiredCapabilities    []string `json:"required_capabilities"`
	RequiredCapabilitiesSet bool     `json:"required_capabilities_set"`
	OutputSchema            *string  `json:"output_schema"`
	OutputSchemaSet         bool     `json:"output_schema_set"`
	EvaluationCriteria      []string `json:"evaluation_criteria"`
	EvaluationCriteriaSet   bool     `json:"evaluation_criteria_set"`
	SemanticTags            []string `json:"semantic_tags"`
	SemanticTagsSet         bool     `json:"semantic_tags_set"`
}

type agentSkillCreateVersionRequest struct {
	Version    string                  `json:"version"`
	FlowJSON   string                  `json:"flow_json"`
	Nodes      []agentSkillNodeRequest `json:"nodes"`
	ChangeNote string                  `json:"change_note"`
	Activate   bool                    `json:"activate"`
	SkillMD    string                  `json:"skill_md"`
}

type agentSkillPreviewRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	FlowJSON    string                  `json:"flow_json"`
	Nodes       []agentSkillNodeRequest `json:"nodes"`
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
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List})
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
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List})
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
	// 修复：改用 base.ProtoResponse 自动透传 gRPC 响应所有字段。
	// 之前用 gin.H 白名单 4 个字段，导致 skill_pool_confidence / memory_pool_confidence
	// / embedding_provider / embedding_model / embedding_dim / candidate_count /
	// query_embedding_latency_ms 等 7 个字段被丢弃；前端用 `|| '-'` 兜底导致这些卡片
	// 一直显示 "-" 或空白。ProtoResponse 通过 protojson.Marshal 序列化整个 proto
	// message，所有声明字段都会出现在 JSON 中；未来新增字段无需再改 web-gin handler。
	base.ProtoResponse(c, resp)
}

func (h *AgentSkillHandler) RegenerateEmbedding(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.EmbeddingConfig.BackfillEmbeddings(c.Request.Context(), &pb.BackfillEmbeddingsRequest{
		ObjectType: "agent_skill",
		ObjectId:   id,
		Force:      true,
		BatchSize:  1,
	})
	if err != nil {
		logger.L().Error("RegenerateAgentSkillEmbedding failed", zap.Int64("skill_id", id), zap.Error(err))
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
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.AgentSkill.GetAgentSkill(c.Request.Context(), &pb.GetAgentSkillRequest{Id: id})
	if err != nil {
		logger.L().Error("GetAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *AgentSkillHandler) Create(c *gin.Context) {
	var body agentSkillCreateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	flowJSON, err := flowJSONFromRequest(body.FlowJSON, body.Nodes)
	if err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	req := pb.CreateAgentSkillRequest{
		Name:                 strings.TrimSpace(body.Name),
		DisplayName:          strings.TrimSpace(body.DisplayName),
		Description:          strings.TrimSpace(body.Description),
		Version:              strings.TrimSpace(body.Version),
		FlowJson:             flowJSON,
		ChangeNote:           strings.TrimSpace(body.ChangeNote),
		IsEnabled:            body.IsEnabled,
		IsEnabledSet:         true,
		IsManualInvocable:    true,
		IsManualInvocableSet: true,
		TriggerKeywords:      body.TriggerKeywords,
		AgentType:            strings.TrimSpace(body.AgentType),
		Category:             strings.TrimSpace(body.Category),
		Scenario:             strings.TrimSpace(body.Scenario),
		Priority:             body.Priority,
		RiskLevel:            strings.TrimSpace(body.RiskLevel),
		RequiredCapabilities: body.RequiredCapabilities,
		OutputSchema:         strings.TrimSpace(body.OutputSchema),
		EvaluationCriteria:   body.EvaluationCriteria,
		SemanticTags:         body.SemanticTags,
	}
	req.Activate = true
	if body.Activate != nil {
		req.Activate = *body.Activate
	}
	if body.IsEnabledSet {
		req.IsEnabled = body.IsEnabled
	}
	if body.IsManualInvocableSet {
		req.IsManualInvocable = body.IsManualInvocable
	}
	req.ActorUserId = currentUserID(c)
	resp, err := h.clients.AgentSkill.CreateAgentSkill(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *AgentSkillHandler) Update(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body agentSkillUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req := pb.UpdateAgentSkillRequest{
		Id:                   id,
		ActorUserId:          currentUserID(c),
		TriggerKeywords:      body.TriggerKeywords,
		TriggerKeywordsSet:   body.TriggerKeywordsSet,
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
	if body.AgentType != nil || body.AgentTypeSet {
		if body.AgentType != nil {
			req.AgentType = strings.TrimSpace(*body.AgentType)
		}
		req.AgentTypeSet = true
	}
	if body.Category != nil || body.CategorySet {
		if body.Category != nil {
			req.Category = strings.TrimSpace(*body.Category)
		}
		req.CategorySet = true
	}
	if body.Scenario != nil || body.ScenarioSet {
		if body.Scenario != nil {
			req.Scenario = strings.TrimSpace(*body.Scenario)
		}
		req.ScenarioSet = true
	}
	if body.Priority != nil || body.PrioritySet {
		if body.Priority != nil {
			req.Priority = *body.Priority
		}
		req.PrioritySet = true
	}
	if body.RiskLevel != nil || body.RiskLevelSet {
		if body.RiskLevel != nil {
			req.RiskLevel = strings.TrimSpace(*body.RiskLevel)
		}
		req.RiskLevelSet = true
	}
	if body.RequiredCapabilitiesSet {
		req.RequiredCapabilities = body.RequiredCapabilities
		req.RequiredCapabilitiesSet = true
	}
	if body.OutputSchema != nil || body.OutputSchemaSet {
		if body.OutputSchema != nil {
			req.OutputSchema = strings.TrimSpace(*body.OutputSchema)
		}
		req.OutputSchemaSet = true
	}
	if body.EvaluationCriteriaSet {
		req.EvaluationCriteria = body.EvaluationCriteria
		req.EvaluationCriteriaSet = true
	}
	if body.SemanticTagsSet {
		req.SemanticTags = body.SemanticTags
		req.SemanticTagsSet = true
	}
	resp, err := h.clients.AgentSkill.UpdateAgentSkill(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *AgentSkillHandler) CreateVersion(c *gin.Context) {
	skillID, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body agentSkillCreateVersionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	flowJSON, err := flowJSONFromRequest(body.FlowJSON, body.Nodes)
	if err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	req := pb.CreateAgentSkillVersionRequest{
		SkillId:     skillID,
		Version:     strings.TrimSpace(body.Version),
		FlowJson:    flowJSON,
		ChangeNote:  strings.TrimSpace(body.ChangeNote),
		Activate:    body.Activate,
		ActorUserId: currentUserID(c),
		SkillMd:     strings.TrimSpace(body.SkillMD),
	}
	resp, err := h.clients.AgentSkill.CreateAgentSkillVersion(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateAgentSkillVersion failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"version": resp.Version})
}

func (h *AgentSkillHandler) ListVersions(c *gin.Context) {
	skillID, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.AgentSkill.ListAgentSkillVersions(c.Request.Context(), &pb.ListAgentSkillVersionsRequest{SkillId: skillID})
	if err != nil {
		logger.L().Error("ListAgentSkillVersions failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

func (h *AgentSkillHandler) ActivateVersion(c *gin.Context) {
	skillID, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	versionID, err := parseIDParam(c, "version_id")
	if err != nil {
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
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *AgentSkillHandler) UpdateStatus(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var req pb.UpdateAgentSkillStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id
	req.ActorUserId = currentUserID(c)
	resp, err := h.clients.AgentSkill.UpdateAgentSkillStatus(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateAgentSkillStatus failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *AgentSkillHandler) Preview(c *gin.Context) {
	var body agentSkillPreviewRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	flowJSON, err := flowJSONFromRequest(body.FlowJSON, body.Nodes)
	if err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	req := pb.PreviewAgentSkillRequest{
		Name:        strings.TrimSpace(body.Name),
		Description: strings.TrimSpace(body.Description),
		FlowJson:    flowJSON,
	}
	resp, err := h.clients.AgentSkill.PreviewAgentSkill(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("PreviewAgentSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"skill_md":         resp.SkillMd,
		"frontmatter_json": resp.FrontmatterJson,
		"body_markdown":    resp.BodyMarkdown,
	})
}

func flowJSONFromRequest(raw string, nodes []agentSkillNodeRequest) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		return raw, nil
	}
	if len(nodes) == 0 {
		return "", nil
	}
	for i := range nodes {
		if nodes[i].Order == 0 {
			nodes[i].Order = int32(i + 1)
		}
	}
	payload := struct {
		Nodes []agentSkillNodeRequest `json:"nodes"`
	}{Nodes: nodes}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
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
