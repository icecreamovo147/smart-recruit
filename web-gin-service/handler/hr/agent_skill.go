package hr

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	base "web-gin-service/handler"
	"web-gin-service/pkg/logger"
	pb "web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
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
	Activate             bool                    `json:"activate"`
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
		Activate:             true,
		IsEnabled:            body.IsEnabled,
		IsEnabledSet:         true,
		IsManualInvocable:    true,
		IsManualInvocableSet: true,
		TriggerKeywords:      body.TriggerKeywords,
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
	var req pb.UpdateAgentSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id
	req.ActorUserId = currentUserID(c)
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
	var req pb.CreateAgentSkillVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.SkillId = skillID
	req.ActorUserId = currentUserID(c)
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
