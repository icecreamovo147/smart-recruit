package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type PlatformUserHandler struct{ clients *rpc.Clients }

func NewPlatformUserHandler(clients *rpc.Clients) *PlatformUserHandler {
	return &PlatformUserHandler{clients: clients}
}

func (h *PlatformUserHandler) List(c *gin.Context) {
	resp, err := h.clients.Admin.ListPlatformUsers(c.Request.Context(), &pb.ListPlatformUsersRequest{Page: int32(queryInt(c, "page", 1)), PageSize: int32(queryInt(c, "page_size", 20)), Status: c.Query("status")})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformUserHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email"`
		Password string `json:"password" binding:"required"`
		RoleKey  string `json:"role_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.CreatePlatformUser(c.Request.Context(), &pb.CreatePlatformUserRequest{AdminId: middleware.UserID(c), Username: req.Username, Email: req.Email, Password: req.Password, RoleKey: req.RoleKey})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformUserHandler) Update(c *gin.Context) {
	userID, ok := platformIDParam(c, "user_id")
	if !ok {
		return
	}
	var req struct {
		RoleKey string `json:"role_key" binding:"required"`
		Status  string `json:"status" binding:"required"`
		Reason  string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.UpdatePlatformUser(c.Request.Context(), &pb.UpdatePlatformUserRequest{AdminId: middleware.UserID(c), UserId: userID, RoleKey: req.RoleKey, Status: req.Status, Reason: strings.TrimSpace(req.Reason)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}
