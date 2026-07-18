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

type EmbeddingConfigHandler struct {
	clients *rpc.Clients
}

func NewEmbeddingConfigHandler(clients *rpc.Clients) *EmbeddingConfigHandler {
	return &EmbeddingConfigHandler{clients: clients}
}

func (h *EmbeddingConfigHandler) ListProviders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.clients.EmbeddingConfig.ListEmbeddingProviders(c.Request.Context(), &pb.ListEmbeddingProvidersRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListEmbeddingProviders failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *EmbeddingConfigHandler) CreateProvider(c *gin.Context) {
	var req pb.CreateEmbeddingProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.EmbeddingConfig.CreateEmbeddingProvider(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateEmbeddingProvider failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"provider": resp.Provider})
}

func (h *EmbeddingConfigHandler) UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req pb.UpdateEmbeddingProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id

	resp, err := h.clients.EmbeddingConfig.UpdateEmbeddingProvider(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateEmbeddingProvider failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"provider": resp.Provider})
}

func (h *EmbeddingConfigHandler) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.EmbeddingConfig.DeleteEmbeddingProvider(c.Request.Context(), &pb.DeleteEmbeddingProviderRequest{Id: id})
	if err != nil {
		logger.L().Error("DeleteEmbeddingProvider failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, nil)
}

func (h *EmbeddingConfigHandler) ListModels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	providerID, _ := strconv.ParseInt(c.DefaultQuery("provider_id", "0"), 10, 64)

	resp, err := h.clients.EmbeddingConfig.ListEmbeddingModels(c.Request.Context(), &pb.ListEmbeddingModelsRequest{
		Page:       int32(page),
		PageSize:   int32(pageSize),
		ProviderId: providerID,
	})
	if err != nil {
		logger.L().Error("ListEmbeddingModels failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *EmbeddingConfigHandler) CreateModel(c *gin.Context) {
	var req pb.CreateEmbeddingModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.EmbeddingConfig.CreateEmbeddingModel(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateEmbeddingModel failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"model": resp.Model})
}

func (h *EmbeddingConfigHandler) UpdateModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req pb.UpdateEmbeddingModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id

	resp, err := h.clients.EmbeddingConfig.UpdateEmbeddingModel(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateEmbeddingModel failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"model": resp.Model})
}

func (h *EmbeddingConfigHandler) SetDefaultModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.EmbeddingConfig.SetDefaultEmbeddingModel(c.Request.Context(), &pb.SetDefaultEmbeddingModelRequest{Id: id})
	if err != nil {
		logger.L().Error("SetDefaultEmbeddingModel failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, nil)
}

func (h *EmbeddingConfigHandler) TestModel(c *gin.Context) {
	var req pb.TestEmbeddingModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.EmbeddingConfig.TestEmbeddingModel(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("TestEmbeddingModel failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"success":    resp.Success,
		"dimension":  resp.Dimension,
		"latency_ms": resp.LatencyMs,
		"request_id": resp.RequestId,
		"detail":     resp.Detail,
	})
}

func (h *EmbeddingConfigHandler) BackfillEmbeddings(c *gin.Context) {
	var req pb.BackfillEmbeddingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.EmbeddingConfig.BackfillEmbeddings(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("BackfillEmbeddings failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"success_count": resp.SuccessCount,
		"failed_count":  resp.FailedCount,
		"skipped_count": resp.SkippedCount,
	})
}
