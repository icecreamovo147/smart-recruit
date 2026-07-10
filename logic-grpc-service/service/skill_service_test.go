package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

func setupSkillServiceTest(t *testing.T) (*SkillService, *repository.SkillRepo) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Skill{}, &model.SkillVersion{}, &model.SkillTool{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := repository.NewSkillRepo(db)
	return NewSkillService(repo), repo
}

func TestValidateSkillManifestRejectsInvalidManifest(t *testing.T) {
	cases := []string{
		`{"version":"1.0.0","runtime":{"type":"http"},"tools":[]}`,
		`{"name":"bad name","version":"1.0.0","runtime":{"type":"http"},"tools":[]}`,
		`{"name":"ok","version":"1.0.0","runtime":{"type":"http"},"tools":[{"name":"call","input_schema":{"type":"object"},"runtime":{"type":"http"},"runtime_config":{"url":"ftp://example.test","method":"POST"}}]}`,
		`{"name":"ok","version":"1.0.0","runtime":{"type":"shell"},"tools":[]}`,
	}
	for _, raw := range cases {
		if _, err := ParseAndValidateSkillManifest(raw); err == nil {
			t.Fatalf("expected invalid manifest error for %s", raw)
		}
	}
}

func TestValidateSkillManifestRejectsUnsafeHTTPRuntimeURLs(t *testing.T) {
	cases := []string{
		"127.0.0.1",
		"10.0.0.1",
		"169.254.169.254",
		"[::1]",
		"[fe80::1]",
		"[fc00::1]",
	}
	for _, host := range cases {
		raw := skillHTTPManifest("unsafe", "http://"+host+"/run", validSkillInputSchema())
		if _, err := ParseAndValidateSkillManifest(raw); err == nil {
			t.Fatalf("expected unsafe URL error for %s", host)
		}
	}
}

func TestValidateSkillManifestRejectsInvalidToolInputSchema(t *testing.T) {
	cases := []string{
		`[]`,
		`{"properties":{}}`,
		`{"type":"string"}`,
		`{"type":"object","properties":[]}`,
		`{"type":"object","properties":{"candidate_id":{"type":"integer"}},"required":"candidate_id"}`,
		`{"type":"object","properties":{"candidate_id":{"type":"integer"}},"required":[123]}`,
		`{"type":"object","properties":{"candidate_id":{"type":"integer"}},"required":["missing"]}`,
	}
	for _, inputSchema := range cases {
		raw := skillHTTPManifest("bad_schema", "http://93.184.216.34/run", inputSchema)
		if _, err := ParseAndValidateSkillManifest(raw); err == nil {
			t.Fatalf("expected invalid input_schema error for %s", inputSchema)
		}
	}
}

func TestValidateSkillManifestRequiresToolInputSchema(t *testing.T) {
	raw := skillHTTPManifest("missing_schema", "http://93.184.216.34/run", "")
	if _, err := ParseAndValidateSkillManifest(raw); err == nil {
		t.Fatalf("expected missing input_schema error")
	}
}

func TestSkillCatalogAndRuntimeOnlyIncludeEnabledBoundTools(t *testing.T) {
	svc, repo := setupSkillServiceTest(t)
	ctx := context.Background()
	if err := seedSkillVersion(ctx, repo, "resume_ops", 1, "parse", 1); err != nil {
		t.Fatalf("seed enabled skill: %v", err)
	}
	if err := seedSkillVersion(ctx, repo, "disabled_skill", 0, "lookup", 1); err != nil {
		t.Fatalf("seed disabled skill: %v", err)
	}
	if err := seedSkillVersion(ctx, repo, "disabled_tool", 1, "rank", 0); err != nil {
		t.Fatalf("seed disabled tool: %v", err)
	}

	caps, err := svc.ListEnabledSkillCapabilities(ctx)
	if err != nil {
		t.Fatalf("ListEnabledSkillCapabilities: %v", err)
	}
	if len(caps) != 1 || caps[0].Key != "resume_ops:parse" {
		t.Fatalf("expected only enabled skill capability, got %+v", caps)
	}

	tools, instructions, err := svc.CollectBoundSkillCallableTools(ctx, map[string]bool{"missing:tool": true})
	if err != nil {
		t.Fatalf("CollectBoundSkillCallableTools: %v", err)
	}
	if len(tools) != 0 || len(instructions) != 0 {
		t.Fatalf("unbound skill should not inject tools/instructions")
	}

	tools, instructions, err = svc.CollectBoundSkillCallableTools(ctx, map[string]bool{"resume_ops:parse": true})
	if err != nil {
		t.Fatalf("CollectBoundSkillCallableTools bound: %v", err)
	}
	if len(tools) != 1 || len(instructions) != 1 {
		t.Fatalf("bound skill should inject one tool and one instruction, got tools=%d instructions=%d", len(tools), len(instructions))
	}
	info, err := tools[0].Info(ctx)
	if err != nil {
		t.Fatalf("tool info: %v", err)
	}
	if info.Name != "skill_resume_ops_parse" {
		t.Fatalf("unexpected runtime tool name %q", info.Name)
	}
}

func TestSkillHTTPToolRejectsRedirectToPrivateAddress(t *testing.T) {
	svc, repo := setupSkillServiceTest(t)
	ctx := context.Background()
	if err := seedSkillVersion(ctx, repo, "resume_ops", 1, "parse", 1); err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	var roundTrips int
	svc.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		roundTrips++
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"http://127.0.0.1/internal"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})}

	tools, _, err := svc.CollectBoundSkillCallableTools(ctx, map[string]bool{"resume_ops:parse": true})
	if err != nil {
		t.Fatalf("CollectBoundSkillCallableTools: %v", err)
	}
	if _, err := tools[0].(tool.InvokableTool).InvokableRun(ctx, `{"candidate_id":123}`); err == nil {
		t.Fatalf("expected private redirect to be rejected")
	}
	if roundTrips != 1 {
		t.Fatalf("redirect target should be rejected before second request, got %d round trips", roundTrips)
	}
}

func TestUpdateSkillToolRejectsUnsafeRuntimeConfig(t *testing.T) {
	svc, repo := setupSkillServiceTest(t)
	ctx := context.Background()
	if err := seedSkillVersion(ctx, repo, "resume_ops", 1, "parse", 1); err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	skill, err := repo.GetSkillByName(ctx, "resume_ops")
	if err != nil {
		t.Fatalf("GetSkillByName: %v", err)
	}
	tools, err := repo.ListTools(ctx, *skill.CurrentVersionID, false)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	_, err = svc.UpdateSkillTool(ctx, &pb.UpdateSkillToolRequest{
		ToolId:            tools[0].ID,
		RuntimeConfigJson: `{"url":"http://169.254.169.254/latest/meta-data","method":"POST"}`,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for unsafe runtime_config, got %v", err)
	}
}

func TestSkillHTTPToolPostsJSONAndReturnsBody(t *testing.T) {
	svc, repo := setupSkillServiceTest(t)
	ctx := context.Background()
	if err := seedSkillVersion(ctx, repo, "resume_ops", 1, "parse", 1); err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	var gotBody string
	svc.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		body, _ := io.ReadAll(req.Body)
		gotBody = string(body)
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	})}

	tools, _, err := svc.CollectBoundSkillCallableTools(ctx, map[string]bool{"resume_ops:parse": true})
	if err != nil {
		t.Fatalf("CollectBoundSkillCallableTools: %v", err)
	}
	result, err := tools[0].(tool.InvokableTool).InvokableRun(ctx, `{"candidate_id":123}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if gotBody != `{"candidate_id":123}` {
		t.Fatalf("unexpected request body %s", gotBody)
	}
	if result != `{"ok":true}` {
		t.Fatalf("unexpected result %s", result)
	}
}

func skillHTTPManifest(name, runtimeURL, inputSchema string) string {
	schemaPart := ""
	if inputSchema != "" {
		schemaPart = `,"input_schema":` + inputSchema
	}
	return `{"name":"` + name + `","version":"1.0.0","runtime":{"type":"http"},"tools":[{"name":"call"` + schemaPart + `,"runtime":{"type":"http"},"runtime_config":{"url":"` + runtimeURL + `","method":"POST"}}]}`
}

func validSkillInputSchema() string {
	return `{"type":"object","properties":{"candidate_id":{"type":"integer"}},"required":["candidate_id"]}`
}

func seedSkillVersion(ctx context.Context, repo *repository.SkillRepo, name string, skillEnabled int32, toolName string, toolEnabled int32) error {
	skill := &model.Skill{Name: name, DisplayName: name, SourceType: "local", IsEnabled: 1}
	if err := repo.CreateSkill(ctx, skill); err != nil {
		return err
	}
	inputSchema := `{"type":"object","properties":{"candidate_id":{"type":"integer","description":"Candidate ID"}},"required":["candidate_id"]}`
	runtimeConfig := `{"url":"http://93.184.216.34/run","method":"POST","headers":{"X-Test":"1"}}`
	version := &model.SkillVersion{
		SkillID:         skill.ID,
		Version:         "1.0.0",
		ManifestJSON:    `{}`,
		Instruction:     "Use this skill only for resume operations.",
		InputSchemaJSON: &inputSchema,
		RuntimeType:     "http",
	}
	if err := repo.CreateVersionWithTools(ctx, version, []model.SkillTool{{
		ToolName:          toolName,
		Description:       "Parse resume",
		InputSchemaJSON:   &inputSchema,
		RuntimeConfigJSON: &runtimeConfig,
		IsEnabled:         1,
	}}, true); err != nil {
		return err
	}
	if skillEnabled == 0 {
		if err := repo.UpdateSkillPartial(ctx, skill.ID, map[string]any{"is_enabled": int32(0)}); err != nil {
			return err
		}
	}
	if toolEnabled == 0 {
		tools, err := repo.ListTools(ctx, version.ID, false)
		if err != nil {
			return err
		}
		if len(tools) > 0 {
			return repo.UpdateToolEnabled(ctx, tools[0].ID, false)
		}
	}
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
