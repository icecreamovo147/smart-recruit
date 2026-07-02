package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	base "web-gin-service/handler"
	"web-gin-service/pkg/logger"
	pb "web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

// MCPHandler handles MCP server and tool HTTP requests.
type MCPHandler struct {
	clients *rpc.Clients
}

func NewMCPHandler(clients *rpc.Clients) *MCPHandler {
	return &MCPHandler{clients: clients}
}

// ── Server CRUD ─────────────────────────────────────────────────────

func (h *MCPHandler) ListMCPServers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.clients.MCP.ListMCPServers(c.Request.Context(), &pb.ListMCPServersRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListMCPServers failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *MCPHandler) CreateMCPServer(c *gin.Context) {
	var req pb.CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.clients.MCP.CreateMCPServer(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateMCPServer failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"server": resp.Server})
}

func (h *MCPHandler) UpdateMCPServer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	var req pb.UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id

	resp, err := h.clients.MCP.UpdateMCPServer(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateMCPServer failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"server": resp.Server})
}

func (h *MCPHandler) DeleteMCPServer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.MCP.DeleteMCPServer(c.Request.Context(), &pb.DeleteMCPServerRequest{Id: id})
	if err != nil {
		logger.L().Error("DeleteMCPServer failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, nil)
}

// ── Tool Policy CRUD ────────────────────────────────────────────────

func (h *MCPHandler) ListMCPToolPolicies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	serverID, _ := strconv.ParseInt(c.DefaultQuery("server_id", "0"), 10, 64)

	resp, err := h.clients.MCP.ListMCPToolPolicies(c.Request.Context(), &pb.ListMCPToolPoliciesRequest{
		ServerId: serverID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListMCPToolPolicies failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

func (h *MCPHandler) CreateMCPToolPolicy(c *gin.Context) {
	var req pb.CreateMCPToolPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	resp, err := h.clients.MCP.CreateMCPToolPolicy(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CreateMCPToolPolicy failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"policy": resp.Policy})
}

func (h *MCPHandler) UpdateMCPToolPolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var req pb.UpdateMCPToolPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.Id = id
	resp, err := h.clients.MCP.UpdateMCPToolPolicy(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateMCPToolPolicy failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"policy": resp.Policy})
}

func (h *MCPHandler) DeleteMCPToolPolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	resp, err := h.clients.MCP.DeleteMCPToolPolicy(c.Request.Context(), &pb.DeleteMCPToolPolicyRequest{Id: id})
	if err != nil {
		logger.L().Error("DeleteMCPToolPolicy failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, nil)
}

func (h *MCPHandler) ListMCPToolLogs(c *gin.Context) {
	serverID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid server_id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.clients.MCP.ListMCPToolLogs(c.Request.Context(), &pb.ListMCPToolLogsRequest{
		ServerId: serverID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListMCPToolLogs failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"total": resp.Total,
		"list":  resp.List,
	})
}

// ── Connection Test ─────────────────────────────────────────────────

func (h *MCPHandler) TestMCPConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}

	resp, err := h.clients.MCP.TestMCPConnection(c.Request.Context(), &pb.TestMCPConnectionRequest{ServerId: id})
	if err != nil {
		logger.L().Error("TestMCPConnection failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"success": resp.Success,
		"detail":  resp.Detail,
	})
}

// ── Tool Listing ────────────────────────────────────────────────────

func (h *MCPHandler) ListMCPTools(c *gin.Context) {
	serverIDStr := c.Param("id")
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid server_id")
		return
	}

	resp, err := h.clients.MCP.ListMCPTools(c.Request.Context(), &pb.ListMCPToolsRequest{ServerId: serverID})
	if err != nil {
		logger.L().Error("ListMCPTools failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

// ── Tool Call ───────────────────────────────────────────────────────

func (h *MCPHandler) CallMCPTool(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		base.BadRequest(c, "invalid server_id")
		return
	}

	var req pb.CallMCPToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	req.ServerId = id

	resp, err := h.clients.MCP.CallMCPTool(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("CallMCPTool failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"result_content":  resp.ResultContent,
		"error_msg":       resp.ErrorMsg,
		"duration_ms":     resp.DurationMs,
		"policy_decision": resp.PolicyDecision,
		"policy_reason":   resp.PolicyReason,
		"policy_id":       resp.PolicyId,
	})
}
