package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"logic-grpc-service/model"
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

// validateRole checks that req.Role is one of the supported values (1=candidate, 2=HR).
// When the invite-code flow is not active, the web frontend passes the intended role directly.
func validateRole(role int32) error {
	switch role {
	case 1, 2:
		return nil
	default:
		return errors.New("无效的账号角色")
	}
}

type AuthService struct {
	users     *repository.UserRepo
	tokens    *repository.RefreshTokenRepo
	jwtSecret string
}

func NewAuthService(users *repository.UserRepo, tokens *repository.RefreshTokenRepo, jwtSecret string) *AuthService {
	return &AuthService{users: users, tokens: tokens, jwtSecret: jwtSecret}
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if err := validatePassword(req.Password); err != nil {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	if err := validateRole(req.Role); err != nil {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	role := req.Role

	log := logger.With(zap.String("username", req.Username), zap.Int32("role", role))

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
	user := &model.User{Username: username, Password: string(hash), Role: role, Email: req.Email}
	if err := s.users.Create(ctx, user); err != nil {
		log.Error("register create user failed", zap.Error(err))
		return nil, err
	}
	log.Info("user registered", zap.Int64("user_id", user.ID))
	return &pb.RegisterResponse{Code: errs.OK, Msg: "注册成功", UserId: user.ID, Username: user.Username, Role: user.Role}, nil
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
		Code:     errs.OK,
		Msg:      "登录成功",
		Token:    plainToken,
		UserId:   user.ID,
		Role:     user.Role,
		Username: user.Username,
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

	return &pb.RefreshTokenResponse{
		Code:             errs.OK,
		Msg:              "刷新成功",
		UserId:           result.UserID,
		Username:         result.Username,
		Role:             result.Role,
		RefreshToken:     newPlainToken,
		RefreshExpiresAt: newExpiresAt.Unix(),
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
