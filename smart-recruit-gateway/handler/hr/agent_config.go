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

// AgentConfigHandler handles agent configuration HTTP requests.
type AgentConfigHandler struct {
	clients *rpc.Clients
}

func NewAgentConfigHandler(clients *rpc.Clients) *AgentConfigHandler {
	return &AgentConfigHandler{clients: clients}
}

// ListAgents returns a paginated list of agent configurations.
func (h *AgentConfigHandler) ListAgents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	agentType := c.DefaultQuery("agent_type", "")

	resp, err := h.clients.AgentConfig.ListAgents(c.Request.Context(), &pb.ListAgentsRequest{
		Page:      int32(page),
		PageSize:  int32(pageSize),
		AgentType: agentType,
	})
	if err != nil {
		logger.L().Error("ListAgents failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *AgentConfigHandler) ListCapabilities(c *gin.Context) {
	agentType := c.DefaultQuery("agent_type", "")

	resp, err := h.clients.AgentConfig.ListCapabilities(c.Request.Context(), &pb.ListCapabilitiesRequest{
		AgentType: agentType,
	})
	if err != nil {
		logger.L().Error("ListCapabilities failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"list": resp.List,
	})
}

// CreateAgent creates a new agent configuration.
func (h *AgentConfigHandler) CreateAgent(c *gin.Context) {
	var req pb.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.AgentConfig.CreateAgent(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateAgent failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"agent": resp.Agent})
}

// UpdateAgent updates an existing agent configuration.
func (h *AgentConfigHandler) UpdateAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req pb.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id

	resp, err := h.clients.AgentConfig.UpdateAgent(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateAgent failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"agent": resp.Agent})
}

// DeleteAgent deletes an agent configuration.
func (h *AgentConfigHandler) DeleteAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.AgentConfig.DeleteAgent(c.Request.Context(), &pb.DeleteAgentRequest{Id: id})
	if err != nil {
		logger.L().Error("DeleteAgent failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, nil)
}
