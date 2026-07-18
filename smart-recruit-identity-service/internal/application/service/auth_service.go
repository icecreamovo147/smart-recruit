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
		if err := s.assignRegistrationRole(ctx, user.ID, plan, inviterID); err != nil {
			return dto.RegisterResult{}, err
		}
	}
	return dto.RegisterResult{UserID: user.ID, Username: user.Username, Role: user.Role}, nil
}

func (s *AuthService) assignRegistrationRole(ctx context.Context, userID int64, plan policy.RegistrationPlan, assignedBy *uint64) error {
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
	if err := s.tokens.Create(ctx, user.ID, plainToken, familyID, expiresAt, cmd.ClientIP, cmd.UserAgent); err != nil {
		return dto.AuthResult{}, err
	}
	roles, permissions := s.loadUserRBAC(ctx, uint64(user.ID))
	return dto.AuthResult{
		UserID:           user.ID,
		Username:         user.Username,
		Role:             user.Role,
		Email:            user.Email,
		AccountType:      user.AccountType,
		Roles:            roles,
		Permissions:      permissions,
		TokenVersion:     user.TokenVersion,
		RefreshToken:     plainToken,
		RefreshExpiresAt: expiresAt.Unix(),
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
	roles, permissions := s.loadUserRBAC(ctx, uint64(session.UserID))
	return dto.AuthResult{
		UserID:           session.UserID,
		Username:         session.Username,
		Role:             session.Role,
		AccountType:      session.AccountType,
		Roles:            roles,
		Permissions:      permissions,
		TokenVersion:     session.TokenVersion,
		RefreshToken:     newPlainToken,
		RefreshExpiresAt: expiresAt.Unix(),
	}, nil
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
	if s.authz == nil {
		return nil, ErrAuthzRepoUnavailable
	}
	return s.authz.LoadPrincipal(ctx, uint64(req.UserID))
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
