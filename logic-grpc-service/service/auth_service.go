package service

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
	jwthelper "logic-grpc-service/pkg/jwt"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type AuthService struct {
	users     *repository.UserRepo
	jwtSecret string
}

func NewAuthService(users *repository.UserRepo, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Role validation is handled at the web-gin gateway layer:
	//   - ALLOW_PUBLIC_HR_REGISTER=true allows direct HR registration (dev only)
	//   - Valid HR_INVITE_CODE allows HR registration
	//   - By default, all public registrations are role=1 (candidate)
	// This gRPC service trusts the role passed by the gateway.
	log := logger.With(zap.String("username", req.Username), zap.Int32("role", req.Role))
	if req.Username == "" || len(req.Password) < 6 || (req.Role != 1 && req.Role != 2) {
		return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: "用户名、密码或角色不合法"}, nil
	}
	existing, err := s.users.GetByUsername(ctx, req.Username)
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
	user := &model.User{Username: req.Username, Password: string(hash), Role: req.Role, Email: req.Email}
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
	token, err := jwthelper.Generate(s.jwtSecret, user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}
	log.Info("user logged in", zap.Int64("user_id", user.ID), zap.Int32("role", user.Role))
	return &pb.LoginResponse{Code: errs.OK, Msg: "登录成功", Token: token, UserId: user.ID, Role: user.Role, Username: user.Username}, nil
}
