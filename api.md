# 接口说明文档

> Base URL（Web Gin 服务）：`http://localhost:8080`  
> 所有需要鉴权的接口须在 Header 携带：`Authorization: Bearer <JWT Token>`  
> 响应体统一格式见下方

---

## 统一响应格式

```json
{
  "code": 0,        // 0=成功，非0=错误码
  "msg": "success", // 提示信息
  "data": {}        // 业务数据，失败时可为 null
}
```

**通用错误码：**

| code | 说明 |
|------|------|
| 0 | 成功 |
| 401 | 未登录或 Token 无效/过期 |
| 403 | 无权限操作 |
| 400 | 请求参数错误 |
| 500 | 服务器内部错误 |

---

## 一、通用认证接口（HR & 候选人共用）

### 1.1 注册

```
POST /api/v1/auth/register
```

**请求体：**

```json
{
  "username": "zhangsan",
  "password": "123456",
  "role": 1   // 1=候选人，2=HR
}
```

**响应：**

```json
{
  "code": 0,
  "msg": "注册成功",
  "data": {
    "user_id": 1,
    "username": "zhangsan",
    "role": 1
  }
}
```

---

### 1.2 登录

```
POST /api/v1/auth/login
```

**请求体：**

```json
{
  "username": "zhangsan",
  "password": "123456"
}
```

**响应：**

```json
{
  "code": 0,
  "msg": "登录成功",
  "data": {
    "token": "eyJhbGci...",
    "user_id": 1,
    "role": 1,
    "username": "zhangsan"
  }
}
```

---

## 二、HR 管理端接口（需 role=2 + JWT）

### 2.1 岗位管理

#### 新增岗位

```
POST /api/v1/hr/jobs
```

**请求体：**

```json
{
  "title": "后端开发工程师",
  "department": "技术部",
  "location": "北京",
  "salary_range": "20k-35k",
  "description": "负责后端服务开发...",
  "requirements": "3年以上Go开发经验..."
}
```

**响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": { "job_id": 10 }
}
```

---

#### 编辑岗位

```
PUT /api/v1/hr/jobs/:job_id
```

> 仅能编辑本人创建的岗位，操作他人岗位返回 403。

**请求体：** 同新增岗位，字段可部分传入

---

#### 下架岗位

```
PATCH /api/v1/hr/jobs/:job_id/offline
```

**响应：**

```json
{
  "code": 0,
  "msg": "岗位已下架"
}
```

---

#### 上线岗位

```
PATCH /api/v1/hr/jobs/:job_id/online
```

**响应：**

```json
{
  "code": 0,
  "msg": "岗位已上线"
}
```

---

#### 获取本人岗位列表（分页）

```
GET /api/v1/hr/jobs?page=1&page_size=10
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "total": 25,
    "list": [
      {
        "job_id": 10,
        "title": "后端开发工程师",
        "status": 1,
        "application_count": 8,
        "created_at": "2025-06-01T10:00:00Z"
      }
    ]
  }
}
```

---

### 2.2 候选人台账查看

#### 获取指定岗位的投递列表（分页）

```
GET /api/v1/hr/jobs/:job_id/applications?page=1&page_size=10
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "total": 8,
    "list": [
      {
        "application_id": 1,
        "user_id": 5,
        "real_name": "李四",
        "phone": "138xxxxxxxx",
        "education": "本科",
        "school": "某某大学",
        "skills": ["Go", "MySQL", "Redis"],
        "applied_at": "2025-06-02T14:00:00Z",
        "resume_url": "https://oss.example.com/sign?...",
        "status": 0
      }
    ]
  }
}
```

> `resume_url` 为后端实时生成的 OSS 签名 URL，有效期 15 分钟，前端直接使用即可

---

#### 更新投递状态

```
PATCH /api/v1/hr/applications/:application_id/status
```

**请求体：**

```json
{
  "status": 2
}
```

> 状态含义：0=待查看，1=已查看，2=通过，3=淘汰。

**响应：**

```json
{
  "code": 0,
  "msg": "投递状态已更新"
}
```

---

### 2.3 AI 对话接口

#### 发送消息

```
POST /api/v1/hr/ai/chat
```

**请求体：**

```json
{
  "message": "后端开发岗位今天投递了多少人？"
}
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "reply": "后端开发工程师岗位今日共收到 3 份投递，累计投递 12 人。",
    "created_at": "2025-06-02T15:00:00Z"
  }
}
```

> 后端流程：解析意图 → 查询 MySQL → 拼接上下文 → 调用 Eino → 返回回答；同时将本次 user/assistant 消息对写入 `ai_chat_history`

---

#### 获取历史对话记录

```
GET /api/v1/hr/ai/history?page=1&page_size=50
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "list": [
      { "role": "user", "content": "投递总人数是多少？", "created_at": "..." },
      { "role": "assistant", "content": "截至目前，平台共收到 47 份投递。", "created_at": "..." }
    ]
  }
}
```

---

## 三、候选人用户端接口

### 3.1 公开接口（无需登录）

#### 浏览岗位列表（分页）

```
GET /api/v1/jobs?page=1&page_size=10&keyword=后端
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "total": 30,
    "list": [
      {
        "job_id": 10,
        "title": "后端开发工程师",
        "department": "技术部",
        "location": "北京",
        "salary_range": "20k-35k",
        "created_at": "2025-06-01T10:00:00Z"
      }
    ]
  }
}
```

---

#### 查看岗位详情

```
GET /api/v1/jobs/:job_id
```

---

### 3.2 需要登录的接口（role=1 + JWT）

#### 获取/更新个人档案

```
GET  /api/v1/candidate/profile
PUT  /api/v1/candidate/profile
```

**PUT 请求体：**

```json
{
  "real_name": "李四",
  "phone": "138xxxxxxxx",
  "education": "本科",
  "school": "某某大学",
  "work_experience": "2022-2024 某公司 后端开发...",
  "skills": "Go,MySQL,Redis,Docker"
}
```

**响应（GET）：**

```json
{
  "code": 0,
  "data": {
    "real_name": "李四",
    "phone": "138xxxxxxxx",
    "education": "本科",
    "school": "某某大学",
    "work_experience": "...",
    "skills": ["Go", "MySQL", "Redis"],
    "is_complete": true
  }
}
```

---

#### 获取当前有效简历

```
GET /api/v1/candidate/resume
```

**响应（未上传）：**

```json
{
  "code": 0,
  "data": { "resume": null }
}
```

**响应（已上传）：**

```json
{
  "code": 0,
  "data": {
    "resume": {
      "resume_id": 3,
      "file_name": "李四简历.pdf",
      "file_type": "pdf",
      "file_size": 204800,
      "uploaded_at": "2025-06-02T15:15:00Z",
      "resume_url": "https://oss.example.com/presign-get?..."
    }
  }
}
```

> `resume_url` 为当前有效 PDF 简历的临时预览 URL，前端可直接用于 iframe / object 预览。

---

#### 获取简历上传签名 URL

```
POST /api/v1/candidate/resume/presign
```

**请求体：**

```json
{
  "file_name": "李四简历.pdf",
  "file_type": "pdf"
}
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "upload_url": "https://oss.example.com/presign-put?...",
    "oss_key": "resumes/user_5/1685702400_resume.pdf",
    "expire_at": "2025-06-02T15:15:00Z"
  }
}
```

> 仅支持 PDF、DOCX 格式简历。前端拿到 `upload_url` 后，直接 HTTP PUT 文件到 OSS，无需经过后端服务器。

---

#### 确认简历上传成功

```
POST /api/v1/candidate/resume/confirm
```

**请求体：**

```json
{
  "oss_key": "resumes/user_5/1685702400_resume.pdf",
  "file_name": "李四简历.pdf",
  "file_type": "pdf",
  "file_size": 204800
}
```

**响应：**

```json
{
  "code": 0,
  "data": { "resume_id": 3 }
}
```

---

#### 一键投递岗位

```
POST /api/v1/candidate/applications
```

**请求体：**

```json
{
  "job_id": 10
}
```

**前置校验（后端执行）：**
1. 用户已登录（JWT 有效）
2. 候选人档案 `is_complete = 1`
3. 存在有效简历记录（`is_valid = 1`）
4. 未重复投递该岗位

**响应（校验失败示例）：**

```json
{
  "code": 4001,
  "msg": "请先完善个人资料后再投递"
}
```

```json
{
  "code": 4002,
  "msg": "请先上传简历后再投递"
}
```

---

#### 查看我的投递记录

```
GET /api/v1/candidate/applications?page=1&page_size=10
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "total": 3,
    "list": [
      {
        "application_id": 1,
        "job_id": 10,
        "job_title": "后端开发工程师",
        "status": 0,
        "applied_at": "2025-06-02T14:00:00Z"
      }
    ]
  }
}
```

---

## 四、接口权限矩阵

| 接口路径 | 游客 | 候选人(role=1) | HR(role=2) |
|----------|------|----------------|------------|
| GET /api/v1/jobs | ✅ | ✅ | ✅ |
| POST /api/v1/auth/register | ✅ | - | - |
| POST /api/v1/auth/login | ✅ | - | - |
| GET/PUT /api/v1/candidate/profile | ❌ | ✅ | ❌ |
| POST /api/v1/candidate/resume/* | ❌ | ✅ | ❌ |
| POST /api/v1/candidate/applications | ❌ | ✅ | ❌ |
| POST /api/v1/hr/jobs | ❌ | ❌ | ✅ |
| GET /api/v1/hr/jobs/:id/applications | ❌ | ❌ | ✅ |
| POST /api/v1/hr/ai/chat | ❌ | ❌ | ✅ |
