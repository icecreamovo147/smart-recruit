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
// @Description  注册求职者账号（角色：candidate）。HR/管理员账号由管理员通过后台创建。
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

// ---- Auth (补充) ----

// @Summary      验证邀请码
// @Description  注册前验证邀请码是否有效
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  object{code=string}  true  "邀请码"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/auth/register/validate-invite-code [post]
func _swag_29() {}

// @Summary      用户登出
// @Description  退出登录，清除 Token
// @Tags         认证
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/auth/logout [post]
// @Security     BearerAuth
func _swag_30() {}

// @Summary      刷新 Token
// @Description  使用 Refresh Token 获取新的 Access Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  object{refresh_token=string}  true  "刷新令牌"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/auth/refresh [post]
func _swag_31() {}

// @Summary      获取当前用户信息
// @Description  获取当前已登录用户的详细信息
// @Tags         认证
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/auth/me [get]
// @Security     BearerAuth
func _swag_32() {}

// ---- HR 岗位管理 (补充) ----

// @Summary      岗位选项列表
// @Description  获取所有岗位的下拉选项（用于筛选器）
// @Tags         HR-岗位管理
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/job-options [get]
// @Security     BearerAuth
func _swag_33() {}

// @Summary      投递状态转换历史
// @Description  获取指定投递的状态变更历史
// @Tags         HR-投递管理
// @Produce      json
// @Param        id  path  int  true  "投递ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/applications/{id}/transitions [get]
// @Security     BearerAuth
func _swag_34() {}

// ---- HR 面试管理 ----

// @Summary      投递的面试列表
// @Description  获取指定投递记录的所有面试安排
// @Tags         HR-面试管理
// @Produce      json
// @Param        id  path  int  true  "投递ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/applications/{id}/interviews [get]
// @Security     BearerAuth
func _swag_35() {}

// @Summary      安排面试
// @Description  为候选人安排一场新的面试
// @Tags         HR-面试管理
// @Accept       json
// @Produce      json
// @Param        body  body  object{application_id=int,interviewer_id=int,scheduled_at=string,round_no=int,mode=string}  true  "面试信息"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/interviews [post]
// @Security     BearerAuth
func _swag_36() {}

// @Summary      更新面试
// @Description  更新面试安排信息
// @Tags         HR-面试管理
// @Accept       json
// @Produce      json
// @Param        interview_id  path  int  true  "面试ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/interviews/{interview_id} [put]
// @Security     BearerAuth
func _swag_37() {}

// @Summary      取消面试
// @Description  取消已安排的面试
// @Tags         HR-面试管理
// @Accept       json
// @Produce      json
// @Param        interview_id  path  int  true  "面试ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/interviews/{interview_id}/cancel [patch]
// @Security     BearerAuth
func _swag_38() {}

// @Summary      面试详情
// @Description  获取面试安排的详细信息
// @Tags         HR-面试管理
// @Produce      json
// @Param        interview_id  path  int  true  "面试ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/interviews/{interview_id} [get]
// @Security     BearerAuth
func _swag_39() {}

// @Summary      面试官列表
// @Description  获取可选面试官列表
// @Tags         HR-面试管理
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/interviewers [get]
// @Security     BearerAuth
func _swag_40() {}

// @Summary      我的面试任务
// @Description  获取当前用户的面试任务列表
// @Tags         HR-面试管理
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/my-interviews [get]
// @Security     BearerAuth
func _swag_41() {}

// @Summary      提交面试反馈
// @Description  面试官提交面试评价
// @Tags         HR-面试管理
// @Accept       json
// @Produce      json
// @Param        interview_id  path  int  true  "面试ID"
// @Param        body          body  object{rating=int,comment=string}  true  "反馈内容"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/interviews/{interview_id}/feedback [post]
// @Security     BearerAuth
func _swag_42() {}

// @Summary      获取面试反馈
// @Description  查看已提交的面试反馈
// @Tags         HR-面试管理
// @Produce      json
// @Param        interview_id  path  int  true  "面试ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/interviews/{interview_id}/feedback [get]
// @Security     BearerAuth
func _swag_43() {}

// ---- HR Offer 管理 ----

// @Summary      创建 Offer
// @Description  为候选人创建录用 Offer
// @Tags         HR-Offer管理
// @Accept       json
// @Produce      json
// @Param        body  body  object{application_id=int,title=string,salary=string,start_date=string}  true  "Offer信息"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/hr/offers [post]
// @Security     BearerAuth
func _swag_44() {}

// @Summary      更新 Offer
// @Description  编辑 Offer 信息
// @Tags         HR-Offer管理
// @Accept       json
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/offers/{offer_id} [put]
// @Security     BearerAuth
func _swag_45() {}

// @Summary      Offer 详情
// @Description  获取 Offer 详细信息
// @Tags         HR-Offer管理
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/offers/{offer_id} [get]
// @Security     BearerAuth
func _swag_46() {}

// @Summary      投递的 Offer 列表
// @Description  获取指定投递记录的 Offer 列表
// @Tags         HR-Offer管理
// @Produce      json
// @Param        id  path  int  true  "投递ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/applications/{id}/offers [get]
// @Security     BearerAuth
func _swag_47() {}

// @Summary      发送 Offer
// @Description  将 Offer 发送给候选人
// @Tags         HR-Offer管理
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/offers/{offer_id}/send [post]
// @Security     BearerAuth
func _swag_48() {}

// @Summary      撤回 Offer
// @Description  撤回已发送的 Offer
// @Tags         HR-Offer管理
// @Accept       json
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/offers/{offer_id}/withdraw [post]
// @Security     BearerAuth
func _swag_49() {}

// @Summary      Offer 事件列表
// @Description  获取 Offer 的状态变更事件
// @Tags         HR-Offer管理
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/offers/{offer_id}/events [get]
// @Security     BearerAuth
func _swag_50() {}

// ---- 候选人-Offer ----

// @Summary      我的 Offer 列表
// @Description  获取发送给我的 Offer 列表
// @Tags         候选人-Offer
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/offers [get]
// @Security     BearerAuth
func _swag_51() {}

// @Summary      Offer 详情
// @Description  查看 Offer 详细信息
// @Tags         候选人-Offer
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/offers/{offer_id} [get]
// @Security     BearerAuth
func _swag_52() {}

// @Summary      接受 Offer
// @Description  候选人接受录用 Offer
// @Tags         候选人-Offer
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/offers/{offer_id}/accept [post]
// @Security     BearerAuth
func _swag_53() {}

// @Summary      拒绝 Offer
// @Description  候选人拒绝录用 Offer
// @Tags         候选人-Offer
// @Accept       json
// @Produce      json
// @Param        offer_id  path  int  true  "Offer ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/offers/{offer_id}/reject [post]
// @Security     BearerAuth
func _swag_54() {}

// ---- 候选人-面试 ----

// @Summary      我的面试列表
// @Description  获取安排给我的面试列表
// @Tags         候选人-面试
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/interviews [get]
// @Security     BearerAuth
func _swag_55() {}

// ---- 候选人-通知 ----

// @Summary      通知列表
// @Description  获取候选人通知列表
// @Tags         候选人-通知
// @Produce      json
// @Param        page      query  int  false  "页码"  default(1)
// @Param        page_size query  int  false  "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/notifications [get]
// @Security     BearerAuth
func _swag_56() {}

// @Summary      未读通知数
// @Description  获取未读通知数量
// @Tags         候选人-通知
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/notifications/unread-count [get]
// @Security     BearerAuth
func _swag_57() {}

// @Summary      通知摘要
// @Description  获取通知分类摘要
// @Tags         候选人-通知
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/notifications/summary [get]
// @Security     BearerAuth
func _swag_58() {}

// @Summary      通知实时流
// @Description  SSE 实时推送新通知
// @Tags         候选人-通知
// @Produce      text/event-stream
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/notifications/stream [get]
// @Security     BearerAuth
func _swag_59() {}

// @Summary      标记通知已读
// @Description  将指定通知标记为已读
// @Tags         候选人-通知
// @Produce      json
// @Param        notification_id  path  int  true  "通知ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/notifications/{notification_id}/read [patch]
// @Security     BearerAuth
func _swag_60() {}

// @Summary      全部标记已读
// @Description  将所有通知标记为已读
// @Tags         候选人-通知
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/notifications/read-all [patch]
// @Security     BearerAuth
func _swag_61() {}

// ---- 候选人-AI ----

// @Summary      AI 会话列表
// @Description  获取候选人的 AI 对话会话列表
// @Tags         候选人-AI
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/ai/sessions [get]
// @Security     BearerAuth
func _swag_62() {}

// @Summary      新建 AI 会话
// @Description  创建新的 AI 对话会话
// @Tags         候选人-AI
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/ai/sessions [post]
// @Security     BearerAuth
func _swag_63() {}

// @Summary      AI 会话消息
// @Description  获取指定会话的消息列表
// @Tags         候选人-AI
// @Produce      json
// @Param        session_id  path  int  true  "会话ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/ai/sessions/{session_id}/messages [get]
// @Security     BearerAuth
func _swag_64() {}

// @Summary      重命名 AI 会话
// @Description  修改会话标题
// @Tags         候选人-AI
// @Accept       json
// @Produce      json
// @Param        session_id  path  int  true  "会话ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/ai/sessions/{session_id} [put]
// @Security     BearerAuth
func _swag_65() {}

// @Summary      删除 AI 会话
// @Description  删除候选人 AI 会话
// @Tags         候选人-AI
// @Produce      json
// @Param        session_id  path  int  true  "会话ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/ai/sessions/{session_id} [delete]
// @Security     BearerAuth
func _swag_66() {}

// @Summary      AI 流式对话
// @Description  候选人端 SSE 流式 AI 对话
// @Tags         候选人-AI
// @Accept       json
// @Produce      text/event-stream
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/candidate/ai/chat/stream [post]
// @Security     BearerAuth
func _swag_67() {}

// ---- HR-通知 ----

// @Summary      通知列表
// @Description  获取 HR 通知列表
// @Tags         HR-通知
// @Produce      json
// @Param        page      query  int  false  "页码"  default(1)
// @Param        page_size query  int  false  "每页数量" default(10)
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notifications [get]
// @Security     BearerAuth
func _swag_68() {}

// @Summary      未读通知数
// @Description  获取 HR 未读通知数量
// @Tags         HR-通知
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notifications/unread-count [get]
// @Security     BearerAuth
func _swag_69() {}

// @Summary      通知摘要
// @Description  获取 HR 通知分类摘要
// @Tags         HR-通知
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notifications/summary [get]
// @Security     BearerAuth
func _swag_70() {}

// @Summary      通知实时流
// @Description  SSE 实时推送新通知
// @Tags         HR-通知
// @Produce      text/event-stream
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notifications/stream [get]
// @Security     BearerAuth
func _swag_71() {}

// @Summary      标记通知已读
// @Description  将指定通知标记为已读
// @Tags         HR-通知
// @Produce      json
// @Param        notification_id  path  int  true  "通知ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notifications/{notification_id}/read [patch]
// @Security     BearerAuth
func _swag_72() {}

// @Summary      全部标记已读
// @Description  将所有通知标记为已读
// @Tags         HR-通知
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notifications/read-all [patch]
// @Security     BearerAuth
func _swag_73() {}

// ---- HR-仪表盘 ----

// @Summary      工作台汇总
// @Description  获取 HR 工作台汇总数据
// @Tags         HR-仪表盘
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/dashboard/summary [get]
// @Security     BearerAuth
func _swag_74() {}

// ---- HR-数据分析 ----

// @Summary      仪表盘报表
// @Description  获取 KPI 概览报表数据
// @Tags         HR-数据分析
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/analytics/dashboard [get]
// @Security     BearerAuth
func _swag_75() {}

// @Summary      漏斗报表
// @Description  获取投递漏斗报表数据
// @Tags         HR-数据分析
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/analytics/funnel [get]
// @Security     BearerAuth
func _swag_76() {}

// @Summary      阶段停留时长
// @Description  获取投递在各阶段的停留时长报表
// @Tags         HR-数据分析
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/analytics/time-in-stage [get]
// @Security     BearerAuth
func _swag_77() {}

// @Summary      面试与 Offer 指标
// @Description  获取面试和 Offer 的关键指标
// @Tags         HR-数据分析
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/analytics/metrics [get]
// @Security     BearerAuth
func _swag_78() {}

// @Summary      安全审计日志
// @Description  获取认证授权审计日志
// @Tags         HR-数据分析
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/auth-audit-logs [get]
// @Security     BearerAuth
func _swag_79() {}

// ---- HR-协作 ----

// @Summary      候选人工作台
// @Description  获取候选人协作工作台数据
// @Tags         HR-协作
// @Produce      json
// @Param        candidate_user_id  path  int  true  "候选人用户ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/candidates/{candidate_user_id}/workspace [get]
// @Security     BearerAuth
func _swag_80() {}

// @Summary      创建备注
// @Description  为候选人创建备注
// @Tags         HR-协作
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notes [post]
// @Security     BearerAuth
func _swag_81() {}

// @Summary      备注列表
// @Description  获取候选人备注列表
// @Tags         HR-协作
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/notes [get]
// @Security     BearerAuth
func _swag_82() {}

// @Summary      标签列表
// @Description  获取所有可用标签
// @Tags         HR-协作
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/tags [get]
// @Security     BearerAuth
func _swag_83() {}

// @Summary      创建标签
// @Description  创建新标签
// @Tags         HR-协作
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/tags [post]
// @Security     BearerAuth
func _swag_84() {}

// @Summary      分配标签
// @Description  为候选人分配标签
// @Tags         HR-协作
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/tags/assign [post]
// @Security     BearerAuth
func _swag_85() {}

// @Summary      取消标签
// @Description  取消候选人的标签
// @Tags         HR-协作
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/tags/unassign [post]
// @Security     BearerAuth
func _swag_86() {}

// @Summary      候选人标签列表
// @Description  获取指定候选人的标签列表
// @Tags         HR-协作
// @Produce      json
// @Param        candidate_user_id  path  int  true  "候选人用户ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/candidates/{candidate_user_id}/tags [get]
// @Security     BearerAuth
func _swag_87() {}

// @Summary      创建跟进任务
// @Description  创建候选人跟进任务
// @Tags         HR-协作
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/follow-up-tasks [post]
// @Security     BearerAuth
func _swag_88() {}

// @Summary      跟进任务列表
// @Description  获取跟进任务列表
// @Tags         HR-协作
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/follow-up-tasks [get]
// @Security     BearerAuth
func _swag_89() {}

// @Summary      完成跟进任务
// @Description  标记跟进任务为已完成
// @Tags         HR-协作
// @Produce      json
// @Param        task_id  path  int  true  "任务ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/follow-up-tasks/{task_id}/complete [patch]
// @Security     BearerAuth
func _swag_90() {}

// @Summary      候选人时间线
// @Description  获取候选人操作时间线
// @Tags         HR-协作
// @Produce      json
// @Param        candidate_user_id  path  int  true  "候选人用户ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/candidates/{candidate_user_id}/timeline [get]
// @Security     BearerAuth
func _swag_91() {}

// ---- 管理后台 ----

// @Summary      创建邀请码
// @Tags         管理后台-邀请码
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/invite-codes [post]
// @Security     BearerAuth
func _swag_92() {}

// @Summary      邀请码列表
// @Tags         管理后台-邀请码
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/invite-codes [get]
// @Security     BearerAuth
func _swag_93() {}

// @Summary      延长邀请码
// @Tags         管理后台-邀请码
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "邀请码ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/invite-codes/{id}/extend [patch]
// @Security     BearerAuth
func _swag_94() {}

// @Summary      撤销邀请码
// @Tags         管理后台-邀请码
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "邀请码ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/invite-codes/{id}/revoke [patch]
// @Security     BearerAuth
func _swag_95() {}

// @Summary      重新激活邀请码
// @Tags         管理后台-邀请码
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "邀请码ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/invite-codes/{id}/reactivate [patch]
// @Security     BearerAuth
func _swag_96() {}

// @Summary      部门列表
// @Tags         管理后台-部门
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments [get]
// @Security     BearerAuth
func _swag_97() {}

// @Summary      创建部门
// @Tags         管理后台-部门
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments [post]
// @Security     BearerAuth
func _swag_98() {}

// @Summary      更新部门
// @Tags         管理后台-部门
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "部门ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments/{id} [put]
// @Security     BearerAuth
func _swag_99() {}

// @Summary      更新部门状态
// @Tags         管理后台-部门
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "部门ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments/{id}/status [patch]
// @Security     BearerAuth
func _swag_100() {}

// @Summary      删除部门
// @Tags         管理后台-部门
// @Produce      json
// @Param        id  path  int  true  "部门ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments/{id} [delete]
// @Security     BearerAuth
func _swag_101() {}

// @Summary      地点列表
// @Tags         管理后台-地点
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/locations [get]
// @Security     BearerAuth
func _swag_102() {}

// @Summary      创建地点
// @Tags         管理后台-地点
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/locations [post]
// @Security     BearerAuth
func _swag_103() {}

// @Summary      更新地点
// @Tags         管理后台-地点
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "地点ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/locations/{id} [put]
// @Security     BearerAuth
func _swag_104() {}

// @Summary      更新地点状态
// @Tags         管理后台-地点
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "地点ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/locations/{id}/status [patch]
// @Security     BearerAuth
func _swag_105() {}

// @Summary      删除地点
// @Tags         管理后台-地点
// @Produce      json
// @Param        id  path  int  true  "地点ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/locations/{id} [delete]
// @Security     BearerAuth
func _swag_106() {}

// @Summary      部门地点映射列表
// @Tags         管理后台-部门
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments/location-map [get]
// @Security     BearerAuth
func _swag_107() {}

// @Summary      部门地点配置
// @Tags         管理后台-部门
// @Produce      json
// @Param        id  path  int  true  "部门ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments/{id}/locations [get]
// @Security     BearerAuth
func _swag_108() {}

// @Summary      更新部门地点配置
// @Tags         管理后台-部门
// @Accept       json
// @Produce      json
// @Param        id  path  int  true  "部门ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/departments/{id}/locations [put]
// @Security     BearerAuth
func _swag_109() {}

// @Summary      第三方服务使用日志
// @Tags         管理后台-审计
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/third-party-usage-logs [get]
// @Security     BearerAuth
func _swag_110() {}

// @Summary      角色列表
// @Tags         管理后台-RBAC
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/roles [get]
// @Security     BearerAuth
func _swag_111() {}

// @Summary      权限列表
// @Tags         管理后台-RBAC
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/permissions [get]
// @Security     BearerAuth
func _swag_112() {}

// @Summary      用户角色
// @Tags         管理后台-RBAC
// @Produce      json
// @Param        user_id  path  int  true  "用户ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/users/{user_id}/roles [get]
// @Security     BearerAuth
func _swag_113() {}

// @Summary      分配角色
// @Tags         管理后台-RBAC
// @Accept       json
// @Produce      json
// @Param        user_id  path  int  true  "用户ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/users/{user_id}/roles/assign [post]
// @Security     BearerAuth
func _swag_114() {}

// @Summary      撤销角色
// @Tags         管理后台-RBAC
// @Accept       json
// @Produce      json
// @Param        user_id  path  int  true  "用户ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/users/{user_id}/roles/revoke [post]
// @Security     BearerAuth
func _swag_115() {}

// @Summary      分配数据范围
// @Tags         管理后台-RBAC
// @Accept       json
// @Produce      json
// @Param        user_id  path  int  true  "用户ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/users/{user_id}/data-scopes [post]
// @Security     BearerAuth
func _swag_116() {}

// @Summary      撤销数据范围
// @Tags         管理后台-RBAC
// @Produce      json
// @Param        scope_id  path  int  true  "范围ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/data-scopes/{scope_id} [delete]
// @Security     BearerAuth
func _swag_117() {}

// @Summary      员工用户列表
// @Tags         管理后台-用户
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/staff-users [get]
// @Security     BearerAuth
func _swag_118() {}

// @Summary      创建员工用户
// @Tags         管理后台-用户
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/hr/admin/staff-users [post]
// @Security     BearerAuth
func _swag_119() {}
