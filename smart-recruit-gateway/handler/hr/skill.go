package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/rpc"
	pb "smart-recruit-proto/recruitment/pb"
)

type SkillHandler struct {
	clients *rpc.Clients
}

func NewSkillHandler(clients *rpc.Clients) *SkillHandler {
	return &SkillHandler{clients: clients}
}

func (h *SkillHandler) ListSkills(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resp, err := h.clients.Skill.ListSkills(c.Request.Context(), &pb.ListSkillsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListSkills failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List})
}

func (h *SkillHandler) CreateSkill(c *gin.Context) {
	var req pb.CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	resp, err := h.clients.Skill.CreateSkill(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *SkillHandler) UpdateSkill(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var req pb.UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id
	resp, err := h.clients.Skill.UpdateSkill(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateSkill failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *SkillHandler) CreateSkillVersion(c *gin.Context) {
	skillID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var req pb.CreateSkillVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.SkillId = skillID
	resp, err := h.clients.Skill.CreateSkillVersion(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateSkillVersion failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"version": resp.Version, "tools": resp.Tools})
}

func (h *SkillHandler) ListSkillVersions(c *gin.Context) {
	skillID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.Skill.ListSkillVersions(c.Request.Context(), &pb.ListSkillVersionsRequest{SkillId: skillID})
	if err != nil {
		logger.L().Error("ListSkillVersions failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

func (h *SkillHandler) ActivateSkillVersion(c *gin.Context) {
	skillID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	versionID, err := strconv.ParseInt(c.Param("version_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid version_id")
		return
	}
	resp, err := h.clients.Skill.ActivateSkillVersion(c.Request.Context(), &pb.ActivateSkillVersionRequest{
		SkillId:   skillID,
		VersionId: versionID,
	})
	if err != nil {
		logger.L().Error("ActivateSkillVersion failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"skill": resp.Skill})
}

func (h *SkillHandler) ListSkillTools(c *gin.Context) {
	skillID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.Skill.ListSkillTools(c.Request.Context(), &pb.ListSkillToolsRequest{
		SkillId:     skillID,
		EnabledOnly: c.DefaultQuery("enabled_only", "") == "true",
	})
	if err != nil {
		logger.L().Error("ListSkillTools failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

func (h *SkillHandler) UpdateSkillTool(c *gin.Context) {
	toolID, err := strconv.ParseInt(c.Param("tool_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid tool_id")
		return
	}
	var req pb.UpdateSkillToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.ToolId = toolID
	resp, err := h.clients.Skill.UpdateSkillTool(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateSkillTool failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"tool": resp.Tool})
}
