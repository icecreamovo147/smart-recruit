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

// LlmConfigHandler handles LLM provider and model config HTTP requests.
type LlmConfigHandler struct {
	clients *rpc.Clients
}

func NewLlmConfigHandler(clients *rpc.Clients) *LlmConfigHandler {
	return &LlmConfigHandler{clients: clients}
}

// ── Providers ───────────────────────────────────────────────────────────

func (h *LlmConfigHandler) ListProviders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.clients.LlmConfig.ListProviders(c.Request.Context(), &pb.ListProvidersRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListProviders failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *LlmConfigHandler) CreateProvider(c *gin.Context) {
	var req pb.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.LlmConfig.CreateProvider(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateProvider failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"provider": resp.Provider})
}

func (h *LlmConfigHandler) UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req pb.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id

	resp, err := h.clients.LlmConfig.UpdateProvider(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateProvider failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"provider": resp.Provider})
}

func (h *LlmConfigHandler) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.LlmConfig.DeleteProvider(c.Request.Context(), &pb.DeleteProviderRequest{Id: id})
	if err != nil {
		logger.L().Error("DeleteProvider failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, nil)
}

func (h *LlmConfigHandler) TestProviderConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.LlmConfig.TestProviderConnection(c.Request.Context(), &pb.TestProviderConnectionRequest{ProviderId: id})
	if err != nil {
		logger.L().Error("TestProviderConnection failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"success": resp.Success,
		"detail":  resp.Detail,
	})
}

func (h *LlmConfigHandler) DiscoverProviderModels(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body struct {
		Refresh bool `json:"refresh"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			base.BadRequest(c, "invalid request body")
			return
		}
	}
	resp, err := h.clients.LlmConfig.DiscoverProviderModels(c.Request.Context(), &pb.DiscoverProviderModelsRequest{ProviderId: id, Refresh: body.Refresh})
	if err != nil {
		logger.L().Error("DiscoverProviderModels failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List, "source": resp.Source, "fetched_at": resp.FetchedAt})
}

func (h *LlmConfigHandler) GetProviderModelPreset(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body struct {
		ModelName string `json:"model_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "model_name is required")
		return
	}
	resp, err := h.clients.LlmConfig.GetProviderModelPreset(c.Request.Context(), &pb.GetProviderModelPresetRequest{ProviderId: id, ModelName: body.ModelName})
	if err != nil {
		logger.L().Error("GetProviderModelPreset failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"model": resp.Model, "source": resp.Source, "fetched_at": resp.FetchedAt})
}

func (h *LlmConfigHandler) TestModelConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.LlmConfig.TestModelConnection(c.Request.Context(), &pb.TestModelConnectionRequest{ModelId: id})
	if err != nil {
		logger.L().Error("TestModelConnection failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"success": resp.Success,
		"detail":  resp.Detail,
	})
}

// ── Models ──────────────────────────────────────────────────────────────

func (h *LlmConfigHandler) ListModels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	providerID, _ := strconv.ParseInt(c.DefaultQuery("provider_id", "0"), 10, 64)

	resp, err := h.clients.LlmConfig.ListModels(c.Request.Context(), &pb.ListModelsRequest{
		Page:       int32(page),
		PageSize:   int32(pageSize),
		ProviderId: providerID,
	})
	if err != nil {
		logger.L().Error("ListModels failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *LlmConfigHandler) ListAvailableModels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "200"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 200
	}

	resp, err := h.clients.LlmConfig.ListModels(c.Request.Context(), &pb.ListModelsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListAvailableModels failed", zap.Error(err))
		base.Internal(c, err)
		return
	}

	list := make([]gin.H, 0, len(resp.List))
	for _, model := range resp.List {
		if model == nil || !model.GetIsEnabled() {
			continue
		}
		list = append(list, gin.H{
			"id":                    model.GetId(),
			"model_name":            model.GetModelName(),
			"display_name":          model.GetDisplayName(),
			"is_enabled":            model.GetIsEnabled(),
			"is_default":            model.GetIsDefault(),
			"max_tokens":            model.GetMaxTokens(),
			"context_window_tokens": model.GetContextWindowTokens(),
		})
	}

	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": int64(len(list)),
		"list":  list,
	})
}

func (h *LlmConfigHandler) CreateModel(c *gin.Context) {
	var req pb.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.LlmConfig.CreateModel(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateModel failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"model": resp.Model})
}

func (h *LlmConfigHandler) UpdateModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req pb.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id

	resp, err := h.clients.LlmConfig.UpdateModel(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateModel failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"model": resp.Model})
}

func (h *LlmConfigHandler) DeleteModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.LlmConfig.DeleteModel(c.Request.Context(), &pb.DeleteModelRequest{Id: id})
	if err != nil {
		logger.L().Error("DeleteModel failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, nil)
}
