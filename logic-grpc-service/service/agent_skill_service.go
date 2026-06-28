package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type AgentSkillService struct {
	pb.UnimplementedAgentSkillServiceServer
	repo *repository.AgentSkillRepo
}

func NewAgentSkillService(repo *repository.AgentSkillRepo) *AgentSkillService {
	return &AgentSkillService{repo: repo}
}

func (s *AgentSkillService) ListAgentSkills(ctx context.Context, req *pb.ListAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	page, pageSize := normalizePage(req.GetPage(), req.GetPageSize())
	skills, total, err := s.repo.ListSkills(ctx, page, pageSize, req.GetEnabledOnly(), req.GetKeyword())
	if err != nil {
		return nil, status.Error(codes.Internal, "list agent skills failed")
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: agentSkillsToPB(skills)}, nil
}

func (s *AgentSkillService) ListAvailableAgentSkills(ctx context.Context, req *pb.ListAvailableAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	page, pageSize := normalizePage(req.GetPage(), req.GetPageSize())
	skills, total, err := s.repo.ListAvailable(ctx, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "list available agent skills failed")
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: agentSkillsToPB(skills)}, nil
}

func (s *AgentSkillService) GetAgentSkill(ctx context.Context, req *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: agentSkillToPB(skill)}, nil
}

func (s *AgentSkillService) CreateAgentSkill(ctx context.Context, req *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	name := strings.TrimSpace(req.GetName())
	if !agentSkillNamePattern.MatchString(name) {
		return nil, status.Error(codes.InvalidArgument, "invalid agent skill name")
	}
	displayName := strings.TrimSpace(req.GetDisplayName())
	if displayName == "" {
		displayName = name
	}
	triggerKeywords, err := marshalStringList(req.GetTriggerKeywords())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid trigger_keywords")
	}
	enabled := boolToInt32(true)
	if req.GetIsEnabledSet() {
		enabled = boolToInt32(req.GetIsEnabled())
	}
	manual := boolToInt32(true)
	if req.GetIsManualInvocableSet() {
		manual = boolToInt32(req.GetIsManualInvocable())
	}
	createdBy := optionalPositiveInt64(req.GetActorUserId())
	skill := &model.AgentSkill{
		Name:              name,
		DisplayName:       displayName,
		Description:       strings.TrimSpace(req.GetDescription()),
		IsEnabled:         enabled,
		IsManualInvocable: manual,
		TriggerKeywords:   triggerKeywords,
		CreatedBy:         createdBy,
		UpdatedBy:         createdBy,
	}
	versionText := strings.TrimSpace(req.GetVersion())
	flowJSON := strings.TrimSpace(req.GetFlowJson())
	skillMDInput := strings.TrimSpace(req.GetSkillMd())
	if flowJSON != "" || skillMDInput != "" || versionText != "" {
		if flowJSON == "" && skillMDInput == "" {
			return nil, status.Error(codes.InvalidArgument, "flow_json or skill_md is required")
		}
		if versionText == "" {
			versionText = time.Now().Format("20060102150405")
		}
		if !agentSkillVersionRegexp.MatchString(versionText) {
			return nil, status.Error(codes.InvalidArgument, "invalid version")
		}
		skillMD, frontmatterJSON, bodyMarkdown, err := buildAgentSkillVersionContent(skill.Name, skill.Description, flowJSON, skillMDInput)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		canonicalFlow := ""
		if strings.TrimSpace(flowJSON) != "" {
			_, canonicalFlow, err = parseAgentSkillFlow(flowJSON)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		version := &model.AgentSkillVersion{
			Version:         versionText,
			FlowJSON:        canonicalFlow,
			SkillMD:         skillMD,
			FrontmatterJSON: frontmatterJSON,
			BodyMarkdown:    bodyMarkdown,
			ChangeNote:      strings.TrimSpace(req.GetChangeNote()),
			CreatedBy:       createdBy,
		}
		if err := s.repo.CreateSkillWithVersion(ctx, skill, version, true); err != nil {
			return nil, status.Error(codes.Internal, "create agent skill failed")
		}
	} else if err := s.repo.CreateSkill(ctx, skill); err != nil {
		return nil, status.Error(codes.Internal, "create agent skill failed")
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: agentSkillToPB(skill)}, nil
}

func (s *AgentSkillService) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	updates := map[string]any{}
	if req.GetDisplayName() != "" {
		updates["display_name"] = strings.TrimSpace(req.GetDisplayName())
	}
	if req.GetDescription() != "" {
		updates["description"] = strings.TrimSpace(req.GetDescription())
	}
	if req.GetTriggerKeywordsSet() {
		triggerKeywords, err := marshalStringList(req.GetTriggerKeywords())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid trigger_keywords")
		}
		updates["trigger_keywords"] = triggerKeywords
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = boolToInt32(req.GetIsEnabled())
	}
	if req.GetIsManualInvocableSet() {
		updates["is_manual_invocable"] = boolToInt32(req.GetIsManualInvocable())
	}
	if actor := optionalPositiveInt64(req.GetActorUserId()); actor != nil {
		updates["updated_by"] = actor
	}
	if len(updates) > 0 {
		if err := s.repo.UpdateSkillPartial(ctx, req.GetId(), updates); err != nil {
			return nil, status.Error(codes.Internal, "update agent skill failed")
		}
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: agentSkillToPB(skill)}, nil
}

func (s *AgentSkillService) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error) {
	if req.GetSkillId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id is required")
	}
	if !agentSkillVersionRegexp.MatchString(strings.TrimSpace(req.GetVersion())) {
		return nil, status.Error(codes.InvalidArgument, "invalid version")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	skillMDInput := strings.TrimSpace(req.GetSkillMd())
	flowJSON := strings.TrimSpace(req.GetFlowJson())
	if flowJSON == "" && skillMDInput == "" {
		return nil, status.Error(codes.InvalidArgument, "flow_json or skill_md is required")
	}
	skillMD, frontmatterJSON, bodyMarkdown, err := buildAgentSkillVersionContent(skill.Name, skill.Description, flowJSON, skillMDInput)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	canonicalFlow := ""
	if flowJSON != "" {
		_, canonicalFlow, err = parseAgentSkillFlow(flowJSON)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	version := &model.AgentSkillVersion{
		SkillID:         skill.ID,
		Version:         strings.TrimSpace(req.GetVersion()),
		FlowJSON:        canonicalFlow,
		SkillMD:         skillMD,
		FrontmatterJSON: frontmatterJSON,
		BodyMarkdown:    bodyMarkdown,
		ChangeNote:      strings.TrimSpace(req.GetChangeNote()),
		CreatedBy:       optionalPositiveInt64(req.GetActorUserId()),
	}
	if err := s.repo.CreateVersion(ctx, version, req.GetActivate()); err != nil {
		return nil, status.Error(codes.Internal, "create agent skill version failed")
	}
	return &pb.AgentSkillVersionResponse{Code: 0, Msg: "success", Version: agentSkillVersionToPB(version)}, nil
}

func (s *AgentSkillService) ListAgentSkillVersions(ctx context.Context, req *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error) {
	if req.GetSkillId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id is required")
	}
	versions, err := s.repo.ListVersions(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.Internal, "list agent skill versions failed")
	}
	list := make([]*pb.AgentSkillVersionInfo, 0, len(versions))
	for i := range versions {
		list = append(list, agentSkillVersionToPB(&versions[i]))
	}
	return &pb.ListAgentSkillVersionsResponse{Code: 0, Msg: "success", List: list}, nil
}

func (s *AgentSkillService) ActivateAgentSkillVersion(ctx context.Context, req *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error) {
	if req.GetSkillId() <= 0 || req.GetVersionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "skill_id and version_id are required")
	}
	if err := s.repo.ActivateVersion(ctx, req.GetSkillId(), req.GetVersionId()); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "agent skill version not found")
		}
		return nil, status.Error(codes.Internal, "activate agent skill version failed")
	}
	if actor := optionalPositiveInt64(req.GetActorUserId()); actor != nil {
		_ = s.repo.UpdateSkillPartial(ctx, req.GetSkillId(), map[string]any{"updated_by": actor})
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetSkillId())
	if err != nil {
		return nil, status.Error(codes.Internal, "get agent skill failed")
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: agentSkillToPB(skill)}, nil
}

func (s *AgentSkillService) UpdateAgentSkillStatus(ctx context.Context, req *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	updates := map[string]any{"is_enabled": boolToInt32(req.GetIsEnabled())}
	if actor := optionalPositiveInt64(req.GetActorUserId()); actor != nil {
		updates["updated_by"] = actor
	}
	if err := s.repo.UpdateSkillPartial(ctx, req.GetId(), updates); err != nil {
		return nil, status.Error(codes.Internal, "update agent skill status failed")
	}
	skill, err := s.repo.GetSkillByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "agent skill not found")
	}
	return &pb.AgentSkillResponse{Code: 0, Msg: "success", Skill: agentSkillToPB(skill)}, nil
}

func (s *AgentSkillService) PreviewAgentSkill(ctx context.Context, req *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error) {
	name := strings.TrimSpace(req.GetName())
	if !agentSkillNamePattern.MatchString(name) {
		return nil, status.Error(codes.InvalidArgument, "invalid agent skill name")
	}
	skillMD, frontmatterJSON, bodyMarkdown, err := GenerateAgentSkillMarkdown(name, strings.TrimSpace(req.GetDescription()), req.GetFlowJson())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.PreviewAgentSkillResponse{Code: 0, Msg: "success", SkillMd: skillMD, FrontmatterJson: frontmatterJSON, BodyMarkdown: bodyMarkdown}, nil
}

func normalizePage(page, pageSize int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func agentSkillsToPB(skills []model.AgentSkill) []*pb.AgentSkillInfo {
	list := make([]*pb.AgentSkillInfo, 0, len(skills))
	for i := range skills {
		list = append(list, agentSkillToPB(&skills[i]))
	}
	return list
}

func agentSkillToPB(skill *model.AgentSkill) *pb.AgentSkillInfo {
	if skill == nil {
		return nil
	}
	var currentVersionID int64
	if skill.CurrentVersionID != nil {
		currentVersionID = *skill.CurrentVersionID
	}
	keywords := []string{}
	_ = json.Unmarshal([]byte(skill.TriggerKeywords), &keywords)
	return &pb.AgentSkillInfo{
		Id:                skill.ID,
		Name:              skill.Name,
		DisplayName:       skill.DisplayName,
		Description:       skill.Description,
		CurrentVersionId:  currentVersionID,
		IsEnabled:         skill.IsEnabled == 1,
		IsManualInvocable: skill.IsManualInvocable == 1,
		TriggerKeywords:   keywords,
		CreatedAt:         formatTime(skill.CreatedAt),
		UpdatedAt:         formatTime(skill.UpdatedAt),
	}
}

func agentSkillVersionToPB(version *model.AgentSkillVersion) *pb.AgentSkillVersionInfo {
	if version == nil {
		return nil
	}
	return &pb.AgentSkillVersionInfo{
		Id:              version.ID,
		SkillId:         version.SkillID,
		Version:         version.Version,
		FlowJson:        version.FlowJSON,
		SkillMd:         version.SkillMD,
		FrontmatterJson: version.FrontmatterJSON,
		BodyMarkdown:    version.BodyMarkdown,
		ChangeNote:      version.ChangeNote,
		CreatedAt:       formatTime(version.CreatedAt),
	}
}

func buildAgentSkillVersionContent(name, description, flowJSON, skillMDInput string) (skillMD, frontmatterJSON, bodyMarkdown string, err error) {
	if strings.TrimSpace(skillMDInput) == "" {
		return GenerateAgentSkillMarkdown(name, description, flowJSON)
	}
	doc, err := ParseAgentSkillMarkdown(skillMDInput)
	if err != nil {
		return "", "", "", err
	}
	if frontmatterString(doc.Frontmatter, "name") != strings.TrimSpace(name) {
		return "", "", "", fmt.Errorf("skill_md frontmatter name must match agent skill name")
	}
	if frontmatterString(doc.Frontmatter, "description") != strings.TrimSpace(description) {
		return "", "", "", fmt.Errorf("skill_md frontmatter description must match agent skill description")
	}
	frontmatterBytes, err := json.Marshal(doc.Frontmatter)
	if err != nil {
		return "", "", "", err
	}
	return strings.TrimSpace(skillMDInput) + "\n", string(frontmatterBytes), doc.Body, nil
}

func marshalStringList(values []string) (string, error) {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			clean = append(clean, value)
		}
	}
	b, err := json.Marshal(clean)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func optionalPositiveInt64(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}
