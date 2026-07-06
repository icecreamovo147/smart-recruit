# ADR-005: LLM Provider 和 Model 配置管理方案

## 状态

已采纳 (2026-06-26)

## 背景

当前 LLM 配置（API Key、Base URL、Model 名称）全部通过环境变量 `AI_API_KEY` / `AI_BASE_URL` / `AI_MODEL` 硬编码，存在以下问题：

1. API Key 明文存储在配置文件和数据库中（第三方日志表）
2. 不支持多 Provider 切换
3. Model 参数（temperature、top_p 等）不可动态配置
4. 无法可视化管理和测试连接

## 决策

### 1. 加密算法选型：AES-256-GCM

选择 AES-256-GCM 作为 API Key 加密方案：

- AES-256-GCM 提供认证加密（AEAD），同时保证机密性和完整性
- Go 标准库 `crypto/aes` + `crypto/cipher` 原生支持，无需额外依赖
- GCM 模式不需要填充，密文长度 = 明文长度 + 12（nonce）+ 16（tag）
- 每次加密生成随机 12 字节 nonce，确保相同明文每次加密结果不同

### 2. ENCRYPTION_KEY 的来源与管理

- 密钥从环境变量 `ENCRYPTION_KEY` 读取
- 密钥要求：32 字节（64 位十六进制字符串或 32 字节原始数据）
- 启动时校验：长度必须为 32 字节，否则 panic
- 部署时通过 K8s Secret / Docker 环境变量注入
- 不存储在数据库或配置文件中

### 3. Provider 表的 extra_headers 用途

`extra_headers` 字段（JSON 格式）用于存储 Provider 特定的 HTTP Header：

- Anthropic API：需要 `x-api-key` header 替代 `Authorization`
- 自定义 Provider：可能需要额外的鉴权 Header
- 格式示例：`{"x-api-key": "sk-ant-xxx", "X-Custom-Header": "value"}`

返回前端时，extra_headers 中的 value 部分需要脱敏（显示前 4 + 后 4 位）。

### 4. 与已有环境变量的兼容策略

为了不影响现有业务，采用**优先级策略**：

1. 优先从 `llm_providers` + `llm_models` 表读取配置
2. 如果表为空或未查询到对应记录，回退到环境变量 `AI_BASE_URL` / `AI_API_KEY` / `AI_MODEL`
3. `eino_client.go` 的 `NewClient` 函数签名保持不变，新增 `NewClientFromConfig` 方法

### 5. 数据库表设计

#### llm_providers 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT AUTO_INCREMENT PK | 主键 |
| name | VARCHAR(128) NOT NULL | Provider 名称 |
| base_url | VARCHAR(512) NOT NULL | API 基础 URL |
| api_key_encrypted | VARCHAR(512) NOT NULL | AES-256-GCM 加密后的 API Key |
| provider_type | VARCHAR(64) NOT NULL | openai_compatible / anthropic / deepseek / ollama |
| extra_headers | JSON | 额外 HTTP Headers |
| is_enabled | TINYINT(1) DEFAULT 1 | 是否启用 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

#### llm_models 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT AUTO_INCREMENT PK | 主键 |
| provider_id | BIGINT NOT NULL FK | 关联 llm_providers.id |
| model_name | VARCHAR(128) NOT NULL | 模型名称（API 使用） |
| display_name | VARCHAR(128) | 展示名称 |
| temperature | DOUBLE DEFAULT 0.7 | 温度参数 |
| top_p | DOUBLE DEFAULT 1.0 | Top P 参数 |
| max_tokens | INT DEFAULT 4096 | 最大 Token 数 |
| max_concurrency | INT DEFAULT 10 | 最大并发数 |
| timeout_seconds | INT DEFAULT 90 | 超时秒数 |
| is_enabled | TINYINT(1) DEFAULT 1 | 是否启用 |
| is_default | TINYINT(1) DEFAULT 0 | 是否默认模型 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### 6. API Key 脱敏方案

返回前端时：
- 只显示前 4 位 + 后 4 位，中间替换为 `****`
- 如果 Key 长度 <= 8，只显示前 2 位 + 后 2 位
- `api_key_encrypted` 字段永远不返回前端
- `extra_headers` 中的 value 也按相同规则脱敏

### 7. 连接测试方案

测试 Provider 连接时：
- 调用 `GET {base_url}/v1/models` 端点（OpenAI 兼容接口）
- 请求 Header：`Authorization: Bearer {decrypted_api_key}`
- 超时设置：10 秒
- 如果返回 200 则连通性正常

## 影响范围

- 新增 2 个数据库表（llm_providers, llm_models）
- 新增 gRPC 服务 LlmConfigService（9 个方法）
- 新增 HTTP API（/hr/admin/llm-providers/*, /hr/admin/llm-models/*）
- 修改 `eino_client.go` 支持从配置读取
- 不影响现有 AI 业务流程

## 回退方案

- 删除 llm_providers 和 llm_models 表
- 恢复依赖环境变量的配置方式
- 回滚 proto 变更
