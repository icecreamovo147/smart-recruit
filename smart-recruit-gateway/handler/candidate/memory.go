package candidate

import (
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

const memoryOwnerRoleCandidate int32 = 1

type MemoryHandler struct {
	clients *rpc.Clients
}

func NewMemoryHandler(clients *rpc.Clients) *MemoryHandler {
	return &MemoryHandler{clients: clients}
}

func (h *MemoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ownerID := uint64(middleware.UserID(c))
	resp, err := h.clients.AI.ListMemories(c.Request.Context(), &pb.ListMemoriesRequest{
		OwnerRole:  memoryOwnerRoleCandidate,
		OwnerId:    ownerID,
		ScopeType:  strings.TrimSpace(c.Query("scope_type")),
		ScopeId:    parseUint64Query(c, "scope_id"),
		MemoryType: strings.TrimSpace(c.Query("memory_type")),
		Status:     strings.TrimSpace(c.Query("status")),
		PiiLevel:   strings.TrimSpace(c.Query("pii_level")),
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		logger.L().Error("ListMemories failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List})
}

func (h *MemoryHandler) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	ownerID := uint64(middleware.UserID(c))
	resp, err := h.clients.AI.GetMemory(c.Request.Context(), &pb.GetMemoryRequest{
		Id:        id,
		OwnerRole: memoryOwnerRoleCandidate,
		OwnerId:   ownerID,
	})
	if err != nil {
		logger.L().Error("GetMemory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"memory": resp.Memory})
}

func (h *MemoryHandler) Create(c *gin.Context) {
	var body candidateMemoryCreateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	ownerID := uint64(middleware.UserID(c))
	resp, err := h.clients.AI.CreateMemory(c.Request.Context(), &pb.CreateMemoryRequest{
		OwnerRole:      memoryOwnerRoleCandidate,
		OwnerId:        ownerID,
		ScopeType:      strings.TrimSpace(body.ScopeType),
		ScopeId:        body.ScopeID,
		MemoryType:     strings.TrimSpace(body.MemoryType),
		Content:        strings.TrimSpace(body.Content),
		Source:         candidateMemorySource(body.Source),
		Confidence:     body.Confidence,
		Importance:     body.Importance,
		CreatedBy:      middleware.UserID(c),
		ConfirmHighPii: body.ConfirmHighPII,
	})
	if err != nil {
		logger.L().Error("CreateMemory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"memory": resp.Memory})
}

func (h *MemoryHandler) Update(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body candidateMemoryUpdateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	ownerID := uint64(middleware.UserID(c))
	req := pb.UpdateMemoryRequest{
		Id:        id,
		OwnerRole: memoryOwnerRoleCandidate,
		OwnerId:   ownerID,
	}
	if body.Content != nil || body.ContentSet {
		req.Content = strings.TrimSpace(derefString(body.Content))
		req.ContentSet = true
	}
	if body.Confidence != nil || body.ConfidenceSet {
		req.Confidence = derefFloat(body.Confidence)
		req.ConfidenceSet = true
	}
	if body.Importance != nil || body.ImportanceSet {
		req.Importance = derefFloat(body.Importance)
		req.ImportanceSet = true
	}
	resp, err := h.clients.AI.UpdateMemory(c.Request.Context(), &req)
	if err != nil {
		logger.L().Error("UpdateMemory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"memory": resp.Memory})
}

func (h *MemoryHandler) Revoke(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body candidateMemoryRevokeBody
	_ = c.ShouldBindJSON(&body)
	ownerID := uint64(middleware.UserID(c))
	resp, err := h.clients.AI.RevokeMemory(c.Request.Context(), &pb.RevokeMemoryRequest{
		Id:           id,
		OwnerRole:    memoryOwnerRoleCandidate,
		OwnerId:      ownerID,
		RevokedBy:    middleware.UserID(c),
		RevokeReason: strings.TrimSpace(body.RevokeReason),
	})
	if err != nil {
		logger.L().Error("RevokeMemory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"memory": resp.Memory})
}

type candidateMemoryCreateBody struct {
	ScopeType      string  `json:"scope_type"`
	ScopeID        uint64  `json:"scope_id"`
	MemoryType     string  `json:"memory_type"`
	Content        string  `json:"content"`
	Source         string  `json:"source"`
	Confidence     float64 `json:"confidence"`
	Importance     float64 `json:"importance"`
	ConfirmHighPII bool    `json:"confirm_high_pii"`
}

type candidateMemoryUpdateBody struct {
	Content       *string  `json:"content"`
	ContentSet    bool     `json:"content_set"`
	Confidence    *float64 `json:"confidence"`
	ConfidenceSet bool     `json:"confidence_set"`
	Importance    *float64 `json:"importance"`
	ImportanceSet bool     `json:"importance_set"`
}

type candidateMemoryRevokeBody struct {
	RevokeReason string `json:"revoke_reason"`
}

func parseUint64Param(c *gin.Context, name string) (uint64, error) {
	return strconv.ParseUint(c.Param(name), 10, 64)
}

func parseUint64Query(c *gin.Context, key string) uint64 {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func candidateMemorySource(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "candidate_manual"
	}
	return raw
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
