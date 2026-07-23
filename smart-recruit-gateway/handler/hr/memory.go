package hr

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

const (
	memoryOwnerRoleCandidate int32 = 1
	memoryOwnerRoleHR        int32 = 2
)

type MemoryHandler struct {
	clients *rpc.Clients
}

func NewMemoryHandler(clients *rpc.Clients) *MemoryHandler {
	return &MemoryHandler{clients: clients}
}

func (h *MemoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ownerID := uint64(currentUserID(c))
	resp, err := h.clients.AI.ListMemories(c.Request.Context(), &pb.ListMemoriesRequest{
		TenantId:   middleware.TenantID(c),
		OwnerRole:  memoryOwnerRoleHR,
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
	id, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	ownerID := uint64(currentUserID(c))
	resp, err := h.clients.AI.GetMemory(c.Request.Context(), &pb.GetMemoryRequest{
		TenantId:  middleware.TenantID(c),
		Id:        uint64(id),
		OwnerRole: memoryOwnerRoleHR,
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
	var body memoryCreateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	ownerID := uint64(currentUserID(c))
	userID := currentUserID(c)
	resp, err := h.clients.AI.CreateMemory(c.Request.Context(), &pb.CreateMemoryRequest{
		TenantId:       middleware.TenantID(c),
		OwnerRole:      memoryOwnerRoleHR,
		OwnerId:        ownerID,
		ScopeType:      strings.TrimSpace(body.ScopeType),
		ScopeId:        body.ScopeID,
		MemoryType:     strings.TrimSpace(body.MemoryType),
		Content:        strings.TrimSpace(body.Content),
		Source:         memorySourceOrDefault(body.Source, "hr_manual"),
		Confidence:     body.Confidence,
		Importance:     body.Importance,
		CreatedBy:      userID,
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
	id, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body memoryUpdateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	ownerID := uint64(currentUserID(c))
	req := pb.UpdateMemoryRequest{
		TenantId:  middleware.TenantID(c),
		Id:        uint64(id),
		OwnerRole: memoryOwnerRoleHR,
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
	if body.Status != nil || body.StatusSet {
		req.Status = strings.TrimSpace(derefString(body.Status))
		req.StatusSet = true
	}
	if body.PiiLevel != nil || body.PiiLevelSet {
		req.PiiLevel = strings.TrimSpace(derefString(body.PiiLevel))
		req.PiiLevelSet = true
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
	id, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body memoryRevokeBody
	_ = c.ShouldBindJSON(&body)
	ownerID := uint64(currentUserID(c))
	resp, err := h.clients.AI.RevokeMemory(c.Request.Context(), &pb.RevokeMemoryRequest{
		TenantId:     middleware.TenantID(c),
		Id:           uint64(id),
		OwnerRole:    memoryOwnerRoleHR,
		OwnerId:      ownerID,
		RevokedBy:    currentUserID(c),
		RevokeReason: strings.TrimSpace(body.RevokeReason),
	})
	if err != nil {
		logger.L().Error("RevokeMemory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"memory": resp.Memory})
}

func (h *MemoryHandler) Recall(c *gin.Context) {
	var body memoryRecallBody
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	limit := body.Limit
	if limit <= 0 {
		limit = 5
	}
	ownerID := uint64(currentUserID(c))
	resp, err := h.clients.AI.RecallMemories(c.Request.Context(), &pb.RecallMemoriesRequest{
		TenantId:    middleware.TenantID(c),
		OwnerRole:   memoryOwnerRoleHR,
		OwnerId:     ownerID,
		ScopeType:   strings.TrimSpace(body.ScopeType),
		ScopeId:     body.ScopeID,
		Query:       strings.TrimSpace(body.Query),
		Limit:       limit,
		MemoryTypes: body.MemoryTypes,
	})
	if err != nil {
		logger.L().Error("RecallMemories failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"items": resp.Items})
}

func (h *MemoryHandler) PlatformList(c *gin.Context) {
	ownerRole, ownerID, ok := parseMemoryOwnerQuery(c)
	if !ok {
		base.BadRequest(c, "owner_role and owner_id are required")
		return
	}
	tenantID := parseUint64Query(c, "tenant_id")
	if ownerRole == memoryOwnerRoleHR && tenantID == 0 {
		base.BadRequest(c, "tenant_id is required for HR memory")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	resp, err := h.clients.AI.ListMemories(c.Request.Context(), &pb.ListMemoriesRequest{
		TenantId:   int64(tenantID),
		OwnerRole:  ownerRole,
		OwnerId:    ownerID,
		ScopeType:  strings.TrimSpace(c.Query("scope_type")),
		ScopeId:    parseUint64Query(c, "scope_id"),
		MemoryType: strings.TrimSpace(c.Query("memory_type")),
		Status:     strings.TrimSpace(c.Query("status")),
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		logger.L().Error("Platform ListMemories failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List})
}

func (h *MemoryHandler) PlatformCreate(c *gin.Context) {
	var body platformMemoryCreateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	if body.OwnerRole <= 0 || body.OwnerID == 0 {
		base.BadRequest(c, "owner_role and owner_id are required")
		return
	}
	if body.OwnerRole == memoryOwnerRoleHR && body.TenantID == 0 {
		base.BadRequest(c, "tenant_id is required for HR memory")
		return
	}
	resp, err := h.clients.AI.CreateMemory(c.Request.Context(), &pb.CreateMemoryRequest{
		TenantId:       int64(body.TenantID),
		OwnerRole:      body.OwnerRole,
		OwnerId:        body.OwnerID,
		ScopeType:      strings.TrimSpace(body.ScopeType),
		ScopeId:        body.ScopeID,
		MemoryType:     strings.TrimSpace(body.MemoryType),
		Content:        strings.TrimSpace(body.Content),
		Source:         memorySourceOrDefault(body.Source, "platform_correction"),
		Confidence:     body.Confidence,
		Importance:     body.Importance,
		CreatedBy:      currentUserID(c),
		ConfirmHighPii: body.ConfirmHighPII,
	})
	if err != nil {
		logger.L().Error("Platform CreateMemory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"memory": resp.Memory})
}

func (h *MemoryHandler) PlatformRevoke(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		base.BadRequest(c, "invalid id")
		return
	}
	var body platformMemoryRevokeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		base.BadRequest(c, "invalid request body")
		return
	}
	if body.OwnerRole <= 0 || body.OwnerID == 0 {
		base.BadRequest(c, "owner_role and owner_id are required")
		return
	}
	if body.OwnerRole == memoryOwnerRoleHR && body.TenantID == 0 {
		base.BadRequest(c, "tenant_id is required for HR memory")
		return
	}
	resp, err := h.clients.AI.RevokeMemory(c.Request.Context(), &pb.RevokeMemoryRequest{
		TenantId:     int64(body.TenantID),
		Id:           uint64(id),
		OwnerRole:    body.OwnerRole,
		OwnerId:      body.OwnerID,
		RevokedBy:    currentUserID(c),
		RevokeReason: memorySourceOrDefault(body.RevokeReason, "platform_correction"),
	})
	if err != nil {
		logger.L().Error("Platform RevokeMemory failed", zap.Error(err))
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"memory": resp.Memory})
}

type memoryCreateBody struct {
	ScopeType      string  `json:"scope_type"`
	ScopeID        uint64  `json:"scope_id"`
	MemoryType     string  `json:"memory_type"`
	Content        string  `json:"content"`
	Source         string  `json:"source"`
	Confidence     float64 `json:"confidence"`
	Importance     float64 `json:"importance"`
	ConfirmHighPII bool    `json:"confirm_high_pii"`
}

type memoryUpdateBody struct {
	Content       *string  `json:"content"`
	ContentSet    bool     `json:"content_set"`
	Confidence    *float64 `json:"confidence"`
	ConfidenceSet bool     `json:"confidence_set"`
	Importance    *float64 `json:"importance"`
	ImportanceSet bool     `json:"importance_set"`
	Status        *string  `json:"status"`
	StatusSet     bool     `json:"status_set"`
	PiiLevel      *string  `json:"pii_level"`
	PiiLevelSet   bool     `json:"pii_level_set"`
}

type memoryRevokeBody struct {
	RevokeReason string `json:"revoke_reason"`
}

type memoryRecallBody struct {
	Query       string   `json:"query"`
	ScopeType   string   `json:"scope_type"`
	ScopeID     uint64   `json:"scope_id"`
	Limit       int32    `json:"limit"`
	MemoryTypes []string `json:"memory_types"`
}

type platformMemoryCreateBody struct {
	TenantID       uint64  `json:"tenant_id"`
	OwnerRole      int32   `json:"owner_role"`
	OwnerID        uint64  `json:"owner_id"`
	ScopeType      string  `json:"scope_type"`
	ScopeID        uint64  `json:"scope_id"`
	MemoryType     string  `json:"memory_type"`
	Content        string  `json:"content"`
	Source         string  `json:"source"`
	Confidence     float64 `json:"confidence"`
	Importance     float64 `json:"importance"`
	ConfirmHighPII bool    `json:"confirm_high_pii"`
}

type platformMemoryRevokeBody struct {
	TenantID     uint64 `json:"tenant_id"`
	OwnerRole    int32  `json:"owner_role"`
	OwnerID      uint64 `json:"owner_id"`
	RevokeReason string `json:"revoke_reason"`
}

func parseMemoryOwnerQuery(c *gin.Context) (int32, uint64, bool) {
	ownerRole, _ := strconv.Atoi(strings.TrimSpace(c.Query("owner_role")))
	ownerID := parseUint64Query(c, "owner_id")
	if ownerRole <= 0 || ownerID == 0 {
		return 0, 0, false
	}
	return int32(ownerRole), ownerID, true
}

func memorySourceOrDefault(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
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
