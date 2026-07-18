package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-identity-service/internal/application/command"
	"smart-recruit-identity-service/internal/application/port"
	"smart-recruit-identity-service/internal/domain/model"
)

func TestAdminServiceRevokeRoleRollsBackWhenTokenSyncCannotFailClosed(t *testing.T) {
	authz := newMemoryAuthz()
	authz.userRoles[7] = []string{model.RoleRecruiter}
	cache := &failingTokenCache{setErr: errors.New("set failed"), deleteErr: errors.New("delete failed")}
	admin := newTestAdminService(t, AdminDeps{Authz: authz, TokenCache: cache, Audit: authz})

	err := admin.RevokeUserRole(context.Background(), command.RevokeUserRole{
		UserID:  7,
		AdminID: 99,
		RoleKey: model.RoleRecruiter,
	})
	if !errors.Is(err, ErrPermissionChangeFailed) {
		t.Fatalf("RevokeUserRole error = %v, want %v", err, ErrPermissionChangeFailed)
	}
	if !authz.restoredRole {
		t.Fatal("expected role restore when token sync cannot fail closed")
	}
	if cache.deletedUserID != 7 {
		t.Fatalf("deleted user id = %d, want 7", cache.deletedUserID)
	}
}

func TestAdminServiceAssignDataScopeAllowsCacheDeleteFailSafe(t *testing.T) {
	authz := newMemoryAuthz()
	cache := &failingTokenCache{setErr: errors.New("set failed")}
	admin := newTestAdminService(t, AdminDeps{Authz: authz, TokenCache: cache, Audit: authz})

	err := admin.AssignDataScope(context.Background(), command.AssignDataScope{
		UserID:       7,
		AdminID:      99,
		ScopeKey:     model.ScopeOwnJobs,
		ResourceType: "",
		ResourceID:   0,
	})
	if err != nil {
		t.Fatalf("AssignDataScope returned error: %v", err)
	}
	if cache.deletedUserID != 7 {
		t.Fatalf("deleted user id = %d, want 7", cache.deletedUserID)
	}
	if got := authz.assignedScopes[7]; len(got) != 1 || got[0] != model.ScopeOwnJobs {
		t.Fatalf("assigned scopes = %#v, want own_jobs", got)
	}
}

func TestAdminServiceListRolesRequiresPermission(t *testing.T) {
	admin := newTestAdminService(t, AdminDeps{Authz: newMemoryAuthz(), Authorizer: denyAuthorizer{err: errors.New("forbidden")}})
	_, err := admin.ListRoles(context.Background())
	if err == nil {
		t.Fatal("expected permission error")
	}
}

func newTestAdminService(t *testing.T, deps AdminDeps) *AdminService {
	t.Helper()
	svc, err := NewAdminService(deps)
	if err != nil {
		t.Fatalf("NewAdminService returned error: %v", err)
	}
	return svc
}

type failingTokenCache struct {
	setErr        error
	deleteErr     error
	deletedUserID uint64
}

func (c *failingTokenCache) SetTokenVersion(context.Context, uint64, int32, time.Duration) error {
	return c.setErr
}

func (c *failingTokenCache) DeleteTokenVersion(_ context.Context, userID uint64) error {
	c.deletedUserID = userID
	return c.deleteErr
}

type denyAuthorizer struct {
	err error
}

func (a denyAuthorizer) AuthorizePermission(context.Context, string) error {
	return a.err
}

var _ port.TokenVersionCache = (*failingTokenCache)(nil)
