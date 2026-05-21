package main

// @title           智能招聘系统 API
// @version         1.0
// @description     智能招聘平台后端接口文档，包含候选人端和 HR 管理端接口。
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer {JWT Token}

// ---- Auth ----

// @Summary      用户注册
// @Description  注册候选人(role=1)或HR(role=2)账号
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  object{username=string,password=string,role=int,email=string}  true  "注册信息"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/auth/register [post]
func _swag_1() {}

// @Summary      用户登录
// @Description  使用用户名密码登录，返回JWT Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  object{username=string,password=string}  true  "登录凭据"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/auth/login [post]
func _swag_2() {}

// ---- Public Jobs ----

// @Summary      浏览岗位列表
// @Description  分页浏览在招岗位，支持关键词搜索
// @Tags         岗位
// @Accept       json
// @Produce      json
// @Param        page      query  int     false  "页码"   default(1)
// @Param        page_size query  int     false  "每页数量" default(10)
// @Param        keyword   query  string  false  "搜索关键词"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/jobs [get]
func _swag_3() {}

// @Summary      查看岗位详情
// @Description  根据岗位ID查看详细信息
// @Tags         岗位
// @Accept       json
// @Produce      json
// @Param        job_id  path  int  true  "岗位ID"
// @Success      200     {object}  map[string]interface{}
// @Router       /api/v1/jobs/{job_id} [get]
func _swag_4() {}

// ---- HR Jobs ----

// @Summary      新增岗位
// @Description  HR创建新的招聘岗位
// @Tags         HR-岗位管理
// @Accept       json
// @Produce      json
// @Param        body  body  object{title=string,department=string,location=string,salary_range=string,description=string,requirements=string}  true  "岗位信息"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/jobs [post]
// @Security     BearerAuth
func _swag_5() {}

// @Summary      编辑岗位
// @Description  HR编辑自己创建的岗位
// @Tags         HR-岗位管理
// @Accept       json
// @Produce      json
// @Param        job_id  path  int  true  "岗位ID"
// @Success      200     {object}  map[string]interface{}
// @Router       /api/v1/hr/jobs/{job_id} [put]
// @Security     BearerAuth
func _swag_6() {}

// @Summary      下架岗位
// @Description  将岗位状态改为已下架
// @Tags         HR-岗位管理
// @Accept       json
// @Produce      json
// @Param        job_id  path  int  true  "岗位ID"
// @Success      200     {object}  map[string]interface{}
// @Router       /api/v1/hr/jobs/{job_id}/offline [patch]
// @Security     BearerAuth
func _swag_7() {}

// @Summary      上线岗位
// @Description  将已下架岗位重新上线
// @Tags         HR-岗位管理
// @Accept       json
// @Produce      json
// @Param        job_id  path  int  true  "岗位ID"
// @Success      200     {object}  map[string]interface{}
// @Router       /api/v1/hr/jobs/{job_id}/online [patch]
// @Security     BearerAuth
func _swag_8() {}

// @Summary      获取HR岗位列表
// @Description  获取当前HR创建的岗位列表
// @Tags         HR-岗位管理
// @Accept       json
// @Produce      json
// @Param        page      query  int  false  "页码"  default(1)
// @Param        page_size query  int  false  "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/jobs [get]
// @Security     BearerAuth
func _swag_9() {}

// ---- HR Applications ----

// @Summary      查看候选人台账
// @Description  查看指定岗位的投递列表
// @Tags         HR-投递管理
// @Accept       json
// @Produce      json
// @Param        job_id    path  int  true  "岗位ID"
// @Param        page      query  int  false  "页码"  default(1)
// @Param        page_size query  int  false  "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/jobs/{job_id}/applications [get]
// @Security     BearerAuth
func _swag_10() {}

// @Summary      更新投递状态
// @Description  更新投递状态:0待查看/1已查看/2通过/3淘汰
// @Tags         HR-投递管理
// @Accept       json
// @Produce      json
// @Param        application_id  path  int    true  "投递ID"
// @Param        body            body  object{status=int}  true  "状态:0/1/2/3"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/applications/{application_id}/status [patch]
// @Security     BearerAuth
func _swag_11() {}

// ---- HR AI ----

// @Summary      AI对话
// @Description  向AI助手发送消息获取回复
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        body  body  object{message=string,application_id=int,session_id=int}  true  "消息内容"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/chat [post]
// @Security     BearerAuth
func _swag_12() {}

// @Summary      AI流式对话
// @Description  SSE流式AI对话
// @Tags         HR-AI助手
// @Accept       json
// @Produce      text/event-stream
// @Param        body  body  object{message=string,application_id=int,session_id=int}  true  "消息内容"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/chat/stream [post]
// @Security     BearerAuth
func _swag_13() {}

// @Summary      对话历史
// @Description  获取AI对话历史记录
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        page      query  int  false  "页码"  default(1)
// @Param        page_size query  int  false  "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/history [get]
// @Security     BearerAuth
func _swag_14() {}

// @Summary      会话列表
// @Description  获取AI对话会话列表
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        page      query  int  false  "页码"  default(1)
// @Param        page_size query  int  false  "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/sessions [get]
// @Security     BearerAuth
func _swag_15() {}

// @Summary      新建会话
// @Description  创建新的AI对话会话
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        body  body  object{title=string}  false  "会话标题"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/sessions [post]
// @Security     BearerAuth
func _swag_16() {}

// @Summary      会话消息
// @Description  获取指定会话的消息列表
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        session_id  path  int  true  "会话ID"
// @Param        page        query int  false "页码"  default(1)
// @Param        page_size   query int  false "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/sessions/{session_id}/messages [get]
// @Security     BearerAuth
func _swag_17() {}

// @Summary      重命名会话
// @Description  修改AI会话名称
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        session_id  path  int    true  "会话ID"
// @Param        body        body  object{title=string}  true  "新名称"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/sessions/{session_id} [put]
// @Security     BearerAuth
func _swag_18() {}

// @Summary      删除会话
// @Description  删除AI会话及其历史
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        session_id  path  int  true  "会话ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/sessions/{session_id} [delete]
// @Security     BearerAuth
func _swag_19() {}

// @Summary      创建简历分析会话
// @Description  为指定投递记录创建AI简历分析会话
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        body  body  object{application_id=int}  true  "投递记录ID"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/application-analysis-sessions [post]
// @Security     BearerAuth
func _swag_20() {}

// @Summary      分析投递
// @Description  分析候选人简历与岗位匹配度
// @Tags         HR-AI助手
// @Accept       json
// @Produce      json
// @Param        body  body  object{application_id=int}  true  "投递记录ID"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/ai/analyze-application [post]
// @Security     BearerAuth
func _swag_21() {}

// ---- Candidate ----

// @Summary      获取个人档案
// @Description  获取候选人个人资料
// @Tags         候选人
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/profile [get]
// @Security     BearerAuth
func _swag_22() {}

// @Summary      更新个人档案
// @Description  更新候选人个人资料
// @Tags         候选人
// @Accept       json
// @Produce      json
// @Param        body  body  object{real_name=string,phone=string,education=string,school=string,work_experience=string,skills=string}  true  "个人资料"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/candidate/profile [put]
// @Security     BearerAuth
func _swag_23() {}

// @Summary      获取简历
// @Description  获取当前有效简历信息
// @Tags         候选人
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/resume [get]
// @Security     BearerAuth
func _swag_24() {}

// @Summary      获取上传签名URL
// @Description  获取OSS预签名上传URL(仅支持PDF)
// @Tags         候选人
// @Accept       json
// @Produce      json
// @Param        body  body  object{file_name=string,file_type=string}  true  "文件信息"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/candidate/resume/presign [post]
// @Security     BearerAuth
func _swag_25() {}

// @Summary      确认上传
// @Description  确认简历文件已上传到OSS
// @Tags         候选人
// @Accept       json
// @Produce      json
// @Param        body  body  object{oss_key=string,file_name=string,file_type=string,file_size=int}  true  "确认信息"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/candidate/resume/confirm [post]
// @Security     BearerAuth
func _swag_26() {}

// @Summary      投递岗位
// @Description  候选人投递指定岗位
// @Tags         候选人
// @Accept       json
// @Produce      json
// @Param        body  body  object{job_id=int}  true  "岗位ID"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/candidate/applications [post]
// @Security     BearerAuth
func _swag_27() {}

// @Summary      我的投递记录
// @Description  查看我的历史投递记录
// @Tags         候选人
// @Accept       json
// @Produce      json
// @Param        page      query  int  false  "页码"  default(1)
// @Param        page_size query  int  false  "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/applications [get]
// @Security     BearerAuth
func _swag_28() {}
