package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// AuthAPI is the Identity-owned gRPC auth contract implemented by the current AuthService.
type AuthAPI interface {
	Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error)
	Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error)
	RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error)
	RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error)
	GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error)
	UpdateEmail(context.Context, *pb.UpdateEmailRequest) (*pb.CommonResponse, error)
	RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error)
}
