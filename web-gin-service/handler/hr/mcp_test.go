package hr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

// mockMCPClient implements pb.MCPServiceClient for testing.
type mockMCPClient struct {
	listFn         func(context.Context, *pb.ListMCPServersRequest, ...grpc.CallOption) (*pb.ListMCPServersResponse, error)
	createFn       func(context.Context, *pb.CreateMCPServerRequest, ...grpc.CallOption) (*pb.MCPServerResponse, error)
	updateFn       func(context.Context, *pb.UpdateMCPServerRequest, ...grpc.CallOption) (*pb.MCPServerResponse, error)
	deleteFn       func(context.Context, *pb.DeleteMCPServerRequest, ...grpc.CallOption) (*pb.CommonResponse, error)
	listPolicyFn   func(context.Context, *pb.ListMCPToolPoliciesRequest, ...grpc.CallOption) (*pb.ListMCPToolPoliciesResponse, error)
	createPolicyFn func(context.Context, *pb.CreateMCPToolPolicyRequest, ...grpc.CallOption) (*pb.MCPToolPolicyResponse, error)
	updatePolicyFn func(context.Context, *pb.UpdateMCPToolPolicyRequest, ...grpc.CallOption) (*pb.MCPToolPolicyResponse, error)
	deletePolicyFn func(context.Context, *pb.DeleteMCPToolPolicyRequest, ...grpc.CallOption) (*pb.CommonResponse, error)
	testFn         func(context.Context, *pb.TestMCPConnectionRequest, ...grpc.CallOption) (*pb.TestMCPConnectionResponse, error)
	listToolFn     func(context.Context, *pb.ListMCPToolsRequest, ...grpc.CallOption) (*pb.ListMCPToolsResponse, error)
	callToolFn     func(context.Context, *pb.CallMCPToolRequest, ...grpc.CallOption) (*pb.CallMCPToolResponse, error)
}

func (m *mockMCPClient) ListMCPServers(ctx context.Context, req *pb.ListMCPServersRequest, opts ...grpc.CallOption) (*pb.ListMCPServersResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req, opts...)
	}
	return &pb.ListMCPServersResponse{Code: 0, Msg: "ok", List: []*pb.MCPServerInfo{}}, nil
}

func (m *mockMCPClient) CreateMCPServer(ctx context.Context, req *pb.CreateMCPServerRequest, opts ...grpc.CallOption) (*pb.MCPServerResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req, opts...)
	}
	return &pb.MCPServerResponse{Code: 0, Msg: "ok", Server: &pb.MCPServerInfo{Id: 1, Name: req.Name}}, nil
}

func (m *mockMCPClient) UpdateMCPServer(ctx context.Context, req *pb.UpdateMCPServerRequest, opts ...grpc.CallOption) (*pb.MCPServerResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req, opts...)
	}
	return &pb.MCPServerResponse{Code: 0, Msg: "ok", Server: &pb.MCPServerInfo{Id: req.Id, Name: req.Name}}, nil
}

func (m *mockMCPClient) DeleteMCPServer(ctx context.Context, req *pb.DeleteMCPServerRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, req, opts...)
	}
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockMCPClient) ListMCPToolPolicies(ctx context.Context, req *pb.ListMCPToolPoliciesRequest, opts ...grpc.CallOption) (*pb.ListMCPToolPoliciesResponse, error) {
	if m.listPolicyFn != nil {
		return m.listPolicyFn(ctx, req, opts...)
	}
	return &pb.ListMCPToolPoliciesResponse{Code: 0, Msg: "ok", List: []*pb.MCPToolPolicyInfo{}}, nil
}

func (m *mockMCPClient) CreateMCPToolPolicy(ctx context.Context, req *pb.CreateMCPToolPolicyRequest, opts ...grpc.CallOption) (*pb.MCPToolPolicyResponse, error) {
	if m.createPolicyFn != nil {
		return m.createPolicyFn(ctx, req, opts...)
	}
	return &pb.MCPToolPolicyResponse{Code: 0, Msg: "ok", Policy: &pb.MCPToolPolicyInfo{Id: 1, ServerId: req.ServerId, ToolName: req.ToolName}}, nil
}

func (m *mockMCPClient) UpdateMCPToolPolicy(ctx context.Context, req *pb.UpdateMCPToolPolicyRequest, opts ...grpc.CallOption) (*pb.MCPToolPolicyResponse, error) {
	if m.updatePolicyFn != nil {
		return m.updatePolicyFn(ctx, req, opts...)
	}
	return &pb.MCPToolPolicyResponse{Code: 0, Msg: "ok", Policy: &pb.MCPToolPolicyInfo{Id: req.Id, ServerId: req.ServerId, ToolName: req.ToolName}}, nil
}

func (m *mockMCPClient) DeleteMCPToolPolicy(ctx context.Context, req *pb.DeleteMCPToolPolicyRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	if m.deletePolicyFn != nil {
		return m.deletePolicyFn(ctx, req, opts...)
	}
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockMCPClient) TestMCPConnection(ctx context.Context, req *pb.TestMCPConnectionRequest, opts ...grpc.CallOption) (*pb.TestMCPConnectionResponse, error) {
	if m.testFn != nil {
		return m.testFn(ctx, req, opts...)
	}
	return &pb.TestMCPConnectionResponse{Code: 0, Msg: "ok", Success: true}, nil
}

func (m *mockMCPClient) ListMCPTools(ctx context.Context, req *pb.ListMCPToolsRequest, opts ...grpc.CallOption) (*pb.ListMCPToolsResponse, error) {
	if m.listToolFn != nil {
		return m.listToolFn(ctx, req, opts...)
	}
	return &pb.ListMCPToolsResponse{Code: 0, Msg: "ok", List: []*pb.MCPToolInfo{}}, nil
}

func (m *mockMCPClient) CallMCPTool(ctx context.Context, req *pb.CallMCPToolRequest, opts ...grpc.CallOption) (*pb.CallMCPToolResponse, error) {
	if m.callToolFn != nil {
		return m.callToolFn(ctx, req, opts...)
	}
	return &pb.CallMCPToolResponse{Code: 0, Msg: "ok", ResultContent: "result"}, nil
}

func TestMCPHandler_ListMCPServers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{
		listFn: func(_ context.Context, req *pb.ListMCPServersRequest, _ ...grpc.CallOption) (*pb.ListMCPServersResponse, error) {
			if req.Page != 1 || req.PageSize != 20 {
				t.Fatalf("expected page=1, page_size=20, got page=%d, page_size=%d", req.Page, req.PageSize)
			}
			return &pb.ListMCPServersResponse{
				Code:  0,
				Msg:   "ok",
				Total: 1,
				List: []*pb.MCPServerInfo{
					{Id: 1, Name: "test-server", Transport: "stdio"},
				},
			}, nil
		},
	}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.GET("/hr/mcp/servers", handler.ListMCPServers)

	req := httptest.NewRequest(http.MethodGet, "/hr/mcp/servers?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "test-server") {
		t.Fatalf("expected response to contain 'test-server', got %s", body)
	}
}

func TestMCPHandler_CreateMCPServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{
		createFn: func(_ context.Context, req *pb.CreateMCPServerRequest, _ ...grpc.CallOption) (*pb.MCPServerResponse, error) {
			if req.Name != "new-server" {
				t.Fatalf("expected name 'new-server', got %q", req.Name)
			}
			return &pb.MCPServerResponse{
				Code: 0,
				Msg:  "ok",
				Server: &pb.MCPServerInfo{
					Id: 2, Name: "new-server", Transport: "sse",
				},
			}, nil
		},
	}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.POST("/hr/mcp/servers", handler.CreateMCPServer)

	body := `{"name":"new-server","transport":"sse","command_or_url":"http://localhost:8080"}`
	req := httptest.NewRequest(http.MethodPost, "/hr/mcp/servers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestMCPHandler_DeleteMCPServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.DELETE("/hr/mcp/servers/:id", handler.DeleteMCPServer)

	req := httptest.NewRequest(http.MethodDelete, "/hr/mcp/servers/5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestMCPHandler_DeleteMCPServer_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.DELETE("/hr/mcp/servers/:id", handler.DeleteMCPServer)

	req := httptest.NewRequest(http.MethodDelete, "/hr/mcp/servers/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// BadRequest returns HTTP 200 with an error code in the body
	body := w.Body.String()
	if !strings.Contains(body, "invalid id") {
		t.Fatalf("expected error message containing 'invalid id', got %s", body)
	}
}

func TestMCPHandler_ListMCPTools(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{
		listToolFn: func(_ context.Context, req *pb.ListMCPToolsRequest, _ ...grpc.CallOption) (*pb.ListMCPToolsResponse, error) {
			if req.ServerId != 3 {
				t.Fatalf("expected ServerId=3, got %d", req.ServerId)
			}
			return &pb.ListMCPToolsResponse{
				Code: 0,
				Msg:  "ok",
				List: []*pb.MCPToolInfo{
					{Name: "tool1", Description: "first tool"},
				},
			}, nil
		},
	}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.GET("/hr/mcp/servers/:id/tools", handler.ListMCPTools)

	req := httptest.NewRequest(http.MethodGet, "/hr/mcp/servers/3/tools", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestMCPHandler_TestConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{
		testFn: func(_ context.Context, req *pb.TestMCPConnectionRequest, _ ...grpc.CallOption) (*pb.TestMCPConnectionResponse, error) {
			if req.ServerId != 1 {
				t.Fatalf("expected ServerId=1, got %d", req.ServerId)
			}
			return &pb.TestMCPConnectionResponse{Code: 0, Msg: "ok", Success: true, Detail: "connected"}, nil
		},
	}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.GET("/hr/mcp/servers/:id/test", handler.TestMCPConnection)

	req := httptest.NewRequest(http.MethodGet, "/hr/mcp/servers/1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestMCPHandler_CallMCPTool_InjectServerID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{
		callToolFn: func(_ context.Context, req *pb.CallMCPToolRequest, _ ...grpc.CallOption) (*pb.CallMCPToolResponse, error) {
			if req.ServerId != 1 {
				t.Fatalf("expected ServerId=1 (from path), got %d", req.ServerId)
			}
			if req.ToolName != "my_tool" {
				t.Fatalf("expected ToolName='my_tool', got %s", req.ToolName)
			}
			if req.ArgsJson != `{"key":"val"}` {
				t.Fatalf("expected ArgsJson='{\"key\":\"val\"}', got %s", req.ArgsJson)
			}
			return &pb.CallMCPToolResponse{
				Code: 0, Msg: "ok", ResultContent: "tool output", DurationMs: 100,
			}, nil
		},
	}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.POST("/hr/mcp/servers/:id/call", handler.CallMCPTool)

	body := `{"tool_name":"my_tool","args_json":"{\"key\":\"val\"}"}`
	req := httptest.NewRequest(http.MethodPost, "/hr/mcp/servers/1/call", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestMCPHandler_CallMCPTool_BodyServerIDIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{
		callToolFn: func(_ context.Context, req *pb.CallMCPToolRequest, _ ...grpc.CallOption) (*pb.CallMCPToolResponse, error) {
			if req.ServerId != 1 {
				t.Fatalf("expected ServerId=1 (from path, not body), got %d", req.ServerId)
			}
			return &pb.CallMCPToolResponse{
				Code: 0, Msg: "ok", ResultContent: "tool output", DurationMs: 100,
			}, nil
		},
	}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.POST("/hr/mcp/servers/:id/call", handler.CallMCPTool)

	body := `{"tool_name":"my_tool","server_id":99,"args_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/hr/mcp/servers/1/call", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestMCPHandler_CallMCPTool_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewMCPHandler(&rpc.Clients{MCP: &mockMCPClient{}})
	router := gin.New()
	router.POST("/hr/mcp/servers/:id/call", handler.CallMCPTool)

	body := `{"tool_name":"my_tool","args_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/hr/mcp/servers/invalid/call", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, "invalid server_id") {
		t.Fatalf("expected error message containing 'invalid server_id', got %s", bodyStr)
	}
}

func TestMCPHandler_CallMCPTool_GRPCError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockMCPClient{
		callToolFn: func(_ context.Context, req *pb.CallMCPToolRequest, _ ...grpc.CallOption) (*pb.CallMCPToolResponse, error) {
			return nil, status.Error(codes.Internal, "gRPC error")
		},
	}
	handler := NewMCPHandler(&rpc.Clients{MCP: mock})
	router := gin.New()
	router.POST("/hr/mcp/servers/:id/call", handler.CallMCPTool)

	body := `{"tool_name":"my_tool","args_json":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/hr/mcp/servers/1/call", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// base.Internal returns HTTP 200 with an error envelope
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, "code") || !strings.Contains(bodyStr, "msg") {
		t.Fatalf("expected error envelope in response, got %s", bodyStr)
	}
}
