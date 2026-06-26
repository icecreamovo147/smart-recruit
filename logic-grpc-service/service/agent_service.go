package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

// AgentConfigService implements the pb.AgentConfigServiceServer interface.
type AgentConfigService struct {
	pb.UnimplementedAgentConfigServiceServer
	repo              *repository.AgentConfigRepo
	modelRepo         *repository.ModelConfigRepo
	promptRepo        *repository.PromptTemplateRepo
}

// NewAgentConfigService creates a new AgentConfigService.
func NewAgentConfigService(
	repo *repository.AgentConfigRepo,
	modelRepo *repository.ModelConfigRepo,
	promptRepo *repository.PromptTemplateRepo,
) *AgentConfigService {
	return &AgentConfigService{
		repo:       repo,
		modelRepo:  modelRepo,
		promptRepo: promptRepo,
	}
}

// ── CRUD ────────────────────────────────────────────────────────────────

// ListAgents returns a paginated list of agent configurations.
func (s *AgentConfigService) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}

	agents, total, err := s.repo.List(ctx, page, pageSize, req.GetAgentType())
	if err != nil {
		logger.L().Error("list agents failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list agents failed")
	}

	list := make([]*pb.AgentConfigInfo, 0, len(agents))
	for _, a := range agents {
		info, err := s.agentToPB(ctx, &a)
		if err != nil {
			logger.L().Warn("convert agent to pb failed", zap.Int64("id", a.ID), zap.Error(err))
			info = agentToPBBasic(&a)
		}
		list = append(list, info)
	}

	return &pb.ListAgentsResponse{
		Code:  0,
		Msg:   "success",
		Total: total,
		List:  list,
	}, nil
}

// CreateAgent creates a new agent configuration.
func (s *AgentConfigService) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.AgentConfigResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if strings.TrimSpace(req.GetAgentType()) == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_type is required")
	}
	if strings.TrimSpace(req.GetDisplayName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "display_name is required")
	}

	// Check name uniqueness.
	exists, err := s.repo.ExistsByName(ctx, req.GetName())
	if err != nil {
		logger.L().Error("check agent name failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "check agent name failed")
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "agent with this name already exists")
	}

	// Validate FK references if specified.
	if req.GetModelId() > 0 {
		if _, err := s.modelRepo.GetByID(ctx, req.GetModelId()); err != nil {
			return nil, status.Error(codes.InvalidArgument, "model_id not found")
		}
	}
	if req.GetPromptTemplateId() > 0 {
		if _, err := s.promptRepo.GetByID(ctx, req.GetPromptTemplateId()); err != nil {
			return nil, status.Error(codes.InvalidArgument, "prompt_template_id not found")
		}
	}

	maxIter := req.GetMaxIterations()
	if maxIter <= 0 {
		maxIter = 5
	}

	var tempOverride *float64
	if req.GetTemperatureOverrideSet() {
		v := req.GetTemperatureOverride()
		tempOverride = &v
	}

	isDefault := int32(0)
	if req.GetIsDefault() {
		isDefault = 1
		// Clear existing defaults for this type.
		if err := s.repo.ClearDefaultsForType(ctx, req.GetAgentType()); err != nil {
			logger.L().Error("clear defaults failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "clear defaults failed")
		}
	}

	cfg := &model.AgentConfig{
		Name:                req.GetName(),
		DisplayName:         req.GetDisplayName(),
		Description:         req.GetDescription(),
		AgentType:           req.GetAgentType(),
		ModelID:             int64Ptr(req.GetModelId(), req.GetModelId() > 0),
		PromptTemplateID:    int64Ptr(req.GetPromptTemplateId(), req.GetPromptTemplateId() > 0),
		Instruction:         req.GetInstruction(),
		MaxIterations:       maxIter,
		TemperatureOverride: tempOverride,
		IsDefault:           isDefault,
		IsEnabled:           1,
	}

	if err := s.repo.Create(ctx, cfg); err != nil {
		logger.L().Error("create agent config failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create agent config failed")
	}

	// Bind tools.
	if len(req.GetToolNames()) > 0 {
		if err := s.repo.ReplaceToolBindings(ctx, cfg.ID, req.GetToolNames()); err != nil {
			logger.L().Error("bind tools failed", zap.Error(err))
			// Non-fatal: agent config was created, tool bindings can be updated later.
		}
	}

	info, err := s.agentToPB(ctx, cfg)
	if err != nil {
		info = agentToPBBasic(cfg)
	}

	return &pb.AgentConfigResponse{
		Code:  0,
		Msg:   "success",
		Agent: info,
	}, nil
}

// UpdateAgent updates an existing agent configuration.
func (s *AgentConfigService) UpdateAgent(ctx context.Context, req *pb.UpdateAgentRequest) (*pb.AgentConfigResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	existing, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "agent config not found")
		}
		logger.L().Error("get agent config failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get agent config failed")
	}

	updates := make(map[string]any)

	if req.GetName() != "" {
		updates["name"] = req.GetName()
	}
	if req.GetDisplayName() != "" {
		updates["display_name"] = req.GetDisplayName()
	}
	if req.GetDescription() != "" {
		updates["description"] = req.GetDescription()
	}
	if req.GetInstruction() != "" {
		updates["instruction"] = req.GetInstruction()
	}
	if req.GetModelIdSet() {
		if req.GetModelId() > 0 {
			if _, err := s.modelRepo.GetByID(ctx, req.GetModelId()); err != nil {
				return nil, status.Error(codes.InvalidArgument, "model_id not found")
			}
		}
		v := req.GetModelId()
		updates["model_id"] = &v
		if v == 0 {
			updates["model_id"] = nil
		}
	}
	if req.GetPromptTemplateIdSet() {
		if req.GetPromptTemplateId() > 0 {
			if _, err := s.promptRepo.GetByID(ctx, req.GetPromptTemplateId()); err != nil {
				return nil, status.Error(codes.InvalidArgument, "prompt_template_id not found")
			}
		}
		v := req.GetPromptTemplateId()
		if v > 0 {
			updates["prompt_template_id"] = v
		} else {
			updates["prompt_template_id"] = nil
		}
	}
	if req.GetMaxIterationsSet() {
		v := req.GetMaxIterations()
		if v <= 0 {
			v = 5
		}
		updates["max_iterations"] = v
	}
	if req.GetTemperatureOverrideSet() {
		v := req.GetTemperatureOverride()
		updates["temperature_override"] = &v
	}
	if req.GetIsDefaultSet() {
		if req.GetIsDefault() {
			if err := s.repo.ClearDefaultsForType(ctx, existing.AgentType); err != nil {
				logger.L().Error("clear defaults failed", zap.Error(err))
				return nil, status.Error(codes.Internal, "clear defaults failed")
			}
		}
		isDefault := int32(0)
		if req.GetIsDefault() {
			isDefault = 1
		}
		updates["is_default"] = isDefault
	}
	if req.GetIsEnabledSet() {
		isEnabled := int32(0)
		if req.GetIsEnabled() {
			isEnabled = 1
		}
		updates["is_enabled"] = isEnabled
	}

	if len(updates) > 0 {
		if err := s.repo.UpdatePartial(ctx, existing.ID, updates); err != nil {
			logger.L().Error("update agent config failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "update agent config failed")
		}
	}

	// Replace tool bindings if explicitly set.
	if req.GetToolNamesSet() {
		if err := s.repo.ReplaceToolBindings(ctx, existing.ID, req.GetToolNames()); err != nil {
			logger.L().Error("replace tool bindings failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "replace tool bindings failed")
		}
	}

	// Fetch updated config.
	updated, err := s.repo.GetByID(ctx, existing.ID)
	if err != nil {
		logger.L().Error("get updated agent config failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get updated agent config failed")
	}

	info, err := s.agentToPB(ctx, updated)
	if err != nil {
		info = agentToPBBasic(updated)
	}

	return &pb.AgentConfigResponse{
		Code:  0,
		Msg:   "success",
		Agent: info,
	}, nil
}

// DeleteAgent deletes an agent configuration.
func (s *AgentConfigService) DeleteAgent(ctx context.Context, req *pb.DeleteAgentRequest) (*pb.CommonResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.repo.Delete(ctx, req.GetId()); err != nil {
		logger.L().Error("delete agent config failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "delete agent config failed")
	}

	return &pb.CommonResponse{Code: 0, Msg: "success"}, nil
}

// ── Internal RPC ─────────────────────────────────────────────────────────

// GetAgentConfig returns the active agent config for a given agent type, with tool bindings.
// This is used internally by the AI service to build agent runtime configurations.
func (s *AgentConfigService) GetAgentConfig(ctx context.Context, req *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error) {
	if strings.TrimSpace(req.GetAgentType()) == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_type is required")
	}

	cfg, err := s.repo.GetByAgentType(ctx, req.GetAgentType())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "no active agent config found for agent_type: "+req.GetAgentType())
		}
		logger.L().Error("get agent config failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get agent config failed")
	}

	info, err := s.agentToPB(ctx, cfg)
	if err != nil {
		info = agentToPBBasic(cfg)
	}

	return &pb.GetAgentConfigResponse{
		Code:  0,
		Msg:   "success",
		Agent: info,
	}, nil
}

// ── Helper functions ─────────────────────────────────────────────────────

func (s *AgentConfigService) agentToPB(ctx context.Context, cfg *model.AgentConfig) (*pb.AgentConfigInfo, error) {
	info := agentToPBBasic(cfg)

	// Join model name.
	if cfg.ModelID != nil && *cfg.ModelID > 0 {
		if m, err := s.modelRepo.GetByID(ctx, *cfg.ModelID); err == nil && m != nil {
			info.ModelName = m.ModelName
		}
	}

	// Join prompt template name.
	if cfg.PromptTemplateID != nil && *cfg.PromptTemplateID > 0 {
		if t, err := s.promptRepo.GetByID(ctx, *cfg.PromptTemplateID); err == nil && t != nil {
			info.PromptTemplateName = t.Name
		}
	}

	// Load tool bindings.
	bindings, err := s.repo.ListToolBindings(ctx, cfg.ID)
	if err == nil {
		pbBindings := make([]*pb.AgentToolBindingInfo, 0, len(bindings))
		for _, b := range bindings {
			pbBindings = append(pbBindings, &pb.AgentToolBindingInfo{
				Id:        b.ID,
				AgentId:   b.AgentID,
				ToolName:  b.ToolName,
				IsEnabled: b.IsEnabled == 1,
				CreatedAt: b.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		info.ToolBindings = pbBindings
	}

	return info, nil
}

func agentToPBBasic(cfg *model.AgentConfig) *pb.AgentConfigInfo {
	var modelID int64
	if cfg.ModelID != nil {
		modelID = *cfg.ModelID
	}
	var promptTemplateID int64
	if cfg.PromptTemplateID != nil {
		promptTemplateID = *cfg.PromptTemplateID
	}
	var tempOverride float64
	if cfg.TemperatureOverride != nil {
		tempOverride = *cfg.TemperatureOverride
	}

	return &pb.AgentConfigInfo{
		Id:                   cfg.ID,
		Name:                 cfg.Name,
		DisplayName:          cfg.DisplayName,
		Description:          cfg.Description,
		AgentType:            cfg.AgentType,
		ModelId:              modelID,
		PromptTemplateId:     promptTemplateID,
		Instruction:          cfg.Instruction,
		MaxIterations:        cfg.MaxIterations,
		TemperatureOverride:  tempOverride,
		IsDefault:            cfg.IsDefault == 1,
		IsEnabled:            cfg.IsEnabled == 1,
		CreatedAt:            cfg.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:            cfg.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func int64Ptr(v int64, set bool) *int64 {
	if !set {
		return nil
	}
	return &v
}

// ── Seed Functions ───────────────────────────────────────────────────────

// SeedDefaultAgents creates the initial agent configurations from hardcoded defaults.
// This is idempotent: it only inserts if no agents exist for the agent_type.
func SeedDefaultAgents(ctx context.Context, agentRepo *repository.AgentConfigRepo) error {
	// Check if HR agent already seeded.
	_, err := agentRepo.GetByAgentType(ctx, "hr_recruiting_agent")
	if err == nil {
		logger.L().Info("agents already seeded, skipping")
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		logger.L().Warn("check existing agents failed, will attempt seed", zap.Error(err))
	}

	// Define HR recruiting agent tool names (from ai/hr_adk_tools.go).
	hrToolNames := []string{
		"query_total_applications",
		"query_today_applications",
		"get_job_heat_ranking",
		"search_candidates",
		"get_job_detail",
		"search_jobs",
		"get_candidate_detail",
		"propose_application_status_update",
		"list_all_applications",
		"list_applications_by_job",
		"list_applications_by_status",
		"get_application_status_summary",
		"get_application_trend",
		"get_job_list",
	}

	candidateToolNames := []string{
		"list_my_applications",
		"get_my_application_detail",
		"get_my_resume_text",
		"list_jobs_for_recommendation",
		"get_job_detail_for_candidate",
		"recommend_jobs_by_resume",
	}

	maxIter := int32(5)

	// Seed HR recruiting agent.
	hrAgent := &model.AgentConfig{
		Name:          "hr_recruiting_agent",
		DisplayName:   "HR 招聘助手",
		Description:   "基于招聘系统数据的 AI HR 助手，支持岗位、候选人、投递等实时数据查询与分析",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: maxIter,
		IsDefault:     1,
		IsEnabled:     1,
	}
	if err := agentRepo.Create(ctx, hrAgent); err != nil {
		return fmt.Errorf("seed hr agent: %w", err)
	}
	if err := agentRepo.ReplaceToolBindings(ctx, hrAgent.ID, hrToolNames); err != nil {
		logger.L().Warn("seed hr agent tool bindings failed", zap.Error(err))
	}
	logger.L().Info("seeded hr_recruiting_agent", zap.Int64("id", hrAgent.ID))

	// Seed candidate assistant agent.
	candidateAgent := &model.AgentConfig{
		Name:          "candidate_assistant",
		DisplayName:   "候选人 AI 助手",
		Description:   "为候选人提供投递进度查询、岗位推荐、简历优化建议的 AI 助手",
		AgentType:     "candidate_assistant",
		MaxIterations: maxIter,
		IsDefault:     1,
		IsEnabled:     1,
	}
	if err := agentRepo.Create(ctx, candidateAgent); err != nil {
		return fmt.Errorf("seed candidate agent: %w", err)
	}
	if err := agentRepo.ReplaceToolBindings(ctx, candidateAgent.ID, candidateToolNames); err != nil {
		logger.L().Warn("seed candidate agent tool bindings failed", zap.Error(err))
	}
	logger.L().Info("seeded candidate_assistant", zap.Int64("id", candidateAgent.ID))

	return nil
}
