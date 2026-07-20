package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type PlatformAIHandler struct {
	clients *rpc.Clients
}

func NewPlatformAIHandler(clients *rpc.Clients) *PlatformAIHandler {
	return &PlatformAIHandler{clients: clients}
}

func (h *PlatformAIHandler) ListCapabilities(c *gin.Context) {
	resp, err := h.clients.PlatformAI.ListPlatformAICapabilities(c.Request.Context(), &pb.ListPlatformAICapabilitiesRequest{})
	if err != nil {
		Internal(c, err)
		return
	}
	From(c, resp.GetCode(), resp.GetMsg(), gin.H{"list": resp.GetList()})
}

func (h *PlatformAIHandler) ListCapabilityVersions(c *gin.Context) {
	capabilityID, err := strconv.ParseInt(c.Param("capability_id"), 10, 64)
	if err != nil || capabilityID <= 0 {
		BadRequest(c, "invalid capability id")
		return
	}
	resp, err := h.clients.PlatformAI.ListPlatformAICapabilityVersions(c.Request.Context(), &pb.ListPlatformAICapabilityVersionsRequest{CapabilityId: capabilityID})
	if err != nil {
		Internal(c, err)
		return
	}
	From(c, resp.GetCode(), resp.GetMsg(), gin.H{"list": resp.GetList()})
}

func (h *PlatformAIHandler) CreateCapabilityDraft(c *gin.Context) {
	capabilityID, err := strconv.ParseInt(c.Param("capability_id"), 10, 64)
	if err != nil || capabilityID <= 0 {
		BadRequest(c, "invalid capability id")
		return
	}
	var body struct {
		SnapshotJSON string `json:"snapshot_json" binding:"required"`
		ChangeNote   string `json:"change_note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "snapshot_json and change_note are required")
		return
	}
	resp, err := h.clients.PlatformAI.CreatePlatformAICapabilityDraft(c.Request.Context(), &pb.CreatePlatformAICapabilityDraftRequest{
		CapabilityId: capabilityID,
		SnapshotJson: body.SnapshotJSON,
		ChangeNote:   body.ChangeNote,
		ActorUserId:  middleware.UserID(c),
		RequestId:    RequestID(c),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	From(c, resp.GetCode(), resp.GetMsg(), gin.H{"version": resp.GetVersion()})
}

func (h *PlatformAIHandler) UpdateCapabilityDraft(c *gin.Context) {
	versionID, err := strconv.ParseInt(c.Param("version_id"), 10, 64)
	if err != nil || versionID <= 0 {
		BadRequest(c, "invalid version id")
		return
	}
	var body struct {
		SnapshotJSON string `json:"snapshot_json" binding:"required"`
		ChangeNote   string `json:"change_note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "snapshot_json and change_note are required")
		return
	}
	resp, err := h.clients.PlatformAI.UpdatePlatformAICapabilityDraft(c.Request.Context(), &pb.UpdatePlatformAICapabilityDraftRequest{
		VersionId:    versionID,
		SnapshotJson: body.SnapshotJSON,
		ChangeNote:   body.ChangeNote,
		ActorUserId:  middleware.UserID(c),
		RequestId:    RequestID(c),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	From(c, resp.GetCode(), resp.GetMsg(), gin.H{"version": resp.GetVersion()})
}

func (h *PlatformAIHandler) PublishCapabilityVersion(c *gin.Context) {
	versionID, err := strconv.ParseInt(c.Param("version_id"), 10, 64)
	if err != nil || versionID <= 0 {
		BadRequest(c, "invalid version id")
		return
	}
	resp, err := h.clients.PlatformAI.PublishPlatformAICapabilityVersion(c.Request.Context(), &pb.PublishPlatformAICapabilityVersionRequest{
		VersionId:   versionID,
		ActorUserId: middleware.UserID(c),
		RequestId:   RequestID(c),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	From(c, resp.GetCode(), resp.GetMsg(), gin.H{"version": resp.GetVersion()})
}

func (h *PlatformAIHandler) AuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	capabilityID, _ := strconv.ParseInt(c.Query("capability_id"), 10, 64)
	resp, err := h.clients.PlatformAI.QueryPlatformAIConfigAuditLogs(c.Request.Context(), &pb.QueryPlatformAIConfigAuditLogsRequest{
		Page:         int32(page),
		PageSize:     int32(pageSize),
		ResourceType: c.Query("resource_type"),
		CapabilityId: capabilityID,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	From(c, resp.GetCode(), resp.GetMsg(), gin.H{"total": resp.GetTotal(), "list": resp.GetList()})
}
