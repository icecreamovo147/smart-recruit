package handler

import (
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"web-gin-service/middleware"
	"web-gin-service/pkg/authz"
	"web-gin-service/pkg/jwt"
	"web-gin-service/pkg/logger"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type AuthHandler struct {
	clients          *rpc.Clients
	defaultCookie    string
	candidateCookie  string
	hrCookie         string
	interviewerCookie string
	cookieSecure     bool
	jwtSecret        string
	rdb              *redis.Client // optional: for token_version cache
}

func NewAuthHandler(clients *rpc.Clients, defaultCookie, candidateCookie, hrCookie, interviewerCookie string, cookieSecure bool, jwtSecret string, rdb *redis.Client) *AuthHandler {
	if defaultCookie == "" {
		defaultCookie = "recruitment_token"
	}
	if candidateCookie == "" {
		candidateCookie = "recruitment_candidate_token"
	}
	if hrCookie == "" {
		hrCookie = "recruitment_hr_token"
	}
	if interviewerCookie == "" {
		interviewerCookie = "recruitment_interviewer_token"
	}
	return &AuthHandler{
		clients:           clients,
		defaultCookie:     defaultCookie,
		candidateCookie:   candidateCookie,
		hrCookie:          hrCookie,
		interviewerCookie: interviewerCookie,
		cookieSecure:      cookieSecure,
		jwtSecret:         jwtSecret,
		rdb:               rdb,
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

	// Candidate self-registration (role=1) is always allowed.
	// Staff registration (role=2,3) requires a valid invite code — validated server-side.
	resp, err := h.clients.Auth.Register(c.Request.Context(), &pb.RegisterRequest{
		Username:   req.Username,
		Password:   req.Password,
		Role:       req.Role,
		Email:      req.Email,
		InviteCode: req.InviteCode,
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
		cookieName := h.loginCookieName(c, resp.AccountType, resp.Role, resp.Roles)
		accessCookie := cookieName + "_access"
		refreshCookie := cookieName + "_refresh"

		// Write access JWT as httpOnly cookie (short TTL, 24h) with full RBAC metadata.
		accessToken, err := jwt.GenerateFull(
			h.jwtSecret, resp.UserId, resp.Username, resp.Role,
			resp.AccountType, resp.Roles, resp.Permissions, resp.TokenVersion,
			jwt.AccessTokenTTL,
		)
		if err != nil {
			logger.L().Error("login: generate access JWT failed", zap.Int64("user_id", resp.UserId), zap.Error(err))
			Internal(c, err)
			return
		}
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
		// Cache token_version so JWT middleware validates against Redis.
		if h.rdb != nil && resp.TokenVersion > 0 {
			if err := middleware.SetTokenVersionCache(c.Request.Context(), h.rdb, resp.UserId, resp.TokenVersion); err != nil {
				logger.L().Warn("failed to cache token_version after login, user may face auth issues",
					zap.Int64("user_id", resp.UserId), zap.Error(err))
			}
		}
		resp.Token = ""
	}
	ProtoResponse(c, resp)
}

// Me returns the current server-side principal by calling GetPrincipal RPC.
// This ensures the response always reflects the latest roles, permissions, and
// token_version from the database, not stale JWT claims.
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.UserID(c)
	resp, err := h.clients.Auth.GetPrincipal(c.Request.Context(), &pb.GetPrincipalRequest{
		UserId: userID,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	if resp.Code != 0 {
		From(c, 401, "会话已过期，请重新登录", nil)
		return
	}
	// Convert data scopes to a serializable form
	dataScopes := make([]gin.H, 0, len(resp.DataScopes))
	for _, ds := range resp.DataScopes {
		dataScopes = append(dataScopes, gin.H{
			"scope_key":     ds.ScopeKey,
			"resource_type": ds.ResourceType,
			"resource_id":   ds.ResourceId,
		})
	}
	OK(c, "ok", gin.H{
		"user_id":      resp.UserId,
		"username":     resp.Username,
		"role":         resp.Role,         // Deprecated: kept for compatibility
		"account_type": resp.AccountType,
		"roles":        resp.Roles,
		"permissions":  resp.Permissions,
		"token_version": resp.TokenVersion,
		"data_scopes":   dataScopes,
	})
}

// Logout clears both cookies and revokes the refresh token via Logic.
func (h *AuthHandler) Logout(c *gin.Context) {
	// Prefer X-Client-App header for cookie selection when available;
	// fall back to JWT claims (account_type/role) for backward compatibility.
	var cookieName string
	if clientApp := c.GetHeader("X-Client-App"); clientApp != "" {
		cookieName = h.requestCookieName(c)
	} else {
		accountType := middleware.AccountType(c)
		legacyRole := middleware.Role(c)
		cookieName = h.loginCookieNameByAccountType(accountType, legacyRole)
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

	// Determine cookie name: prefer X-Client-App header to keep the
	// interviewer cookie namespace consistent across refresh cycles.
	if clientApp := c.GetHeader("X-Client-App"); clientApp != "" {
		cookieName = h.requestCookieName(c)
	} else if resp.AccountType != "" {
		cookieName = h.loginCookieNameByAccountType(resp.AccountType, resp.Role)
	}

	// Write new access JWT cookie with full RBAC metadata.
	accessToken, err := jwt.GenerateFull(
		h.jwtSecret, resp.UserId, resp.Username, resp.Role,
		resp.AccountType, resp.Roles, resp.Permissions, resp.TokenVersion,
		jwt.AccessTokenTTL,
	)
	if err != nil {
		logger.L().Error("refresh: generate access JWT failed", zap.Int64("user_id", resp.UserId), zap.Error(err))
		Internal(c, err)
		return
	}
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

	// Cache token_version so JWT middleware can validate.
	if h.rdb != nil && resp.TokenVersion > 0 {
		if err := middleware.SetTokenVersionCache(c.Request.Context(), h.rdb, resp.UserId, resp.TokenVersion); err != nil {
			logger.L().Warn("failed to cache token_version after refresh, user may face auth issues",
				zap.Int64("user_id", resp.UserId), zap.Error(err))
		}
	}
	// JSON body carries no token.
	resp.RefreshToken = ""
	ProtoResponse(c, resp)
}

// ValidateInviteCode checks whether an invite code is valid (active and not expired).
// Used by the registration page to give immediate feedback before form submission.
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

// Deprecated: loginCookieNameByRole uses legacy numeric role comparison.
// Use loginCookieNameByAccountType instead, which derives the cookie name
// from the account_type field. This function is retained only as a fallback
// for callers that don't have account_type available.
func (h *AuthHandler) loginCookieNameByRole(role int32) string {
	if role >= 2 {
		return h.hrCookie
	}
	return h.candidateCookie
}

// loginCookieNameByAccountType uses account_type as the primary signal,
// falling back to legacy role for backward compatibility.
func (h *AuthHandler) loginCookieNameByAccountType(accountType string, legacyRole int32) string {
	if accountType == "staff" {
		return h.hrCookie
	}
	if accountType == "candidate" {
		return h.candidateCookie
	}
	// Fallback to legacy role check
	return h.loginCookieNameByRole(legacyRole)
}

func (h *AuthHandler) requestCookieName(c *gin.Context) string {
	switch c.GetHeader("X-Client-App") {
	case "hr":
		return h.hrCookie
	case "candidate", "user":
		return h.candidateCookie
	case "interviewer":
		return h.interviewerCookie
	default:
		return h.defaultCookie
	}
}

// loginCookieName selects the cookie namespace for a login request.
// When X-Client-App is "interviewer", it validates that the user holds the
// interviewer role and returns the interviewer cookie namespace.
func (h *AuthHandler) loginCookieName(c *gin.Context, accountType string, legacyRole int32, roles []string) string {
	if c.GetHeader("X-Client-App") == "interviewer" {
		if slices.Contains(roles, authz.RoleInterviewer) {
			return h.interviewerCookie
		}
		// Non-interviewer account attempting to log in via interviewer client.
		// Return interviewer cookie anyway — the frontend will detect the
		// missing role and force logout with an appropriate message.
		return h.interviewerCookie
	}
	return h.loginCookieNameByAccountType(accountType, legacyRole)
}
