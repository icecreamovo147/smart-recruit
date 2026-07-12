package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	api AuthAPI
}

func NewAuthServer(api AuthAPI) (*AuthServer, error) {
	if api == nil {
		return nil, fmt.Errorf("identity auth api is required")
	}
	return &AuthServer{api: api}, nil
}

func (s *AuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return s.api.Register(ctx, req)
}

func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.api.Login(ctx, req)
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return s.api.RefreshToken(ctx, req)
}

func (s *AuthServer) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return s.api.RevokeRefreshToken(ctx, req)
}

func (s *AuthServer) RecordAuthDecision(ctx context.Context, req *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return s.api.RecordAuthDecision(ctx, req)
}

func (s *AuthServer) GetPrincipal(ctx context.Context, req *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return s.api.GetPrincipal(ctx, req)
}

func (s *AuthServer) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateEmail(ctx, req)
}
