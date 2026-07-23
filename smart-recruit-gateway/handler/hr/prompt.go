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

// PromptHandler handles prompt template HTTP requests.
type PromptHandler struct {
	clients *rpc.Clients
}

// NewPromptHandler creates a new PromptHandler.
func NewPromptHandler(clients *rpc.Clients) *PromptHandler {
	return &PromptHandler{clients: clients}
}

// ── Template CRUD ─────────────────────────────────────────────────────────

func (h *PromptHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	agentType := c.DefaultQuery("agent_type", "")

	resp, err := h.clients.Prompt.ListPromptTemplates(c.Request.Context(), &pb.ListPromptTemplatesRequest{
		Page:      int32(page),
		PageSize:  int32(pageSize),
		AgentType: agentType,
	})
	if err != nil {
		logger.L().Error("ListPromptTemplates failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *PromptHandler) Create(c *gin.Context) {
	var req pb.CreatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.Prompt.CreatePromptTemplate(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreatePromptTemplate failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"template": resp.Template})
}

func (h *PromptHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req pb.UpdatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id

	resp, err := h.clients.Prompt.UpdatePromptTemplate(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdatePromptTemplate failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"template": resp.Template})
}

func (h *PromptHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.Prompt.DeletePromptTemplate(c.Request.Context(), &pb.DeletePromptTemplateRequest{Id: id})
	if err != nil {
		logger.L().Error("DeletePromptTemplate failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ── Version Management ────────────────────────────────────────────────────

func (h *PromptHandler) ListVersions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.clients.Prompt.GetPromptVersionHistory(c.Request.Context(), &pb.GetPromptVersionHistoryRequest{
		TemplateId: id,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		logger.L().Error("GetPromptVersionHistory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *PromptHandler) Rollback(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req struct {
		Version    int32  `json:"version"`
		UpdatedBy  int64  `json:"updated_by"`
		ChangeNote string `json:"change_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.Prompt.RollbackPromptVersion(c.Request.Context(), &pb.RollbackPromptVersionRequest{
		TemplateId: id,
		Version:    req.Version,
		UpdatedBy:  req.UpdatedBy,
		ChangeNote: req.ChangeNote,
	})
	if err != nil {
		logger.L().Error("RollbackPromptVersion failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"template": resp.Template})
}
