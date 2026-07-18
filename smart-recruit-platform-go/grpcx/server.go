package grpcx

import (
	"errors"
	"strings"

	"google.golang.org/grpc"
)

type ServerConfig struct {
	ServiceName        string
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
	ServerOptions      []grpc.ServerOption
}

func (cfg ServerConfig) Validate() error {
	if strings.TrimSpace(cfg.ServiceName) == "" {
		return errors.New("grpc server service name is required")
	}
	return nil
}

func NewServer(cfg ServerConfig) (*grpc.Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	options := make([]grpc.ServerOption, 0, len(cfg.ServerOptions)+2)
	if len(cfg.UnaryInterceptors) > 0 {
		options = append(options, grpc.ChainUnaryInterceptor(cfg.UnaryInterceptors...))
	}
	if len(cfg.StreamInterceptors) > 0 {
		options = append(options, grpc.ChainStreamInterceptor(cfg.StreamInterceptors...))
	}
	options = append(options, cfg.ServerOptions...)
	return grpc.NewServer(options...), nil
}
