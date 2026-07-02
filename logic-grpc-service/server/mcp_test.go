package server

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
	"logic-grpc-service/service"
)

func TestServerForwardsListMCPToolLogs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.MCPServer{}, &model.MCPToolLog{}, &model.MCPToolPolicy{}); err != nil {
		t.Fatalf("auto-migrate MCP models: %v", err)
	}

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterMCPServiceServer(grpcServer, New(&service.Services{
		MCP: service.NewMCPService(repository.NewMCPRepo(db), config.Config{}),
	}))
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	resp, err := pb.NewMCPServiceClient(conn).ListMCPToolLogs(ctx, &pb.ListMCPToolLogsRequest{
		ServerId: 1,
		Page:     1,
		PageSize: 20,
	})
	if status.Code(err) == codes.Unimplemented {
		t.Fatalf("ListMCPToolLogs returned Unimplemented; server is not forwarding the RPC")
	}
	if err != nil {
		t.Fatalf("ListMCPToolLogs failed: %v", err)
	}
	if resp.GetCode() != 0 || resp.GetTotal() != 0 {
		t.Fatalf("ListMCPToolLogs response = code %d total %d, want code 0 total 0", resp.GetCode(), resp.GetTotal())
	}
}
