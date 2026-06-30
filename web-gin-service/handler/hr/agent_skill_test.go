package hr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type mockAgentSkillClient struct {
	createVersionFn func(context.Context, *pb.CreateAgentSkillVersionRequest, ...grpc.CallOption) (*pb.AgentSkillVersionResponse, error)
	updateFn        func(context.Context, *pb.UpdateAgentSkillRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error)
}

func (m *mockAgentSkillClient) ListAgentSkills(context.Context, *pb.ListAgentSkillsRequest, ...grpc.CallOption) (*pb.ListAgentSkillsResponse, error) {
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) GetAgentSkill(context.Context, *pb.GetAgentSkillRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
	return &pb.AgentSkillResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) CreateAgentSkill(context.Context, *pb.CreateAgentSkillRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
	return &pb.AgentSkillResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest, opts ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req, opts...)
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "ok", Skill: &pb.AgentSkillInfo{Id: req.Id}}, nil
}

func (m *mockAgentSkillClient) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest, opts ...grpc.CallOption) (*pb.AgentSkillVersionResponse, error) {
	if m.createVersionFn != nil {
		return m.createVersionFn(ctx, req, opts...)
	}
	return &pb.AgentSkillVersionResponse{Code: 0, Msg: "ok", Version: &pb.AgentSkillVersionInfo{Id: 1, SkillId: req.SkillId, Version: req.Version}}, nil
}

func (m *mockAgentSkillClient) ListAgentSkillVersions(context.Context, *pb.ListAgentSkillVersionsRequest, ...grpc.CallOption) (*pb.ListAgentSkillVersionsResponse, error) {
	return &pb.ListAgentSkillVersionsResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) ActivateAgentSkillVersion(context.Context, *pb.ActivateAgentSkillVersionRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
	return &pb.AgentSkillResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) UpdateAgentSkillStatus(context.Context, *pb.UpdateAgentSkillStatusRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
	return &pb.AgentSkillResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) PreviewAgentSkill(context.Context, *pb.PreviewAgentSkillRequest, ...grpc.CallOption) (*pb.PreviewAgentSkillResponse, error) {
	return &pb.PreviewAgentSkillResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) ListAvailableAgentSkills(context.Context, *pb.ListAvailableAgentSkillsRequest, ...grpc.CallOption) (*pb.ListAgentSkillsResponse, error) {
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "ok"}, nil
}

func TestAgentSkillHandlerCreateVersionConvertsNodesToFlowJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var captured *pb.CreateAgentSkillVersionRequest
	handler := NewAgentSkillHandler(&rpc.Clients{AgentSkill: &mockAgentSkillClient{
		createVersionFn: func(_ context.Context, req *pb.CreateAgentSkillVersionRequest, _ ...grpc.CallOption) (*pb.AgentSkillVersionResponse, error) {
			captured = req
			return &pb.AgentSkillVersionResponse{
				Code:    0,
				Msg:     "ok",
				Version: &pb.AgentSkillVersionInfo{Id: 2, SkillId: req.SkillId, Version: req.Version},
			}, nil
		},
	}})
	router := gin.New()
	router.POST("/hr/agent-skills/:id/versions", handler.CreateVersion)

	body := `{"version":"1.0.0","nodes":[{"id":"trigger","type":"trigger","title":"Trigger","content":"Use this"},{"id":"instruction","type":"instruction","title":"Instruction","content":"Do it","order":5}],"activate":true}`
	req := httptest.NewRequest(http.MethodPost, "/hr/agent-skills/9/versions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	if captured == nil {
		t.Fatal("CreateAgentSkillVersion was not called")
	}
	if captured.SkillId != 9 || captured.Version != "1.0.0" || !captured.Activate {
		t.Fatalf("unexpected request: %+v", captured)
	}
	var flow struct {
		Nodes []agentSkillNodeRequest `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(captured.FlowJson), &flow); err != nil {
		t.Fatalf("flow_json is not valid JSON: %v", err)
	}
	if len(flow.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(flow.Nodes))
	}
	if flow.Nodes[0].Order != 1 || flow.Nodes[1].Order != 5 {
		t.Fatalf("unexpected node order conversion: %+v", flow.Nodes)
	}
}

func TestAgentSkillHandlerUpdateSetsEmptyStringFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var captured *pb.UpdateAgentSkillRequest
	handler := NewAgentSkillHandler(&rpc.Clients{AgentSkill: &mockAgentSkillClient{
		updateFn: func(_ context.Context, req *pb.UpdateAgentSkillRequest, _ ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
			captured = req
			return &pb.AgentSkillResponse{Code: 0, Msg: "ok", Skill: &pb.AgentSkillInfo{Id: req.Id}}, nil
		},
	}})
	router := gin.New()
	router.PUT("/hr/agent-skills/:id", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/hr/agent-skills/7", strings.NewReader(`{"display_name":"","description":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	if captured == nil {
		t.Fatal("UpdateAgentSkill was not called")
	}
	if !captured.DisplayNameSet || !captured.DescriptionSet {
		t.Fatalf("expected explicit set flags, got %+v", captured)
	}
	if captured.DisplayName != "" || captured.Description != "" {
		t.Fatalf("expected empty string values, got display=%q description=%q", captured.DisplayName, captured.Description)
	}
}
