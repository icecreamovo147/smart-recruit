package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/jwt"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

// validatePassword enforces minimum 8-character length and complexity requirements
// (uppercase, lowercase, digit, or special character).
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度至少8个字符")
	}
	if len(password) > 128 {
		return errors.New("密码长度不能超过128个字符")
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}
	if hasUpper && hasLower && hasDigit {
		return nil
	}
	if hasUpper && hasLower && hasSpecial {
		return nil
	}
	if hasUpper && hasDigit && hasSpecial {
		return nil
	}
	if hasLower && hasDigit && hasSpecial {
		return nil
	}
	return errors.New("密码必须包含大小写字母、数字或特殊字符中的至少三类")
}

// validateRegistrationRole checks that the requested role is valid for registration.
// Candidate (role=1) self-registration is always allowed from public endpoints.
// Staff registration via invite code always creates a recruiter account;
// the client-provided role is ignored — admin roles must be assigned by existing admins.
func validateRegistrationRole(role int32, hasInviteCode bool) error {
	switch role {
	case 1:
		return nil // candidate self-registration always allowed
	case 2:
		if !hasInviteCode {
			return errors.New("HR 账号注册需要有效的邀请码")
		}
		return nil
	default:
		return errors.New("无效的账号角色")
	}
}

type AuthService struct {
	users       *repository.UserRepo
	tokens      *repository.RefreshTokenRepo
	authz       *repository.AuthzRepo
	inviteCodes *repository.InviteCodeRepo
	jwtSecret   string
}

func NewAuthService(users *repository.UserRepo, tokens *repository.RefreshTokenRepo, authzRepo *repository.AuthzRepo, inviteCodes *repository.InviteCodeRepo, jwtSecret string) *AuthService {
	return &AuthService{users: users, tokens: tokens, authz: authzRepo, inviteCodes: inviteCodes, jwtSecret: jwtSecret}
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if err := validatePassword(req.Password); err != nil {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	role := req.Role
	hasInviteCode := req.InviteCode != ""

	if err := validateRegistrationRole(role, hasInviteCode); err != nil {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	log := logger.With(zap.String("username", req.Username), zap.Int32("role", role))

	// Validate invite code for staff registration
	var inviteCodeRecord *model.InviteCode
	if hasInviteCode {
		if s.inviteCodes == nil {
			return &pb.RegisterResponse{Code: errs.ErrInternal, Msg: "邀请码验证服务不可用"}, nil
		}
		var err error
		inviteCodeRecord, err = s.inviteCodes.GetByCode(ctx, req.InviteCode)
		if err != nil || inviteCodeRecord == nil {
			log.Warn("invalid or expired invite code used for registration", zap.String("invite_code", req.InviteCode))
			return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: "邀请码无效或已过期"}, nil
		}
		log = log.With(zap.Int64("invite_code_id", inviteCodeRecord.ID), zap.Int64("invited_by", inviteCodeRecord.CreatedBy))
	}

	if req.Username == "" || len(req.Username) > 50 {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: "用户名不能为空且不超过50字符"}, nil
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: "用户名不能为空"}, nil
	}
	existing, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		log.Error("register check username failed", zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: "用户名已存在"}, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Determine account type and status
	accountType := "candidate"
	status := "active"
	if role >= 2 {
		accountType = "staff"
	}

	user := &model.User{
		Username:    username,
		Password:    string(hash),
		Role:        role,
		Email:       req.Email,
		AccountType: accountType,
		Status:      status,
		TokenVersion: 1,
	}
	if err := s.users.Create(ctx, user); err != nil {
		log.Error("register create user failed", zap.Error(err))
		return nil, err
	}

	// Assign RBAC roles based on registration type
	if s.authz != nil {
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
			// Staff registration via invite code — always creates recruiter.
			// Admin roles must be explicitly granted by an existing admin.
			inviterID := uint64(inviteCodeRecord.CreatedBy)
			if err := s.assignStaffRolesAndScopes(ctx, user.ID, inviterID, log); err != nil {
				log.Error("staff role assignment failed during registration", zap.Error(err))
				return &pb.RegisterResponse{Code: errs.ErrInternal, Msg: "账号创建失败，请稍后重试"}, nil
			}
		}
	}

	log.Info("user registered", zap.Int64("user_id", user.ID))
	return &pb.RegisterResponse{
		Code:     errs.OK,
		Msg:      "注册成功",
		UserId:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

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

// loadUserRBAC fetches the user's RBAC roles and permissions from the database.
func (s *AuthService) loadUserRBAC(ctx context.Context, userID uint64) (roles, perms []string) {
	if s.authz == nil {
		return nil, nil
	}
	roles, _ = s.authz.GetUserRoles(ctx, userID)
	perms, _ = s.authz.GetUserPermissions(ctx, userID)
	return roles, perms
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log := logger.With(zap.String("username", req.Username))
	user, err := s.users.GetByUsername(ctx, req.Username)
	if err != nil {
		log.Error("login find user failed", zap.Error(err))
		return nil, err
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		log.Warn("login failed, wrong credentials")
		return &pb.LoginResponse{Code: errs.ErrUnauthorized, Msg: "用户名或密码错误"}, nil
	}

	// Load RBAC metadata
	roles, perms := s.loadUserRBAC(ctx, uint64(user.ID))

	// Generate opaque refresh token and store its hash.
	plainToken, err := jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	familyID := uuid.NewString()
	expiresAt := time.Now().Add(jwt.RefreshTokenTTL)
	if err := s.tokens.Create(ctx, user.ID, plainToken, familyID, expiresAt, "", ""); err != nil {
		log.Error("create refresh token failed", zap.Error(err))
		return nil, err
	}

	log.Info("user logged in", zap.Int64("user_id", user.ID), zap.Int32("role", user.Role))
	return &pb.LoginResponse{
		Code:         errs.OK,
		Msg:          "登录成功",
		Token:        plainToken,
		UserId:       user.ID,
		Role:         user.Role,
		Username:     user.Username,
		AccountType:  user.AccountType,
		Roles:        roles,
		Permissions:  perms,
		TokenVersion: user.TokenVersion,
	}, nil
}

// RefreshToken rotates the opaque refresh token and returns a new one.
// If the provided token is revoked (reused), the entire family is invalidated.
func (s *AuthService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	newPlainToken, err := jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	newExpiresAt := time.Now().Add(jwt.RefreshTokenTTL)

	result, err := s.tokens.Rotate(ctx, req.RefreshToken, newPlainToken, newExpiresAt, req.ClientIp, req.UserAgent)
	if err != nil {
		log := logger.L()
		if errors.Is(err, repository.ErrTokenNotFound) || errors.Is(err, repository.ErrTokenExpired) {
			log.Warn("refresh token invalid or expired", zap.Error(err))
			return &pb.RefreshTokenResponse{Code: errs.ErrUnauthorized, Msg: "令牌无效或已过期，请重新登录"}, nil
		}
		if errors.Is(err, repository.ErrTokenReuseDetected) {
			log.Warn("refresh token reuse detected, family revoked")
			return &pb.RefreshTokenResponse{Code: errs.ErrUnauthorized, Msg: "会话异常，请重新登录"}, nil
		}
		return nil, err
	}

	// Load RBAC metadata for the refreshed token
	roles, perms := s.loadUserRBAC(ctx, uint64(result.UserID))

	return &pb.RefreshTokenResponse{
		Code:             errs.OK,
		Msg:              "刷新成功",
		UserId:           result.UserID,
		Username:         result.Username,
		Role:             result.Role,
		RefreshToken:     newPlainToken,
		RefreshExpiresAt: newExpiresAt.Unix(),
		AccountType:      result.AccountType,
		Roles:            roles,
		Permissions:      perms,
		TokenVersion:     result.TokenVersion,
	}, nil
}

// RevokeRefreshToken invalidates a single refresh token.
func (s *AuthService) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	if req.RefreshToken == "" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "令牌不能为空"}, nil
	}
	if err := s.tokens.Revoke(ctx, req.RefreshToken); err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "已撤销"}, nil
}

// RecordAuthDecision persists an authorization audit event to the database.
func (s *AuthService) RecordAuthDecision(ctx context.Context, req *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	if s.authz == nil {
		return &pb.CommonResponse{Code: errs.OK, Msg: "audit skipped (no repo)"}, nil
	}
	if err := s.authz.RecordAuthDecision(ctx,
		uint64(req.ActorUserId), req.ActorRoles, req.PermissionKey,
		req.ResourceType, req.ResourceId, req.Decision, req.Reason,
		req.RequestId, req.ClientIp,
	); err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "audited"}, nil
}

// GetPrincipal loads the current server-side principal for a user.
// Unlike JWT claims which may be stale, this always reads from the database.
func (s *AuthService) GetPrincipal(ctx context.Context, req *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	if s.authz == nil {
		return &pb.GetPrincipalResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	principal, err := s.authz.LoadPrincipal(ctx, uint64(req.UserId))
	if err != nil {
		return nil, err
	}
	scopeAssignments := make([]*pb.ScopeAssignment, 0, len(principal.DataScopes))
	for _, ds := range principal.DataScopes {
		scopeAssignments = append(scopeAssignments, &pb.ScopeAssignment{
			ScopeKey:     ds.ScopeKey,
			ResourceType: ds.ResourceType,
			ResourceId:   ds.ResourceID,
		})
	}
	return &pb.GetPrincipalResponse{
		Code:         errs.OK,
		Msg:          "success",
		UserId:       principal.UserID,
		Username:     principal.Username,
		AccountType:  principal.AccountType,
		Role:         principal.LegacyRole,
		Roles:        principal.Roles,
		Permissions:  principal.Permissions,
		TokenVersion: principal.TokenVersion,
		DataScopes:   scopeAssignments,
	}, nil
}
