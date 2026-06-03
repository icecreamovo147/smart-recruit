package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// ── Audit logging ──────────────────────────────────────────────────────

// AuthAuditEntry is emitted for every authorization decision.
type AuthAuditEntry struct {
	ActorUserID   int64
	ActorRoles    string
	PermissionKey string
	ResourceType  string
	ResourceID    uint64
	Decision      string // "allowed" | "denied"
	Reason        string
	RequestID     string
	ClientIP      string
}

// AuditLogger receives authorization audit events. The caller must ensure
// the implementation is non-blocking (channel-buffered or async).
type AuditLogger func(entry AuthAuditEntry)

var (
	auditLogger   AuditLogger
	auditLoggerMu sync.RWMutex
)

// SetAuditLogger configures the global audit logger. Safe for concurrent use.
func SetAuditLogger(logger AuditLogger) {
	auditLoggerMu.Lock()
	defer auditLoggerMu.Unlock()
	auditLogger = logger
}

func emitAudit(entry AuthAuditEntry) {
	auditLoggerMu.RLock()
	fn := auditLogger
	auditLoggerMu.RUnlock()
	if fn != nil {
		fn(entry)
	}
}
// ── RBAC middleware ────────────────────────────────────────────────────

func forbiddenResponse(c *gin.Context, requiredPerm string, reason string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"code": 403,
		"msg":  "无权限操作",
		"data": gin.H{
			"required_permission": requiredPerm,
			"reason":              reason,
		},
		"request_id": requestID(c),
	})
}

func auditDenied(c *gin.Context, permKey, reason string) {
	emitAudit(AuthAuditEntry{
		ActorUserID:   UserID(c),
		ActorRoles:    strings.Join(Roles(c), ","),
		PermissionKey: permKey,
		Decision:      "denied",
		Reason:        reason,
		RequestID:     requestID(c),
		ClientIP:      c.ClientIP(),
	})
}

// RequireRoleByKey checks that the authenticated user has the given RBAC role key.
func RequireRoleByKey(roleKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasRole(c, roleKey) {
			auditDenied(c, "", fmt.Sprintf("missing_role:%s", roleKey))
			forbiddenResponse(c, "", fmt.Sprintf("missing_role:%s", roleKey))
			return
		}
		c.Next()
	}
}

// RequireAnyRole checks that the authenticated user has at least one of the given role keys.
func RequireAnyRole(roleKeys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := Roles(c)
		for _, ur := range userRoles {
			for _, rk := range roleKeys {
				if ur == rk {
					c.Next()
					return
				}
			}
		}
		reason := fmt.Sprintf("missing_any_role:%s", strings.Join(roleKeys, ","))
		auditDenied(c, "", reason)
		forbiddenResponse(c, "", reason)
	}
}

// RequirePermission checks that the authenticated user has the given permission.
func RequirePermission(permKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasPermission(c, permKey) {
			auditDenied(c, permKey, "permission_denied")
			forbiddenResponse(c, permKey, "permission_denied")
			return
		}
		c.Next()
	}
}

// RequireAnyPermission checks that the authenticated user has at least one of the given permissions.
func RequireAnyPermission(permKeys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasAnyPermission(c, permKeys...) {
			permStr := strings.Join(permKeys, ",")
			auditDenied(c, permStr, "any_permission_denied")
			forbiddenResponse(c, permStr, "any_permission_denied")
			return
		}
		c.Next()
	}
}
