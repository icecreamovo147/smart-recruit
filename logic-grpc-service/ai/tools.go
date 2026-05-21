package ai

import "github.com/cloudwego/eino/schema"

func RecruitingTools() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		{
			Name: "query_total_applications",
			Desc: "查询当前 HR 所有岗位的累计投递总数",
		},
		{
			Name: "query_today_applications",
			Desc: "查询今日新增投递数",
		},
		{
			Name: "get_job_heat_ranking",
			Desc: "查询投递数最高的岗位排行",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"top_n": {
					Type:     schema.Integer,
					Desc:     "返回前几名，默认 5",
					Required: false,
				},
			}),
		},
		{
			Name: "search_candidates",
			Desc: "搜索候选人投递记录。姓名精确匹配，电话和岗位名称支持模糊匹配。如果未找到精确匹配的姓名，会尝试按电话或岗位名称搜索。只有工具返回空列表时才能说未找到",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"keyword": {
					Type:     schema.String,
					Desc:     "搜索关键词，匹配候选人姓名、电话或岗位名称",
					Required: true,
				},
			}),
		},
		{
			Name: "get_job_detail",
			Desc: "查询指定岗位的完整信息、岗位要求以及该岗位的投递状态分布。用户询问某个岗位详情、要求、薪资、状态或该岗位概况时调用",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"job_id": {
					Type:     schema.Integer,
					Desc:     "岗位 ID",
					Required: true,
				},
			}),
		},
		{
			Name: "search_jobs",
			Desc: "按关键词和上下架状态搜索当前 HR 发布的岗位。用户提到岗位名称、部门、地点，或需要先定位岗位 ID 时调用",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"keyword": {
					Type:     schema.String,
					Desc:     "搜索关键词，可匹配岗位名称、部门、地点、描述或要求；为空时列出岗位",
					Required: false,
				},
				"status": {
					Type:     schema.Integer,
					Desc:     "岗位状态：1 招募中，0 已下架；不传则不限状态",
					Required: false,
				},
				"page": {
					Type:     schema.Integer,
					Desc:     "页码，从 1 开始，默认 1",
					Required: false,
				},
				"page_size": {
					Type:     schema.Integer,
					Desc:     "每页数量，默认 10，最多 50",
					Required: false,
				},
			}),
		},
		{
			Name: "get_candidate_detail",
			Desc: "获取指定投递记录的完整信息，包括岗位信息和候选人上传简历的解析正文，用于简历分析。分析候选人匹配度时必须优先调用此工具，并以 resume_text 为主要依据；不要使用候选人资料页字段补充判断",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"application_id": {
					Type:     schema.Integer,
					Desc:     "投递记录 ID",
					Required: true,
				},
			}),
		},
		{
			Name: "propose_application_status_update",
			Desc: "当 HR 明确要求将某个投递标记为通过或淘汰时调用。此工具只生成待确认动作，不会直接修改数据库；最终回复必须请求 HR 确认",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"application_id": {
					Type:     schema.Integer,
					Desc:     "投递记录 ID",
					Required: true,
				},
				"status": {
					Type:     schema.Integer,
					Desc:     "目标状态：2 表示通过，3 表示淘汰",
					Required: true,
				},
			}),
		},
		{
			Name: "list_all_applications",
			Desc: "分页列出当前 HR 所有岗位下的全部投递候选人，用于浏览整体候选人情况",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"page": {
					Type:     schema.Integer,
					Desc:     "页码，从 1 开始，默认 1",
					Required: false,
				},
				"page_size": {
					Type:     schema.Integer,
					Desc:     "每页数量，默认 10",
					Required: false,
				},
			}),
		},
		{
			Name: "list_applications_by_job",
			Desc: "分页列出指定岗位下的投递候选人，可按投递状态筛选。用户询问某岗位有哪些候选人、待查看/通过/淘汰名单时调用",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"job_id": {
					Type:     schema.Integer,
					Desc:     "岗位 ID",
					Required: true,
				},
				"status": {
					Type:     schema.Integer,
					Desc:     "投递状态：0 待查看，1 已查看，2 通过，3 淘汰；不传则不限状态",
					Required: false,
				},
				"current_only": {
					Type:     schema.Boolean,
					Desc:     "是否只看当前有效投递，默认 true",
					Required: false,
				},
				"page": {
					Type:     schema.Integer,
					Desc:     "页码，从 1 开始，默认 1",
					Required: false,
				},
				"page_size": {
					Type:     schema.Integer,
					Desc:     "每页数量，默认 10，最多 50",
					Required: false,
				},
			}),
		},
		{
			Name: "list_applications_by_status",
			Desc: "分页列出当前 HR 所有岗位下某个状态的投递候选人，也可限定岗位。用户询问待处理、已通过、已淘汰、已查看候选人名单时调用",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"status": {
					Type:     schema.Integer,
					Desc:     "投递状态：0 待查看，1 已查看，2 通过，3 淘汰",
					Required: true,
				},
				"job_id": {
					Type:     schema.Integer,
					Desc:     "岗位 ID；不传则查询所有岗位",
					Required: false,
				},
				"current_only": {
					Type:     schema.Boolean,
					Desc:     "是否只看当前有效投递，默认 true",
					Required: false,
				},
				"page": {
					Type:     schema.Integer,
					Desc:     "页码，从 1 开始，默认 1",
					Required: false,
				},
				"page_size": {
					Type:     schema.Integer,
					Desc:     "每页数量，默认 10，最多 50",
					Required: false,
				},
			}),
		},
		{
			Name: "get_application_status_summary",
			Desc: "查询投递状态分布统计，可按岗位限定。用户询问待查看、已查看、通过、淘汰各有多少人，或招聘漏斗概况时调用",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"job_id": {
					Type:     schema.Integer,
					Desc:     "岗位 ID；不传则统计所有岗位",
					Required: false,
				},
			}),
		},
		{
			Name: "get_application_trend",
			Desc: "查询近 N 天投递趋势，可按岗位限定。用户询问最近几天、近一周、近一个月每天投递变化时调用",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"days": {
					Type:     schema.Integer,
					Desc:     "最近天数，默认 7，最多 90",
					Required: false,
				},
				"job_id": {
					Type:     schema.Integer,
					Desc:     "岗位 ID；不传则统计所有岗位",
					Required: false,
				},
			}),
		},
		{
			Name: "get_job_list",
			Desc: "查询当前 HR 发布的所有在招岗位列表",
		},
	}
}
