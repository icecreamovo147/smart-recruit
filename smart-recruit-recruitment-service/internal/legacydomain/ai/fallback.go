package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildHRFallbackReply returns a conservative HR-side answer based purely on the
// structured tool results that were already collected during this turn. It is
// intended for cases where the LLM final answer failed (timeout, empty reply,
// round-limit, budget-exhausted) but tool calls had already produced data.
//
// The reply is prefixed with a clear note that this is a fallback. It NEVER
// invents data, NEVER uses LLM common sense to fill gaps, and NEVER applies
// natural-language keyword heuristics on the user's question — it speaks only
// from the deterministic tool result map.
func BuildHRFallbackReply(traces []ToolTrace) string {
	if len(traces) == 0 {
		return "AI 服务响应失败，且本次没有查询到可用数据，请稍后重试或换一种问法。"
	}
	var b strings.Builder
	b.WriteString("AI 模型回答失败，以下基于已查询到的数据给出保守回复：\n\n")
	any := false
	for _, t := range traces {
		section := summarizeHRToolTrace(t)
		if section == "" {
			continue
		}
		any = true
		b.WriteString(section)
		b.WriteString("\n")
	}
	if !any {
		return "AI 模型回答失败，且工具结果暂无可总结的数据，请稍后重试。"
	}
	b.WriteString("\n以上为系统已查询的数据。如需更详细分析，请稍后重试。")
	return b.String()
}

// BuildCandidateFallbackReply is the candidate-side counterpart of BuildHRFallbackReply.
func BuildCandidateFallbackReply(traces []ToolTrace) string {
	if len(traces) == 0 {
		return "AI 服务响应失败，且本次没有查询到你的相关数据，请稍后重试。"
	}
	var b strings.Builder
	b.WriteString("AI 模型回答失败，以下基于已查询到的数据给出保守回复：\n\n")
	any := false
	for _, t := range traces {
		section := summarizeCandidateToolTrace(t)
		if section == "" {
			continue
		}
		any = true
		b.WriteString(section)
		b.WriteString("\n")
	}
	if !any {
		return "AI 模型回答失败，且工具结果暂无可总结的数据，请稍后重试。"
	}
	b.WriteString("\n以上为系统已查询的数据。如需更详细分析，请稍后重试。")
	return b.String()
}

func summarizeHRToolTrace(t ToolTrace) string {
	if t.Error != nil {
		return fmt.Sprintf("- %s: 工具调用失败（%s），数据查询失败。", t.ToolName, t.Error.Error())
	}
	switch t.ToolName {
	case "query_total_applications":
		return formatScalarTool(t.Result, "total_applications", "累计投递总数")
	case "query_today_applications":
		return formatScalarTool(t.Result, "today_applications", "今日新增投递数")
	case "get_job_heat_ranking":
		return formatHotJobs(t.Result)
	case "get_application_status_summary":
		return formatStatusSummary(t.Result)
	case "get_application_trend":
		return formatApplicationTrend(t.Result)
	case "get_job_list":
		return formatJobList(t.Result, "jobs")
	case "search_jobs":
		return formatJobList(t.Result, "jobs")
	case "list_all_applications", "list_applications_by_job", "list_applications_by_status":
		return formatApplicationList(t.Result)
	case "search_candidates":
		return formatSearchCandidates(t.Result)
	case "propose_application_status_update":
		return formatProposeAction(t.Result)
	case "get_candidate_detail", "get_job_detail":
		return formatGenericKV(t.ToolName, t.Result)
	default:
		return ""
	}
}

func summarizeCandidateToolTrace(t ToolTrace) string {
	if t.Error != nil {
		return fmt.Sprintf("- %s: 工具调用失败（%s），数据查询失败。", t.ToolName, t.Error.Error())
	}
	switch t.ToolName {
	case "list_my_applications":
		return formatMyApplications(t.Result)
	case "get_my_application_detail":
		return formatGenericKV("我的投递", t.Result)
	case "get_my_resume_text":
		return formatMyResumeStatus(t.Result)
	case "list_jobs_for_recommendation":
		return formatJobList(t.Result, "jobs")
	case "get_job_detail_for_candidate":
		return formatGenericKV("岗位", t.Result)
	case "recommend_jobs_by_resume":
		return "- 已根据简历查询到候选岗位列表。AI 推荐生成失败，请稍后重试。"
	default:
		return ""
	}
}

// ---- formatters ----

func formatScalarTool(raw, key, label string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	v, ok := data[key]
	if !ok {
		return ""
	}
	switch n := v.(type) {
	case float64:
		return fmt.Sprintf("- %s：%d", label, int64(n))
	case int64:
		return fmt.Sprintf("- %s：%d", label, n)
	case int:
		return fmt.Sprintf("- %s：%d", label, n)
	}
	return ""
}

func formatHotJobs(raw string) string {
	var data struct {
		HotJobs []struct {
			Title string `json:"title"`
			Total int64  `json:"total"`
		} `json:"hot_jobs"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if len(data.HotJobs) == 0 {
		return "- 热门岗位排行：暂无数据。"
	}
	var b strings.Builder
	b.WriteString("- 热门岗位排行：")
	parts := make([]string, 0, len(data.HotJobs))
	for _, j := range data.HotJobs {
		parts = append(parts, fmt.Sprintf("%s（%d 人）", j.Title, j.Total))
	}
	b.WriteString(strings.Join(parts, "、"))
	return b.String()
}

func formatStatusSummary(raw string) string {
	var data struct {
		JobID  int64 `json:"job_id"`
		Counts []struct {
			StatusText string `json:"status_text"`
			Total      int64  `json:"total"`
		} `json:"counts"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if len(data.Counts) == 0 {
		return "- 投递状态分布：暂无投递记录。"
	}
	parts := make([]string, 0, len(data.Counts))
	for _, c := range data.Counts {
		parts = append(parts, fmt.Sprintf("%s %d", c.StatusText, c.Total))
	}
	prefix := "投递状态分布"
	if data.JobID > 0 {
		prefix = fmt.Sprintf("岗位 %d 状态分布", data.JobID)
	}
	return fmt.Sprintf("- %s：%s", prefix, strings.Join(parts, "、"))
}

func formatApplicationTrend(raw string) string {
	var data struct {
		Days  int   `json:"days"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	return fmt.Sprintf("- 近 %d 天投递总数：%d", data.Days, data.Total)
}

func formatJobList(raw, key string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	jobsRaw, ok := data[key].([]any)
	if !ok || len(jobsRaw) == 0 {
		return "- 岗位列表：未查询到岗位。"
	}
	var b strings.Builder
	b.WriteString("- 岗位列表（前几条）：")
	max := 5
	if len(jobsRaw) < max {
		max = len(jobsRaw)
	}
	parts := make([]string, 0, max)
	for i := 0; i < max; i++ {
		j, ok := jobsRaw[i].(map[string]any)
		if !ok {
			continue
		}
		title, _ := j["title"].(string)
		dept, _ := j["department"].(string)
		if title == "" {
			continue
		}
		if dept != "" {
			parts = append(parts, fmt.Sprintf("%s（%s）", title, dept))
		} else {
			parts = append(parts, title)
		}
	}
	b.WriteString(strings.Join(parts, "、"))
	if len(jobsRaw) > max {
		b.WriteString(fmt.Sprintf("，共 %d 条", len(jobsRaw)))
	}
	return b.String()
}

func formatApplicationList(raw string) string {
	var data struct {
		Total      int64 `json:"total"`
		Candidates []struct {
			RealName   string `json:"real_name"`
			JobTitle   string `json:"job_title"`
			StatusText string `json:"status_text"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if len(data.Candidates) == 0 {
		return fmt.Sprintf("- 投递列表：未查询到符合条件的投递记录（共 %d 条）。", data.Total)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- 投递列表（共 %d 条，前 %d 条）：\n", data.Total, len(data.Candidates)))
	for _, c := range data.Candidates {
		name := c.RealName
		if name == "" {
			name = "候选人"
		}
		b.WriteString(fmt.Sprintf("  - %s · %s · %s\n", name, c.JobTitle, c.StatusText))
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatSearchCandidates(raw string) string {
	var data struct {
		Candidates []struct {
			RealName string `json:"real_name"`
			JobTitle string `json:"job_title"`
			Status   string `json:"status"`
		} `json:"candidates"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if len(data.Candidates) == 0 {
		if data.Message != "" {
			return "- 候选人搜索：" + data.Message
		}
		return "- 候选人搜索：未找到匹配的候选人。"
	}
	var b strings.Builder
	b.WriteString("- 候选人搜索结果：\n")
	for _, c := range data.Candidates {
		b.WriteString(fmt.Sprintf("  - %s · %s · %s\n", c.RealName, c.JobTitle, c.Status))
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatMyApplications(raw string) string {
	var data struct {
		Applications []struct {
			JobTitle   string `json:"job_title"`
			StatusText string `json:"status_text"`
			RoundNo    int32  `json:"round_no"`
			AppliedAt  string `json:"applied_at"`
		} `json:"applications"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if len(data.Applications) == 0 {
		if data.Message != "" {
			return "- " + data.Message
		}
		return "- 你目前还没有投递记录。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- 你的投递记录（共 %d 条）：\n", len(data.Applications)))
	for _, a := range data.Applications {
		b.WriteString(fmt.Sprintf("  - %s · %s · 第 %d 轮 · %s\n", a.JobTitle, a.StatusText, a.RoundNo, a.AppliedAt))
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatMyResumeStatus(raw string) string {
	var data struct {
		Available bool   `json:"resume_available"`
		FileName  string `json:"file_name"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if !data.Available {
		if data.Message != "" {
			return "- " + data.Message
		}
		return "- 你的简历当前不可用，请上传或重新上传简历。"
	}
	if data.FileName != "" {
		return fmt.Sprintf("- 已读取简历《%s》，可用于后续分析。", data.FileName)
	}
	return "- 你的简历可用。"
}

func formatProposeAction(raw string) string {
	var data struct {
		Action        string `json:"action"`
		ApplicationID int64  `json:"application_id"`
		ActionStatus  int32  `json:"action_status"`
		CandidateName string `json:"candidate_name"`
		JobTitle      string `json:"job_title"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if data.ApplicationID <= 0 {
		return ""
	}
	verb := "更新状态"
	switch data.ActionStatus {
	case 2:
		verb = "通过"
	case 3:
		verb = "淘汰"
	}
	return fmt.Sprintf("- 已生成待确认动作：将投递 %d（%s · %s）%s。当前工具不会直接修改数据库，请向系统按钮二次确认后再执行。",
		data.ApplicationID, data.CandidateName, data.JobTitle, verb)
}

func formatGenericKV(label, raw string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ""
	}
	if msg, ok := data["error"].(string); ok && msg != "" {
		return fmt.Sprintf("- %s：%s", label, msg)
	}
	keys := []string{"title", "job_title", "candidate_name", "status_text", "department", "location", "salary_range"}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := data[k].(string); ok && v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("- %s：%s", label, strings.Join(parts, " · "))
}
