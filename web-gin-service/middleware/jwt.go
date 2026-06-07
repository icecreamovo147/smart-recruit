package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	golangjwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"web-gin-service/pkg/contextkeys"
	webjwt "web-gin-service/pkg/jwt"
	"web-gin-service/pkg/logger"
)

// JWTAuth parses the access token from the Authorization header or the named cookie
// and injects identity + authorization context into the Gin context.
func JWTAuth(secret, cookieName string) gin.HandlerFunc {
	return JWTAuthWithTokenVersion(secret, cookieName, nil)
}

// JWTAuthWithTokenVersion is like JWTAuth but optionally validates token_version
// against a Redis cache. If rdb is nil, token_version validation is skipped.
func JWTAuthWithTokenVersion(secret, cookieName string, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only consume Authorization if it's a Bearer header.
		// Other schemes (Basic, Digest, proxy-injected headers) must not
		// block the httpOnly cookie fallback.
		var tokenText string
		if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			tokenText = authHeader[7:]
		}
		if tokenText == "" {
			if cookie, err := c.Cookie(cookieName); err == nil {
				tokenText = cookie
			}
		}
		if tokenText == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录或 Token 无效", "data": nil, "request_id": requestID(c)})
			return
		}
		claims := &webjwt.Claims{}
		token, err := golangjwt.ParseWithClaims(tokenText, claims, func(token *golangjwt.Token) (any, error) {
			// Explicitly require HS256 to prevent algorithm confusion attacks.
			if _, ok := token.Method.(*golangjwt.SigningMethodHMAC); !ok {
				return nil, golangjwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, golangjwt.WithValidMethods([]string{"HS256"}), golangjwt.WithLeeway(30*time.Second))
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录或 Token 无效", "data": nil, "request_id": requestID(c)})
			return
		}

		// ── Server-side token_version validation ──────────────────────────
		// If Redis is configured, check that the token's version matches the
		// current server-side version. A mismatch means permissions changed
		// after this token was issued — reject and force re-login.
		if rdb != nil && claims.UserID > 0 {
			if !validateTokenVersion(c.Request.Context(), rdb, claims.UserID, claims.TokenVersion) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code": 401, "msg": "权限已变更，请重新登录", "data": nil, "request_id": requestID(c),
				})
				return
			}
		}

		// Populate context with full principal data.
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role) // Deprecated: kept for compatibility
		c.Set("account_type", claims.AccountType)
		c.Set("roles", claims.Roles)             // []string of role keys
		c.Set("permissions", claims.Permissions) // []string of permission keys
		c.Set("token_version", claims.TokenVersion)

		// Also inject into the Go context so gRPC metadata forwarding
		// can propagate the authenticated actor to the logic service.
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, contextkeys.UserID, claims.UserID)
		ctx = context.WithValue(ctx, contextkeys.AccountType, claims.AccountType)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// CurrentPrincipal is the database-backed identity used to revalidate JWT claims.
type CurrentPrincipal struct {
	UserID       int64
	Username     string
	Role         int32
	AccountType  string
	Roles        []string
	Permissions  []string
	TokenVersion int32
}

// PrincipalLoader loads the current identity from the source of truth.
type PrincipalLoader func(context.Context, int64) (*CurrentPrincipal, error)

// ValidateCurrentPrincipal rejects JWTs for deleted users and refreshes the
// authorization context from the database-backed principal.
func ValidateCurrentPrincipal(load PrincipalLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if load == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "msg": "会话验证不可用，请重新登录", "data": nil, "request_id": requestID(c),
			})
			return
		}

		userID := UserID(c)
		principal, err := load(c.Request.Context(), userID)
		if err != nil || principal == nil || principal.UserID != userID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "msg": "账号不存在或会话已失效，请重新登录", "data": nil, "request_id": requestID(c),
			})
			return
		}
		if principal.TokenVersion != TokenVersion(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "msg": "权限已变更，请重新登录", "data": nil, "request_id": requestID(c),
			})
			return
		}

		c.Set("username", principal.Username)
		c.Set("role", principal.Role)
		c.Set("account_type", principal.AccountType)
		c.Set("roles", principal.Roles)
		c.Set("permissions", principal.Permissions)
		c.Set("token_version", principal.TokenVersion)

		ctx := context.WithValue(c.Request.Context(), contextkeys.AccountType, principal.AccountType)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// validateTokenVersion checks the JWT token_version against the cached server-side
// version in Redis. Returns true if the token is still valid.
// Uses the cache key: "token_version:{user_id}".
//
// Strategy:
//   - Redis hit: compare stored version <= jwt version, reject if stored > jwt.
//   - Redis miss (redis.Nil): reject — key genuinely doesn't exist, force re-login.
//   - Redis error (unreachable): allow through in degraded mode — JWT signature is
//     still verified, we just can't check token version revocation.
func validateTokenVersion(ctx context.Context, rdb *redis.Client, userID int64, jwtVersion int32) bool {
	key := fmt.Sprintf("token_version:%d", userID)
	stored, err := rdb.Get(ctx, key).Int()
	if err != nil {
		// redis.Nil means the key genuinely doesn't exist — the version was
		// never cached (e.g. Redis restart) → reject (fail-closed).
		if errors.Is(err, redis.Nil) {
			return false
		}
		// Redis unreachable (network error, OOM, etc.) → allow through in
		// degraded mode. The JWT signature itself is still verified, so the
		// token is not forged — we just can't check if it was revoked.
		// Logged at Error level so monitoring/alerting can detect prolonged
		// Redis outages that would prevent token revocation from taking effect.
		logger.L().Error("token_version check skipped: redis unavailable — revocation enforcement degraded",
			zap.Int64("user_id", userID), zap.Error(err))
		return true
	}
	return int32(stored) <= jwtVersion
}

// SetTokenVersionCache writes the current token_version to Redis so the
// JWT middleware can validate it. Call this after permission changes.
func SetTokenVersionCache(ctx context.Context, rdb *redis.Client, userID int64, version int32) error {
	key := fmt.Sprintf("token_version:%d", userID)
	// Cache with a TTL matching the access token lifetime.
	return rdb.Set(ctx, key, version, webjwt.AccessTokenTTL).Err()
}

// JWTAuthByClient selects the appropriate cookie based on the X-Client-App header
// and delegates to JWTAuthWithTokenVersion when rdb is provided, or JWTAuth otherwise.
func JWTAuthByClient(secret, candidateCookie, hrCookie, interviewerCookie, fallbackCookie string, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieName := fallbackCookie
		switch c.GetHeader("X-Client-App") {
		case "hr":
			cookieName = hrCookie
		case "candidate", "user":
			cookieName = candidateCookie
		case "interviewer":
			cookieName = interviewerCookie
		}
		// Read from the short-lived access token cookie, with optional token_version validation.
		JWTAuthWithTokenVersion(secret, cookieName+"_access", rdb)(c)
	}
}

// ── Context helpers ───────────────────────────────────────────────────

func requestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}

// UserID returns the authenticated user ID from the Gin context.
func UserID(c *gin.Context) int64 {
	value, _ := c.Get("user_id")
	userID, _ := value.(int64)
	return userID
}

// Username returns the authenticated username from the Gin context.
func Username(c *gin.Context) string {
	value, _ := c.Get("username")
	username, _ := value.(string)
	return username
}

// Role returns the legacy numeric role from the Gin context (deprecated).
func Role(c *gin.Context) int32 {
	value, _ := c.Get("role")
	role, _ := value.(int32)
	return role
}

// AccountType returns the account type (candidate/staff) from the Gin context.
func AccountType(c *gin.Context) string {
	value, _ := c.Get("account_type")
	at, _ := value.(string)
	return at
}

// Roles returns the RBAC role keys from the Gin context.
func Roles(c *gin.Context) []string {
	value, _ := c.Get("roles")
	roles, _ := value.([]string)
	return roles
}

// Permissions returns the RBAC permission keys from the Gin context.
func Permissions(c *gin.Context) []string {
	value, _ := c.Get("permissions")
	perms, _ := value.([]string)
	return perms
}

// TokenVersion returns the token version from the Gin context.
func TokenVersion(c *gin.Context) int32 {
	value, _ := c.Get("token_version")
	tv, _ := value.(int32)
	return tv
}

// HasRole checks if the authenticated user has a specific RBAC role.
func HasRole(c *gin.Context, roleKey string) bool {
	for _, r := range Roles(c) {
		if r == roleKey {
			return true
		}
	}
	return false
}

// HasPermission checks if the authenticated user has a specific permission.
func HasPermission(c *gin.Context, permKey string) bool {
	for _, p := range Permissions(c) {
		if p == permKey {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the authenticated user has any of the given permissions.
func HasAnyPermission(c *gin.Context, permKeys ...string) bool {
	perms := Permissions(c)
	for _, p := range perms {
		for _, target := range permKeys {
			if p == target {
				return true
			}
		}
	}
	return false
}
