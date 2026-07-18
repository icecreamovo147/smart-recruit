package grpcx

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"smart-recruit-platform-go/servicemeta"
)

type ClientConfig struct {
	Target        string
	InternalToken string
	Timeout       time.Duration
	DialOptions   []grpc.DialOption
}

func (cfg ClientConfig) Validate() error {
	if strings.TrimSpace(cfg.Target) == "" {
		return errors.New("grpc client target is required")
	}
	if cfg.Timeout < 0 {
		return errors.New("grpc client timeout cannot be negative")
	}
	return nil
}

func DialOptions(cfg ClientConfig) ([]grpc.DialOption, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if strings.TrimSpace(cfg.InternalToken) != "" {
		options = append(options, grpc.WithUnaryInterceptor(internalTokenUnaryClientInterceptor(cfg.InternalToken)))
	}
	options = append(options, cfg.DialOptions...)
	return options, nil
}

func internalTokenUnaryClientInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		next, err := servicemeta.AppendInternalToken(ctx, token)
		if err != nil {
			return err
		}
		return invoker(next, method, req, reply, cc, opts...)
	}
}
