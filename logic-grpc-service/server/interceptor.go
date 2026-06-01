package server

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcMetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/pkg/metadata"
)

const internalTokenHeader = "x-internal-token"

// authMode reads GRPC_INTERNAL_AUTH env: "required" rejects invalid requests,
// while the default "optional" keeps plain go run main.go friendly for local
// coursework and automated smoke tests.
func authMode() string {
	mode := os.Getenv("GRPC_INTERNAL_AUTH")
	if mode == "required" {
		return "required"
	}
	return "optional"
}

// UnaryAuthInterceptor validates x-internal-token for non-health unary RPCs.
// When GRPC_INTERNAL_TOKEN is not configured (empty), auth is skipped with a warning.
func UnaryAuthInterceptor() grpc.UnaryServerInterceptor {
	internalToken := os.Getenv("GRPC_INTERNAL_TOKEN")
	if internalToken == "" {
		logger.L().Warn("GRPC_INTERNAL_TOKEN is empty — gRPC internal auth DISABLED. Set it in production.")
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}
	mode := authMode()

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isHealthCheck(info.FullMethod) {
			return handler(ctx, req)
		}
		if !checkToken(ctx, internalToken, mode, info.FullMethod) {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid internal token")
		}
		ctx = injectMetadataIntoContext(ctx)
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor validates x-internal-token for non-health stream RPCs.
// When GRPC_INTERNAL_TOKEN is not configured (empty), auth is skipped with a warning.
func StreamAuthInterceptor() grpc.StreamServerInterceptor {
	internalToken := os.Getenv("GRPC_INTERNAL_TOKEN")
	if internalToken == "" {
		logger.L().Warn("GRPC_INTERNAL_TOKEN is empty — gRPC internal auth DISABLED. Set it in production.")
		return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}
	mode := authMode()

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if isHealthCheck(info.FullMethod) {
			return handler(srv, ss)
		}
		if !checkToken(ss.Context(), internalToken, mode, info.FullMethod) {
			return status.Error(codes.Unauthenticated, "missing or invalid internal token")
		}
		ctx := injectMetadataIntoContext(ss.Context())
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

func isHealthCheck(fullMethod string) bool {
	return fullMethod == "/grpc.health.v1.Health/Check" ||
		fullMethod == "/grpc.health.v1.Health/Watch"
}

func checkToken(ctx context.Context, expectedToken, mode, fullMethod string) bool {
	md, ok := grpcMetadata.FromIncomingContext(ctx)
	if !ok {
		return authDecision(mode, fullMethod, "no metadata in request")
	}
	values := md.Get(internalTokenHeader)
	if len(values) == 0 {
		return authDecision(mode, fullMethod, "missing x-internal-token")
	}
	token := values[0]
	if token != expectedToken {
		return authDecision(mode, fullMethod, "invalid x-internal-token")
	}
	return true
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

func injectMetadataIntoContext(ctx context.Context) context.Context {
	md, ok := grpcMetadata.FromIncomingContext(ctx)
	if ok {
		if vals := md.Get("x-request-id"); len(vals) > 0 {
			ctx = context.WithValue(ctx, metadata.KeyRequestID, vals[0])
		}
		if vals := md.Get("x-client-ip"); len(vals) > 0 {
			ctx = context.WithValue(ctx, metadata.KeyClientIP, vals[0])
		}
	}
	return ctx
}

// ValidateInternalToken fails when GRPC_INTERNAL_AUTH=required and GRPC_INTERNAL_TOKEN is empty.
// This check is independent of ALLOW_INSECURE_DEV_CONFIG — gRPC internal auth must
// always be explicitly configured. For local dev without Docker, set GRPC_INTERNAL_AUTH=optional.
func ValidateInternalToken() error {
	mode := authMode()
	if mode != "required" {
		return nil
	}
	if os.Getenv("GRPC_INTERNAL_TOKEN") == "" {
		return fmt.Errorf("GRPC_INTERNAL_TOKEN is empty while GRPC_INTERNAL_AUTH=required: production requires a non-empty internal token. Set GRPC_INTERNAL_AUTH=optional only for local development without gRPC auth")
	}
	return nil
}

func authDecision(mode, fullMethod, reason string) bool {
	if mode == "required" {
		logger.L().Warn("gRPC auth rejected",
			zap.String("method", fullMethod),
			zap.String("reason", reason),
			zap.String("mode", mode),
		)
		return false
	}
	logger.L().Info("gRPC auth missing but optional",
		zap.String("method", fullMethod),
		zap.String("reason", reason),
		zap.String("mode", mode),
	)
	return true
}
