# 数据库设计文档

> 数据库名：`recruitment`  
> 字符集：`utf8mb4`  
> 排序规则：`utf8mb4_unicode_ci`

---

## 表清单

| 表名 | 说明 |
|------|------|
| `users` | 用户账号表（HR + 候选人共用，角色字段区分） |
| `jobs` | 招聘岗位表 |
| `candidate_profiles` | 候选人结构化档案表 |
| `resumes` | 简历 OSS 存储记录表 |
| `applications` | 岗位投递关联表 |
| `ai_chat_history` | AI 对话历史记录表 |
| `ai_chat_sessions` | AI 会话表 |
| `ai_session_summaries` | AI 会话滚动摘要表 |
| `ai_tool_traces` | AI 工具调用轨迹表 |
| `ai_memories` | AI 长期记忆表 |
| `notifications` | 站内通知表 |
| `event_outbox` | 事务消息 outbox 表 |

---

## 详细表结构

### 1. users — 用户账号表

```sql
CREATE TABLE `users` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username`     VARCHAR(64)     NOT NULL COMMENT '用户名（唯一）',
  `password`     VARCHAR(255)    NOT NULL COMMENT 'bcrypt 哈希密码',
  `role`         TINYINT         NOT NULL DEFAULT 1 COMMENT '角色：1=候选人 2=HR',
  `email`        VARCHAR(128)    DEFAULT NULL COMMENT '邮箱（可选）',
  `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账号表';
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT | 自增主键 |
| username | VARCHAR(64) | 登录用户名，全局唯一 |
| password | VARCHAR(255) | bcrypt 加密后存储，禁止明文 |
| role | TINYINT | 1=候选人，2=HR，一个账号只有一种角色 |

---

### 2. jobs — 招聘岗位表

```sql
CREATE TABLE `jobs` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '岗位ID',
  `hr_id`        BIGINT UNSIGNED NOT NULL COMMENT '发布该岗位的 HR 用户ID',
  `title`        VARCHAR(128)    NOT NULL COMMENT '岗位名称',
  `department`   VARCHAR(64)     DEFAULT NULL COMMENT '所属部门',
  `location`     VARCHAR(128)    DEFAULT NULL COMMENT '工作地点',
  `salary_range` VARCHAR(64)     DEFAULT NULL COMMENT '薪资范围，如 15k-25k',
  `description`  TEXT            DEFAULT NULL COMMENT '岗位详情描述',
  `requirements` TEXT            DEFAULT NULL COMMENT '任职要求',
  `status`       TINYINT         NOT NULL DEFAULT 1 COMMENT '状态：1=招募中 0=已下架',
  `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_id` (`hr_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='招聘岗位表';
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| hr_id | BIGINT | 关联 users.id，HR 只能管理自己创建的岗位 |
| status | TINYINT | 1=招募中（公开可见），0=已下架（隐藏） |

---

### 3. candidate_profiles — 候选人结构化档案表

```sql
CREATE TABLE `candidate_profiles` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id`         BIGINT UNSIGNED NOT NULL COMMENT '关联 users.id，一人一条档案',
  `real_name`       VARCHAR(64)     DEFAULT NULL COMMENT '真实姓名',
  `phone`           VARCHAR(20)     DEFAULT NULL COMMENT '联系电话',
  `education`       VARCHAR(32)     DEFAULT NULL COMMENT '最高学历，如：本科、硕士、博士',
  `school`          VARCHAR(128)    DEFAULT NULL COMMENT '毕业院校',
  `work_experience` TEXT            DEFAULT NULL COMMENT '工作/项目经历（富文本或 JSON）',
  `skills`          VARCHAR(512)    DEFAULT NULL COMMENT '核心技能标签，逗号分隔',
  `is_complete`     TINYINT         NOT NULL DEFAULT 0 COMMENT '档案是否完整：0=不完整 1=完整',
  `created_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人结构化档案表';
```

**完整性判断规则：**  
`real_name + phone + education + school + work_experience + skills` 六项全部非空时，`is_complete = 1`，否则为 0。

---

### 4. resumes — 简历 OSS 存储记录表

```sql
CREATE TABLE `resumes` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id`     BIGINT UNSIGNED NOT NULL COMMENT '候选人用户ID',
  `oss_key`     VARCHAR(512)    NOT NULL COMMENT 'OSS 对象 Key（相对路径）',
  `file_name`   VARCHAR(255)    NOT NULL COMMENT '原始文件名',
  `file_type`   VARCHAR(16)     NOT NULL COMMENT '文件类型：pdf / docx',
  `file_size`   INT UNSIGNED    DEFAULT NULL COMMENT '文件大小（字节）',
  `parsed_text` MEDIUMTEXT      NULL COMMENT 'PDF 简历解析文本',
  `parsed_at`   DATETIME        NULL COMMENT '简历解析时间',
  `is_valid`    TINYINT         NOT NULL DEFAULT 1 COMMENT '是否有效：1=有效 0=已失效',
  `uploaded_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `valid_key`   TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_valid` = 1 THEN 1 ELSE NULL END
  ) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_valid_resume` (`user_id`, `valid_key`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_is_valid` (`is_valid`),
  KEY `idx_user_uploaded` (`user_id`, `uploaded_at`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='简历 OSS 存储记录表';
```

**说明：**
- `oss_key` 仅存储 OSS 对象 Key，不存储完整 URL（URL 通过签名动态生成）
- 每次候选人上传新简历，旧记录 `is_valid` 置为 0，新增一条有效记录
- 本地不缓存任何简历文件，只缓存 PDF/DOCX 解析出的文本，供 AI 简历分析使用
- `valid_key` 是 MySQL 8 生成列，与 `uk_user_valid_resume` 唯一索引配合，保证同一候选人在数据库层面最多只有一条 `is_valid=1` 的简历
- 已有数据库在添加 `uk_user_valid_resume` 前，必须先把同一用户多条 `is_valid=1` 的历史简历清理为仅保留最新一条，否则唯一索引创建会失败；迁移脚本 `logic-grpc-service/migrations/000003_add_idempotency_constraints.sql` 已包含该清理步骤

如果已有数据库需要支持 AI 简历分析，需要执行：

```sql
ALTER TABLE `resumes`
  ADD COLUMN `parsed_text` MEDIUMTEXT NULL COMMENT 'PDF 简历解析文本' AFTER `file_size`,
  ADD COLUMN `parsed_at` DATETIME NULL COMMENT '简历解析时间' AFTER `parsed_text`;
```

---

### 5. applications — 岗位投递关联表

```sql
CREATE TABLE `applications` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id`       BIGINT UNSIGNED NOT NULL COMMENT '投递的岗位ID',
  `user_id`      BIGINT UNSIGNED NOT NULL COMMENT '投递的候选人用户ID',
  `resume_id`    BIGINT UNSIGNED NOT NULL COMMENT '投递时使用的简历ID',
  `status`       TINYINT         NOT NULL DEFAULT 0 COMMENT '投递状态：0=待查看 1=已查看 2=通过 3=淘汰',
  `round_no`     INT             NOT NULL DEFAULT 1 COMMENT '同一候选人同一岗位的第几次投递',
  `is_current`   TINYINT         NOT NULL DEFAULT 1 COMMENT '是否当前有效投递：1=当前流程 0=历史流程',
  `applied_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `active_key`   TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_current` = 1 AND `status` <> 3 THEN 1 ELSE NULL END
  ) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_active_application` (`job_id`, `user_id`, `active_key`),
  KEY `idx_job_user_current_status` (`job_id`, `user_id`, `is_current`, `status`) COMMENT '查询候选人同岗位当前流程',
  KEY `idx_job_status_current` (`job_id`, `status`, `is_current`) COMMENT '岗位下按状态筛选当前投递',
  KEY `idx_job_current_applied` (`job_id`, `is_current`, `applied_at`, `id`),
  KEY `idx_user_applied` (`user_id`, `applied_at`, `id`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位投递关联表';
```

**说明：**
- `active_key` 是 MySQL 8 生成列，仅当 `is_current=1` 且 `status<>3`（非淘汰）时值=1，否则为 NULL
- `uk_active_application` 唯一索引利用 MySQL 中 NULL 不参与唯一比较的特性，保证同一候选人在同一岗位最多只有一条当前有效流程
- 淘汰后（status=3）的旧投递不会触发唯一冲突，候选人可以重新投递
- 已有数据库在添加 `uk_active_application` 前，必须先归档同一候选人同一岗位的重复当前有效投递，否则唯一索引创建会失败；迁移脚本 `logic-grpc-service/migrations/000003_add_idempotency_constraints.sql` 会保留最新一条并将旧记录 `is_current` 置为 0
- `logic-grpc-service/migrations/000006_add_performance_indexes.sql` 补充了游标分页和高频查询复合索引，其中 `idx_job_status_current` 用于岗位投递按状态筛选/统计

如果已有数据库仍保留旧的同岗位唯一投递约束，需要执行一次：

```sql
ALTER TABLE `applications` DROP INDEX `uk_job_user`;
ALTER TABLE `applications`
  ADD COLUMN `round_no` INT NOT NULL DEFAULT 1 COMMENT '同一候选人同一岗位的第几次投递' AFTER `status`,
  ADD COLUMN `is_current` TINYINT NOT NULL DEFAULT 1 COMMENT '是否当前有效投递：1=当前流程 0=历史流程' AFTER `round_no`;
CREATE INDEX `idx_job_user_current_status` ON `applications` (`job_id`, `user_id`, `is_current`, `status`);
```

如果迁移前已经产生过同一候选人同一岗位的多条投递，建议再执行一次历史轮次回填：

```sql
UPDATE `applications` a
JOIN (
  SELECT
    id,
    ROW_NUMBER() OVER (PARTITION BY job_id, user_id ORDER BY applied_at ASC, id ASC) AS fixed_round_no
  FROM `applications`
) ranked ON ranked.id = a.id
SET a.round_no = ranked.fixed_round_no;
```

---

### 6. ai_chat_sessions — AI 会话表

```sql
CREATE TABLE `ai_chat_sessions` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id`          BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `owner_role`     TINYINT         NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR',
  `owner_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID',
  `title`          VARCHAR(255)    NOT NULL COMMENT '会话标题',
  `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '绑定的投递记录ID，0表示普通数据问答',
  `created_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at`     DATETIME        NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_hr_updated_at` (`hr_id`, `updated_at`),
  KEY `idx_hr_deleted_updated` (`hr_id`, `deleted_at`, `updated_at`),
  KEY `idx_owner_deleted_updated` (`owner_role`, `owner_id`, `deleted_at`, `updated_at`),
  KEY `idx_application_id` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 会话表';
```

**说明：**
- `owner_role + owner_id` 是通用归属字段，替代单一 `hr_id` 角色模型，同时支持候选人 AI 会话
- 现有 HR 会话可通过 `owner_role=2, owner_id=hr_id` 回填
- 新建 HR 会话和消息写入时，后端仓储层会自动补齐 `owner_role=2, owner_id=hr_id`，保证新旧数据归属语义一致

已有数据库升级到软删除可执行：

```sql
ALTER TABLE `ai_chat_sessions`
  ADD COLUMN `deleted_at` DATETIME NULL COMMENT '软删除时间',
  ADD KEY `idx_hr_deleted_updated` (`hr_id`, `deleted_at`, `updated_at`);
```

### 7. ai_chat_history — AI 对话历史记录表

```sql
CREATE TABLE `ai_chat_history` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'AI 会话ID',
  `hr_id`      BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `owner_role` TINYINT         NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR',
  `owner_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID',
  `role`       VARCHAR(16)     NOT NULL COMMENT '消息角色：user / assistant',
  `content`    TEXT            NOT NULL COMMENT '消息内容',
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_session_id` (`session_id`),
  KEY `idx_hr_id_created` (`hr_id`, `created_at`),
  KEY `idx_owner_session_created` (`owner_role`, `owner_id`, `session_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 对话历史记录表';
```

**说明：**
- 每条消息通过 `session_id` 归属到独立会话，不同会话记录彼此隔离
- 候选人台账发起的 AI 分析会创建一个绑定 `application_id` 的新会话
- `owner_role + owner_id` 与 `ai_chat_sessions` 一致，支持候选人/HR 双角色查询

已有数据库升级到多会话 AI 助手可执行：

```sql
CREATE TABLE IF NOT EXISTS `ai_chat_sessions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `title` VARCHAR(255) NOT NULL COMMENT '会话标题',
  `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '绑定的投递记录ID，0表示普通数据问答',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_updated_at` (`hr_id`, `updated_at`),
  KEY `idx_application_id` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 会话表';

ALTER TABLE `ai_chat_history`
  ADD COLUMN `session_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'AI 会话ID' AFTER `id`,
  ADD INDEX `idx_session_id` (`session_id`);

INSERT INTO `ai_chat_sessions` (`hr_id`, `title`, `created_at`, `updated_at`)
SELECT `hr_id`, '历史对话', MIN(`created_at`), MAX(`created_at`)
FROM `ai_chat_history`
WHERE `session_id` = 0
GROUP BY `hr_id`;

UPDATE `ai_chat_history` h
JOIN `ai_chat_sessions` s
  ON s.hr_id = h.hr_id
 AND s.title = '历史对话'
SET h.session_id = s.id
WHERE h.session_id = 0;
```

---

### 8. ai_session_summaries — AI 会话滚动摘要表

```sql
CREATE TABLE `ai_session_summaries` (
  `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id`         BIGINT UNSIGNED NOT NULL COMMENT 'AI 会话ID',
  `hr_id`              BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `summary`            TEXT            NOT NULL COMMENT '会话摘要',
  `covered_message_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '摘要覆盖到的最大消息ID',
  `message_count`      INT             NOT NULL DEFAULT 0 COMMENT '摘要覆盖消息数量',
  `created_at`         DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`         DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session_id` (`session_id`),
  KEY `idx_hr_session` (`hr_id`, `session_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 会话滚动摘要表';
```

**说明：**
- 一个会话一条摘要，通过 `session_id` 唯一约束
- `covered_message_id` 记录摘要已覆盖到的最大消息 ID，用于判断增量更新范围
- 当会话消息数超过阈值（默认 30 条）时，异步滚动更新摘要

### 9. ai_tool_traces — AI 工具调用轨迹表

```sql
CREATE TABLE `ai_tool_traces` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `hr_id`          BIGINT UNSIGNED NOT NULL,
  `tool_call_id`   VARCHAR(128)    NOT NULL DEFAULT '',
  `tool_name`      VARCHAR(128)    NOT NULL,
  `arguments_json` JSON            NULL,
  `result_json`    JSON            NULL,
  `result_summary` TEXT            NULL,
  `status`         VARCHAR(32)     NOT NULL DEFAULT 'success' COMMENT 'success / error',
  `error_message`  TEXT            NULL,
  `created_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_session_created` (`session_id`, `created_at`),
  KEY `idx_hr_tool_created` (`hr_id`, `tool_name`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 工具调用轨迹表';
```

**说明：**
- 每次 LLM 工具调用都会写入一条轨迹记录
- `result_json` 保存原始工具返回，用于审计
- `result_summary` 为大型结果的简短摘要，避免大 JSON 长期进入 prompt
- 写入失败只打日志，不影响主流程

### 10. ai_memories — AI 长期记忆表

```sql
CREATE TABLE `ai_memories` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id`       BIGINT UNSIGNED NOT NULL COMMENT '记忆归属 HR',
  `scope_type`  VARCHAR(32)     NOT NULL COMMENT 'hr / job / application / candidate',
  `scope_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '作用域ID',
  `memory_type` VARCHAR(32)     NOT NULL COMMENT 'preference / fact / conclusion / warning',
  `content`     TEXT            NOT NULL COMMENT '记忆内容',
  `source`      VARCHAR(32)     NOT NULL DEFAULT 'agent' COMMENT 'user / tool / agent / system',
  `confidence`  DECIMAL(4,3)    NOT NULL DEFAULT 1.000,
  `expires_at`  DATETIME        NULL COMMENT '可选过期时间',
  `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_scope` (`hr_id`, `scope_type`, `scope_id`),
  KEY `idx_hr_type` (`hr_id`, `memory_type`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 长期记忆表';
```

**说明：**
- 第一阶段使用结构化检索（MySQL），暂不引入向量数据库
- `scope_type` 支持 hr（HR 级偏好）、job（岗位级）、application（投递级）、candidate（候选人级）
- `memory_type` 支持 preference（偏好）、fact（事实）、conclusion（分析结论）、warning（风险提示）
- 写入策略为保守规则触发：用户明确表达偏好、简历分析完成后、状态变更确认后
- 每轮最多注入 10 条记忆，总字符数不超过 1500

### 11. notifications — 站内通知表

```sql
CREATE TABLE `notifications` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '通知ID',
  `event_id`      VARCHAR(64)     NULL COMMENT '来源 outbox 事件ID，用于 MQ 重复投递幂等',
  `receiver_id`   BIGINT UNSIGNED NOT NULL COMMENT '接收用户ID',
  `receiver_role` TINYINT         NOT NULL COMMENT '接收者角色：1=候选人 2=HR',
  `type`          VARCHAR(64)     NOT NULL COMMENT '通知类型',
  `title`         VARCHAR(128)    NOT NULL COMMENT '通知标题',
  `content`       VARCHAR(512)    NOT NULL COMMENT '通知内容',
  `link`          VARCHAR(255)    DEFAULT NULL COMMENT '点击跳转路径',
  `biz_type`      VARCHAR(64)     DEFAULT NULL COMMENT '业务对象类型，如 application/job',
  `biz_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '业务对象ID',
  `is_read`       TINYINT         NOT NULL DEFAULT 0 COMMENT '是否已读：0=未读 1=已读',
  `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `read_at`       DATETIME        NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_notification_event_id` (`event_id`),
  UNIQUE KEY `uk_notification_once` (`receiver_id`, `receiver_role`, `biz_type`, `biz_id`, `type`),
  KEY `idx_receiver_read_created` (`receiver_id`, `receiver_role`, `is_read`, `created_at`),
  KEY `idx_receiver_created` (`receiver_id`, `receiver_role`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='站内通知表';
```

**说明：**
- 所有通知通过 `receiver_id + receiver_role` 归属到具体用户，查询时强制校验防止越权
- `type` 枚举：`new_application`（新投递）、`application_approved`（通过）、`application_rejected`（淘汰）
- `link` 仅为站内相对路径，不允许外部 URL
- 通知内容不包含手机号、OSS key、简历正文等敏感信息
- 业务触发点：候选人投递成功 → 通知 HR；HR 更新状态 → 通知候选人
- `event_id` 对应 outbox 事件，防止 RabbitMQ 至少一次投递造成重复通知
- `uk_notification_once` 兜底限制同一接收人、同一业务对象、同一通知类型只创建一次，避免多 consumer 并发重复写入

**已有数据库迁移：**

```sql
ALTER TABLE `notifications`
  ADD COLUMN `event_id` VARCHAR(64) NULL COMMENT '来源 outbox 事件ID，用于 MQ 重复投递幂等' AFTER `id`;

ALTER TABLE `notifications`
  DROP INDEX `idx_receiver_biz_type`,
  ADD UNIQUE KEY `uk_notification_event_id` (`event_id`),
  ADD UNIQUE KEY `uk_notification_once` (`receiver_id`, `receiver_role`, `biz_type`, `biz_id`, `type`);
```

---

## 表关联关系

```
users (id)
  ├── jobs.hr_id                    HR 发布岗位
  ├── candidate_profiles.user_id    候选人档案（1:1）
  ├── resumes.user_id               候选人简历（1:N）
  ├── applications.user_id          候选人投递记录（1:N）
  ├── ai_chat_sessions.hr_id        HR AI 会话（1:N）
  └── ai_chat_history.hr_id         HR 对话消息（1:N）

jobs (id)
  └── applications.job_id           岗位收到的投递（1:N）

resumes (id)
  └── applications.resume_id        投递使用的简历版本

ai_chat_sessions (id)
  └── ai_chat_history.session_id    会话消息（1:N）
```

### 12. event_outbox — 事务消息 outbox 表

```sql
CREATE TABLE `event_outbox` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(64) NOT NULL COMMENT '全局唯一事件ID',
  `event_type` VARCHAR(64) NOT NULL COMMENT 'notification.create / resume.parse',
  `aggregate_type` VARCHAR(64) NOT NULL COMMENT 'application / resume / notification',
  `aggregate_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `routing_key` VARCHAR(128) NOT NULL,
  `payload` JSON NOT NULL,
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=pending 1=published 2=dead 3=processing',
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME NULL,
  `last_error` TEXT NULL,
  `locked_at` DATETIME NULL,
  `locked_by` VARCHAR(128) NOT NULL DEFAULT '',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_event_id` (`event_id`),
  KEY `idx_status_next_retry` (`status`, `next_retry_at`, `locked_at`, `id`),
  KEY `idx_aggregate` (`aggregate_type`, `aggregate_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务消息 outbox 表';
```

**说明：**
- 业务操作成功后同步写入 outbox（同一数据库，无需分布式事务）
- outbox publisher 先将 pending/stale processing 事件 claim 为 processing，再发布到 RabbitMQ
- 发布成功后标记为 published，失败后恢复 pending、记录错误并设置下次重试时间
- `event_id` 全局唯一，防止重复发布
- `locked_at`/`locked_by` 用于多实例部署下的抢占锁和 stale processing 恢复
- 通知写入、简历解析等异步任务通过 outbox 解耦

---

## 初始化 SQL

```sql
CREATE DATABASE IF NOT EXISTS `recruitment`
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `recruitment`;

-- 依次执行上方各表的 CREATE TABLE 语句
```
