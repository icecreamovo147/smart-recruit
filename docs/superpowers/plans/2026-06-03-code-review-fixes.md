# Code Review Fixes — `feature/role-permission-redesign` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 20 issues (1 Critical, 6 High, 7 Medium, 6 Low) identified in code review of the RBAC permission redesign branch.

**Architecture:** The fixes span three layers — Go gRPC backend (`logic-grpc-service`), Go Gin web gateway (`web-gin-service`), and Vue 3 frontends (`hr-frontend`, `user-frontend`). Changes are organized by file to minimize churn: each task modifies one file (except the scope dedup which touches two). Backend changes are independent and can be compiled/tested individually; frontend changes are independent of backend changes.

**Tech Stack:** Go 1.x (GORM, gRPC, Gin, Redis, JWT), Vue 3 + TypeScript (Pinia, Vue Router, Element Plus)

---

## Tasks Overview

| # | Severity | Issue | File(s) |
|---|----------|-------|---------|
| 1 | 🔴 CR-1 | MigrateAllLegacyUsers never called | `logic-grpc-service/main.go` |
| 2 | 🟠 HI-1 | Token version increment failure silently tolerated | `logic-grpc-service/service/admin_service.go` |
| 3 | 🟠 HI-2, M-1 | SetTokenVersionCache + JWT generation errors discarded | `web-gin-service/handler/auth.go` |
| 4 | 🟠 HI-3 | validateTokenVersion fail-closed on Redis errors | `web-gin-service/middleware/jwt.go` |
| 5 | 🟠 HI-4 | Self-revoke protection bypassed on LoadPrincipal failure | `logic-grpc-service/service/admin_service.go` |
| 6 | 🟠 HI-5 | Registration RBAC role assignment failure silently swallowed | `logic-grpc-service/service/auth_service.go` |
| 7 | 🟠 HI-6 | Password complexity frontend validation mismatch | `hr-frontend/src/views/RegisterView.vue` |
| 8 | 🟡 M-2 | ValidateInviteCode treats DB errors as "invalid" | `logic-grpc-service/service/admin_service.go` |
| 9 | 🟡 M-3 | Transaction .Update() return value not checked | `logic-grpc-service/service/application_service.go` |
| 10 | 🟡 M-4, M-5 | AssignRole TOCTOU + RevokeDataScope dead code | `logic-grpc-service/repository/authz_repo.go` + `admin_service.go` |
| 11 | 🟡 M-6 | CreateStaffUser bypasses password complexity validation | `logic-grpc-service/service/admin_service.go` |
| 12 | 🟡 M-7 | Scope check logic duplicated across job/application services | `job_service.go` + `application_service.go` |
| 13 | 🟢 L-1 | user-frontend auth store roles/permissions no `|| []` fallback | `user-frontend/src/stores/auth.ts` |
| 14 | 🟢 L-2 | hr-frontend restoreSession catch swallows all errors | `hr-frontend/src/stores/auth.ts` |
| 15 | 🟢 L-3 | useAuthStore() called twice in router guard | `hr-frontend/src/router/index.ts` |
| 16 | 🟢 L-4, L-5 | Outbox duplicate + IncrementTokenVersion race | `outbox_publisher.go` + `authz_repo.go` |
| 17 | 🟢 L-6 | Workbench and audit log share Monitor icon | `hr-frontend/src/App.vue` |

---

### Task 1: CR-1 — Call MigrateAllLegacyUsers on startup

**Files:**
- Modify: `logic-grpc-service/main.go:189-196`

- [ ] **Step 1: Add MigrateAllLegacyUsers call after admin bootstrap block**

In `main.go`, after the `INITIAL_ADMIN_USERNAME` bootstrap block (after line 196's closing `}`), add the legacy user migration call:

```go
	// Migrate all legacy users (those without user_roles records) to RBAC.
	// Must run before gRPC server starts so all existing users have roles.
	migrated, err := authzRepo.MigrateAllLegacyUsers(ctx)
	if err != nil {
		log.Warn("legacy user migration completed with errors", zap.Error(err))
	} else {
		log.Info("legacy user migration completed", zap.Int64("migrated", migrated))
	}
```

The insertion point is after line 196 (`}` closing the admin bootstrap block) and before line 198 (`// Start background workers`).

- [ ] **Step 2: Build and verify compilation**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds without errors.

---

### Task 2: HI-1 — Return error when IncrementTokenVersion fails in admin mutations

**Files:**
- Modify: `logic-grpc-service/service/admin_service.go:355-356, 411-412, 436-437, 470-471`

**Context:** Four admin mutation methods (`AssignUserRole`, `RevokeUserRole`, `AssignDataScope`, `RevokeDataScope`) silently tolerate `IncrementTokenVersion` failure, then proceed to success response. The permission change is committed to DB but token version isn't bumped → old JWT remains valid for 24h.

- [ ] **Step 1: Fix AssignUserRole (line 355-356)**

Replace:
```go
		if newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId)); err != nil {
			logger.L().Warn("increment token version failed after role assign", zap.Error(err))
		} else if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
```

With:
```go
		newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId))
		if err != nil {
			logger.L().Error("increment token version failed after role assign, permission change may not be enforced",
				zap.Error(err), zap.Int64("user_id", req.UserId))
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
		if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
```

- [ ] **Step 2: Fix RevokeUserRole (line 411-412)**

Replace:
```go
		if newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId)); err != nil {
			logger.L().Warn("increment token version failed after role revoke", zap.Error(err))
		} else if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
```

With:
```go
		newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId))
		if err != nil {
			logger.L().Error("increment token version failed after role revoke, permission change may not be enforced",
				zap.Error(err), zap.Int64("user_id", req.UserId))
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
		if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
```

- [ ] **Step 3: Fix AssignDataScope (line 436-437)**

Replace:
```go
		if newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId)); err != nil {
			logger.L().Warn("increment token version failed after scope assign", zap.Error(err))
		} else if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
```

With:
```go
		newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId))
		if err != nil {
			logger.L().Error("increment token version failed after scope assign, permission change may not be enforced",
				zap.Error(err), zap.Int64("user_id", req.UserId))
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
		if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
```

- [ ] **Step 4: Fix RevokeDataScope (line 470-471)**

Replace:
```go
		if scopeUserID > 0 {
			if newVersion, err := s.authz.IncrementTokenVersion(ctx, scopeUserID); err != nil {
				logger.L().Warn("increment token version failed after scope revoke", zap.Error(err))
			} else if err := s.setTokenVersionCache(ctx, scopeUserID, newVersion); err != nil {
```

With:
```go
		if scopeUserID > 0 {
			newVersion, err := s.authz.IncrementTokenVersion(ctx, scopeUserID)
			if err != nil {
				logger.L().Error("increment token version failed after scope revoke, permission change may not be enforced",
					zap.Error(err), zap.Uint64("user_id", scopeUserID))
				return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
			}
			if err := s.setTokenVersionCache(ctx, scopeUserID, newVersion); err != nil {
```

- [ ] **Step 5: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 3: HI-2 + M-1 — Fix error handling in web-gin-service auth handler

**Files:**
- Modify: `web-gin-service/handler/auth.go:96-100, 114-116, 230-234, 248-250`

**Issues:**
- HI-2: `SetTokenVersionCache` errors discarded with `_ =` at Login and RefreshToken → can cause infinite 401 loop
- M-1: `jwt.GenerateFull` errors discarded with `_` at Login and RefreshToken → silent JWT generation failure

- [ ] **Step 1: Fix Login — JWT generation error (line 96)**

Replace:
```go
			accessToken, _ := jwt.GenerateFull(
				h.jwtSecret, resp.UserId, resp.Username, resp.Role,
				resp.AccountType, resp.Roles, resp.Permissions, resp.TokenVersion,
				jwt.AccessTokenTTL,
			)
```

With:
```go
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
```

Note: `logger` import from `web-gin-service/pkg/logger` needs to be available. Check if already imported.

- [ ] **Step 1b: Fix Login — SetTokenVersionCache error (lines 114-116)**

Replace:
```go
				if h.rdb != nil && resp.TokenVersion > 0 {
					_ = middleware.SetTokenVersionCache(c.Request.Context(), h.rdb, resp.UserId, resp.TokenVersion)
				}
```

With:
```go
				if h.rdb != nil && resp.TokenVersion > 0 {
					if err := middleware.SetTokenVersionCache(c.Request.Context(), h.rdb, resp.UserId, resp.TokenVersion); err != nil {
						logger.L().Warn("failed to cache token_version after login, user may face auth issues",
							zap.Int64("user_id", resp.UserId), zap.Error(err))
					}
				}
```

- [ ] **Step 2: Fix RefreshToken — JWT generation error (line 230)**

Replace:
```go
		accessToken, _ := jwt.GenerateFull(
			h.jwtSecret, resp.UserId, resp.Username, resp.Role,
			resp.AccountType, resp.Roles, resp.Permissions, resp.TokenVersion,
			jwt.AccessTokenTTL,
		)
```

With:
```go
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
```

- [ ] **Step 2b: Fix RefreshToken — SetTokenVersionCache error (lines 248-250)**

Replace:
```go
		if h.rdb != nil && resp.TokenVersion > 0 {
			_ = middleware.SetTokenVersionCache(c.Request.Context(), h.rdb, resp.UserId, resp.TokenVersion)
		}
```

With:
```go
		if h.rdb != nil && resp.TokenVersion > 0 {
			if err := middleware.SetTokenVersionCache(c.Request.Context(), h.rdb, resp.UserId, resp.TokenVersion); err != nil {
				logger.L().Warn("failed to cache token_version after refresh, user may face auth issues",
					zap.Int64("user_id", resp.UserId), zap.Error(err))
			}
		}
```

- [ ] **Step 3: Add required imports**

Check that `web-gin-service/handler/auth.go` imports:
- `"go.uber.org/zap"` — for `zap.Error`, `zap.Int64`
- `"web-gin-service/pkg/logger"` — for `logger.L()`

If `zap` is not imported, add `"go.uber.org/zap"` to the import block.
If `logger` is not imported, add `"web-gin-service/pkg/logger"` to the import block.

- [ ] **Step 4: Build and verify**

```bash
cd web-gin-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 4: HI-3 — Fix validateTokenVersion to distinguish Redis Nil from Redis errors

**Files:**
- Modify: `web-gin-service/middleware/jwt.go:91-99`

**Context:** When Redis is unreachable, `validateTokenVersion` returns `false` for every request → all users get 401. Need to differentiate "key genuinely not found" (fail-closed) from "Redis unreachable" (fail-open with degraded mode).

- [ ] **Step 1: Add required imports**

Ensure `web-gin-service/middleware/jwt.go` imports:
```go
	"errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"web-gin-service/pkg/logger"
```

Check which are already present. `"github.com/redis/go-redis/v9"` is already imported. Need to add `"errors"`, `"go.uber.org/zap"`, and `"web-gin-service/pkg/logger"` if not present.

- [ ] **Step 2: Fix validateTokenVersion function (lines 91-99)**

Replace the entire `validateTokenVersion` function:

```go
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
		logger.L().Warn("token_version check skipped: redis unavailable",
			zap.Int64("user_id", userID), zap.Error(err))
		return true
	}
	return int32(stored) <= jwtVersion
}
```

Also update the doc comment on line 85-89 to reflect the new behavior:

Replace:
```go
// Fail-closed strategy:
// - Redis hit: compare stored version <= jwt version, reject if stored > jwt.
// - Redis miss (key not found) or error: reject — cannot verify, deny for safety.
//   SetTokenVersionCache is always called at login/refresh before returning the
//   access token, so cache absence means Redis restart/eviction — force refresh.
```

With:
```go
// Strategy:
// - Redis hit: compare stored version <= jwt version, reject if stored > jwt.
// - Redis miss (redis.Nil): reject — key genuinely doesn't exist, force re-login.
// - Redis error (unreachable): allow through in degraded mode — JWT signature is
//   still verified, we just can't check token version revocation.
```

- [ ] **Step 3: Build and verify**

```bash
cd web-gin-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 5: HI-4 — Fix self-revoke protection when LoadPrincipal fails

**Files:**
- Modify: `logic-grpc-service/service/admin_service.go:394-403`

**Context:** In `RevokeUserRole`, when the admin is revoking their own `system_admin` role, `LoadPrincipal` failure silently skips the self-revoke guard — the revoke proceeds and the admin loses all access with no recovery.

- [ ] **Step 1: Fix the self-revoke guard in RevokeUserRole (lines 394-403)**

Replace:
```go
			if uint64(req.UserId) == adminID {
				principal, err := s.authz.LoadPrincipal(ctx, adminID)
				if err == nil && principal != nil {
					hasOtherAdmin := principal.HasRole("recruiting_admin")
					if !hasOtherAdmin {
						s.auditAdminAction(ctx, adminID, "revoke_role", uint64(req.UserId), "denied",
							"self-revoke of own system_admin blocked", "", "")
						return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "不能移除自己的系统管理员角色，请让其他系统管理员操作"}, nil
					}
				}
			}
```

With:
```go
			if uint64(req.UserId) == adminID {
				principal, err := s.authz.LoadPrincipal(ctx, adminID)
				if err != nil {
					logger.L().Error("cannot verify admin role safety, blocking self-revoke",
						zap.Error(err), zap.Int64("user_id", req.UserId))
					return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "无法验证管理员角色，请稍后重试"}, nil
				}
				if principal != nil {
					hasOtherAdmin := principal.HasRole("recruiting_admin")
					if !hasOtherAdmin {
						s.auditAdminAction(ctx, adminID, "revoke_role", uint64(req.UserId), "denied",
							"self-revoke of own system_admin blocked", "", "")
						return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "不能移除自己的系统管理员角色，请让其他系统管理员操作"}, nil
					}
				}
			}
```

- [ ] **Step 2: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 6: HI-5 — Fix registration RBAC role assignment error handling

**Files:**
- Modify: `logic-grpc-service/service/auth_service.go:160-174, 189-201`

**Context:** During registration, RBAC role assignment failures are silently swallowed. If the roles table is empty (seed not run), users are created without roles and can't access anything.

- [ ] **Step 1: Fix candidate role assignment (lines 161-167)**

Replace:
```go
		if role == 1 {
			// Candidate self-registration
			candidateRole, err := s.authz.GetRoleByKey(ctx, authz.RoleCandidate)
			if err == nil {
				_ = s.authz.AssignRole(ctx, uint64(user.ID), candidateRole.ID, nil)
			}
		} else if inviteCodeRecord != nil {
```

With:
```go
		if role == 1 {
			// Candidate self-registration
			candidateRole, err := s.authz.GetRoleByKey(ctx, authz.RoleCandidate)
			if err != nil || candidateRole == nil {
				log.Error("candidate role not found, RBAC seed may be missing", zap.Error(err))
				return &pb.RegisterResponse{Code: errs.ErrInternal, Msg: "系统初始化未完成，请联系管理员"}, nil
			}
			if err := s.authz.AssignRole(ctx, uint64(user.ID), candidateRole.ID, nil); err != nil {
				log.Error("assign candidate role failed during registration", zap.Error(err))
				return &pb.RegisterResponse{Code: errs.ErrInternal, Msg: "账号创建失败，请稍后重试"}, nil
			}
		} else if inviteCodeRecord != nil {
```

- [ ] **Step 2: Fix assignStaffRolesAndScopes to propagate errors (lines 189-201)**

Replace the entire `assignStaffRolesAndScopes` function to return errors instead of logging warnings:

```go
// assignStaffRolesAndScopes assigns the recruiter RBAC role and own_jobs data scope
// for staff users registering via invite code. Returns error if RBAC assignment fails.
func (s *AuthService) assignStaffRolesAndScopes(ctx context.Context, userID int64, inviterID uint64, log *zap.Logger) error {
	uid := uint64(userID)

	recruiterRole, err := s.authz.GetRoleByKey(ctx, authz.RoleRecruiter)
	if err != nil || recruiterRole == nil {
		log.Error("recruiter role not found, RBAC seed may be missing", zap.Error(err))
		return fmt.Errorf("recruiter role not found: %w", err)
	}
	if err := s.authz.AssignRole(ctx, uid, recruiterRole.ID, &inviterID); err != nil {
		log.Error("assign recruiter role failed during registration", zap.Error(err))
		return fmt.Errorf("assign recruiter role: %w", err)
	}
	if err := s.authz.AssignDataScope(ctx, uid, "own_jobs", "", 0, &inviterID); err != nil {
		log.Error("assign own_jobs scope failed during registration", zap.Error(err))
		return fmt.Errorf("assign own_jobs scope: %w", err)
	}
	return nil
}
```

Note: Need to add `"fmt"` to imports if not already present.

- [ ] **Step 3: Update the call site in Register (line 172)**

Replace:
```go
			s.assignStaffRolesAndScopes(ctx, user.ID, inviterID, log)
```

With:
```go
			if err := s.assignStaffRolesAndScopes(ctx, user.ID, inviterID, log); err != nil {
				log.Error("staff role assignment failed during registration", zap.Error(err))
				return &pb.RegisterResponse{Code: errs.ErrInternal, Msg: "账号创建失败，请稍后重试"}, nil
			}
```

- [ ] **Step 4: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 7: HI-6 — Fix password complexity frontend validation to match backend

**Files:**
- Modify: `hr-frontend/src/views/RegisterView.vue:27-38`

**Context:** Frontend regex requires (lowercase AND uppercase AND (digit OR special)) — a strict 3-category subset. Backend accepts any 3 of 4 categories. Two valid backend passwords (`abc123!@#`, `ABC123!@#`) are incorrectly rejected by the frontend.

- [ ] **Step 1: Replace the password validation rule (lines 27-38)**

Replace:
```typescript
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
    {
      pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d|.*[!@#$%^&*])/,
      message: '密码需包含大小写字母、数字或特殊字符中的至少三类',
      trigger: 'blur',
    },
  ],
}
```

With:
```typescript
const validatePasswordComplexity = (_rule: any, value: string, callback: any) => {
  if (!value) { callback(); return }
  let categories = 0
  if (/[a-z]/.test(value)) categories++
  if (/[A-Z]/.test(value)) categories++
  if (/\d/.test(value)) categories++
  if (/[!@#$%^&*]/.test(value)) categories++
  if (categories >= 3) { callback(); return }
  callback(new Error('密码需包含大小写字母、数字或特殊字符中的至少三类'))
}

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
    { validator: validatePasswordComplexity, trigger: 'blur' },
  ],
}
```

- [ ] **Step 2: Run TypeScript type check**

```bash
cd hr-frontend && npx vue-tsc --noEmit
```

Expected: No new type errors.

---

### Task 8: M-2 — Distinguish gorm.ErrRecordNotFound from DB errors in ValidateInviteCode

**Files:**
- Modify: `logic-grpc-service/service/admin_service.go:185-189`

- [ ] **Step 1: Check imports**

Ensure `"errors"` and `"gorm.io/gorm"` are in the import block. `gorm.io/gorm` should already be imported (it's used elsewhere). `"errors"` may need to be added.

- [ ] **Step 2: Fix ValidateInviteCode (lines 185-189)**

Replace:
```go
	_, err := s.inviteCodes.GetByCode(ctx, req.InviteCode)
	if err != nil {
		return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: false}, nil
	}
```

With:
```go
	_, err := s.inviteCodes.GetByCode(ctx, req.InviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: false}, nil
		}
		logger.L().Error("validate invite code: db error", zap.Error(err))
		return nil, err
	}
```

- [ ] **Step 3: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 9: M-3 — Check .Update() return value in transaction

**Files:**
- Modify: `logic-grpc-service/service/application_service.go:286-291`

- [ ] **Step 1: Fix the transaction update calls (lines 286-291)**

Replace:
```go
		if rows > 0 && req.Status == 3 {
			tx.Model(&model.Application{}).Where("id = ?", req.ApplicationId).Update("is_current", 0)
		}
		if rows > 0 && req.Status == 2 && detail.Status == 3 {
			tx.Model(&model.Application{}).Where("user_id = ? AND job_id = ? AND is_current = 1", detail.UserID, detail.JobID).Update("is_current", 0)
			tx.Model(&model.Application{}).Where("id = ?", req.ApplicationId).Update("is_current", 1)
		}
```

With:
```go
		if rows > 0 && req.Status == 3 {
			if err := tx.Model(&model.Application{}).Where("id = ?", req.ApplicationId).Update("is_current", 0).Error; err != nil {
				return err
			}
		}
		if rows > 0 && req.Status == 2 && detail.Status == 3 {
			if err := tx.Model(&model.Application{}).Where("user_id = ? AND job_id = ? AND is_current = 1", detail.UserID, detail.JobID).Update("is_current", 0).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Application{}).Where("id = ?", req.ApplicationId).Update("is_current", 1).Error; err != nil {
				return err
			}
		}
```

- [ ] **Step 2: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 10: M-4 + M-5 — Wrap AssignRole in transaction + remove RevokeDataScope dead code

**Files:**
- Modify: `logic-grpc-service/repository/authz_repo.go:98-117`
- Modify: `logic-grpc-service/service/admin_service.go:456-458`

- [ ] **Step 1: Fix AssignRole TOCTOU (authz_repo.go lines 98-117)**

Replace the `AssignRole` function body with a transaction-wrapped version:

```go
// AssignRole grants a role to a user. Uses a transaction to prevent TOCTOU race
// between the duplicate check and the insert. Returns an error if the assignment
// already exists and is active.
func (r *AuthzRepo) AssignRole(ctx context.Context, userID, roleID uint64, assignedBy *uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.UserRole{}).
			Where("user_id = ? AND role_id = ? AND revoked_at IS NULL", userID, roleID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("user %d already has role %d", userID, roleID)
		}

		ur := &model.UserRole{
			UserID:     userID,
			RoleID:     roleID,
			AssignedBy: assignedBy,
			AssignedAt: time.Now(),
		}
		return tx.Create(ur).Error
	})
}
```

- [ ] **Step 2: Remove RevokeDataScope dead code (admin_service.go lines 455-458)**

Replace:
```go
	// Look up the scope to get the affected user_id before revoking.
	scopes, err := s.authz.GetUserDataScopes(ctx, 0) // GetUserDataScopes filters by user_id; we need a different approach
	_ = scopes // placeholder
	// Instead, look up the specific scope record to find its user_id.
	// We'll query the scope directly through the authz repo.
	scopeUserID, err := s.authz.GetScopeOwnerID(ctx, req.ScopeId)
```

With:
```go
	// Look up the scope owner before revoking.
	scopeUserID, err := s.authz.GetScopeOwnerID(ctx, req.ScopeId)
```

- [ ] **Step 3: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 11: M-6 — Call validatePassword in CreateStaffUser

**Files:**
- Modify: `logic-grpc-service/service/admin_service.go:522-525`

- [ ] **Step 1: Fix CreateStaffUser password validation (lines 522-525)**

Replace:
```go
	// Validate password
	if len(req.Password) < 8 {
		return &pb.CreateStaffUserResponse{Code: errs.ErrBadRequest, Msg: "密码长度至少8个字符"}, nil
	}
```

With:
```go
	// Validate password — use the same complexity rules as registration.
	if err := validatePassword(req.Password); err != nil {
		return &pb.CreateStaffUserResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
```

- [ ] **Step 2: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds (note: `validatePassword` is defined in `auth_service.go` within the same package, so it's directly callable).

---

### Task 12: M-7 — Extract common scope evaluator to reduce duplication

**Files:**
- Modify: `logic-grpc-service/service/job_service.go:42-144`
- Modify: `logic-grpc-service/service/application_service.go:50-137`
- Create: `logic-grpc-service/service/scope_evaluator.go` (new file)

**Context:** `checkJobScope` and `checkApplicationJobScope` implement nearly identical scope-checking logic (scan scopes → check full access → check own_jobs → check dept/loc → check interviews). Extracting the common logic prevents drift when new scope types are added.

- [ ] **Step 1: Create the new scope_evaluator.go file**

Create `logic-grpc-service/service/scope_evaluator.go`:

```go
package service

import (
	"context"
	"fmt"

	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/repository"
)

// scopeEvaluator extracts the common data-scope authorization logic shared by
// JobService and ApplicationService. It answers: given a user's scope keys and
// an optional job, what level of access does the user have?
type scopeEvaluator struct {
	authzRepo *repository.AuthzRepo
}

// evalScope returns the effective access level for a user on a given job.
// Pass jobID=0 for create-like operations where no existing job is checked.
// Pass a non-nil jobGetter to fetch job details on demand (avoids a DB call
// when scope is already full-access).
func (e *scopeEvaluator) evalScope(ctx context.Context, userID uint64, jobGetter func() (*jobScopeTarget, error)) (ScopeLevel, error) {
	if e.authzRepo == nil {
		return scopeFull, nil
	}

	scopeKeys, err := e.authzRepo.GetUserScopeKeys(ctx, userID)
	if err != nil {
		return scopeDenied, fmt.Errorf("scope lookup failed: %w", err)
	}

	// Full access scopes — bypass all checks.
	for _, sk := range scopeKeys {
		if sk == authz.ScopeRecruitingAll || sk == authz.ScopeSystemAll {
			return scopeFull, nil
		}
	}

	// Detect scope categories present.
	hasOwnJobs, hasDept, hasLoc, hasInterview := false, false, false, false
	for _, sk := range scopeKeys {
		switch sk {
		case authz.ScopeOwnJobs:
			hasOwnJobs = true
		case authz.ScopeDepartment:
			hasDept = true
		case authz.ScopeLocation:
			hasLoc = true
		case authz.ScopeAssignedInterviews:
			hasInterview = true
		}
	}

	// If no job to check against (create-like operations), any scope is enough.
	if jobGetter == nil {
		if hasOwnJobs || hasDept || hasLoc {
			return scopeOwned, nil
		}
		return scopeDenied, fmt.Errorf("no valid data scope assigned")
	}

	// Fetch the target job to validate ownership/dept/loc/interview scope.
	job, err := jobGetter()
	if err != nil {
		return scopeDenied, err
	}
	if job == nil {
		return scopeDenied, fmt.Errorf("target job not found")
	}

	if hasOwnJobs && job.HrID == int64(userID) {
		return scopeOwned, nil
	}
	if hasDept {
		deptIDs, err := e.authzRepo.GetUserDepartmentIDs(ctx, userID)
		if err != nil {
			return scopeDenied, fmt.Errorf("department scope lookup: %w", err)
		}
		if job.DepartmentID != nil {
			for _, dID := range deptIDs {
				if uint64(*job.DepartmentID) == dID {
					return scopeDepartmentOrLocation, nil
				}
			}
		}
	}
	if hasLoc {
		locIDs, err := e.authzRepo.GetUserLocationIDs(ctx, userID)
		if err != nil {
			return scopeDenied, fmt.Errorf("location scope lookup: %w", err)
		}
		if job.LocationID != nil {
			for _, lID := range locIDs {
				if uint64(*job.LocationID) == lID {
					return scopeDepartmentOrLocation, nil
				}
			}
		}
	}
	if hasInterview {
		isInterviewer, err := e.authzRepo.IsInterviewerForJob(ctx, userID, uint64(job.ID))
		if err != nil {
			return scopeDenied, fmt.Errorf("interviewer scope lookup: %w", err)
		}
		if isInterviewer {
			return scopeOwned, nil
		}
	}

	return scopeDenied, fmt.Errorf("scope denied for user %d", userID)
}

// jobScopeTarget carries the fields needed for scope evaluation against a job.
type jobScopeTarget struct {
	ID           int64
	HrID         int64
	DepartmentID *int
	LocationID   *int
}
```

- [ ] **Step 2: Add scopeEvaluator to Services struct and constructor**

Check the `Services` struct in `logic-grpc-service/service/services.go` for where to add the field. Add:
```go
	scopeEval *scopeEvaluator
```

In `NewServices`, after `authzRepo` is available, add:
```go
	scopeEval := &scopeEvaluator{authzRepo: authzRepo}
```

And pass `scopeEval` to both `JobService` and `ApplicationService` constructors.

- [ ] **Step 3: Update JobService to use scopeEvaluator**

In `job_service.go`:
- Replace `checkJobScope` method body with a call to `s.scopeEval.evalScope`
- The method signature remains the same for backward compatibility

```go
func (s *JobService) checkJobScope(ctx context.Context, userID int64, _permKey string, jobID int64) (ScopeLevel, error) {
	var jobGetter func() (*jobScopeTarget, error)
	if jobID > 0 {
		jobGetter = func() (*jobScopeTarget, error) {
			job, err := s.jobs.GetByID(ctx, jobID)
			if err != nil {
				return nil, fmt.Errorf("job lookup failed: %w", err)
			}
			if job == nil {
				return nil, fmt.Errorf("job %d not found", jobID)
			}
			return &jobScopeTarget{
				ID:           job.ID,
				HrID:         job.HrID,
				DepartmentID: job.DepartmentID,
				LocationID:   job.LocationID,
			}, nil
		}
	}
	return s.scopeEval.evalScope(ctx, uint64(userID), jobGetter)
}
```

- [ ] **Step 4: Update ApplicationService to use scopeEvaluator**

In `application_service.go`, replace `checkApplicationJobScope` body similarly:

```go
func (s *ApplicationService) checkApplicationJobScope(ctx context.Context, userID int64, jobID int64) (ScopeLevel, error) {
	var jobGetter func() (*jobScopeTarget, error)
	if jobID > 0 {
		jobGetter = func() (*jobScopeTarget, error) {
			job, err := s.jobs.GetByID(ctx, jobID)
			if err != nil {
				return nil, err
			}
			if job == nil {
				return nil, fmt.Errorf("job %d not found", jobID)
			}
			return &jobScopeTarget{
				ID:           job.ID,
				HrID:         job.HrID,
				DepartmentID: job.DepartmentID,
				LocationID:   job.LocationID,
			}, nil
		}
	}
	return s.scopeEval.evalScope(ctx, uint64(userID), jobGetter)
}
```

- [ ] **Step 5: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 13: L-1 — Add `|| []` fallback for roles/permissions in user-frontend auth store

**Files:**
- Modify: `user-frontend/src/stores/auth.ts:27-28`

- [ ] **Step 1: Add fallback to login action**

Replace lines 27-28:
```typescript
        roles: data.roles,
        permissions: data.permissions,
```

With:
```typescript
        roles: data.roles || [],
        permissions: data.permissions || [],
```

- [ ] **Step 2: Run TypeScript type check**

```bash
cd user-frontend && npx vue-tsc --noEmit
```

Expected: No new type errors.

---

### Task 14: L-2 — Improve hr-frontend restoreSession error handling

**Files:**
- Modify: `hr-frontend/src/stores/auth.ts:107-109`

- [ ] **Step 1: Add error logging to catch block**

Replace:
```typescript
      } catch {
        return false
      }
```

With:
```typescript
      } catch (err) {
        console.error('[auth] restoreSession failed:', err)
        return false
      }
```

- [ ] **Step 2: Run TypeScript type check**

```bash
cd hr-frontend && npx vue-tsc --noEmit
```

Expected: No new type errors.

---

### Task 15: L-3 — Avoid duplicate useAuthStore() call in router guard

**Files:**
- Modify: `hr-frontend/src/router/index.ts:93-117`

- [ ] **Step 1: Consolidate to single useAuthStore() call**

Replace the entire `beforeEach` callback (lines 93-117):

```typescript
router.beforeEach(async (to, _from, next) => {
  const auth = useAuthStore()
  let user = getUser()

  if (to.meta.requiresAuth && !user) {
    // Try restoring session from httpOnly cookie before rejecting.
    await auth.restoreSession()
    user = getUser()
  }
  if (to.meta.requiresAuth && !user) {
    next('/login')
    return
  }

  // Permission-based guard
  const requiredPerm = to.meta.requiresPermission as string | undefined
  if (requiredPerm && user) {
    if (!auth.hasPermission(requiredPerm)) {
      next('/403')
      return
    }
  }

  next()
})
```

- [ ] **Step 2: Run TypeScript type check**

```bash
cd hr-frontend && npx vue-tsc --noEmit
```

Expected: No new type errors.

---

### Task 16: L-4 + L-5 — Outbox MarkPublished error logging + IncrementTokenVersion transaction

**Files:**
- Modify: `logic-grpc-service/service/outbox_publisher.go:164-166`
- Modify: `logic-grpc-service/repository/authz_repo.go:561-572`

- [ ] **Step 1: Improve outbox MarkPublished error logging (outbox_publisher.go lines 164-166)**

Replace:
```go
	if err := p.repo.MarkPublished(ctx, ev.ID); err != nil {
		logger.L().Error("outbox mark published error", zap.Error(err))
	}
```

With:
```go
	if err := p.repo.MarkPublished(ctx, ev.ID); err != nil {
		// Message was already published to MQ — a duplicate delivery is possible.
		// The consumer must be idempotent (e.g. dedup by event_id).
		logger.L().Error("outbox mark published failed after MQ publish, duplicate delivery possible",
			zap.String("event_id", ev.EventID), zap.Error(err))
	}
```

- [ ] **Step 2: Wrap IncrementTokenVersion in transaction (authz_repo.go lines 561-572)**

Replace:
```go
func (r *AuthzRepo) IncrementTokenVersion(ctx context.Context, userID uint64) (int32, error) {
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		return 0, err
	}
	var user model.User
	if err := r.db.WithContext(ctx).Select("token_version").First(&user, userID).Error; err != nil {
		return 0, err
	}
	return user.TokenVersion, nil
}
```

With:
```go
func (r *AuthzRepo) IncrementTokenVersion(ctx context.Context, userID uint64) (int32, error) {
	var newVersion int32
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).
			Where("id = ?", userID).
			Update("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
			return err
		}
		var user model.User
		if err := tx.Select("token_version").First(&user, userID).Error; err != nil {
			return err
		}
		newVersion = user.TokenVersion
		return nil
	})
	return newVersion, err
}
```

- [ ] **Step 3: Build and verify**

```bash
cd logic-grpc-service && go build ./...
```

Expected: Compilation succeeds.

---

### Task 17: L-6 — Use distinct icon for audit log sidebar item

**Files:**
- Modify: `hr-frontend/src/App.vue:118-121`

- [ ] **Step 1: Check imports for available icons**

Check `hr-frontend/src/App.vue` for the icon import block. Need to find an appropriate icon to replace `Monitor` for the audit log entry. `DataAnalysis`, `Document`, or `List` would be appropriate. If Element Plus icons don't have a perfect match, `DataLine` or `Histogram` work well for audit/analytics.

First, check what's imported. Then:

Replace line 119:
```html
        <el-icon><Monitor /></el-icon>
```

With (using `DataAnalysis` which is commonly available in `@element-plus/icons-vue`):
```html
        <el-icon><DataAnalysis /></el-icon>
```

Add the import for `DataAnalysis` in the script section if not already present.

- [ ] **Step 2: Run TypeScript type check**

```bash
cd hr-frontend && npx vue-tsc --noEmit
```

Expected: No new type errors.

---

## Verification Checklist

After implementing all tasks, run the full verification suite:

```bash
# Backend compilation
cd logic-grpc-service && go build ./...
cd web-gin-service && go build ./...

# Frontend type check
cd hr-frontend && npx vue-tsc --noEmit
cd user-frontend && npx vue-tsc --noEmit

# Tests (if integration environment available)
cd logic-grpc-service
go test -tags=integration ./... -run "TestIntegration" -v

# Service-level RBAC tests
go test ./service/... -run "RBAC" -v
```

---

## Risk Assessment

| Task | Risk | Mitigation |
|------|------|------------|
| 1 (CR-1) | Low — adds new startup call, no existing behavior changed | Migration is idempotent (skips users with existing user_roles) |
| 2 (HI-1) | Low — changes warn→error for token version failures | Restores proper semantics; errors were already possible in the else-branch |
| 3 (HI-2, M-1) | Medium — changes control flow in login/refresh | JWT.GenerateFull only fails on extreme edge cases; SetTokenVersionCache change is warn-only |
| 4 (HI-3) | Medium — changes fail-closed to fail-open on Redis errors | Degraded mode still verifies JWT signature; worse case is stale tokens not being rejected during Redis outage |
| 5 (HI-4) | Low — adds early return on error | Conservative: blocks operation when can't verify safety |
| 6 (HI-5) | Medium — changes registration flow to return errors | Users who would have gotten zombie accounts now get clear error messages |
| 7 (HI-6) | Low — frontend-only validation change | Relaxes frontend check to match backend; fewer false rejections |
| 8-11 (M-2–M-6) | Low — targeted fixes with clear semantics | Each change is isolated and well-understood |
| 12 (M-7) | Medium — refactoring shared logic | Scope evaluator has identical logic to original; verify with existing tests |
| 13-17 (L-1–L-6) | Low — cosmetic/documentation improvements | No behavioral changes to core logic |
