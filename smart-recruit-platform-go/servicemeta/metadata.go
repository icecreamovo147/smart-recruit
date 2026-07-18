package servicemeta

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	InternalTokenHeader = "x-internal-token"
	RequestIDHeader     = "x-request-id"
	TraceParentHeader   = "traceparent"
)

type Service struct {
	Name     string
	Env      string
	Version  string
	Instance string
}

func (svc Service) Validate() error {
	if strings.TrimSpace(svc.Name) == "" {
		return errors.New("service name is required")
	}
	if strings.TrimSpace(svc.Env) == "" {
		return errors.New("service env is required")
	}
	if strings.TrimSpace(svc.Version) == "" {
		return errors.New("service version is required")
	}
	return nil
}

func AppendInternalToken(ctx context.Context, token string) (context.Context, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("internal token is required")
	}
	return metadata.AppendToOutgoingContext(ctx, InternalTokenHeader, strings.TrimSpace(token)), nil
}

func InternalTokenFromIncoming(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(InternalTokenHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func RequestIDFromIncoming(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(RequestIDHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
