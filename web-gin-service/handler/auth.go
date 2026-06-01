package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"web-gin-service/pkg/jwt"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type AuthHandler struct {
	clients         *rpc.Clients
	defaultCookie   string
	candidateCookie string
	hrCookie        string
	cookieSecure    bool
	jwtSecret       string
}

func NewAuthHandler(clients *rpc.Clients, defaultCookie, candidateCookie, hrCookie string, cookieSecure bool, jwtSecret string) *AuthHandler {
	if defaultCookie == "" {
		defaultCookie = "recruitment_token"
	}
	if candidateCookie == "" {
		candidateCookie = "recruitment_candidate_token"
	}
	if hrCookie == "" {
		hrCookie = "recruitment_hr_token"
	}
	return &AuthHandler{
		clients:         clients,
		defaultCookie:   defaultCookie,
		candidateCookie: candidateCookie,
		hrCookie:        hrCookie,
		cookieSecure:    cookieSecure,
		jwtSecret:       jwtSecret,
	}
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

	// TODO(invite-code): The invite-code flow is reserved; validation is not yet wired.
	resp, err := h.clients.Auth.Register(c.Request.Context(), &pb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Role:     req.Role,
		Email:    req.Email,
	})
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
	if resp.Code == 0 && resp.Token != "" {
		cookieName := h.loginCookieName(resp.Role)
		accessCookie := cookieName + "_access"
		refreshCookie := cookieName + "_refresh"

		// Write access JWT as httpOnly cookie (short TTL, 24h).
		accessToken, _ := jwt.GenerateWithTTL(h.jwtSecret, resp.UserId, resp.Username, resp.Role, jwt.AccessTokenTTL)
		http.SetCookie(c.Writer, &http.Cookie{
			Name: accessCookie, Value: accessToken,
			Path: "/", Expires: time.Now().Add(jwt.AccessTokenTTL),
			HttpOnly: true, Secure: h.secureCookie(c), SameSite: http.SameSiteStrictMode,
		})
		// Write opaque refresh token as httpOnly cookie (long TTL, 30d).
		http.SetCookie(c.Writer, &http.Cookie{
			Name: refreshCookie, Value: resp.Token,
			Path: "/", Expires: time.Now().Add(jwt.RefreshTokenTTL),
			HttpOnly: true, Secure: h.secureCookie(c), SameSite: http.SameSiteStrictMode,
		})
		// Erase token from JSON body — it lives in the httpOnly cookie only.
		resp.Token = ""
	}
	ProtoResponse(c, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	OK(c, "ok", gin.H{"user_id": userID, "username": username, "role": role})
}

// Logout clears both cookies and revokes the refresh token via Logic.
func (h *AuthHandler) Logout(c *gin.Context) {
	cookieName := h.requestCookieName(c)
	if role, ok := c.Get("role"); ok {
		if r, ok := role.(int32); ok {
			cookieName = h.loginCookieName(r)
		}
	}

	// Attempt to revoke the refresh token on Logic; ignore errors — cookie must be cleared regardless.
	refreshCookieName := cookieName + "_refresh"
	if rt, err := c.Cookie(refreshCookieName); err == nil && rt != "" {
		_, _ = h.clients.Auth.RevokeRefreshToken(c.Request.Context(), &pb.RevokeRefreshTokenRequest{RefreshToken: rt})
	}

	for _, name := range []string{cookieName + "_access", refreshCookieName} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.secureCookie(c),
			SameSite: http.SameSiteStrictMode,
		})
	}
	OK(c, "已退出登录", nil)
}

// RefreshToken exchanges a valid refresh token cookie for a new access + refresh token pair.
// This endpoint does NOT require an access token; it is authenticated by the refresh cookie.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	cookieName := h.requestCookieName(c)
	refreshCookieName := cookieName + "_refresh"
	plainToken, err := c.Cookie(refreshCookieName)
	if err != nil || plainToken == "" {
		From(c, 401, "刷新令牌不存在，请重新登录", nil)
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	resp, err := h.clients.Auth.RefreshToken(c.Request.Context(), &pb.RefreshTokenRequest{
		RefreshToken: plainToken,
		ClientIp:     clientIP,
		UserAgent:    userAgent,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	if resp.Code != 0 {
		// Token invalid/expired/reused — clear cookies and require re-login.
		for _, name := range []string{cookieName + "_access", refreshCookieName} {
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     name,
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				Secure:   h.secureCookie(c),
				SameSite: http.SameSiteStrictMode,
			})
		}
		ProtoResponse(c, resp)
		return
	}

	// Write new access JWT cookie.
	accessToken, _ := jwt.GenerateWithTTL(h.jwtSecret, resp.UserId, resp.Username, resp.Role, jwt.AccessTokenTTL)
	http.SetCookie(c.Writer, &http.Cookie{
		Name: cookieName + "_access", Value: accessToken,
		Path: "/", Expires: time.Now().Add(jwt.AccessTokenTTL),
		HttpOnly: true, Secure: h.secureCookie(c), SameSite: http.SameSiteStrictMode,
	})
	// Write new opaque refresh token cookie.
	http.SetCookie(c.Writer, &http.Cookie{
		Name: refreshCookieName, Value: resp.RefreshToken,
		Path: "/", Expires: time.Now().Add(jwt.RefreshTokenTTL),
		HttpOnly: true, Secure: h.secureCookie(c), SameSite: http.SameSiteStrictMode,
	})

	// JSON body carries no token.
	resp.RefreshToken = ""
	ProtoResponse(c, resp)
}

// ValidateInviteCode is a reserved capability; invite-code validation is not yet active.
func (h *AuthHandler) ValidateInviteCode(c *gin.Context) {
	var req struct {
		InviteCode string `json:"invite_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "邀请码不能为空")
		return
	}
	resp, err := h.clients.Admin.ValidateInviteCode(c.Request.Context(), &pb.ValidateInviteCodeRequest{
		InviteCode: req.InviteCode,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *AuthHandler) secureCookie(c *gin.Context) bool {
	return h.cookieSecure || c.Request.TLS != nil
}

func (h *AuthHandler) loginCookieName(role int32) string {
	if role >= 2 {
		return h.hrCookie
	}
	return h.candidateCookie
}

func (h *AuthHandler) requestCookieName(c *gin.Context) string {
	switch c.GetHeader("X-Client-App") {
	case "hr":
		return h.hrCookie
	case "candidate", "user":
		return h.candidateCookie
	default:
		return h.defaultCookie
	}
}
