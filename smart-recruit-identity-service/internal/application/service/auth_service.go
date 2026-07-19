package service

import (
	"context"
	"errors"
	"fmt"

	"smart-recruit-identity-service/internal/application/command"
	"smart-recruit-identity-service/internal/application/dto"
	"smart-recruit-identity-service/internal/application/port"
	"smart-recruit-identity-service/internal/application/query"
	"smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/policy"
	"smart-recruit-identity-service/internal/domain/repository"
)

var (
	ErrInviteServiceUnavailable = errors.New("邀请码验证服务不可用")
	ErrInviteInvalid            = errors.New("邀请码无效或已过期")
	ErrUsernameExists           = errors.New("用户名已存在")
	ErrRoleSeedMissing          = errors.New("系统初始化未完成，请联系管理员")
	ErrAccountCreateFailed      = errors.New("账号创建失败，请稍后重试")
	ErrInvalidCredentials       = errors.New("用户名或密码错误")
	ErrRefreshTokenInvalid      = errors.New("令牌无效或已过期，请重新登录")
	ErrRefreshTokenReused       = errors.New("会话异常，请重新登录")
	ErrRefreshTokenRequired     = errors.New("令牌不能为空")
	ErrAuthzRepoUnavailable     = errors.New("authz repo not configured")
	ErrUserNotFound             = errors.New("用户不存在")
)

type AuthDeps struct {
	Users         repository.UserRepository
	Tokens        repository.RefreshTokenRepository
	Authz         repository.AuthzRepository
	Tenants       repository.TenantRepository
	Invites       repository.InviteCodeRepository
	Audit         repository.AuditRepository
	Passwords     port.PasswordService
	TokenFactory  port.TokenGenerator
	ActorVerifier port.ActorVerifier
	Clock         port.Clock
}

type AuthService struct {
	users         repository.UserRepository
	tokens        repository.RefreshTokenRepository
	authz         repository.AuthzRepository
	tenants       repository.TenantRepository
	invites       repository.InviteCodeRepository
	audit         repository.AuditRepository
	passwords     port.PasswordService
	tokenFactory  port.TokenGenerator
	actorVerifier port.ActorVerifier
	clock         port.Clock
}

func NewAuthService(deps AuthDeps) (*AuthService, error) {
	if deps.Users == nil {
		return nil, errors.New("user repository is required")
	}
	if deps.Passwords == nil {
		return nil, errors.New("password service is required")
	}
	clock := deps.Clock
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &AuthService{
		users:         deps.Users,
		tokens:        deps.Tokens,
		authz:         deps.Authz,
		tenants:       deps.Tenants,
		invites:       deps.Invites,
		audit:         deps.Audit,
		passwords:     deps.Passwords,
		tokenFactory:  deps.TokenFactory,
		actorVerifier: deps.ActorVerifier,
		clock:         clock,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, cmd command.Register) (dto.RegisterResult, error) {
	if err := policy.ValidatePassword(cmd.Password); err != nil {
		return dto.RegisterResult{}, err
	}
	plan, err := policy.RegistrationPlanFor(cmd.Role, cmd.InviteCode)
	if err != nil {
		return dto.RegisterResult{}, err
	}
	var inviterID *uint64
	var inviteTenantID int64
	if plan.RequiresInvite {
		if s.invites == nil {
			return dto.RegisterResult{}, ErrInviteServiceUnavailable
		}
		invite, err := s.invites.GetByCode(ctx, cmd.InviteCode)
		if err != nil || invite == nil || !invite.UsableAt(s.clock.Now()) {
			return dto.RegisterResult{}, ErrInviteInvalid
		}
		id := uint64(invite.CreatedBy)
		inviterID = &id
		inviteTenantID = invite.TenantID
	}
	username, err := policy.NormalizeUsername(cmd.Username)
	if err != nil {
		return dto.RegisterResult{}, err
	}
	existing, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return dto.RegisterResult{}, err
	}
	if existing != nil {
		return dto.RegisterResult{}, ErrUsernameExists
	}
	hash, err := s.passwords.HashPassword(cmd.Password)
	if err != nil {
		return dto.RegisterResult{}, err
	}
	user := &model.User{
		Username:     username,
		PasswordHash: hash,
		Role:         plan.LegacyRole,
		Email:        cmd.Email,
		AccountType:  plan.AccountType,
		Status:       plan.Status,
		TokenVersion: 1,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return dto.RegisterResult{}, err
	}
	if s.authz != nil && plan.RoleKey != "" {
		if err := s.assignRegistrationRole(ctx, user.ID, plan, inviterID, inviteTenantID); err != nil {
			return dto.RegisterResult{}, err
		}
	}
	return dto.RegisterResult{UserID: user.ID, Username: user.Username, Role: user.Role}, nil
}

func (s *AuthService) assignRegistrationRole(ctx context.Context, userID int64, plan policy.RegistrationPlan, assignedBy *uint64, tenantID int64) error {
	role, err := s.authz.GetRoleByKey(ctx, plan.RoleKey)
	if err != nil || role == nil {
		return ErrRoleSeedMissing
	}
	if err := s.authz.AssignRole(ctx, uint64(userID), role.ID, assignedBy); err != nil {
		return fmt.Errorf("%w: %v", ErrAccountCreateFailed, err)
	}
	if plan.ScopeKey != "" {
		if err := s.authz.AssignDataScope(ctx, uint64(userID), plan.ScopeKey, "", 0, assignedBy); err != nil {
			return fmt.Errorf("%w: %v", ErrAccountCreateFailed, err)
		}
	}
	if plan.AccountType == model.AccountTypeStaff && s.tenants != nil {
		if tenantID <= 0 {
			defaultTenant, tenantErr := s.tenants.GetDefault(ctx)
			if tenantErr != nil {
				return fmt.Errorf("%w: resolve default tenant: %v", ErrAccountCreateFailed, tenantErr)
			}
			if defaultTenant != nil {
				tenantID = defaultTenant.ID
			}
		}
		if tenantID <= 0 {
			return fmt.Errorf("%w: staff invite has no tenant", ErrAccountCreateFailed)
		}
		membership, err := s.tenants.EnsureMembership(ctx, userID, tenantID, "active")
		if err != nil || membership == nil {
			return fmt.Errorf("%w: create tenant membership: %v", ErrAccountCreateFailed, err)
		}
		if err := s.tenants.AssignMembershipRole(ctx, membership.ID, role.ID, assignedBy); err != nil {
			return fmt.Errorf("%w: assign tenant role: %v", ErrAccountCreateFailed, err)
		}
		if plan.ScopeKey != "" {
			if err := s.tenants.AssignMembershipDataScope(ctx, membership.ID, plan.ScopeKey, "", 0, assignedBy); err != nil {
				return fmt.Errorf("%w: assign tenant scope: %v", ErrAccountCreateFailed, err)
			}
		}
	}
	return nil
}

func (s *AuthService) Login(ctx context.Context, cmd command.Login) (dto.AuthResult, error) {
	if s.tokens == nil {
		return dto.AuthResult{}, errors.New("refresh token repository is required")
	}
	if s.tokenFactory == nil {
		return dto.AuthResult{}, errors.New("token generator is required")
	}
	user, err := s.users.GetByUsername(ctx, cmd.Username)
	if err != nil {
		return dto.AuthResult{}, err
	}
	if user == nil || !s.passwords.VerifyPassword(user.PasswordHash, cmd.Password) {
		return dto.AuthResult{}, ErrInvalidCredentials
	}
	plainToken, err := s.tokenFactory.NewRefreshToken()
	if err != nil {
		return dto.AuthResult{}, err
	}
	familyID, err := s.tokenFactory.NewFamilyID()
	if err != nil {
		return dto.AuthResult{}, err
	}
	expiresAt := s.clock.Now().Add(port.RefreshTokenTTL)
	principal, memberships, apps, err := s.resolveLoginPrincipal(ctx, user, cmd.ClientApp, cmd.RequestedTenantID)
	if err != nil {
		return dto.AuthResult{}, err
	}
	if err := s.tokens.Create(ctx, user.ID, plainToken, familyID, principal.ClientApp, principal.TenantID, principal.MembershipID, expiresAt, cmd.ClientIP, cmd.UserAgent); err != nil {
		return dto.AuthResult{}, err
	}
	return dto.AuthResult{
		UserID:           user.ID,
		Username:         user.Username,
		Role:             user.Role,
		Email:            user.Email,
		AccountType:      principal.AccountType,
		Roles:            principal.Roles,
		Permissions:      principal.Permissions,
		TokenVersion:     user.TokenVersion,
		RefreshToken:     plainToken,
		RefreshExpiresAt: expiresAt.Unix(),
		TenantID:         principal.TenantID,
		MembershipID:     principal.MembershipID,
		ClientApp:        principal.ClientApp,
		AvailableApps:    apps,
		Memberships:      memberships,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, cmd command.RefreshToken) (dto.AuthResult, error) {
	if s.tokens == nil {
		return dto.AuthResult{}, errors.New("refresh token repository is required")
	}
	if s.tokenFactory == nil {
		return dto.AuthResult{}, errors.New("token generator is required")
	}
	newPlainToken, err := s.tokenFactory.NewRefreshToken()
	if err != nil {
		return dto.AuthResult{}, err
	}
	expiresAt := s.clock.Now().Add(port.RefreshTokenTTL)
	session, err := s.tokens.Rotate(ctx, cmd.RefreshToken, newPlainToken, expiresAt, cmd.ClientIP, cmd.UserAgent)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTokenNotFound), errors.Is(err, repository.ErrTokenExpired):
			return dto.AuthResult{}, ErrRefreshTokenInvalid
		case errors.Is(err, repository.ErrTokenReuseDetected):
			return dto.AuthResult{}, ErrRefreshTokenReused
		default:
			return dto.AuthResult{}, err
		}
	}
	principal, err := s.principalForSession(ctx, session)
	if err != nil {
		return dto.AuthResult{}, err
	}
	return dto.AuthResult{
		UserID:           session.UserID,
		Username:         session.Username,
		Role:             session.Role,
		AccountType:      principal.AccountType,
		Roles:            principal.Roles,
		Permissions:      principal.Permissions,
		TokenVersion:     session.TokenVersion,
		RefreshToken:     newPlainToken,
		RefreshExpiresAt: expiresAt.Unix(),
		TenantID:         principal.TenantID,
		MembershipID:     principal.MembershipID,
		ClientApp:        principal.ClientApp,
	}, nil
}

// SwitchTenant validates the target membership against the current refresh
// session, then atomically rotates and rebinds that session to the target.
// Callers never get to assert a tenant merely by supplying an HTTP header.
func (s *AuthService) SwitchTenant(ctx context.Context, cmd command.SwitchTenant) (dto.AuthResult, error) {
	if cmd.RefreshToken == "" || cmd.TenantID <= 0 {
		return dto.AuthResult{}, ErrRefreshTokenInvalid
	}
	if s.tokens == nil || s.tenants == nil || s.tokenFactory == nil {
		return dto.AuthResult{}, errors.New("tenant session dependencies are required")
	}
	current, err := s.tokens.GetActive(ctx, cmd.RefreshToken)
	if err != nil {
		return dto.AuthResult{}, mapRefreshTokenError(err)
	}
	if current.AccountType != model.AccountTypeStaff || current.ClientApp != "staff" {
		return dto.AuthResult{}, ErrInvalidCredentials
	}
	target, err := s.tenants.LoadTenantPrincipal(ctx, current.UserID, cmd.TenantID, "staff")
	if err != nil || target == nil || target.MembershipID <= 0 {
		return dto.AuthResult{}, ErrInvalidCredentials
	}
	newPlainToken, err := s.tokenFactory.NewRefreshToken()
	if err != nil {
		return dto.AuthResult{}, err
	}
	expiresAt := s.clock.Now().Add(port.RefreshTokenTTL)
	session, err := s.tokens.RotateToTenant(ctx, cmd.RefreshToken, newPlainToken, target.TenantID, target.MembershipID, expiresAt, cmd.ClientIP, cmd.UserAgent)
	if err != nil {
		return dto.AuthResult{}, mapRefreshTokenError(err)
	}
	memberships, err := s.tenants.ListUserMemberships(ctx, session.UserID)
	if err != nil {
		return dto.AuthResult{}, err
	}
	return dto.AuthResult{
		UserID: session.UserID, Username: session.Username, Role: session.Role,
		AccountType: session.AccountType, Roles: target.Roles, Permissions: target.Permissions,
		TokenVersion: session.TokenVersion, RefreshToken: newPlainToken, RefreshExpiresAt: expiresAt.Unix(),
		TenantID: target.TenantID, MembershipID: target.MembershipID, ClientApp: "staff",
		AvailableApps: []string{"staff"}, Memberships: memberships,
	}, nil
}

func mapRefreshTokenError(err error) error {
	switch {
	case errors.Is(err, repository.ErrTokenNotFound), errors.Is(err, repository.ErrTokenExpired):
		return ErrRefreshTokenInvalid
	case errors.Is(err, repository.ErrTokenReuseDetected):
		return ErrRefreshTokenReused
	default:
		return err
	}
}

func (s *AuthService) RevokeRefreshToken(ctx context.Context, cmd command.RevokeRefreshToken) error {
	if cmd.RefreshToken == "" {
		return ErrRefreshTokenRequired
	}
	if s.tokens == nil {
		return errors.New("refresh token repository is required")
	}
	return s.tokens.Revoke(ctx, cmd.RefreshToken)
}

func (s *AuthService) RecordAuthDecision(ctx context.Context, cmd command.RecordAuthDecision) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.RecordAuthDecision(ctx, model.AuthAuditDecision{
		TenantID:      cmd.TenantID,
		MembershipID:  cmd.MembershipID,
		ActorUserID:   cmd.ActorUserID,
		ActorRoles:    cmd.ActorRoles,
		PermissionKey: cmd.PermissionKey,
		ResourceType:  cmd.ResourceType,
		ResourceID:    cmd.ResourceID,
		Decision:      cmd.Decision,
		Reason:        cmd.Reason,
		RequestID:     cmd.RequestID,
		ClientIP:      cmd.ClientIP,
	})
}

func (s *AuthService) GetPrincipal(ctx context.Context, req query.GetPrincipal) (*model.Principal, error) {
	if req.TenantID > 0 {
		if s.tenants == nil {
			return nil, errors.New("tenant repository is required")
		}
		principal, err := s.tenants.LoadTenantPrincipal(ctx, req.UserID, req.TenantID, req.ClientApp)
		if err != nil || principal == nil {
			return principal, err
		}
		if req.MembershipID > 0 && principal.MembershipID != req.MembershipID {
			return nil, errors.New("tenant membership mismatch")
		}
		memberships, listErr := s.tenants.ListUserMemberships(ctx, req.UserID)
		if listErr != nil {
			return nil, listErr
		}
		principal.Memberships = memberships
		principal.AvailableApps = []string{"staff"}
		return principal, nil
	}
	if s.authz == nil {
		return nil, ErrAuthzRepoUnavailable
	}
	if req.ClientApp == "platform" {
		return s.authz.LoadPlatformPrincipal(ctx, uint64(req.UserID))
	}
	return s.authz.LoadPrincipal(ctx, uint64(req.UserID))
}

func (s *AuthService) resolveLoginPrincipal(ctx context.Context, user *model.User, clientApp string, requestedTenantID int64) (*model.Principal, []model.TenantMembership, []string, error) {
	if clientApp == "" {
		if user.AccountType == model.AccountTypeCandidate {
			clientApp = "candidate"
		} else {
			clientApp = "staff"
		}
	}
	if user.AccountType == model.AccountTypeCandidate {
		if clientApp != "candidate" && clientApp != "user" {
			return nil, nil, nil, ErrInvalidCredentials
		}
		if s.authz == nil {
			return nil, nil, nil, ErrAuthzRepoUnavailable
		}
		principal, err := s.authz.LoadPrincipal(ctx, uint64(user.ID))
		if err != nil {
			return nil, nil, nil, err
		}
		principal.ClientApp = "candidate"
		principal.AvailableApps = []string{"candidate"}
		return principal, nil, []string{"candidate"}, nil
	}
	if clientApp == "platform" {
		if s.authz == nil {
			return nil, nil, nil, ErrAuthzRepoUnavailable
		}
		principal, err := s.authz.LoadPlatformPrincipal(ctx, uint64(user.ID))
		if err != nil {
			return nil, nil, nil, ErrInvalidCredentials
		}
		return principal, nil, []string{"platform"}, nil
	}
	if clientApp != "staff" && clientApp != "hr" && clientApp != "interviewer" {
		return nil, nil, nil, ErrInvalidCredentials
	}
	if s.tenants == nil {
		return nil, nil, nil, errors.New("tenant repository is required for staff login")
	}
	memberships, err := s.tenants.ListUserMemberships(ctx, user.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	active := make([]model.TenantMembership, 0, len(memberships))
	for _, membership := range memberships {
		if membership.IsActive() {
			active = append(active, membership)
		}
	}
	if len(active) == 0 {
		return nil, memberships, nil, ErrInvalidCredentials
	}
	tenantID := requestedTenantID
	if tenantID == 0 {
		tenantID = active[0].TenantID
	}
	principal, err := s.tenants.LoadTenantPrincipal(ctx, user.ID, tenantID, "staff")
	if err != nil || principal == nil {
		return nil, memberships, nil, ErrInvalidCredentials
	}
	principal.Memberships = memberships
	principal.AvailableApps = []string{"staff"}
	return principal, memberships, []string{"staff"}, nil
}

func (s *AuthService) principalForSession(ctx context.Context, session *model.RefreshSession) (*model.Principal, error) {
	if session.TenantID > 0 {
		if s.tenants == nil {
			return nil, errors.New("tenant repository is required")
		}
		principal, err := s.tenants.LoadTenantPrincipal(ctx, session.UserID, session.TenantID, session.ClientApp)
		if err != nil || principal == nil || (session.MembershipID > 0 && principal.MembershipID != session.MembershipID) {
			return nil, ErrRefreshTokenInvalid
		}
		return principal, nil
	}
	if s.authz == nil {
		return nil, ErrAuthzRepoUnavailable
	}
	if session.ClientApp == "platform" {
		principal, err := s.authz.LoadPlatformPrincipal(ctx, uint64(session.UserID))
		if err != nil {
			return nil, ErrRefreshTokenInvalid
		}
		return principal, nil
	}
	principal, err := s.authz.LoadPrincipal(ctx, uint64(session.UserID))
	if err != nil {
		return nil, err
	}
	principal.ClientApp = session.ClientApp
	return principal, nil
}

func (s *AuthService) UpdateEmail(ctx context.Context, cmd command.UpdateEmail) error {
	if s.actorVerifier != nil {
		if err := s.actorVerifier.VerifyActorMatch(ctx, cmd.UserID); err != nil {
			return err
		}
	}
	email, err := policy.NormalizeEmail(cmd.Email)
	if err != nil {
		return err
	}
	user, err := s.users.GetByID(ctx, cmd.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	return s.users.UpdateEmail(ctx, cmd.UserID, email)
}

func (s *AuthService) loadUserRBAC(ctx context.Context, userID uint64) ([]string, []string) {
	if s.authz == nil {
		return nil, nil
	}
	roles, _ := s.authz.GetUserRoles(ctx, userID)
	permissions, _ := s.authz.GetUserPermissions(ctx, userID)
	return roles, permissions
}
