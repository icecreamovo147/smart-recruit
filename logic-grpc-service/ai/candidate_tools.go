package ai

import "github.com/cloudwego/eino/schema"

// CandidateTools returns the tool definitions available to the candidate AI assistant.
func CandidateTools() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		{
			Name: "list_my_applications",
			Desc: "查询当前候选人自己的投递列表和状态。返回每条投递的岗位名称、状态、投递时间、轮次等信息",
		},
		{
			Name: "get_my_application_detail",
			Desc: "查询某条本人投递的详细进度，包括岗位信息和当前状态",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"application_id": {
					Type:     schema.Integer,
					Desc:     "投递记录 ID",
					Required: true,
				},
			}),
		},
		{
			Name: "get_my_resume_text",
			Desc: "获取当前候选人有效上传简历的解析文本。如果候选人没有上传简历或简历解析为空，会返回提示信息",
		},
		{
			Name: "list_jobs_for_recommendation",
			Desc: "查询系统内岗位列表，并标记当前候选人是否已投递每个岗位。返回岗位 ID、名称、部门、地点、薪资、招募状态、has_applied 字段",
		},
		{
			Name: "get_job_detail_for_candidate",
			Desc: "查询指定岗位的详情，并标记当前候选人是否已投递该岗位",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"job_id": {
					Type:     schema.Integer,
					Desc:     "岗位 ID",
					Required: true,
				},
			}),
		},
		{
			Name: "recommend_jobs_by_resume",
			Desc: "基于当前候选人上传简历的解析文本和系统内所有岗位，生成 3-5 个推荐岗位，包含匹配理由、不足点和建议投递优先级。如果候选人没有上传简历或简历解析无效，返回提示信息。每个推荐岗位包含 has_applied 标记",
		},
	}
}
