package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/logger"
	"smart-recruit-proto/recruitment/pb"
)

type SkillService struct {
	pb.UnimplementedSkillServiceServer
	repo       *repository.SkillRepo
	httpClient *http.Client
}

func NewSkillService(repo *repository.SkillRepo) *SkillService {
	return &SkillService{
		repo:       repo,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SkillService) ListSkills(ctx context.Context, req *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	page, pageSize := normalizeManagementPage(req.GetPage(), req.GetPageSize())
	skills, total, err := s.repo.ListSkills(ctx, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "list skills failed")
	}
	list := make([]*pb.SkillInfo, 0, len(skills))
	for i := range skills {
		list = append(list, skillToPB(&skills[i]))
	}
	return &pb.ListSkillsResponse{Code: 0, Msg: "success", Total: total, List: list}, nil
}

func (s *SkillService) CreateSkill(ctx context.Context, req *pb.CreateSkillRequest) (*pb.SkillResponse, error) {
	name := strings.TrimSpace(req.GetName())
	if !skillNamePattern.MatchString(name) {
		return nil, status.Error(codes.InvalidArgument, "invalid skill name")
	}
	sourceType := normalizeSkillSourceType(req.GetSourceType())
	if sourceType == "" {
		sourceType = "local"
	}
	if !isSupportedSkillSourceType(sourceType) {
		return nil, status.Error(codes.InvalidArgument, "invalid source_type")
	}
	displayName := strings.TrimSpace(req.GetDisplayName())
	if displayName == "" {
		displayName = name
	}
	isEnabled := int32(1)
	if req.GetIsEnabledSet() && !req.GetIsEnabled() {
		isEnabled = 0
	}
	skill := &model.Skill{
		Name:        name,
		DisplayName: displayName,
		Description: req.GetDescription(),
		SourceType:  sourceType,
		SourceURI:   req.GetSourceUri(),
		IsEnabled:   isEnabled,
	}
	if err := s.repo.CreateSkill(ctx, skill); err != nil {
		logger.L().Error("create skill failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create skill failed")
	}
	return &pb.SkillResponse{Code: 0, Msg: "success", Skill: skillToPB(skill)}, nil
}

func (s *SkillService) UpdateSkill(ctx context.Context, req *pb.UpdateSkillRequest) (*pb.SkillResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	updates := map[string]any{}
	if req.GetDisplayName() != "" {
		updates["display_name"] = req.GetDisplayName()
	}
	if req.GetDescription() != "" {
		updates["description"] = req.GetDescription()
	}
	if req.GetSourceType() != "" {
		sourceType := normalizeSkillSourceType(req.GetSourceType())
		if !isSupportedSkillSourceType(sourceType) {
			return nil, status.Error(codes.InvalidArgument, "invalid source_type")
		}
		updates["source_type"] = sourceType
	}
	if req.GetSourceUri() != "" {
		updates["source_uri"] = req.GetSourceUri()
	}
	if req.GetIsEnabledSet() {
		if req.GetIsEnabled() {
			updates["is_enabled"] = int32(1)
		} else {
			updates["is_enabled"] = int32(0)
		}
	}
	if len(updates) > 0 {
		if err := s.repo.UpdateSkillPartial(ctx, req.GetId(), updates); err != nil {
			return nil, status.Error(codes.Internal, "update skill failed")
		}
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "skill not found")
	}
	return &pb.SkillResponse{Code: 0, Msg: "success", Skill: skillToPB(skill)}, nil
}

func (s *SkillService) CreateSkillVersion(ctx context.Context, req *pb.CreateSkillVersionRequest) (*pb.SkillVersionResponse, error) {
	if req.GetSkillId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id is required")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "skill not found")
	}
	manifest, err := ParseAndValidateSkillManifest(req.GetManifestJson())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if manifest.Name != skill.Name {
		return nil, status.Error(codes.InvalidArgument, "manifest.name must match skill.name")
	}
	manifestJSON, _ := json.Marshal(manifest)
	version := &model.SkillVersion{
		SkillID:          skill.ID,
		Version:          manifest.Version,
		ManifestJSON:     string(manifestJSON),
		Instruction:      manifest.Instruction,
		InputSchemaJSON:  rawJSONPtr(manifest.InputSchema),
		OutputSchemaJSON: rawJSONPtr(manifest.OutputSchema),
		RuntimeType:      manifest.Runtime.Type,
	}
	tools := make([]model.SkillTool, 0, len(manifest.Tools))
	for _, t := range manifest.Tools {
		runtimeConfig := t.RuntimeConfig
		if len(runtimeConfig) == 0 || string(runtimeConfig) == "null" {
			runtimeConfig = t.Runtime.Config
		}
		tools = append(tools, model.SkillTool{
			ToolName:          t.Name,
			Description:       t.Description,
			InputSchemaJSON:   rawJSONPtr(t.InputSchema),
			RuntimeConfigJSON: rawJSONPtr(runtimeConfig),
			IsEnabled:         1,
		})
	}
	if err := s.repo.CreateVersionWithTools(ctx, version, tools, req.GetActivate()); err != nil {
		return nil, status.Error(codes.Internal, "create skill version failed")
	}
	storedTools, _ := s.repo.ListTools(ctx, version.ID, false)
	return &pb.SkillVersionResponse{
		Code:    0,
		Msg:     "success",
		Version: skillVersionToPB(version),
		Tools:   skillToolsToPB(skill.Name, storedTools),
	}, nil
}

func (s *SkillService) ListSkillVersions(ctx context.Context, req *pb.ListSkillVersionsRequest) (*pb.ListSkillVersionsResponse, error) {
	if req.GetSkillId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id is required")
	}
	versions, err := s.repo.ListVersions(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.Internal, "list skill versions failed")
	}
	list := make([]*pb.SkillVersionInfo, 0, len(versions))
	for i := range versions {
		list = append(list, skillVersionToPB(&versions[i]))
	}
	return &pb.ListSkillVersionsResponse{Code: 0, Msg: "success", List: list}, nil
}

func (s *SkillService) ActivateSkillVersion(ctx context.Context, req *pb.ActivateSkillVersionRequest) (*pb.SkillResponse, error) {
	if req.GetSkillId() <= 0 || req.GetVersionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id and version_id are required")
	}
	if err := s.repo.ActivateVersion(ctx, req.GetSkillId(), req.GetVersionId()); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "skill version not found")
		}
		return nil, status.Error(codes.Internal, "activate skill version failed")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.Internal, "get skill failed")
	}
	return &pb.SkillResponse{Code: 0, Msg: "success", Skill: skillToPB(skill)}, nil
}

func (s *SkillService) ListSkillTools(ctx context.Context, req *pb.ListSkillToolsRequest) (*pb.ListSkillToolsResponse, error) {
	versionID := req.GetSkillVersionId()
	skillName := ""
	if versionID <= 0 {
		if req.GetSkillId() <= 0 {
			return nil, status.Error(codes.InvalidArgument, "skill_id or skill_version_id is required")
		}
		skill, err := s.repo.GetSkillByID(ctx, req.GetSkillId())
		if err != nil {
			return nil, status.Error(codes.NotFound, "skill not found")
		}
		if skill.CurrentVersionID == nil || *skill.CurrentVersionID <= 0 {
			return &pb.ListSkillToolsResponse{Code: 0, Msg: "success"}, nil
		}
		versionID = *skill.CurrentVersionID
		skillName = skill.Name
	}
	if skillName == "" {
		version, err := s.repo.GetVersionByID(ctx, versionID)
		if err != nil {
			return nil, status.Error(codes.NotFound, "skill version not found")
		}
		skill, err := s.repo.GetSkillByID(ctx, version.SkillID)
		if err != nil {
			return nil, status.Error(codes.NotFound, "skill not found")
		}
		skillName = skill.Name
	}
	tools, err := s.repo.ListTools(ctx, versionID, req.GetEnabledOnly())
	if err != nil {
		return nil, status.Error(codes.Internal, "list skill tools failed")
	}
	return &pb.ListSkillToolsResponse{Code: 0, Msg: "success", List: skillToolsToPB(skillName, tools)}, nil
}

func (s *SkillService) UpdateSkillTool(ctx context.Context, req *pb.UpdateSkillToolRequest) (*pb.SkillToolResponse, error) {
	if req.GetToolId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}
	updates := map[string]any{}
	if req.GetDescription() != "" {
		updates["description"] = req.GetDescription()
	}
	if strings.TrimSpace(req.GetRuntimeConfigJson()) != "" {
		if err := validateSkillHTTPRuntimeConfigString(req.GetRuntimeConfigJson()); err != nil {
			return nil, status.Error(codes.InvalidArgument, "runtime_config_json is invalid: "+err.Error())
		}
		updates["runtime_config_json"] = req.GetRuntimeConfigJson()
	}
	if req.GetIsEnabledSet() {
		if req.GetIsEnabled() {
			updates["is_enabled"] = int32(1)
		} else {
			updates["is_enabled"] = int32(0)
		}
	}
	if len(updates) > 0 {
		if err := s.repo.UpdateToolPartial(ctx, req.GetToolId(), updates); err != nil {
			return nil, status.Error(codes.Internal, "update skill tool failed")
		}
	}
	updated, err := s.repo.GetToolByID(ctx, req.GetToolId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "skill tool not found")
	}
	version, err := s.repo.GetVersionByID(ctx, updated.SkillVersionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "skill version not found")
	}
	skill, err := s.repo.GetSkillByID(ctx, version.SkillID)
	if err != nil {
		return nil, status.Error(codes.Internal, "skill not found")
	}
	return &pb.SkillToolResponse{Code: 0, Msg: "success", Tool: skillToolsToPB(skill.Name, []model.SkillTool{*updated})[0]}, nil
}

func (s *SkillService) ListEnabledSkillCapabilities(ctx context.Context) ([]*pb.CapabilityInfo, error) {
	rows, err := s.repo.ListEnabledCurrentSkillTools(ctx)
	if err != nil {
		return nil, err
	}
	caps := make([]*pb.CapabilityInfo, 0, len(rows))
	for _, row := range rows {
		caps = append(caps, &pb.CapabilityInfo{
			Source:       "skill",
			Key:          SkillCapabilityKey(row.SkillName, row.ToolName),
			Name:         SkillRuntimeToolName(row.SkillName, row.ToolName),
			DisplayName:  row.SkillDisplayName + " / " + row.ToolName,
			Description:  firstNonEmpty(row.ToolDescription, row.SkillDescription),
			IsAvailable:  true,
			SkillId:      row.SkillID,
			SkillName:    row.SkillName,
			SkillVersion: row.Version,
			RuntimeType:  row.RuntimeType,
		})
	}
	return caps, nil
}

func (s *SkillService) CollectBoundSkillCallableTools(ctx context.Context, allowedKeys map[string]bool) ([]tool.BaseTool, []string, error) {
	rows, err := s.repo.ListEnabledCurrentSkillTools(ctx)
	if err != nil {
		return nil, nil, err
	}
	var tools []tool.BaseTool
	instructions := make([]string, 0)
	seenInstruction := map[int64]bool{}
	for _, row := range rows {
		key := SkillCapabilityKey(row.SkillName, row.ToolName)
		if allowedKeys != nil && !allowedKeys[key] {
			continue
		}
		if strings.TrimSpace(row.Instruction) != "" && !seenInstruction[row.SkillVersionID] {
			instructions = append(instructions, fmt.Sprintf("SKILL %s@%s:\n%s", row.SkillName, row.Version, row.Instruction))
			seenInstruction[row.SkillVersionID] = true
		}
		if row.RuntimeType != "http" {
			continue
		}
		toolInfo := &schema.ToolInfo{
			Name: SkillRuntimeToolName(row.SkillName, row.ToolName),
			Desc: firstNonEmpty(row.ToolDescription, row.SkillDescription),
		}
		if params := jsonSchemaToEinoParams(row.InputSchemaJSON); len(params) > 0 {
			toolInfo.ParamsOneOf = schema.NewParamsOneOfByParams(params)
		}
		runtimeConfig := ""
		if row.RuntimeConfigJSON != nil {
			runtimeConfig = *row.RuntimeConfigJSON
		}
		tools = append(tools, s.newHTTPTool(toolInfo, runtimeConfig))
	}
	return tools, instructions, nil
}

func (s *SkillService) CollectBoundSkillToolInfos(ctx context.Context, allowedKeys map[string]bool) ([]*schema.ToolInfo, []string, error) {
	callable, instructions, err := s.CollectBoundSkillCallableTools(ctx, allowedKeys)
	if err != nil {
		return nil, nil, err
	}
	infos := make([]*schema.ToolInfo, 0, len(callable))
	for _, t := range callable {
		info, err := t.Info(ctx)
		if err == nil {
			infos = append(infos, info)
		}
	}
	return infos, instructions, nil
}

func (s *SkillService) newHTTPTool(info *schema.ToolInfo, runtimeConfigJSON string) tool.BaseTool {
	return utils.NewTool(info, func(ctx context.Context, args map[string]any) (string, error) {
		var cfg SkillHTTPRuntimeConfig
		if err := json.Unmarshal([]byte(runtimeConfigJSON), &cfg); err != nil {
			return "", fmt.Errorf("invalid http runtime_config: %w", err)
		}
		if err := validateSkillHTTPURL(ctx, cfg.URL); err != nil {
			return "", fmt.Errorf("invalid http runtime_config: %w", err)
		}
		method := strings.ToUpper(strings.TrimSpace(cfg.Method))
		if method == "" {
			method = "POST"
		}
		if method != "POST" {
			return "", fmt.Errorf("unsupported skill http method %s", method)
		}
		client := skillHTTPClient(s.httpClient, cfg.TimeoutSeconds)
		payload, err := json.Marshal(args)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/plain")
		for k, v := range cfg.Headers {
			if strings.TrimSpace(k) != "" {
				req.Header.Set(k, v)
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("skill http status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return strings.TrimSpace(string(body)), nil
	})
}

func skillHTTPClient(base *http.Client, timeoutSeconds int) *http.Client {
	if base == nil {
		base = http.DefaultClient
	}
	client := *base
	if client.Transport == nil {
		client.Transport = safeSkillHTTPTransport()
	}
	if timeoutSeconds > 0 {
		client.Timeout = time.Duration(timeoutSeconds) * time.Second
	}
	previousCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := validateSkillHTTPURL(req.Context(), req.URL.String()); err != nil {
			return err
		}
		if previousCheckRedirect != nil {
			return previousCheckRedirect(req, via)
		}
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	}
	return &client
}

func safeSkillHTTPTransport() http.RoundTripper {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultTransport
	}
	transport := base.Clone()
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if isLocalhostName(host) {
			return nil, fmt.Errorf("url host is not allowed")
		}
		addrs, err := resolvePublicSkillHTTPAddrs(ctx, host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, addr := range addrs {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
	return transport
}

func SkillCapabilityKey(skillName, toolName string) string {
	return strings.TrimSpace(skillName) + ":" + strings.TrimSpace(toolName)
}

var nonToolNameChars = regexp.MustCompile(`[^A-Za-z0-9_]`)

func SkillRuntimeToolName(skillName, toolName string) string {
	name := "skill_" + strings.ReplaceAll(skillName, "-", "_") + "_" + strings.ReplaceAll(toolName, "-", "_")
	name = nonToolNameChars.ReplaceAllString(name, "_")
	return strings.Trim(name, "_")
}

func rawJSONPtr(raw json.RawMessage) *string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	v := string(raw)
	return &v
}

func normalizeSkillSourceType(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func isSupportedSkillSourceType(v string) bool {
	switch v {
	case "local", "git", "http", "mcp", "builtin":
		return true
	default:
		return false
	}
}

func jsonSchemaToEinoParams(schemaJSON *string) map[string]*schema.ParameterInfo {
	params := map[string]*schema.ParameterInfo{}
	if schemaJSON == nil || strings.TrimSpace(*schemaJSON) == "" {
		return params
	}
	var root struct {
		Required   []string                  `json:"required"`
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal([]byte(*schemaJSON), &root); err != nil {
		return params
	}
	required := map[string]bool{}
	for _, r := range root.Required {
		required[r] = true
	}
	for name, prop := range root.Properties {
		param := &schema.ParameterInfo{
			Type:     schemaTypeFromJSONSchema(getStringField(prop, "type")),
			Desc:     getStringField(prop, "description"),
			Required: required[name],
		}
		params[name] = param
	}
	return params
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func skillToPB(skill *model.Skill) *pb.SkillInfo {
	var currentVersionID int64
	if skill.CurrentVersionID != nil {
		currentVersionID = *skill.CurrentVersionID
	}
	return &pb.SkillInfo{
		Id:               skill.ID,
		Name:             skill.Name,
		DisplayName:      skill.DisplayName,
		Description:      skill.Description,
		SourceType:       skill.SourceType,
		SourceUri:        skill.SourceURI,
		CurrentVersionId: currentVersionID,
		IsEnabled:        skill.IsEnabled == 1,
		CreatedAt:        skill.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:        skill.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func skillVersionToPB(version *model.SkillVersion) *pb.SkillVersionInfo {
	return &pb.SkillVersionInfo{
		Id:               version.ID,
		SkillId:          version.SkillID,
		Version:          version.Version,
		ManifestJson:     version.ManifestJSON,
		Instruction:      version.Instruction,
		InputSchemaJson:  stringPtrValue(version.InputSchemaJSON),
		OutputSchemaJson: stringPtrValue(version.OutputSchemaJSON),
		RuntimeType:      version.RuntimeType,
		CreatedAt:        version.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func skillToolsToPB(skillName string, tools []model.SkillTool) []*pb.SkillToolInfo {
	list := make([]*pb.SkillToolInfo, 0, len(tools))
	for i := range tools {
		t := &tools[i]
		list = append(list, &pb.SkillToolInfo{
			Id:                t.ID,
			SkillVersionId:    t.SkillVersionID,
			ToolName:          t.ToolName,
			Description:       t.Description,
			InputSchemaJson:   stringPtrValue(t.InputSchemaJSON),
			RuntimeConfigJson: stringPtrValue(t.RuntimeConfigJSON),
			IsEnabled:         t.IsEnabled == 1,
			CapabilityKey:     SkillCapabilityKey(skillName, t.ToolName),
			RuntimeToolName:   SkillRuntimeToolName(skillName, t.ToolName),
			CreatedAt:         t.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:         t.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return list
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
