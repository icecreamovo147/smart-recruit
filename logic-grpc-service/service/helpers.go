package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"logic-grpc-service/ai"
	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
	"logic-grpc-service/resumeparser"
)

// wrapAIError wraps an AI client error as a gRPC Internal status. It encodes the
// AIError type into the message as "ai:<TYPE>: <user-facing message>" so the
// gateway can route to the correct user prompt without re-classifying.
//
// Pre-existing gRPC status errors are returned as-is. Pure context.Canceled errors
// are also returned as-is so callers can detect user-initiated abort and avoid
// firing error toasts (per plan: 用户取消不是系统错误).
func wrapAIError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	aiErr := ai.ClassifyAIError(err)
	if aiErr.Type == ai.AIContextCanceled {
		return err
	}
	return status.Errorf(codes.Internal, "ai:%s: %s", aiErr.Type, aiErr.UserMessage)
}

// wrapOSSError wraps an OSS client error as a gRPC Internal status with "oss:" prefix.
func wrapOSSError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Errorf(codes.Internal, "oss: %v", err)
}

// ---- Pagination ----

func page(value int32) int32 {
	if value < 1 {
		return 1
	}
	return value
}

func pageSize(value int32) int32 {
	if value < 1 || value > 100 {
		return 10
	}
	return value
}

// ---- Time ----

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ---- Validation ----

func allNotEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

var allowedExtensions = map[string]bool{
	"pdf":  true,
	"docx": true,
}

func allowedFileType(fileName, fileType string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if fileType != "" && ext != strings.ToLower(fileType) {
		return false
	}
	return allowedExtensions[ext]
}

func allowedFileTypesText() string {
	return "PDF、DOCX"
}

// ---- String helpers ----

func splitSkills(skills string) []string {
	parts := strings.Split(skills, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func put(fields map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		fields[key] = value
	}
}

func sanitizeFileName(fileName string) string {
	base := filepath.Base(fileName)
	if base == "." || base == "/" {
		return uuid.NewString()
	}
	return strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(base)
}

func limitSessionTitle(title string) string {
	title = strings.TrimSpace(title)
	runes := []rune(title)
	if len(runes) <= 60 {
		return title
	}
	return string(runes[:60])
}

// ---- Application status ----

func applicationStatusText(status int32) string {
	switch status {
	case 0:
		return "待查看"
	case 1:
		return "已查看"
	case 2:
		return "通过"
	case 3:
		return "淘汰"
	default:
		return "未知"
	}
}

// ---- Display helpers ----

func displayCandidateName(row *repository.ApplicationDetailRow) string {
	if strings.TrimSpace(row.RealName) != "" {
		return row.RealName
	}
	return fmt.Sprintf("候选人 %d", row.UserID)
}

func candidateDisplayName(realName string, userID int64) string {
	if strings.TrimSpace(realName) != "" {
		return realName
	}
	return fmt.Sprintf("候选人 %d", userID)
}

// ---- PB Conversion helpers ----

func toPBProfile(profile *model.CandidateProfile) *pb.CandidateProfile {
	return &pb.CandidateProfile{
		RealName: profile.RealName, Phone: profile.Phone,
		Education: profile.Education, School: profile.School,
		WorkExperience: profile.WorkExperience,
		Skills:         splitSkills(profile.Skills),
		IsComplete:     profile.IsComplete == 1,
	}
}

func toPBResume(resume *model.Resume, resumeURL string) *pb.CandidateResume {
	return &pb.CandidateResume{
		ResumeId: resume.ID, FileName: resume.FileName,
		FileType: resume.FileType, FileSize: resume.FileSize,
		UploadedAt: formatTime(resume.UploadedAt), ResumeUrl: resumeURL,
	}
}

func toPBChatMessages(rows []model.AIChatHistory) []*pb.ChatMessage {
	list := make([]*pb.ChatMessage, 0, len(rows))
	for _, row := range rows {
		list = append(list, &pb.ChatMessage{Role: row.Role, Content: row.Content, CreatedAt: formatTime(row.CreatedAt)})
	}
	return list
}

func toPBChatSession(session model.AIChatSession) *pb.ChatSession {
	return &pb.ChatSession{SessionId: session.ID, Title: session.Title, ApplicationId: session.ApplicationID, CreatedAt: formatTime(session.CreatedAt), UpdatedAt: formatTime(session.UpdatedAt)}
}

func jobsToPB(ctx context.Context, jobs []model.Job, jobRepo *repository.JobRepo) []*pb.Job {
	ids := make([]int64, len(jobs))
	for i, job := range jobs {
		ids[i] = job.ID
	}
	counts, _ := jobRepo.BatchApplicationCounts(ctx, ids)
	list := make([]*pb.Job, 0, len(jobs))
	for _, job := range jobs {
		j := &pb.Job{
			JobId: job.ID, HrId: job.HrID, Title: job.Title,
			Department: job.Department, Location: job.Location,
			SalaryRange: job.SalaryRange, Description: job.Description,
			Requirements: job.Requirements, Status: job.Status,
			ApplicationCount: counts[job.ID], CreatedAt: formatTime(job.CreatedAt),
		}
		if job.DepartmentID != nil {
			j.DepartmentId = *job.DepartmentID
		}
		if job.LocationID != nil {
			j.LocationId = *job.LocationID
		}
		list = append(list, j)
	}
	return list
}

// ---- Resume text extraction (shared by CandidateService and AIService) ----

func extractAndStoreResumeText(ctx context.Context, resumeID int64, fileType, ossKey string, ossClient oss.Storage, resumeRepo *repository.ResumeRepo) (string, error) {
	start := time.Now()
	data, err := ossClient.DownloadObject(ctx, ossKey)
	if err != nil {
		return "", wrapOSSError(err)
	}
	logger.L().Info("oss resume file downloaded",
		zap.Int64("resume_id", resumeID),
		zap.String("file_type", fileType),
		zap.Int("size_bytes", len(data)),
		zap.Duration("cost", time.Since(start)),
	)

	// Validate magic bytes against claimed file type.
	if len(data) >= 4 && !resumeparser.ValidateMagicBytes(fileType, data[:min(len(data), 8)]) {
		return "", fmt.Errorf("文件头魔数与声明的格式 %s 不匹配，文件可能已损坏或扩展名被篡改", strings.ToUpper(fileType))
	}

	parser, err := resumeparser.DefaultRegistry.GetParser(fileType)
	if err != nil {
		logger.L().Info("no parser for resume format, skipping text extraction",
			zap.Int64("resume_id", resumeID),
			zap.String("file_type", fileType),
		)
		return "", fmt.Errorf("暂不支持 %s 格式的简历文档，请使用 PDF 或 DOCX 格式", strings.ToUpper(fileType))
	}

	parseStart := time.Now()
	text, err := parser.ExtractText(ctx, data)
	logger.L().Info("resume parsed",
		zap.Int64("resume_id", resumeID),
		zap.String("file_type", fileType),
		zap.Int("text_chars", len([]rune(text))),
		zap.Duration("cost", time.Since(parseStart)),
		zap.Error(err),
	)
	if strings.TrimSpace(text) == "" {
		if err == nil {
			err = fmt.Errorf("未能从 %s 文件中提取到可用文本", strings.ToUpper(fileType))
		}
		return "", err
	}
	if updateErr := resumeRepo.UpdateParsedText(ctx, resumeID, text); updateErr != nil {
		return text, updateErr
	}
	return text, err
}

// ---- Circuit breaker for resume parsing ----

// resumeParseBreaker prevents repeated OSS download + parse attempts for resumes
// that have already failed, avoiding wasted bandwidth and CPU on corrupted files.
type resumeParseBreaker struct {
	mu            sync.Mutex
	failures      map[int64]int
	cooldownUntil map[int64]time.Time
	maxRetries    int
	cooldown      time.Duration
}

func newResumeParseBreaker(maxRetries int, cooldown time.Duration) *resumeParseBreaker {
	return &resumeParseBreaker{
		failures:      make(map[int64]int),
		cooldownUntil: make(map[int64]time.Time),
		maxRetries:    maxRetries,
		cooldown:      cooldown,
	}
}

func (cb *resumeParseBreaker) allow(resumeID int64) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if until, ok := cb.cooldownUntil[resumeID]; ok && time.Now().Before(until) {
		return false
	}
	return cb.failures[resumeID] < cb.maxRetries
}

func (cb *resumeParseBreaker) recordFailure(resumeID int64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures[resumeID]++
	cb.cooldownUntil[resumeID] = time.Now().Add(cb.cooldown)
}

func (cb *resumeParseBreaker) reset(resumeID int64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	delete(cb.failures, resumeID)
	delete(cb.cooldownUntil, resumeID)
}

// Global breaker: max 3 retries per resume, 5-minute cooldown between retries.
var globalResumeBreaker = newResumeParseBreaker(3, 5*time.Minute)

// resumeTextResult holds the outcome of loading/refreshing resume text for AI analysis.
type resumeTextResult struct {
	Text string
	Note string
}

// loadOrRefreshResumeText is a pipeline stage that ensures usable resume text is available.
// It checks the cached parsed_text, re-downloads and re-parses from OSS if the cache is
// garbled or empty, and returns the best available text with a human-readable note.
func loadOrRefreshResumeText(ctx context.Context, detail *repository.ApplicationDetailRow, ossClient oss.Storage, resumeRepo *repository.ResumeRepo) resumeTextResult {
	text := detail.ParsedText
	fileLabel := strings.ToUpper(detail.FileType)
	if fileLabel == "" {
		fileLabel = "简历"
	}
	note := fmt.Sprintf("未读取到 %s 简历文本。", fileLabel)

	if strings.TrimSpace(text) != "" {
		analysisText, textStats := resumeparser.PrepareForAnalysis(text)
		if !resumeparser.IsAnalysisTextUseful(analysisText, textStats) && detail.ResumeID > 0 && strings.TrimSpace(detail.OSSKey) != "" {
			if globalResumeBreaker.allow(detail.ResumeID) {
				if fresh, err := extractAndStoreResumeText(ctx, detail.ResumeID, detail.FileType, detail.OSSKey, ossClient, resumeRepo); err == nil {
					globalResumeBreaker.reset(detail.ResumeID)
					text = fresh
					note = fmt.Sprintf("缓存的 %s 解析文本疑似乱码，已从 OSS 重新读取候选人上传的 %s 简历《%s》，提取文本约 %d 个字符。", fileLabel, fileLabel, detail.FileName, len([]rune(text)))
				} else {
					globalResumeBreaker.recordFailure(detail.ResumeID)
					text = ""
					note = fmt.Sprintf("缓存的 %s 解析文本疑似乱码，且重新解析候选人上传的 %s 简历《%s》失败：%s。", fileLabel, fileLabel, detail.FileName, err.Error())
				}
			} else {
				text = ""
				note = fmt.Sprintf("缓存的 %s 解析文本疑似乱码，且该简历近期已重试解析多次均失败，已跳过重新解析。可稍后重试。", fileLabel)
			}
		} else {
			note = fmt.Sprintf("已使用候选人上传的 %s 简历《%s》的缓存解析文本，约 %d 个字符。", fileLabel, detail.FileName, len([]rune(text)))
		}
	}

	if strings.TrimSpace(text) == "" && detail.ResumeID > 0 && strings.TrimSpace(detail.OSSKey) != "" {
		if globalResumeBreaker.allow(detail.ResumeID) {
			if fresh, err := extractAndStoreResumeText(ctx, detail.ResumeID, detail.FileType, detail.OSSKey, ossClient, resumeRepo); err == nil {
				globalResumeBreaker.reset(detail.ResumeID)
				text = fresh
				note = fmt.Sprintf("已读取候选人上传的 %s 简历《%s》，提取文本约 %d 个字符。", fileLabel, detail.FileName, len([]rune(text)))
			} else {
				globalResumeBreaker.recordFailure(detail.ResumeID)
				note = fmt.Sprintf("候选人上传了 %s 简历《%s》，但文本解析失败：%s。", fileLabel, detail.FileName, err.Error())
			}
		} else {
			note = fmt.Sprintf("候选人上传了 %s 简历《%s》，但近期已重试解析多次均失败，已跳过重新解析。可稍后重试。", fileLabel, detail.FileName)
		}
	}

	return resumeTextResult{Text: text, Note: note}
}

// prepareResumeForAI cleans and filters the raw resume text for AI consumption.
// It removes garbled/noisy lines, deduplicates, and returns the sanitized text
// with an updated note describing what was done.
func prepareResumeForAI(rawText, currentNote string) (string, string) {
	if strings.TrimSpace(rawText) == "" {
		return "", currentNote
	}
	analysisText, textStats := resumeparser.PrepareForAnalysis(rawText)
	note := currentNote
	if resumeparser.IsAnalysisTextUseful(analysisText, textStats) {
		if textStats.RemovedLines > 0 || textStats.CleanedChars < textStats.OriginalChars {
			note = fmt.Sprintf("%s 已在提交 AI 前过滤乱码/重复噪声，保留有效文本约 %d 个字符，移除疑似噪声行 %d 行。", note, textStats.CleanedChars, textStats.RemovedLines)
		}
		return analysisText, note
	}
	return "", fmt.Sprintf("%s 解析文本经过净化后仍疑似乱码或有效信息不足，已停止将残余乱码提交给 AI。", note)
}

const standardMarkdownReplyRules = `## Markdown 输出硬性规范
你的正文回复会交给标准 CommonMark/Markdown 渲染器展示，必须严格输出合法 Markdown，不要依赖前端容错修正。

1. 列表项必须写成 "- 内容"，短横线后必须有 1 个空格。禁止写成 "-内容"、"-**标题**"。
2. 粗体必须写成 "**文字**"，星号内侧不能有空格。禁止写成 "** 文字**"、"**文字 **"。
3. 粗体前后如果紧贴普通文字、数字、日期或中文标点，必须补空格或自然分隔。例如写 "2026-05-25 **淘汰**（第 4 轮面试）"，不要写 "2026-05-25** 淘汰**（第 4 轮面试）"。
4. 标签式字段必须写成 "**字段：** 内容"，冒号后的正文前保留 1 个空格。例如 "**薪资：** 10000 元"，不要写 "**薪资：**10000 元"。
5. 标题必须写成 "## 标题"，井号后必须有空格；标题前后各空一行。
6. 多条记录必须使用 Markdown 列表逐条输出，每条记录一行；不要把多条记录直接堆成普通换行文本。
7. 段落、标题、列表之间使用空行分隔；不要输出 HTML 标签。`

// buildToolCallingMessages constructs the message list for the AI with tool calling.
// It includes identity context, session summary, long-term memories, recent conversation
// history, and the current user message. The LLM will decide which tools to call.
func buildToolCallingMessages(actx *AgentContext, currentMessage string) []*schema.Message {
	contextLine := "当前没有绑定特定投递记录。"
	if actx.ApplicationID > 0 {
		contextLine = fmt.Sprintf("当前会话绑定的投递记录 ID 是 %d。涉及该候选人的简历分析、匹配度追问、通过/淘汰等问题时，应优先使用这个 application_id 调用工具。", actx.ApplicationID)
	}

	// Build summary section.
	summarySection := "暂无会话摘要。"
	if actx.SessionSummary != "" {
		summarySection = actx.SessionSummary
	}

	// Build long-term memory section.
	memorySection := "暂无相关长期记忆。"
	if len(actx.LongTermMemories) > 0 {
		lines := make([]string, 0, len(actx.LongTermMemories))
		for i, m := range actx.LongTermMemories {
			lines = append(lines, fmt.Sprintf("%d. [%s/%s] %s", i+1, m.ScopeType, m.MemoryType, m.Content))
		}
		memorySection = strings.Join(lines, "\n")
	}

	systemPrompt := "你是智能招聘系统的 AI 数据助手。你可以使用提供的工具查询真实的招聘数据来回答 HR 的问题。你只能回答与招聘系统相关的问题，如果用户询问与招聘无关的内容（如产品评测、技术算法、生活建议等），必须礼貌拒绝并引导回到招聘话题。\n\n" +
		fmt.Sprintf("身份上下文：\n- HR ID: %d\n- 当前会话 ID: %d\n- %s\n\n", actx.HrID, actx.SessionID, contextLine) +
		"会话摘要：\n" + summarySection + "\n\n" +
		"相关长期记忆：\n" + memorySection + "\n\n" +
		"你必须通过是否调用工具来表达当前意图识别结果：\n" +
		"- 如果用户只是问候、感谢、询问你能做什么、请求使用说明，不要调用工具，直接简洁回答。\n" +
		"- 如果用户询问岗位、候选人、投递、简历、趋势、统计、状态等实时招聘数据，必须调用最匹配的工具。\n" +
		"- 如果用户明确要求通过、淘汰、拒绝、录用、进入下一轮等投递状态变更，必须调用 propose_application_status_update 工具生成待确认动作。\n" +
		"- 如果缺少必要参数，不要猜测，应直接追问用户补充。\n" +
		"- 如果工具返回错误或空结果，应基于工具结果向用户说明，不得编造数据。\n\n" +
		"重要规则：\n" +
		"1. 当前用户消息优先级最高。不得仅凭会话摘要、长期记忆或历史对话为当前简短问候补全查询意图，也不得擅自添加用户没有提到的岗位关键词、候选人姓名或时间范围\n" +
		"2. 必须基于工具返回的真实数据回答，不得编造任何数据\n" +
		"3. 工具实时查询结果优先于摘要和长期记忆；如果记忆与工具结果冲突，以工具结果为准\n" +
		"4. 长期记忆可能过期，涉及实时数据（统计、状态、最新简历等）必须调用工具查询\n" +
		"5. 搜索候选人时，姓名采用精确匹配。只有工具返回空列表时才能说未找到\n" +
		"6. 当 total <= 20 时直接列出全部结果。只有 total > 50 时才建议筛选\n" +
		"7. 回答要精炼、专业、中文输出。分析候选人时只给核心匹配点和风险点，每条不超过两行。列出多条数据时每条记录控制在 1-2 句话。\n" +
		"8. 如果问题涉及多个方面，请依次调用相关工具\n" +
		"9. 分析候选人简历或岗位匹配度时，必须调用 get_candidate_detail，并以工具返回的 resume_text 和岗位信息为主要依据；如果 resume_text 为空，要明确说明无法充分基于简历正文判断，不得编造经历\n" +
		"10. 状态变更必须调用 propose_application_status_update 生成待确认动作，回复中请 HR 确认，不要声称已经更新；状态变更必须请求 HR 确认后才能执行\n" +
		"11. 对于不明确的问题，可以请求 HR 补充信息\n\n" +
		standardMarkdownReplyRules

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
	}
	for _, h := range actx.RecentMessages {
		role := schema.Assistant
		if h.Role == "user" {
			role = schema.User
		}
		messages = append(messages, &schema.Message{Role: role, Content: h.Content})
	}
	if !currentMessageAlreadyPersisted(actx, currentMessage) {
		messages = append(messages, schema.UserMessage(currentMessage))
	}
	return messages
}
