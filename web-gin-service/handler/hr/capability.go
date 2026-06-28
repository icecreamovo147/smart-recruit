package hr

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	base "web-gin-service/handler"
	"web-gin-service/pkg/logger"
	pb "web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

const capabilityPageSize int32 = 200

type CapabilityHandler struct {
	clients *rpc.Clients
}

func NewCapabilityHandler(clients *rpc.Clients) *CapabilityHandler {
	return &CapabilityHandler{clients: clients}
}

type CapabilityResponse struct {
	ID          int64                    `json:"id"`
	SkillID     int64                    `json:"skill_id"`
	Name        string                   `json:"name"`
	DisplayName string                   `json:"display_name"`
	Category    string                   `json:"category"`
	Description string                   `json:"description"`
	Scenarios   []string                 `json:"scenarios"`
	Status      string                   `json:"status"`
	ToolsCount  int                      `json:"tools_count"`
	Tools       []CapabilityToolResponse `json:"tools"`
	UpdatedAt   string                   `json:"updated_at"`
}

type CapabilityToolResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateCapabilityFromTemplateRequest struct {
	TemplateKey string   `json:"template_key"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Scenarios   []string `json:"scenarios"`
	Instruction string   `json:"instruction"`
}

func (h *CapabilityHandler) List(c *gin.Context) {
	skills, code, msg, err := h.listAllSkills(c)
	if err != nil {
		logger.L().Error("List capability skills failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	if code != 0 {
		base.From(c, code, msg, nil)
		return
	}

	list := make([]CapabilityResponse, 0, len(skills))
	for _, skill := range skills {
		if !skill.GetIsEnabled() {
			continue
		}
		capability, err := h.buildCapability(c, skill)
		if err != nil {
			logger.L().Error("Build capability failed", zap.Int64("skill_id", skill.GetId()), zap.Error(err))
			base.Internal(c, err)
			return
		}
		list = append(list, capability)
	}

	base.OK(c, "success", gin.H{"total": len(list), "list": list})
}

func (h *CapabilityHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		base.BadRequest(c, "能力 ID 不合法")
		return
	}

	skills, code, msg, err := h.listAllSkills(c)
	if err != nil {
		logger.L().Error("List capability skills failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	if code != 0 {
		base.From(c, code, msg, nil)
		return
	}

	for _, skill := range skills {
		if skill.GetId() != id || !skill.GetIsEnabled() {
			continue
		}
		capability, err := h.buildCapability(c, skill)
		if err != nil {
			logger.L().Error("Build capability failed", zap.Int64("skill_id", skill.GetId()), zap.Error(err))
			base.Internal(c, err)
			return
		}
		base.OK(c, "success", gin.H{"capability": capability})
		return
	}

	base.From(c, 404, "能力不存在或未启用", nil)
}

func (h *CapabilityHandler) CreateFromTemplate(c *gin.Context) {
	var req CreateCapabilityFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求内容不合法")
		return
	}
	req.TemplateKey = strings.TrimSpace(req.TemplateKey)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Description = strings.TrimSpace(req.Description)
	req.Instruction = strings.TrimSpace(req.Instruction)
	if req.TemplateKey == "" {
		req.TemplateKey = "custom"
	}
	if req.DisplayName == "" {
		base.BadRequest(c, "请输入能力名称")
		return
	}
	if req.Description == "" {
		base.BadRequest(c, "请输入能力说明")
		return
	}
	if req.Instruction == "" {
		req.Instruction = defaultCapabilityInstruction(req)
	}

	skillName := generatedSkillName(req.TemplateKey)
	skillResp, err := h.clients.Skill.CreateSkill(c.Request.Context(), &pb.CreateSkillRequest{
		Name:         skillName,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		SourceType:   "builtin",
		IsEnabled:    true,
		IsEnabledSet: true,
	})
	if err != nil {
		logger.L().Error("Create capability skill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	if skillResp.GetCode() != 0 {
		base.From(c, skillResp.GetCode(), skillResp.GetMsg(), nil)
		return
	}

	manifestJSON, err := buildCapabilityManifestJSON(skillName, req)
	if err != nil {
		logger.L().Error("Build capability manifest failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	versionResp, err := h.clients.Skill.CreateSkillVersion(c.Request.Context(), &pb.CreateSkillVersionRequest{
		SkillId:      skillResp.GetSkill().GetId(),
		ManifestJson: manifestJSON,
		Activate:     true,
	})
	if err != nil {
		logger.L().Error("Create capability version failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	if versionResp.GetCode() != 0 {
		base.From(c, versionResp.GetCode(), versionResp.GetMsg(), nil)
		return
	}

	createdSkill := skillResp.GetSkill()
	if createdSkill != nil && versionResp.GetVersion() != nil {
		createdSkill.CurrentVersionId = versionResp.GetVersion().GetId()
	}
	capability, err := h.buildCapability(c, createdSkill)
	if err != nil {
		logger.L().Error("Build created capability failed", zap.Int64("skill_id", skillResp.GetSkill().GetId()), zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.OK(c, "能力已添加", gin.H{"capability": capability})
}

func (h *CapabilityHandler) listAllSkills(c *gin.Context) ([]*pb.SkillInfo, int32, string, error) {
	var all []*pb.SkillInfo
	for page := int32(1); ; page++ {
		resp, err := h.clients.Skill.ListSkills(c.Request.Context(), &pb.ListSkillsRequest{
			Page:     page,
			PageSize: capabilityPageSize,
		})
		if err != nil {
			return nil, 0, "", err
		}
		if resp.GetCode() != 0 {
			return nil, resp.GetCode(), resp.GetMsg(), nil
		}
		all = append(all, resp.GetList()...)
		if len(resp.GetList()) == 0 || int64(len(all)) >= resp.GetTotal() {
			return all, 0, resp.GetMsg(), nil
		}
	}
}

func (h *CapabilityHandler) buildCapability(c *gin.Context, skill *pb.SkillInfo) (CapabilityResponse, error) {
	tools := []CapabilityToolResponse{}
	if skill.GetCurrentVersionId() > 0 {
		resp, err := h.clients.Skill.ListSkillTools(c.Request.Context(), &pb.ListSkillToolsRequest{
			SkillId:     skill.GetId(),
			EnabledOnly: true,
		})
		if err != nil {
			return CapabilityResponse{}, err
		}
		if resp.GetCode() == 0 {
			tools = capabilityTools(resp.GetList())
		}
	}

	status := "available"
	if skill.GetCurrentVersionId() <= 0 || len(tools) == 0 {
		status = "unavailable"
	}

	return CapabilityResponse{
		ID:          skill.GetId(),
		SkillID:     skill.GetId(),
		Name:        skill.GetName(),
		DisplayName: skill.GetDisplayName(),
		Category:    inferCapabilityCategory(skill),
		Description: skill.GetDescription(),
		Scenarios:   capabilityScenarios(skill, tools),
		Status:      status,
		ToolsCount:  len(tools),
		Tools:       tools,
		UpdatedAt:   skill.GetUpdatedAt(),
	}, nil
}

func capabilityTools(items []*pb.SkillToolInfo) []CapabilityToolResponse {
	tools := make([]CapabilityToolResponse, 0, len(items))
	for _, item := range items {
		status := "available"
		if !item.GetIsEnabled() {
			status = "unavailable"
		}
		tools = append(tools, CapabilityToolResponse{
			ID:          item.GetId(),
			Name:        item.GetToolName(),
			DisplayName: capabilityToolDisplayName(item.GetToolName()),
			Description: item.GetDescription(),
			Status:      status,
			UpdatedAt:   item.GetUpdatedAt(),
		})
	}
	return tools
}

func capabilityToolDisplayName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "能力点"
	}
	return strings.ReplaceAll(trimmed, "_", " ")
}

func capabilityScenarios(skill *pb.SkillInfo, tools []CapabilityToolResponse) []string {
	scenarios := make([]string, 0, len(tools))
	seen := map[string]bool{}
	for _, tool := range tools {
		text := strings.TrimSpace(tool.Description)
		if text == "" {
			text = strings.TrimSpace(tool.Name)
		}
		if text == "" || seen[text] {
			continue
		}
		scenarios = append(scenarios, text)
		seen[text] = true
		if len(scenarios) >= 5 {
			return scenarios
		}
	}
	if len(scenarios) == 0 && strings.TrimSpace(skill.GetDescription()) != "" {
		scenarios = append(scenarios, strings.TrimSpace(skill.GetDescription()))
	}
	return scenarios
}

func inferCapabilityCategory(skill *pb.SkillInfo) string {
	text := strings.ToLower(skill.GetName() + " " + skill.GetDisplayName() + " " + skill.GetDescription())
	switch {
	case strings.Contains(text, "resume"), strings.Contains(text, "candidate"), strings.Contains(text, "screen"), strings.Contains(text, "简历"), strings.Contains(text, "候选人"), strings.Contains(text, "筛选"):
		return "candidate_screening"
	case strings.Contains(text, "interview"), strings.Contains(text, "面试"):
		return "interview"
	case strings.Contains(text, "offer"), strings.Contains(text, "录用"):
		return "offer"
	case strings.Contains(text, "job"), strings.Contains(text, "jd"), strings.Contains(text, "职位"):
		return "job"
	default:
		return "recruiting"
	}
}

func generatedSkillName(templateKey string) string {
	key := strings.ToLower(strings.TrimSpace(templateKey))
	var b strings.Builder
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	normalized := strings.Trim(b.String(), "_-")
	if normalized == "" || (normalized[0] >= '0' && normalized[0] <= '9') {
		normalized = "custom_" + normalized
	}
	if len(normalized) > 96 {
		normalized = normalized[:96]
	}
	return fmt.Sprintf("%s_%d", normalized, time.Now().UnixNano())
}

func defaultCapabilityInstruction(req CreateCapabilityFromTemplateRequest) string {
	scenarios := strings.Join(cleanScenarios(req.Scenarios), "、")
	if scenarios == "" {
		scenarios = "招聘流程"
	}
	return fmt.Sprintf("你是招聘团队的 AI 能力：%s。请围绕%s提供清晰、可执行、适合 HR 使用的分析或生成结果。能力说明：%s", req.DisplayName, scenarios, req.Description)
}

func buildCapabilityManifestJSON(skillName string, req CreateCapabilityFromTemplateRequest) (string, error) {
	toolDescription := req.Description
	if scenarios := strings.Join(cleanScenarios(req.Scenarios), "、"); scenarios != "" {
		toolDescription = fmt.Sprintf("%s。适用场景：%s", req.Description, scenarios)
	}
	manifest := map[string]any{
		"name":         skillName,
		"display_name": req.DisplayName,
		"description":  req.Description,
		"version":      time.Now().Format("20060102150405"),
		"instruction":  req.Instruction,
		"runtime": map[string]any{
			"type": "tool",
		},
		"tools": []map[string]any{
			{
				"name":        "assist",
				"description": toolDescription,
				"runtime": map[string]any{
					"type": "tool",
				},
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"context": map[string]any{
							"type":        "string",
							"description": "候选人、岗位或招聘流程上下文",
						},
						"request": map[string]any{
							"type":        "string",
							"description": "HR 希望该能力完成的具体任务",
						},
					},
					"required": []string{"context"},
				},
			},
		},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func cleanScenarios(items []string) []string {
	result := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		text := strings.TrimSpace(item)
		if text == "" || seen[text] {
			continue
		}
		result = append(result, text)
		seen[text] = true
	}
	return result
}
