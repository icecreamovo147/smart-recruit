package handler

import (
	"os"

	"github.com/gin-gonic/gin"

	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type AuthHandler struct {
	clients *rpc.Clients
}

func NewAuthHandler(clients *rpc.Clients) *AuthHandler {
	return &AuthHandler{clients: clients}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		Role       int32  `json:"role" binding:"required"`
		Email      string `json:"email"`
		InviteCode string `json:"invite_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	// Security: validate invite code against database for HR registration.
	// Falls back to ALLOW_PUBLIC_HR_REGISTER=true (dev only) to skip validation.
	effectiveRole := req.Role
	if req.Role == 2 && os.Getenv("ALLOW_PUBLIC_HR_REGISTER") != "true" {
		if req.InviteCode == "" {
			effectiveRole = 1
		} else {
			validResp, err := h.clients.Admin.ValidateInviteCode(c.Request.Context(), &pb.ValidateInviteCodeRequest{InviteCode: req.InviteCode})
			if err != nil || !validResp.GetValid() {
				effectiveRole = 1
			}
		}
	}

	resp, err := h.clients.Auth.Register(c.Request.Context(), &pb.RegisterRequest{Username: req.Username, Password: req.Password, Role: effectiveRole, Email: req.Email})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Auth.Login(c.Request.Context(), &pb.LoginRequest{Username: req.Username, Password: req.Password})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}
