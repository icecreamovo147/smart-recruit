package server

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// TransportSecurityOption returns a gRPC server TLS option when both cert and
// key files are configured. Empty values preserve local insecure transport.
func TransportSecurityOption(certFile, keyFile string) (grpc.ServerOption, bool, error) {
	certFile = strings.TrimSpace(certFile)
	keyFile = strings.TrimSpace(keyFile)
	if certFile == "" && keyFile == "" {
		return nil, false, nil
	}
	if certFile == "" || keyFile == "" {
		return nil, false, fmt.Errorf("both GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE are required to enable gRPC TLS")
	}
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		return nil, false, fmt.Errorf("load gRPC TLS credentials: %w", err)
	}
	return grpc.Creds(creds), true, nil
}
