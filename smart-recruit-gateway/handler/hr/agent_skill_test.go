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

	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type mockAgentSkillClient struct {
	createFn        func(context.Context, *pb.CreateAgentSkillRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error)
	createVersionFn func(context.Context, *pb.CreateAgentSkillVersionRequest, ...grpc.CallOption) (*pb.AgentSkillVersionResponse, error)
	updateFn        func(context.Context, *pb.UpdateAgentSkillRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error)
	previewFn       func(context.Context, *pb.PreviewAgentSkillRequest, ...grpc.CallOption) (*pb.PreviewAgentSkillResponse, error)
}

func (m *mockAgentSkillClient) ListAgentSkills(context.Context, *pb.ListAgentSkillsRequest, ...grpc.CallOption) (*pb.ListAgentSkillsResponse, error) {
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) GetAgentSkill(context.Context, *pb.GetAgentSkillRequest, ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
	return &pb.AgentSkillResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) CreateAgentSkill(ctx context.Context, req *pb.CreateAgentSkillRequest, opts ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req, opts...)
	}
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

func (m *mockAgentSkillClient) PreviewAgentSkill(ctx context.Context, req *pb.PreviewAgentSkillRequest, opts ...grpc.CallOption) (*pb.PreviewAgentSkillResponse, error) {
	if m.previewFn != nil {
		return m.previewFn(ctx, req, opts...)
	}
	return &pb.PreviewAgentSkillResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) ListAvailableAgentSkills(context.Context, *pb.ListAvailableAgentSkillsRequest, ...grpc.CallOption) (*pb.ListAgentSkillsResponse, error) {
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "ok"}, nil
}

func (m *mockAgentSkillClient) DebugSemanticRetrieval(context.Context, *pb.DebugSemanticRetrievalRequest, ...grpc.CallOption) (*pb.DebugSemanticRetrievalResponse, error) {
	return &pb.DebugSemanticRetrievalResponse{Code: 0, Msg: "ok"}, nil
}

func packageJSON(risk, role, outputMode string) string {
	return `{
		"manifest":{
			"schema_version":2,
			"skill_name":"candidate_screening",
			"display_name":"Candidate Screening",
			"description":"Screen candidates",
			"agent_type":"hr_recruiting_agent",
			"category":"screening",
			"scenario":"candidate_screening",
			"priority":30,
			"risk":"` + risk + `",
			"composition":{"role":"` + role + `"},
			"output_contract":{"mode":"` + outputMode + `","schema_id":"","schema_json":""},
			"required_capabilities":[" builtin:evaluate_candidate_match "],
			"trigger_keywords":["screen"],
			"semantic_tags":["resume"],
			"evaluation_criteria":["cite evidence"]
		},
		"core_markdown":"Always cite evidence.",
		"sections":[{
			"section_key":"scoring_rules",
			"title":"Scoring rules",
			"content_markdown":"Use the published rubric.",
			"trigger_terms":["score"],
			"semantic_tags":["rubric"],
			"planner_intents":["evaluate"],
			"priority":10,
			"ordinal":1
		}],
		"authoring_json":"{\"editor\":\"package-v2\"}"
	}`
}

func TestAgentSkillHandlerCreateMapsTypedPackage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var captured *pb.CreateAgentSkillRequest
	handler := NewAgentSkillHandler(&rpc.Clients{AgentSkill: &mockAgentSkillClient{
		createFn: func(_ context.Context, req *pb.CreateAgentSkillRequest, _ ...grpc.CallOption) (*pb.AgentSkillResponse, error) {
			captured = req
			return &pb.AgentSkillResponse{Code: 0, Msg: "ok", Skill: &pb.AgentSkillInfo{
				Id: 3, Name: req.Package.Manifest.SkillName, DisplayName: req.Package.Manifest.DisplayName,
			}}, nil
		},
	}})
	router := gin.New()
	router.POST("/hr/agent-skills", handler.Create)

	body := `{"version":"2.0.0","change_note":"v2","activate":true,"package":` + packageJSON("high", "primary", "advisory") + `}`
	req := httptest.NewRequest(http.MethodPost, "/hr/agent-skills", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK || captured == nil {
		t.Fatalf("status=%d captured=%#v body=%s", w.Code, captured, w.Body.String())
	}
	if captured.GetVersion() != "2.0.0" || captured.GetPackage() == nil {
		t.Fatalf("unexpected request: %#v", captured)
	}
	manifest := captured.GetPackage().GetManifest()
	if manifest.GetRisk() != pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH ||
		manifest.GetActivationPolicy() != pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM ||
		manifest.GetComposition().GetRole() != pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY {
		t.Fatalf("unexpected manifest enums: %#v", manifest)
	}
	if len(captured.GetPackage().GetSections()) != 1 ||
		captured.GetPackage().GetSections()[0].GetSectionKey() != "scoring_rules" {
		t.Fatalf("sections were not mapped: %#v", captured.GetPackage().GetSections())
	}
	if got := manifest.GetRequiredCapabilities(); len(got) != 1 || got[0] != "builtin:evaluate_candidate_match" {
		t.Fatalf("required capabilities were not trimmed: %#v", got)
	}
}

func TestAgentSkillHandlerCreateVersionMapsPackage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var captured *pb.CreateAgentSkillVersionRequest
	handler := NewAgentSkillHandler(&rpc.Clients{AgentSkill: &mockAgentSkillClient{
		createVersionFn: func(_ context.Context, req *pb.CreateAgentSkillVersionRequest, _ ...grpc.CallOption) (*pb.AgentSkillVersionResponse, error) {
			captured = req
			return &pb.AgentSkillVersionResponse{Code: 0, Msg: "ok", Version: &pb.AgentSkillVersionInfo{
				Id: 2, SkillId: req.SkillId, Version: req.Version, Package: &pb.AgentSkillPackageInfo{
					Manifest: req.Package.Manifest, CoreMarkdown: req.Package.CoreMarkdown,
				},
			}}, nil
		},
	}})
	router := gin.New()
	router.POST("/hr/agent-skills/:id/versions", handler.CreateVersion)
	body := `{"version":"2.1.0","activate":true,"package":` + packageJSON("medium", "primary", "none") + `}`
	req := httptest.NewRequest(http.MethodPost, "/hr/agent-skills/9/versions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK || captured == nil {
		t.Fatalf("status=%d captured=%#v body=%s", w.Code, captured, w.Body.String())
	}
	if captured.GetSkillId() != 9 || captured.GetPackage().GetManifest().GetActivationPolicy() !=
		pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO {
		t.Fatalf("unexpected request: %#v", captured)
	}
	if strings.Contains(w.Body.String(), "skill_md") || strings.Contains(w.Body.String(), "flow_json") {
		t.Fatalf("legacy fields leaked into response: %s", w.Body.String())
	}
}

func TestAgentSkillHandlerRejectsLegacyAndInvalidPackageRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAgentSkillHandler(&rpc.Clients{AgentSkill: &mockAgentSkillClient{}})
	router.POST("/hr/agent-skills", handler.Create)

	tests := []struct {
		name string
		body string
	}{
		{"legacy field", `{"name":"legacy","version":"1","skill_md":"old","package":` + packageJSON("low", "primary", "none") + `}`},
		{"activation mismatch", `{"version":"2","package":` + strings.Replace(packageJSON("high", "primary", "none"), `"risk":"high"`, `"risk":"high","activation_policy":"auto"`, 1) + `}`},
		{"supporting output", `{"version":"2","package":` + packageJSON("low", "supporting", "advisory") + `}`},
		{"invalid output schema", `{"version":"2","package":` + strings.Replace(packageJSON("low", "primary", "none"), `"schema_json":""`, `"schema_json":"not-json"`, 1) + `}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/hr/agent-skills", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			var response struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != 400 {
				t.Fatalf("expected code 400, got %d: %s", response.Code, w.Body.String())
			}
		})
	}
}

func TestAgentSkillHandlerUpdateOnlyMapsRegistryFields(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPut, "/hr/agent-skills/7", strings.NewReader(
		`{"display_name":"","description":"","is_enabled":false,"is_manual_invocable":true}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK || captured == nil {
		t.Fatalf("status=%d captured=%#v body=%s", w.Code, captured, w.Body.String())
	}
	if !captured.GetDisplayNameSet() || !captured.GetDescriptionSet() ||
		!captured.GetIsEnabledSet() || !captured.GetIsManualInvocableSet() {
		t.Fatalf("explicit registry fields were not mapped: %#v", captured)
	}
}

func TestAgentSkillHandlerPreviewReturnsTypedPackage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAgentSkillHandler(&rpc.Clients{AgentSkill: &mockAgentSkillClient{
		previewFn: func(_ context.Context, req *pb.PreviewAgentSkillRequest, _ ...grpc.CallOption) (*pb.PreviewAgentSkillResponse, error) {
			return &pb.PreviewAgentSkillResponse{Code: 0, Msg: "ok", Package: &pb.AgentSkillPackageInfo{
				Manifest:               req.Package.Manifest,
				CoreMarkdown:           req.Package.CoreMarkdown,
				CompiledMarkdown:       "# compiled",
				CompiledHash:           "abc123",
				CoreEstimatedTokens:    20,
				PackageEstimatedTokens: 40,
			}}, nil
		},
	}})
	router := gin.New()
	router.POST("/hr/agent-skills/preview", handler.Preview)
	req := httptest.NewRequest(http.MethodPost, "/hr/agent-skills/preview", strings.NewReader(
		`{"package":`+packageJSON("critical", "primary", "strict")+`}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"compiled_hash":"abc123"`) ||
		!strings.Contains(w.Body.String(), `"risk":"critical"`) ||
		!strings.Contains(w.Body.String(), `"activation_policy":"manual_only"`) {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}

func TestAgentSkillPayloadSkipsNilEntriesAndNormalizesRepeatedFields(t *testing.T) {
	list := agentSkillInfoListPayload([]*pb.AgentSkillInfo{nil, {Id: 7, Name: "screening"}})
	if len(list) != 1 || list[0].ID != 7 {
		t.Fatalf("nil skill entries must be skipped: %#v", list)
	}

	payload := agentSkillPackageToPayload(&pb.AgentSkillPackageInfo{
		Manifest: &pb.AgentSkillManifest{
			SchemaVersion:    2,
			SkillName:        "screening",
			DisplayName:      "Screening",
			Risk:             pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW,
			ActivationPolicy: pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO,
			Composition: &pb.AgentSkillComposition{
				Role: pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY,
			},
		},
		Sections: []*pb.AgentSkillSectionInfo{
			nil,
			{Id: 3, SectionKey: "rubric", Title: "Rubric"},
		},
		CompiledHash: "package-hash",
	})
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal package: %v", err)
	}
	body := string(data)
	for _, expected := range []string{
		`"required_capabilities":[]`,
		`"trigger_keywords":[]`,
		`"semantic_tags":[]`,
		`"evaluation_criteria":[]`,
		`"trigger_terms":[]`,
		`"planner_intents":[]`,
		`"risk":"low"`,
		`"activation_policy":"auto"`,
		`"composition":{"role":"primary"}`,
		`"compiled_hash":"package-hash"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %s in %s", expected, body)
		}
	}
}
