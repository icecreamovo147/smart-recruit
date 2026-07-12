package runtime

import (
	"context"

	platformconfig "smart-recruit-platform-go/config"
	"smart-recruit-platform-go/observability"
	"smart-recruit-platform-go/servicemeta"
	pb "smart-recruit-proto/recruitment/pb"
)

const ServiceName = "smart-recruit-gateway"

func Metadata(version string) servicemeta.Service {
	if version == "" {
		version = "dev"
	}
	return servicemeta.Service{Name: ServiceName, Env: "local", Version: version}
}

func ProtoServiceNames() []string {
	return []string{
		pb.AuthService_ServiceDesc.ServiceName,
		pb.JobService_ServiceDesc.ServiceName,
		pb.CandidateService_ServiceDesc.ServiceName,
		pb.ApplicationService_ServiceDesc.ServiceName,
	}
}

func PlatformBootstrap() (platformconfig.Bootstrap, error) {
	return platformconfig.LoadWithLookup(func(key string) string {
		switch key {
		case "SERVICE_NAME":
			return ServiceName
		case "SERVICE_ENV":
			return "local"
		case "SERVICE_VERSION":
			return "dev"
		case "GRPC_ADDR":
			return "127.0.0.1:50051"
		default:
			return ""
		}
	})
}

func NewLogger() error {
	logger, err := observability.NewLogger(observability.LoggerConfig{ServiceName: ServiceName, Env: "local"})
	if err != nil {
		return err
	}
	return logger.Sync()
}

func NewTraceRuntime(ctx context.Context) (*observability.TraceRuntime, error) {
	return observability.NewTraceRuntime(ctx, observability.TraceConfig{ServiceName: ServiceName, ServiceVersion: "dev", Env: "local"})
}
