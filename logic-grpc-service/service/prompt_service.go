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

// PromptService implements the pb.PromptServiceServer interface.
type PromptService struct {
	pb.UnimplementedPromptServiceServer
	repo *repository.PromptTemplateRepo
}

// NewPromptService creates a new PromptService.
func NewPromptService(repo *repository.PromptTemplateRepo) *PromptService {
	return &PromptService{repo: repo}
}

// ── Template CRUD ────────────────────────────────────────────────────────

// ListPromptTemplates returns a paginated list of prompt templates.
func (s *PromptService) ListPromptTemplates(ctx context.Context, req *pb.ListPromptTemplatesRequest) (*pb.ListPromptTemplatesResponse, error) {
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}

	templates, total, err := s.repo.List(ctx, page, pageSize, req.GetAgentType())
	if err != nil {
		logger.L().Error("list prompt templates failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list prompt templates failed")
	}

	list := make([]*pb.PromptTemplateInfo, 0, len(templates))
	for _, t := range templates {
		list = append(list, templateToPB(&t))
	}

	return &pb.ListPromptTemplatesResponse{
		Code:  0,
		Msg:   "success",
		Total: total,
		List:  list,
	}, nil
}

// CreatePromptTemplate creates a new prompt template.
func (s *PromptService) CreatePromptTemplate(ctx context.Context, req *pb.CreatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if strings.TrimSpace(req.GetContent()) == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	if strings.TrimSpace(req.GetAgentType()) == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_type is required")
	}

	promptRole := req.GetPromptRole()
	if promptRole == "" {
		promptRole = "system"
	}

	variables := req.GetVariablesJson()
	var variablesPtr *string
	if variables != "" {
		variablesPtr = &variables
	}

	createdBy := req.GetCreatedBy()
	var createdByPtr *int64
	if createdBy > 0 {
		createdByPtr = &createdBy
	}

	t := &model.PromptTemplate{
		Name:       req.GetName(),
		Content:    req.GetContent(),
		Variables:  variablesPtr,
		Version:    1,
		IsActive:   1,
		AgentType:  req.GetAgentType(),
		PromptRole: promptRole,
		CreatedBy:  createdByPtr,
		UpdatedBy:  createdByPtr,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		logger.L().Error("create prompt template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create prompt template failed")
	}

	// Create initial version record.
	if err := s.repo.CreateVersion(ctx, &model.PromptVersion{
		TemplateID: t.ID,
		Version:    1,
		Content:    req.GetContent(),
		ChangedBy:  createdByPtr,
		ChangeNote: "initial version",
	}); err != nil {
		logger.L().Error("create initial version failed", zap.Error(err))
		// Non-fatal: template was created, version history is a secondary concern.
	}

	return &pb.PromptTemplateResponse{
		Code:     0,
		Msg:      "success",
		Template: templateToPB(t),
	}, nil
}

// UpdatePromptTemplate updates a prompt template. If content changes, a new version is created.
func (s *PromptService) UpdatePromptTemplate(ctx context.Context, req *pb.UpdatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	existing, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "prompt template not found")
		}
		logger.L().Error("get prompt template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get prompt template failed")
	}

	updatedBy := req.GetUpdatedBy()
	var updatedByPtr *int64
	if updatedBy > 0 {
		updatedByPtr = &updatedBy
	}

	updates := make(map[string]any)

	if req.GetName() != "" {
		updates["name"] = req.GetName()
	}
	if req.GetVariablesJson() != "" {
		updates["variables"] = req.GetVariablesJson()
	}
	if req.GetIsActiveSet() {
		isActive := int32(0)
		if req.GetIsActive() {
			isActive = 1
		}
		updates["is_active"] = isActive
	}
	if updatedByPtr != nil {
		updates["updated_by"] = *updatedByPtr
	}

	contentChanged := req.GetContent() != "" && req.GetContent() != existing.Content
	if contentChanged {
		newVersion := existing.Version + 1
		updates["content"] = req.GetContent()
		updates["version"] = newVersion
	}

	if len(updates) == 0 {
		return &pb.PromptTemplateResponse{
			Code:     0,
			Msg:      "no changes",
			Template: templateToPB(existing),
		}, nil
	}

	if err := s.repo.UpdatePartial(ctx, existing.ID, updates); err != nil {
		logger.L().Error("update prompt template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "update prompt template failed")
	}

	// Create version record if content changed.
	if contentChanged {
		newVersion := existing.Version + 1
		changeNote := req.GetChangeNote()
		if changeNote == "" {
			changeNote = "updated content"
		}
		if err := s.repo.CreateVersion(ctx, &model.PromptVersion{
			TemplateID: existing.ID,
			Version:    newVersion,
			Content:    req.GetContent(),
			ChangedBy:  updatedByPtr,
			ChangeNote: changeNote,
		}); err != nil {
			logger.L().Error("create version record failed", zap.Error(err))
		}
	}

	// Fetch updated template.
	updated, err := s.repo.GetByID(ctx, existing.ID)
	if err != nil {
		logger.L().Error("get updated template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get updated template failed")
	}

	return &pb.PromptTemplateResponse{
		Code:     0,
		Msg:      "success",
		Template: templateToPB(updated),
	}, nil
}

// DeletePromptTemplate soft-deletes a prompt template.
func (s *PromptService) DeletePromptTemplate(ctx context.Context, req *pb.DeletePromptTemplateRequest) (*pb.CommonResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.repo.Delete(ctx, req.GetId()); err != nil {
		logger.L().Error("delete prompt template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "delete prompt template failed")
	}

	return &pb.CommonResponse{Code: 0, Msg: "success"}, nil
}

// ── Version Management ──────────────────────────────────────────────────

// GetPromptVersionHistory returns paginated version history for a template.
func (s *PromptService) GetPromptVersionHistory(ctx context.Context, req *pb.GetPromptVersionHistoryRequest) (*pb.GetPromptVersionHistoryResponse, error) {
	if req.GetTemplateId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}

	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}

	versions, total, err := s.repo.ListVersions(ctx, req.GetTemplateId(), page, pageSize)
	if err != nil {
		logger.L().Error("list versions failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list versions failed")
	}

	list := make([]*pb.PromptVersionInfo, 0, len(versions))
	for _, v := range versions {
		list = append(list, versionToPB(&v))
	}

	return &pb.GetPromptVersionHistoryResponse{
		Code:  0,
		Msg:   "success",
		Total: total,
		List:  list,
	}, nil
}

// RollbackPromptVersion rolls back a template to a specific version.
func (s *PromptService) RollbackPromptVersion(ctx context.Context, req *pb.RollbackPromptVersionRequest) (*pb.PromptTemplateResponse, error) {
	if req.GetTemplateId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if req.GetVersion() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "version is required")
	}

	// Get the target version snapshot.
	versionRec, err := s.repo.GetVersion(ctx, req.GetTemplateId(), req.GetVersion())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "version not found")
		}
		logger.L().Error("get version failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get version failed")
	}

	// Get current template.
	existing, err := s.repo.GetByID(ctx, req.GetTemplateId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "prompt template not found")
		}
		logger.L().Error("get template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get template failed")
	}

	newVersion := existing.Version + 1
	updatedBy := req.GetUpdatedBy()
	var updatedByPtr *int64
	if updatedBy > 0 {
		updatedByPtr = &updatedBy
	}

	changeNote := req.GetChangeNote()
	if changeNote == "" {
		changeNote = fmt.Sprintf("rolled back to version %d", req.GetVersion())
	}

	// Update template content to the old version's content, increment version.
	if err := s.repo.UpdatePartial(ctx, existing.ID, map[string]any{
		"content":    versionRec.Content,
		"version":    newVersion,
		"updated_by": updatedBy,
	}); err != nil {
		logger.L().Error("rollback template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "rollback template failed")
	}

	// Create new version record for the rollback.
	if err := s.repo.CreateVersion(ctx, &model.PromptVersion{
		TemplateID: existing.ID,
		Version:    newVersion,
		Content:    versionRec.Content,
		ChangedBy:  updatedByPtr,
		ChangeNote: changeNote,
	}); err != nil {
		logger.L().Error("create rollback version record failed", zap.Error(err))
	}

	// Fetch updated template.
	updated, err := s.repo.GetByID(ctx, existing.ID)
	if err != nil {
		logger.L().Error("get updated template after rollback failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get updated template failed")
	}

	return &pb.PromptTemplateResponse{
		Code:     0,
		Msg:      "success",
		Template: templateToPB(updated),
	}, nil
}

// ── Internal RPC methods ─────────────────────────────────────────────────

// RenderPrompt performs variable interpolation on a prompt template.
func (s *PromptService) RenderPrompt(ctx context.Context, req *pb.RenderPromptRequest) (*pb.RenderPromptResponse, error) {
	if req.GetTemplateId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}

	t, err := s.repo.GetByID(ctx, req.GetTemplateId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "prompt template not found")
		}
		logger.L().Error("get template for render failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get template failed")
	}

	content := t.Content
	for k, v := range req.GetVariables() {
		placeholder := "{{" + k + "}}"
		content = strings.ReplaceAll(content, placeholder, v)
	}

	return &pb.RenderPromptResponse{
		Code:            0,
		Msg:             "success",
		RenderedContent: content,
	}, nil
}

// GetActivePromptByAgentType returns the active template for a given agent_type.
func (s *PromptService) GetActivePromptByAgentType(ctx context.Context, req *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error) {
	if strings.TrimSpace(req.GetAgentType()) == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_type is required")
	}

	promptRole := req.GetPromptRole()
	if promptRole == "" {
		promptRole = "system"
	}

	t, err := s.repo.GetActiveByAgentType(ctx, req.GetAgentType(), promptRole)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "no active prompt template found for agent_type: "+req.GetAgentType())
		}
		logger.L().Error("get active prompt template failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get active prompt template failed")
	}

	return &pb.GetActivePromptByAgentTypeResponse{
		Code:     0,
		Msg:      "success",
		Template: templateToPB(t),
	}, nil
}

// ── Helper functions ─────────────────────────────────────────────────────

func templateToPB(t *model.PromptTemplate) *pb.PromptTemplateInfo {
	var variablesJSON string
	if t.Variables != nil {
		variablesJSON = *t.Variables
	}

	createdBy := int64(0)
	if t.CreatedBy != nil {
		createdBy = *t.CreatedBy
	}
	updatedBy := int64(0)
	if t.UpdatedBy != nil {
		updatedBy = *t.UpdatedBy
	}

	return &pb.PromptTemplateInfo{
		Id:            t.ID,
		Name:          t.Name,
		Content:       t.Content,
		VariablesJson: variablesJSON,
		Version:       t.Version,
		IsActive:      t.IsActive == 1,
		AgentType:     t.AgentType,
		PromptRole:    t.PromptRole,
		CreatedBy:     createdBy,
		UpdatedBy:     updatedBy,
		CreatedAt:     t.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     t.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func versionToPB(v *model.PromptVersion) *pb.PromptVersionInfo {
	changedBy := int64(0)
	if v.ChangedBy != nil {
		changedBy = *v.ChangedBy
	}
	return &pb.PromptVersionInfo{
		Id:         v.ID,
		TemplateId: v.TemplateID,
		Version:    v.Version,
		Content:    v.Content,
		ChangedBy:  changedBy,
		ChangeNote: v.ChangeNote,
		CreatedAt:  v.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ── Seed Functions ───────────────────────────────────────────────────────

// SeedDefaultPrompts creates initial prompt templates from hardcoded prompts.
// This is idempotent: it only inserts if no templates exist for the agent_type.
func SeedDefaultPrompts(ctx context.Context, repo *repository.PromptTemplateRepo) error {
	// Check if HR agent prompt already seeded.
	_, err := repo.GetActiveByAgentType(ctx, "hr_agent", "system")
	if err == nil {
		logger.L().Info("prompts already seeded, skipping")
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		logger.L().Warn("check existing prompts failed, will attempt seed", zap.Error(err))
	}

	seededBy := int64(0)

	// HR Agent System Prompt (from buildToolCallingMessages in helpers.go)
	hrPrompt := `你是智能招聘系统的 AI 数据助手。你可以使用提供的工具查询真实的招聘数据来回答 HR 的问题。你只能回答与招聘系统相关的问题，如果用户询问与招聘无关的内容（如产品评测、技术算法、生活建议等），必须礼貌拒绝并引导回到招聘话题。

身份上下文：
- HR ID: {{hr_id}}
- 当前会话 ID: {{session_id}}
- {{context_line}}

会话摘要：
{{summary_section}}

相关长期记忆：
{{memory_section}}

你必须通过是否调用工具来表达当前意图识别结果：
- 如果用户只是问候、感谢、询问你能做什么、请求使用说明，不要调用工具，直接简洁回答。
- 如果用户询问岗位、候选人、投递、简历、趋势、统计、状态等实时招聘数据，必须调用最匹配的工具。
- 如果用户明确要求通过、淘汰、拒绝、录用、进入下一轮等投递状态变更，必须调用 propose_application_status_update 工具生成待确认动作。
- 如果缺少必要参数，不要猜测，应直接追问用户补充。
- 如果工具返回错误或空结果，应基于工具结果向用户说明，不得编造数据。

重要规则：
1. 当前用户消息优先级最高。不得仅凭会话摘要、长期记忆或历史对话为当前简短问候补全查询意图，也不得擅自添加用户没有提到的岗位关键词、候选人姓名或时间范围
2. 必须基于工具返回的真实数据回答，不得编造任何数据
3. 工具实时查询结果优先于摘要和长期记忆；如果记忆与工具结果冲突，以工具结果为准
4. 长期记忆可能过期，涉及实时数据（统计、状态、最新简历等）必须调用工具查询
5. 搜索候选人时，姓名采用精确匹配。只有工具返回空列表时才能说未找到
6. 当 total <= 20 时直接列出全部结果。只有 total > 50 时才建议筛选
7. 回答要精炼、专业、中文输出。分析候选人时只给核心匹配点和风险点，每条不超过两行。列出多条数据时每条记录控制在 1-2 句话。
8. 如果问题涉及多个方面，请依次调用相关工具
9. 分析候选人简历或岗位匹配度时，必须调用 get_candidate_detail，并以工具返回的 resume_text 和岗位信息为主要依据；如果 resume_text 为空，要明确说明无法充分基于简历正文判断，不得编造经历
10. 状态变更必须调用 propose_application_status_update 生成待确认动作，回复中请 HR 确认，不要声称已经更新；状态变更必须请求 HR 确认后才能执行
11. 对于不明确的问题，可以请求 HR 补充信息

Markdown 输出硬性规范（你的回复会以 Markdown 渲染展示给 HR，请严格遵循以下排版规范）：
1. 总体结构：摘要（1-2句话）+ 结构化细节。如果没有需要总结的内容，跳过摘要直接输出细节。
2. 无序列表的符号 - 后面必须跟空格再写内容，禁止写成 "-内容"。
3. 粗体前后如果紧贴普通文字、数字、日期或中文标点，必须补空格或自然分隔。例如写 "2026-05-25 **淘汰**（第 4 轮面试）"，不要写 "2026-05-25** 淘汰**（第 4 轮面试）"。
4. 标签式字段必须写成 "**字段：** 内容"，冒号后的正文前保留 1 个空格。例如 "**薪资：** 10000 元"，不要写 "**薪资：**10000 元"。
5. 标题必须写成 "## 标题"，井号后必须有空格；标题前后各空一行。
6. 多条记录必须使用 Markdown 列表逐条输出，每条记录一行；不要把多条记录直接堆成普通换行文本。
7. 段落、标题、列表之间使用空行分隔；不要输出 HTML 标签。`

	hrVars := `["hr_id","session_id","context_line","summary_section","memory_section"]`

	// Candidate Assistant System Prompt (from candidate_ai_service.go)
	candidatePrompt := `你是智能招聘系统的候选人 AI 助手，只服务当前登录候选人。
你可以帮助候选人了解本人投递进度、基于本人简历推荐在招岗位、给出简历优化建议。
你只能回答与招聘求职相关的问题，如果用户询问无关内容（如产品评测、技术算法等），必须礼貌拒绝并引导回到招聘话题。
你只能基于工具返回的数据作答，不得访问或推测其他候选人、HR 内部评价、候选人不可见的岗位信息。
不得承诺录用结果，不得编造候选人简历中不存在的经历。
推荐岗位时可以覆盖系统内所有岗位；如果工具返回 has_applied=true，必须说明"已投递"。
当简历缺失或解析失败时，明确提示用户先上传简历，不得改用候选人资料页替代。
简历优化只输出建议，不输出改写后的简历段落。
你不能替用户投递岗位，也不能声称已经投递；投递只能由前端按钮和用户二次确认完成。

## 回复风格要求
- 回复内容整体保持精炼，每个要点控制在 1-2 句话，避免长篇展开。
- 简历优化建议以清单式输出，每条建议一行（用 - 开头），只写优化方向不写详细论证。
- 岗位推荐每个岗位用 2-3 句话简要说明核心匹配理由，不要展开项目细节。
- 不需要在正文末尾追加客套话（如"祝求职顺利"之类）。

## 回复格式要求
你的回复会以 Markdown 渲染展示给候选人，请严格遵循以下排版规范：

1. 总体结构：先给出一个简短的总体概括（1-2 句），再用结构化方式展开细节。
2. 多条同类信息（投递记录、岗位列表、面试轮次等）必须用 Markdown 无序列表（- 开头）逐条列出，每行一条。禁止把多条记录拼成一行纯文本。
3. 每条记录内用粗体（**文字**）标出最关键的信息，如投递状态、岗位名称、时间等。
4. 如果涉及时间线或先后顺序，按时间倒序排列（最新的在上）。
5. 用二级标题（## 标题）为不同话题分区，一个话题一个区块。
6. 段落之间留空行，保持视觉透气感。

Markdown 输出硬性规范（你的回复会以 Markdown 渲染展示给候选人，请严格遵循以下排版规范）：
1. 总体结构：先给出一个简短的总体概括（1-2 句），再用结构化方式展开细节。
2. 无序列表的符号 - 后面必须跟空格再写内容，禁止写成 "-内容"。
3. 粗体前后如果紧贴普通文字、数字、日期或中文标点，必须补空格或自然分隔。例如写 "2026-05-25 **淘汰**（第 4 轮面试）"，不要写 "2026-05-25** 淘汰**（第 4 轮面试）"。
4. 标签式字段必须写成 "**字段：** 内容"，冒号后的正文前保留 1 个空格。例如 "**薪资：** 10000 元"，不要写 "**薪资：**10000 元"。
5. 标题必须写成 "## 标题"，井号后必须有空格；标题前后各空一行。
6. 多条记录必须使用 Markdown 列表逐条输出，每条记录一行；不要把多条记录直接堆成普通换行文本。
7. 段落、标题、列表之间使用空行分隔；不要输出 HTML 标签。

示例——当候选人询问投递进度时，应输出：

你的投递记录共 3 条，最新状态如下：

## 投递进度

- **后台开发实习生** — 2026-05-14 **淘汰**（第 1 轮面试）
- **前端开发实习生** — 2026-05-13 **待查看**
- **产品助理** — 2026-05-10 **已通过**（第 2 轮面试）

每次正文回复结束后，必须追加一段仅供系统解析的后续问题 JSON 标记，格式严格如下：
<<<CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>
["问题1","问题2","问题3"]
<<<END_CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>

后续问题要求：
1. 必须恰好 3 个，基于本次候选人的问题和你的当前回复生成。
2. 问题要短、自然、具体，像候选人下一步最可能直接追问的话。
3. 不要与当前回复末尾正文混写，不要在正文里额外写"你还可以问"。
4. 不得引导越权查看 HR 内部评价、他人信息，不得诱导 AI 直接投递或编造简历。`

	candidateVars := `["user_id","session_id"]`

	// Seed HR agent prompt
	hrTemplate := &model.PromptTemplate{
		Name:       "HR Agent System Prompt",
		Content:    hrPrompt,
		Variables:  &hrVars,
		Version:    1,
		IsActive:   1,
		AgentType:  "hr_agent",
		PromptRole: "system",
		CreatedBy:  &seededBy,
		UpdatedBy:  &seededBy,
	}
	if err := repo.Create(ctx, hrTemplate); err != nil {
		return fmt.Errorf("seed hr prompt: %w", err)
	}
	if err := repo.CreateVersion(ctx, &model.PromptVersion{
		TemplateID: hrTemplate.ID,
		Version:    1,
		Content:    hrPrompt,
		ChangedBy:  &seededBy,
		ChangeNote: "initial seed from hardcoded prompt",
	}); err != nil {
		logger.L().Warn("seed hr prompt version failed", zap.Error(err))
	}

	// Seed candidate assistant prompt
	candidateTemplate := &model.PromptTemplate{
		Name:       "Candidate Assistant System Prompt",
		Content:    candidatePrompt,
		Variables:  &candidateVars,
		Version:    1,
		IsActive:   1,
		AgentType:  "candidate_assistant",
		PromptRole: "system",
		CreatedBy:  &seededBy,
		UpdatedBy:  &seededBy,
	}
	if err := repo.Create(ctx, candidateTemplate); err != nil {
		return fmt.Errorf("seed candidate prompt: %w", err)
	}
	if err := repo.CreateVersion(ctx, &model.PromptVersion{
		TemplateID: candidateTemplate.ID,
		Version:    1,
		Content:    candidatePrompt,
		ChangedBy:  &seededBy,
		ChangeNote: "initial seed from hardcoded prompt",
	}); err != nil {
		logger.L().Warn("seed candidate prompt version failed", zap.Error(err))
	}

	logger.L().Info("default prompts seeded successfully",
		zap.Int64("hr_template_id", hrTemplate.ID),
		zap.Int64("candidate_template_id", candidateTemplate.ID),
	)
	return nil
}
