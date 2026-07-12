package logger

import (
	"context"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"logic-grpc-service/pkg/metadata"
	"logic-grpc-service/pkg/observability"
)

// UnaryServerInterceptor logs every unary gRPC call and recovers panics.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()

		if metadata.GetTraceID(ctx) == "" {
			ctx = metadata.WithTraceContext(ctx, "", "")
		}
		fields := extractLogFields(ctx)
		if len(fields) > 0 {
			ctx = WithRequestLogger(ctx, fields...)
		}

		defer func() {
			if r := recover(); r != nil {
				elapsed := time.Since(start)
				observability.DefaultMetrics.RecordRPCPanic(info.FullMethod)
				observability.DefaultMetrics.ObserveRPC(info.FullMethod, codes.Internal.String(), elapsed)
				l := GetRequestLogger(ctx)
				l.Error("panic recovered",
					append(fields,
						zap.String("method", info.FullMethod),
						zap.Any("panic", r),
						zap.String("stack", string(debug.Stack())),
					)...,
				)
				resp = nil
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		resp, err = handler(ctx, req)

		elapsed := time.Since(start)
		observability.DefaultMetrics.ObserveRPC(info.FullMethod, status.Code(err).String(), elapsed)
		l := GetRequestLogger(ctx)
		logFields := append(fields,
			zap.String("method", info.FullMethod),
			zap.String("grpc_code", status.Code(err).String()),
			zap.Duration("elapsed", elapsed),
		)
		if err != nil {
			logFields = append(logFields, zap.Error(err))
			l.Error("grpc", logFields...)
		} else {
			l.Info("grpc", logFields...)
		}
		return resp, err
	}
}

// StreamServerInterceptor logs streaming gRPC call start/end and recovers panics.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()

		if metadata.GetTraceID(ctx) == "" {
			ctx = metadata.WithTraceContext(ctx, "", "")
			ss = &wrappedServerStream{ServerStream: ss, ctx: ctx}
		}
		fields := extractLogFields(ctx)
		if len(fields) > 0 {
			ctx = WithRequestLogger(ctx, fields...)
			ss = &wrappedServerStream{ServerStream: ss, ctx: ctx}
		}

		defer func() {
			if r := recover(); r != nil {
				elapsed := time.Since(start)
				observability.DefaultMetrics.RecordRPCPanic(info.FullMethod)
				observability.DefaultMetrics.ObserveRPC(info.FullMethod, codes.Internal.String(), elapsed)
				l := GetRequestLogger(ctx)
				l.Error("panic recovered",
					append(fields,
						zap.String("method", info.FullMethod),
						zap.Any("panic", r),
						zap.String("stack", string(debug.Stack())),
					)...,
				)
				panic(r)
			}
		}()

		err := handler(srv, ss)

		elapsed := time.Since(start)
		observability.DefaultMetrics.ObserveRPC(info.FullMethod, status.Code(err).String(), elapsed)
		l := GetRequestLogger(ctx)
		logFields := append(fields,
			zap.String("method", info.FullMethod),
			zap.String("grpc_code", status.Code(err).String()),
			zap.Duration("elapsed", elapsed),
		)
		if err != nil {
			logFields = append(logFields, zap.Error(err))
			l.Error("grpc stream", logFields...)
		} else {
			l.Info("grpc stream", logFields...)
		}
		return err
	}
}

func extractLogFields(ctx context.Context) []zap.Field {
	var fields []zap.Field
	if rid := metadata.GetRequestID(ctx); rid != "" {
		fields = append(fields, zap.String("request_id", rid))
	}
	if ip := metadata.GetClientIP(ctx); ip != "" {
		fields = append(fields, zap.String("client_ip", ip))
	}
	if uid := metadata.GetAuthUserID(ctx); uid > 0 {
		fields = append(fields, zap.Int64("user_id", uid))
	}
	if acct := metadata.GetAuthAccountType(ctx); acct != "" {
		fields = append(fields, zap.String("account_type", acct))
	}
	if traceID := metadata.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if spanID := metadata.GetSpanID(ctx); spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}
	return fields
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context { return w.ctx }
