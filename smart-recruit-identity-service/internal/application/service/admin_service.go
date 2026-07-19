package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commonsquota "smart-recruit-commons/quota"
	"smart-recruit-identity-service/internal/application/command"
	"smart-recruit-identity-service/internal/application/dto"
	"smart-recruit-identity-service/internal/application/port"
	"smart-recruit-identity-service/internal/application/query"
	"smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/policy"
	"smart-recruit-identity-service/internal/domain/repository"
	platformmetadata "smart-recruit-platform-go/metadata"
)

var (
	ErrRoleNotFound              = errors.New("角色不存在")
	ErrRoleNotHeld               = errors.New("该用户未持有此角色")
	ErrPermissionChangeFailed    = errors.New("权限变更失败，请稍后重试")
	ErrPermissionTokenSyncFailed = errors.New("权限变更成功但令牌同步失败，请重试")
	ErrAuditQueryFailed          = errors.New("查询安全审计日志失败")
)

type AdminDeps struct {
	Users      repository.UserRepository
	Authz      repository.AuthzRepository
	Audit      repository.AuditRepository
	Passwords  port.PasswordService
	Authorizer port.AdminAuthorizer
	TokenCache port.TokenVersionCache
	Tenants    repository.TenantRepository
}

type AdminService struct {
	users      repository.UserRepository
	authz      repository.AuthzRepository
	audit      repository.AuditRepository
	passwords  port.PasswordService
	authorizer port.AdminAuthorizer
	tokenCache port.TokenVersionCache
	tenants    repository.TenantRepository
}

func NewAdminService(deps AdminDeps) (*AdminService, error) {
	if deps.Authz == nil {
		return nil, errors.New("authz repository is required")
	}
	return &AdminService{
		users:      deps.Users,
		authz:      deps.Authz,
		audit:      deps.Audit,
		passwords:  deps.Passwords,
		authorizer: deps.Authorizer,
		tokenCache: deps.TokenCache,
		tenants:    deps.Tenants,
	}, nil
}

func (s *AdminService) ListRoles(ctx context.Context) ([]model.Role, error) {
	if err := s.requireAdminPermission(ctx, model.PermAdminRoleManage); err != nil {
		return nil, err
	}
	return s.authz.ListRoles(ctx)
}

func (s *AdminService) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	if err := s.requireAdminPermission(ctx, model.PermAdminRoleManage); err != nil {
		return nil, err
	}
	return s.authz.ListPermissions(ctx)
}

func (s *AdminService) GetUserRoles(ctx context.Context, req query.GetUserRoles) (dto.UserRolesResult, error) {
	if err := s.requireAdminPermission(ctx, model.PermAdminRoleManage); err != nil {
		return dto.UserRolesResult{}, err
	}
	if s.tenants != nil {
		tenantID := platformmetadata.GetAuthTenantID(ctx)
		principal, err := s.tenants.LoadTenantPrincipal(ctx, req.UserID, tenantID, "staff")
		if err != nil || principal == nil {
			return dto.UserRolesResult{}, ErrRoleNotHeld
		}
		scopes := make([]model.DataScope, len(principal.DataScopes))
		for i, scope := range principal.DataScopes {
			scopes[i] = model.DataScope{ID: uint64(scope.ID), UserID: uint64(req.UserID), ScopeKey: scope.ScopeKey, ResourceType: scope.ResourceType, ResourceID: uint64(scope.ResourceID), AssignedAt: scope.AssignedAt}
		}
		return dto.UserRolesResult{RoleKeys: principal.Roles, PermissionKeys: principal.Permissions, DataScopes: scopes}, nil
	}
	roleKeys, err := s.authz.GetUserRoles(ctx, uint64(req.UserID))
	if err != nil {
		return dto.UserRolesResult{}, err
	}
	permissionKeys, err := s.authz.GetUserPermissions(ctx, uint64(req.UserID))
	if err != nil {
		return dto.UserRolesResult{}, err
	}
	scopes, err := s.authz.GetUserDataScopes(ctx, uint64(req.UserID))
	if err != nil {
		return dto.UserRolesResult{}, err
	}
	return dto.UserRolesResult{RoleKeys: roleKeys, PermissionKeys: permissionKeys, DataScopes: scopes}, nil
}

func (s *AdminService) AssignUserRole(ctx context.Context, cmd command.AssignUserRole) error {
	role, err := s.roleByKey(ctx, cmd.RoleKey)
	if err != nil {
		return err
	}
	adminID := uint64(cmd.AdminID)
	if s.tenants != nil {
		tenantID := platformmetadata.GetAuthTenantID(ctx)
		membership, err := s.tenants.GetActiveMembership(ctx, cmd.UserID, tenantID)
		if err != nil || membership == nil || role.ScopeType != model.RoleScopeTenant {
			return ErrRoleNotFound
		}
		if err := s.tenants.AssignMembershipRole(ctx, membership.ID, role.ID, &adminID); err != nil {
			return err
		}
		return s.bumpAuthorizationVersion(ctx, uint64(cmd.UserID))
	}
	if err := s.authz.AssignRole(ctx, uint64(cmd.UserID), role.ID, &adminID); err != nil {
		return err
	}
	newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(cmd.UserID))
	if err != nil {
		return ErrPermissionTokenSyncFailed
	}
	if err := s.syncTokenVersion(ctx, uint64(cmd.UserID), newVersion); err != nil {
		return ErrPermissionTokenSyncFailed
	}
	s.auditAdminAction(ctx, adminID, "assign_role", uint64(cmd.UserID), "allowed", "assigned role "+cmd.RoleKey)
	return nil
}

func (s *AdminService) RevokeUserRole(ctx context.Context, cmd command.RevokeUserRole) error {
	role, err := s.roleByKey(ctx, cmd.RoleKey)
	if err != nil {
		return err
	}
	adminID := uint64(cmd.AdminID)
	if s.tenants != nil {
		tenantID := platformmetadata.GetAuthTenantID(ctx)
		membership, err := s.tenants.GetActiveMembership(ctx, cmd.UserID, tenantID)
		if err != nil || membership == nil || role.ScopeType != model.RoleScopeTenant {
			return ErrRoleNotHeld
		}
		if role.RoleKey == model.RoleRecruitingAdmin {
			count, err := s.tenants.CountActiveTenantMembersWithRole(ctx, tenantID, role.ID)
			if err != nil {
				return err
			}
			if count <= 1 {
				return policy.ErrLastSystemAdmin
			}
		}
		revoked, err := s.tenants.RevokeMembershipRole(ctx, membership.ID, role.ID)
		if err != nil {
			return err
		}
		if !revoked {
			return ErrRoleNotHeld
		}
		return s.bumpAuthorizationVersion(ctx, uint64(cmd.UserID))
	}
	if role.RoleKey == model.RoleSystemAdmin && uint64(cmd.UserID) == adminID {
		principal, err := s.authz.LoadPrincipal(ctx, adminID)
		if err != nil {
			return fmt.Errorf("无法验证管理员角色，请稍后重试: %w", err)
		}
		activeAdmins, err := s.authz.CountActiveUsersWithRole(ctx, model.RoleSystemAdmin)
		if err != nil {
			return err
		}
		if err := policy.CheckSystemAdminRevocation(uint64(cmd.UserID), adminID, role.RoleKey, principal, activeAdmins); err != nil {
			s.auditAdminAction(ctx, adminID, "revoke_role", uint64(cmd.UserID), "denied", err.Error())
			return err
		}
	}
	revoked, err := s.authz.RevokeRoleWithLastAdminGuard(ctx, uint64(cmd.UserID), role.ID, role.RoleKey, &adminID)
	if err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			s.auditAdminAction(ctx, adminID, "revoke_role", uint64(cmd.UserID), "denied", policy.ErrLastSystemAdmin.Error())
			return policy.ErrLastSystemAdmin
		}
		if errors.Is(err, repository.ErrUserRoleNotFound) {
			return ErrRoleNotHeld
		}
		return err
	}
	if !revoked {
		return ErrPermissionChangeFailed
	}
	newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(cmd.UserID))
	if err != nil {
		s.restoreRole(ctx, uint64(cmd.UserID), role.ID)
		return ErrPermissionChangeFailed
	}
	if err := s.syncTokenVersion(ctx, uint64(cmd.UserID), newVersion); err != nil {
		s.restoreRole(ctx, uint64(cmd.UserID), role.ID)
		return ErrPermissionChangeFailed
	}
	s.auditAdminAction(ctx, adminID, "revoke_role", uint64(cmd.UserID), "allowed", "revoked role "+cmd.RoleKey)
	return nil
}

func (s *AdminService) AssignDataScope(ctx context.Context, cmd command.AssignDataScope) error {
	adminID := uint64(cmd.AdminID)
	if s.tenants != nil {
		membership, err := s.tenants.GetActiveMembership(ctx, cmd.UserID, platformmetadata.GetAuthTenantID(ctx))
		if err != nil || membership == nil {
			return ErrRoleNotHeld
		}
		if err := s.tenants.AssignMembershipDataScope(ctx, membership.ID, cmd.ScopeKey, cmd.ResourceType, cmd.ResourceID, &adminID); err != nil {
			return err
		}
		return s.bumpAuthorizationVersion(ctx, uint64(cmd.UserID))
	}
	if err := s.authz.AssignDataScope(ctx, uint64(cmd.UserID), cmd.ScopeKey, cmd.ResourceType, cmd.ResourceID, &adminID); err != nil {
		return err
	}
	newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(cmd.UserID))
	if err != nil {
		return ErrPermissionTokenSyncFailed
	}
	if err := s.syncTokenVersion(ctx, uint64(cmd.UserID), newVersion); err != nil {
		return ErrPermissionTokenSyncFailed
	}
	s.auditAdminAction(ctx, adminID, "assign_scope", uint64(cmd.UserID), "allowed", "assigned scope "+cmd.ScopeKey)
	return nil
}

func (s *AdminService) RevokeDataScope(ctx context.Context, cmd command.RevokeDataScope) error {
	if s.tenants != nil {
		userID, revoked, err := s.tenants.RevokeTenantDataScope(ctx, platformmetadata.GetAuthTenantID(ctx), int64(cmd.ScopeID))
		if err != nil {
			return err
		}
		if !revoked {
			return ErrRoleNotHeld
		}
		return s.bumpAuthorizationVersion(ctx, uint64(userID))
	}
	scopeUserID, err := s.authz.GetScopeOwnerID(ctx, cmd.ScopeID)
	if err != nil {
		return err
	}
	if err := s.authz.RevokeDataScope(ctx, cmd.ScopeID); err != nil {
		return err
	}
	if scopeUserID > 0 {
		newVersion, err := s.authz.IncrementTokenVersion(ctx, scopeUserID)
		if err != nil {
			return ErrPermissionTokenSyncFailed
		}
		if err := s.syncTokenVersion(ctx, scopeUserID, newVersion); err != nil {
			return ErrPermissionTokenSyncFailed
		}
	}
	s.auditAdminAction(ctx, uint64(cmd.AdminID), "revoke_scope", scopeUserID, "allowed", fmt.Sprintf("revoked scope id %d", cmd.ScopeID))
	return nil
}

func (s *AdminService) ListStaffUsers(ctx context.Context, req query.ListStaffUsers) (dto.StaffUsersResult, error) {
	if s.users == nil {
		return dto.StaffUsersResult{}, errors.New("user repo not configured")
	}
	if err := s.requireAdminPermission(ctx, model.PermAdminUserManage); err != nil {
		return dto.StaffUsersResult{}, err
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if s.tenants != nil {
		memberships, total, err := s.tenants.ListTenantMemberships(ctx, platformmetadata.GetAuthTenantID(ctx), int((page-1)*pageSize), int(pageSize))
		if err != nil {
			return dto.StaffUsersResult{}, err
		}
		list := make([]dto.StaffUser, 0, len(memberships))
		for _, membership := range memberships {
			user, err := s.users.GetByID(ctx, membership.UserID)
			if err != nil || user == nil || (req.Status != "" && user.Status != req.Status) {
				continue
			}
			list = append(list, dto.StaffUser{UserID: user.ID, Username: user.Username, Email: user.Email, Status: user.Status, AccountType: user.AccountType, Roles: membership.Roles, TokenVersion: user.TokenVersion, CreatedAt: user.CreatedAt.Format(time.RFC3339)})
		}
		return dto.StaffUsersResult{Total: total, List: list}, nil
	}
	users, total, err := s.users.ListStaff(ctx, page, pageSize, req.Status)
	if err != nil {
		return dto.StaffUsersResult{}, err
	}
	list := make([]dto.StaffUser, len(users))
	for i, user := range users {
		roleKeys, _ := s.authz.GetUserRoles(ctx, uint64(user.ID))
		if roleKeys == nil {
			roleKeys = []string{}
		}
		list[i] = dto.StaffUser{
			UserID:       user.ID,
			Username:     user.Username,
			Email:        user.Email,
			Status:       user.Status,
			AccountType:  user.AccountType,
			Roles:        roleKeys,
			TokenVersion: user.TokenVersion,
			CreatedAt:    user.CreatedAt.Format(time.RFC3339),
		}
	}
	return dto.StaffUsersResult{Total: total, List: list}, nil
}

func (s *AdminService) CreateStaffUser(ctx context.Context, cmd command.CreateStaffUser) (int64, error) {
	if s.users == nil {
		return 0, errors.New("user repo not configured")
	}
	if s.passwords == nil {
		return 0, errors.New("password service is required")
	}
	if err := policy.ValidatePassword(cmd.Password); err != nil {
		return 0, err
	}
	username, err := policy.NormalizeUsername(cmd.Username)
	if err != nil {
		return 0, err
	}
	existing, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrUsernameExists
	}
	hash, err := s.passwords.HashPassword(cmd.Password)
	if err != nil {
		return 0, err
	}
	user := &model.User{
		Username:     username,
		PasswordHash: hash,
		Email:        cmd.Email,
		Role:         model.LegacyRoleAdmin,
		AccountType:  model.AccountTypeStaff,
		Status:       model.UserStatusActive,
		TokenVersion: 1,
	}
	tenantID := int64(0)
	if s.tenants != nil {
		tenantID = platformmetadata.GetAuthTenantID(ctx)
		if err := s.tenants.CheckTenantQuota(ctx, tenantID, "members.max", 1); err != nil {
			if errors.Is(err, commonsquota.ErrLimitExceeded) {
				return 0, errors.New("已达到当前套餐的有效成员上限")
			}
			return 0, err
		}
	}
	if err := s.users.Create(ctx, user); err != nil {
		return 0, err
	}
	adminID := uint64(cmd.AdminID)
	if s.tenants != nil {
		membership, err := s.tenants.EnsureMembership(ctx, user.ID, tenantID, "active")
		if err != nil || membership == nil {
			return 0, ErrAccountCreateFailed
		}
		for _, roleKey := range cmd.RoleKeys {
			role, err := s.authz.GetRoleByKey(ctx, roleKey)
			if err == nil && role != nil && role.ScopeType == model.RoleScopeTenant {
				_ = s.tenants.AssignMembershipRole(ctx, membership.ID, role.ID, &adminID)
			}
		}
		if containsRoleKey(cmd.RoleKeys, model.RoleRecruiter) {
			if err := s.tenants.AssignMembershipDataScope(ctx, membership.ID, model.ScopeOwnJobs, "", 0, &adminID); err != nil {
				return 0, ErrAccountCreateFailed
			}
		}
		s.auditAdminAction(ctx, adminID, "create_staff", uint64(user.ID), "allowed", "created tenant staff user "+username)
		return user.ID, nil
	}
	for _, roleKey := range cmd.RoleKeys {
		role, err := s.authz.GetRoleByKey(ctx, roleKey)
		if err != nil || role == nil {
			continue
		}
		_ = s.authz.AssignRole(ctx, uint64(user.ID), role.ID, &adminID)
	}
	if containsRoleKey(cmd.RoleKeys, model.RoleRecruiter) {
		if err := s.authz.AssignDataScope(ctx, uint64(user.ID), model.ScopeOwnJobs, "", 0, &adminID); err != nil {
			return 0, ErrAccountCreateFailed
		}
	}
	s.auditAdminAction(ctx, adminID, "create_staff", uint64(user.ID), "allowed", "created staff user "+username)
	return user.ID, nil
}

func (s *AdminService) ListPlatformAccounts(ctx context.Context, page, pageSize int32, status string) ([]model.PlatformAccount, int64, error) {
	if err := s.requireAdminPermission(ctx, model.PermPlatformUserManage); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if status != "" && status != model.UserStatusActive && status != "disabled" {
		return nil, 0, ErrAccountCreateFailed
	}
	return s.users.ListPlatformAccounts(ctx, page, pageSize, status)
}

func (s *AdminService) CreatePlatformAccount(ctx context.Context, adminID int64, username, email, password, roleKey string) (int64, error) {
	if err := s.requireAdminPermission(ctx, model.PermPlatformUserManage); err != nil {
		return 0, err
	}
	if s.users == nil || s.passwords == nil || !isPlatformRole(roleKey) {
		return 0, ErrAccountCreateFailed
	}
	if err := policy.ValidatePassword(password); err != nil {
		return 0, err
	}
	normalized, err := policy.NormalizeUsername(username)
	if err != nil {
		return 0, err
	}
	existing, err := s.users.GetByUsername(ctx, normalized)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrUsernameExists
	}
	role, err := s.authz.GetRoleByKey(ctx, roleKey)
	if err != nil || role == nil || role.ScopeType != model.RoleScopePlatform {
		return 0, ErrRoleNotFound
	}
	hash, err := s.passwords.HashPassword(password)
	if err != nil {
		return 0, err
	}
	user := &model.User{Username: normalized, PasswordHash: hash, Email: strings.TrimSpace(email), Role: model.LegacyRoleAdmin, AccountType: model.AccountTypeStaff, Status: model.UserStatusActive, TokenVersion: 1}
	actor := uint64(adminID)
	if err := s.users.CreatePlatformAccount(ctx, user, role.ID, &actor); err != nil {
		return 0, ErrAccountCreateFailed
	}
	return user.ID, nil
}

func (s *AdminService) UpdatePlatformAccount(ctx context.Context, adminID, userID int64, roleKey, status, reason string) (*model.PlatformAccount, error) {
	if err := s.requireAdminPermission(ctx, model.PermPlatformUserManage); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if userID <= 0 || !isPlatformRole(roleKey) || (status != model.UserStatusActive && status != "disabled") || reason == "" || len(reason) > 500 {
		return nil, ErrAccountCreateFailed
	}
	role, err := s.authz.GetRoleByKey(ctx, roleKey)
	if err != nil || role == nil || role.ScopeType != model.RoleScopePlatform {
		return nil, ErrRoleNotFound
	}
	return s.users.UpdatePlatformAccount(ctx, userID, role.ID, status, reason, adminID)
}

func isPlatformRole(roleKey string) bool {
	return roleKey == model.RolePlatformAdmin || roleKey == model.RolePlatformOperator || roleKey == model.RolePlatformAuditor
}

func (s *AdminService) QueryAuthAuditLogs(ctx context.Context, req query.QueryAuthAuditLogs) (dto.AuditLogsResult, error) {
	if s.audit == nil {
		return dto.AuditLogsResult{}, errors.New("audit repo not configured")
	}
	if err := s.requireAdminPermission(ctx, model.PermAuditSecurityRead); err != nil {
		return dto.AuditLogsResult{}, err
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	rows, total, err := s.audit.QueryAuthAuditLogs(ctx, req.ActorUserID, req.PermissionKey, req.Decision, int((page-1)*pageSize), int(pageSize))
	if err != nil {
		return dto.AuditLogsResult{}, fmt.Errorf("%w: %v", ErrAuditQueryFailed, err)
	}
	return dto.AuditLogsResult{Total: total, List: rows}, nil
}

func (s *AdminService) requireAdminPermission(ctx context.Context, permissionKey string) error {
	if s.authorizer == nil {
		return errors.New("authenticated user not found in context - gRPC metadata x-authenticated-user-id is required for admin operations")
	}
	return s.authorizer.AuthorizePermission(ctx, permissionKey)
}

func (s *AdminService) roleByKey(ctx context.Context, roleKey string) (*model.Role, error) {
	role, err := s.authz.GetRoleByKey(ctx, roleKey)
	if err != nil || role == nil {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

func (s *AdminService) syncTokenVersion(ctx context.Context, userID uint64, version int32) error {
	if s.tokenCache == nil {
		return nil
	}
	if err := s.tokenCache.SetTokenVersion(ctx, userID, version, port.AccessTokenTTL); err != nil {
		if delErr := s.tokenCache.DeleteTokenVersion(ctx, userID); delErr != nil {
			return fmt.Errorf("token_version sync failed: SET error=%w, DEL error=%w", err, delErr)
		}
	}
	return nil
}

func (s *AdminService) bumpAuthorizationVersion(ctx context.Context, userID uint64) error {
	newVersion, err := s.authz.IncrementTokenVersion(ctx, userID)
	if err != nil {
		return ErrPermissionTokenSyncFailed
	}
	if err := s.syncTokenVersion(ctx, userID, newVersion); err != nil {
		return ErrPermissionTokenSyncFailed
	}
	return nil
}

func (s *AdminService) restoreRole(ctx context.Context, userID, roleID uint64) {
	_ = s.authz.RestoreRole(ctx, userID, roleID)
}

func (s *AdminService) auditAdminAction(ctx context.Context, actorID uint64, action string, targetUserID uint64, decision, detail string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.RecordAuthDecision(ctx, model.AuthAuditDecision{
		ActorUserID:   actorID,
		PermissionKey: "admin." + action,
		ResourceType:  "user",
		ResourceID:    targetUserID,
		Decision:      decision,
		Reason:        detail,
	})
}

func containsRoleKey(roleKeys []string, target string) bool {
	for _, roleKey := range roleKeys {
		if roleKey == target {
			return true
		}
	}
	return false
}
