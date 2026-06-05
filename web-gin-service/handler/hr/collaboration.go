package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type CollaborationHandler struct {
	clients *rpc.Clients
}

func NewCollaborationHandler(clients *rpc.Clients) *CollaborationHandler {
	return &CollaborationHandler{clients: clients}
}

// GetCandidateWorkspace returns aggregated candidate workspace data.
func (h *CollaborationHandler) GetCandidateWorkspace(c *gin.Context) {
	candidateUserID, err := strconv.ParseInt(c.Param("candidate_user_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "候选人用户ID不合法")
		return
	}
	resp, err := h.clients.Collaboration.GetCandidateWorkspace(c.Request.Context(), &pb.GetCandidateWorkspaceRequest{
		StaffUserId:     middleware.UserID(c),
		CandidateUserId: candidateUserID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// CreateNote creates an internal note for a candidate.
func (h *CollaborationHandler) CreateNote(c *gin.Context) {
	var req struct {
		CandidateUserID uint64 `json:"candidate_user_id" binding:"required"`
		ApplicationID   uint64 `json:"application_id"`
		Content         string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误："+err.Error())
		return
	}
	resp, err := h.clients.Collaboration.CreateNote(c.Request.Context(), &pb.CreateNoteRequest{
		StaffUserId:     middleware.UserID(c),
		CandidateUserId: req.CandidateUserID,
		ApplicationId:   req.ApplicationID,
		Content:         req.Content,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ListNotes lists notes for a candidate.
func (h *CollaborationHandler) ListNotes(c *gin.Context) {
	candidateUserID, err := strconv.ParseUint(c.Query("candidate_user_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "候选人用户ID不合法")
		return
	}
	appID, _ := strconv.ParseUint(c.Query("application_id"), 10, 64)

	resp, err := h.clients.Collaboration.ListNotes(c.Request.Context(), &pb.ListNotesRequest{
		StaffUserId:     middleware.UserID(c),
		CandidateUserId: candidateUserID,
		ApplicationId:   appID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ── Tags ──────────────────────────────────────────────────────────────

// CreateTag creates a new tag definition.
func (h *CollaborationHandler) CreateTag(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误："+err.Error())
		return
	}
	resp, err := h.clients.Collaboration.CreateTag(c.Request.Context(), &pb.CreateTagRequest{
		StaffUserId: middleware.UserID(c),
		Name:        req.Name,
		Color:       req.Color,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ListTags lists all tag definitions.
func (h *CollaborationHandler) ListTags(c *gin.Context) {
	resp, err := h.clients.Collaboration.ListTags(c.Request.Context(), &pb.ListTagsRequest{
		StaffUserId: middleware.UserID(c),
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// AssignTag assigns a tag to a candidate.
func (h *CollaborationHandler) AssignTag(c *gin.Context) {
	var req struct {
		TagID           uint64 `json:"tag_id" binding:"required"`
		CandidateUserID uint64 `json:"candidate_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误："+err.Error())
		return
	}
	resp, err := h.clients.Collaboration.AssignTag(c.Request.Context(), &pb.AssignTagRequest{
		StaffUserId:     middleware.UserID(c),
		TagId:           req.TagID,
		CandidateUserId: req.CandidateUserID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// UnassignTag removes a tag from a candidate.
func (h *CollaborationHandler) UnassignTag(c *gin.Context) {
	var req struct {
		TagID           uint64 `json:"tag_id" binding:"required"`
		CandidateUserID uint64 `json:"candidate_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误："+err.Error())
		return
	}
	resp, err := h.clients.Collaboration.UnassignTag(c.Request.Context(), &pb.UnassignTagRequest{
		StaffUserId:     middleware.UserID(c),
		TagId:           req.TagID,
		CandidateUserId: req.CandidateUserID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ListCandidateTags lists tags assigned to a candidate.
func (h *CollaborationHandler) ListCandidateTags(c *gin.Context) {
	candidateUserID, err := strconv.ParseUint(c.Param("candidate_user_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "候选人用户ID不合法")
		return
	}
	resp, err := h.clients.Collaboration.ListCandidateTags(c.Request.Context(), &pb.ListCandidateTagsRequest{
		StaffUserId:     middleware.UserID(c),
		CandidateUserId: candidateUserID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ── Follow-up Tasks ───────────────────────────────────────────────────

// CreateFollowUpTask creates a follow-up task.
func (h *CollaborationHandler) CreateFollowUpTask(c *gin.Context) {
	var req struct {
		CandidateUserID uint64 `json:"candidate_user_id" binding:"required"`
		ApplicationID   uint64 `json:"application_id"`
		AssigneeUserID  uint64 `json:"assignee_user_id" binding:"required"`
		Title           string `json:"title" binding:"required"`
		Description     string `json:"description"`
		DueAt           string `json:"due_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误："+err.Error())
		return
	}
	resp, err := h.clients.Collaboration.CreateFollowUpTask(c.Request.Context(), &pb.CreateFollowUpTaskRequest{
		StaffUserId:     middleware.UserID(c),
		CandidateUserId: req.CandidateUserID,
		ApplicationId:   req.ApplicationID,
		AssigneeUserId:  req.AssigneeUserID,
		Title:           req.Title,
		Description:     req.Description,
		DueAt:           req.DueAt,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ListFollowUpTasks lists follow-up tasks with optional filters.
func (h *CollaborationHandler) ListFollowUpTasks(c *gin.Context) {
	candidateUserID, _ := strconv.ParseUint(c.Query("candidate_user_id"), 10, 64)
	assigneeUserID, _ := strconv.ParseUint(c.Query("assignee_user_id"), 10, 64)
	status := c.Query("status")

	resp, err := h.clients.Collaboration.ListFollowUpTasks(c.Request.Context(), &pb.ListFollowUpTasksRequest{
		StaffUserId:     middleware.UserID(c),
		CandidateUserId: candidateUserID,
		AssigneeUserId:  assigneeUserID,
		Status:          status,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// CompleteFollowUpTask marks a task as completed.
func (h *CollaborationHandler) CompleteFollowUpTask(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("task_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "任务ID不合法")
		return
	}
	resp, err := h.clients.Collaboration.CompleteFollowUpTask(c.Request.Context(), &pb.CompleteFollowUpTaskRequest{
		StaffUserId: middleware.UserID(c),
		TaskId:      taskID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ListTimelineEvents returns aggregated timeline events for a candidate.
func (h *CollaborationHandler) ListTimelineEvents(c *gin.Context) {
	candidateUserID, err := strconv.ParseInt(c.Param("candidate_user_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "候选人用户ID不合法")
		return
	}
	resp, err := h.clients.Collaboration.ListTimelineEvents(c.Request.Context(), &pb.ListTimelineEventsRequest{
		StaffUserId:     middleware.UserID(c),
		CandidateUserId: uint64(candidateUserID),
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}
