CREATE DATABASE IF NOT EXISTS `recruitment`
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `recruitment`;

SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username` VARCHAR(64) NOT NULL COMMENT '用户名（唯一）',
  `password` VARCHAR(255) NOT NULL COMMENT 'bcrypt 哈希密码',
  `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色：1=候选人 2=HR 3=HR管理员（Deprecated: 保留用于兼容，新授权逻辑使用 RBAC 表）',
  `email` VARCHAR(128) DEFAULT NULL COMMENT '邮箱（可选）',
  `account_type` VARCHAR(32) NOT NULL DEFAULT 'candidate' COMMENT '账号类型：candidate | staff | service',
  `status` VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT '账号状态：active | disabled | locked | pending',
  `token_version` INT NOT NULL DEFAULT 1 COMMENT '令牌版本号，权限变更时递增以失效旧令牌',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账号表';

CREATE TABLE IF NOT EXISTS `refresh_tokens` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '刷新令牌ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 users.id',
  `token_hash` CHAR(64) NOT NULL COMMENT '明文 refresh token 的 sha256 哈希',
  `family_id` VARCHAR(64) NOT NULL COMMENT '登录会话族ID，轮换时保持不变',
  `expires_at` DATETIME(3) NOT NULL COMMENT '刷新令牌过期时间',
  `revoked_at` DATETIME(3) NULL COMMENT '令牌被轮换或撤销的时间',
  `replaced_by_hash` CHAR(64) NULL COMMENT '替换它的新 refresh token 哈希',
  `reuse_detected_at` DATETIME(3) NULL COMMENT '已撤销令牌被复用的检测时间',
  `created_ip` VARCHAR(64) NULL COMMENT '创建该令牌的客户端IP',
  `created_user_agent` VARCHAR(255) NULL COMMENT '创建该令牌的 User-Agent',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_refresh_tokens_token_hash` (`token_hash`),
  KEY `idx_refresh_tokens_user_id` (`user_id`),
  KEY `idx_refresh_tokens_family_id` (`family_id`),
  KEY `idx_refresh_tokens_expires_at` (`expires_at`),
  CONSTRAINT `fk_refresh_tokens_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='刷新令牌存储表';

CREATE TABLE IF NOT EXISTS `jobs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '岗位ID',
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT '发布该岗位的 HR 用户ID',
  `title` VARCHAR(128) NOT NULL COMMENT '岗位名称',
  `department` VARCHAR(64) DEFAULT NULL COMMENT '所属部门',
  `location` VARCHAR(128) DEFAULT NULL COMMENT '工作地点',
  `salary_range` VARCHAR(64) DEFAULT NULL COMMENT '薪资范围，如 15k-25k',
  `description` TEXT DEFAULT NULL COMMENT '岗位详情描述',
  `requirements` TEXT DEFAULT NULL COMMENT '任职要求',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1=招募中 0=已下架',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_id` (`hr_id`),
  KEY `idx_status` (`status`),
  KEY `idx_status_created_id` (`status`, `created_at`, `id`),
  KEY `idx_hr_created_id` (`hr_id`, `created_at`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='招聘岗位表';

CREATE TABLE IF NOT EXISTS `candidate_profiles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 users.id，一人一条档案',
  `real_name` VARCHAR(64) DEFAULT NULL COMMENT '真实姓名',
  `phone` VARCHAR(20) DEFAULT NULL COMMENT '联系电话',
  `education` VARCHAR(32) DEFAULT NULL COMMENT '最高学历，如：本科、硕士、博士',
  `school` VARCHAR(128) DEFAULT NULL COMMENT '毕业院校',
  `work_experience` TEXT DEFAULT NULL COMMENT '工作/项目经历（富文本或 JSON）',
  `skills` VARCHAR(512) DEFAULT NULL COMMENT '核心技能标签，逗号分隔',
  `is_complete` TINYINT NOT NULL DEFAULT 0 COMMENT '档案是否完整：0=不完整 1=完整',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人结构化档案表';

CREATE TABLE IF NOT EXISTS `resumes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人用户ID',
  `oss_key` VARCHAR(512) NOT NULL COMMENT 'OSS 对象 Key（相对路径）',
  `file_name` VARCHAR(255) NOT NULL COMMENT '原始文件名',
  `file_type` VARCHAR(16) NOT NULL COMMENT '文件类型：pdf / doc / docx',
  `file_size` INT UNSIGNED DEFAULT NULL COMMENT '文件大小（字节）',
  `parsed_text` MEDIUMTEXT NULL COMMENT 'PDF 简历解析文本',
  `parsed_at` DATETIME NULL COMMENT '简历解析时间',
  `is_valid` TINYINT NOT NULL DEFAULT 1 COMMENT '是否有效：1=有效 0=已失效',
  `uploaded_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `valid_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_valid` = 1 THEN 1 ELSE NULL END
  ) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_valid_resume` (`user_id`, `valid_key`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_is_valid` (`is_valid`),
  KEY `idx_user_uploaded` (`user_id`, `uploaded_at`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='简历 OSS 存储记录表';

CREATE TABLE IF NOT EXISTS `applications` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id` BIGINT UNSIGNED NOT NULL COMMENT '投递的岗位ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '投递的候选人用户ID',
  `resume_id` BIGINT UNSIGNED NOT NULL COMMENT '投递时使用的简历ID',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '投递状态（旧数值）：0=待查看 1=已查看 2=通过 3=淘汰（Phase 1 迁移兼容保留）',
  `status_key` VARCHAR(64) NOT NULL DEFAULT 'applied' COMMENT '投递状态键（Phase 1 新状态机）',
  `round_no` INT NOT NULL DEFAULT 1 COMMENT '同一候选人同一岗位的第几次投递',
  `is_current` TINYINT NOT NULL DEFAULT 1 COMMENT '是否当前有效投递：1=当前流程 0=历史流程',
  `applied_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `active_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_current` = 1 AND `status_key` NOT IN ('rejected', 'withdrawn', 'offer_rejected', 'hired') THEN 1 ELSE NULL END
  ) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_active_application` (`job_id`, `user_id`, `active_key`),
  KEY `idx_job_user_current_status` (`job_id`, `user_id`, `is_current`, `status`),
  KEY `idx_job_status_current` (`job_id`, `status`, `is_current`),
  KEY `idx_job_current_applied` (`job_id`, `is_current`, `applied_at`, `id`),
  KEY `idx_user_applied` (`user_id`, `applied_at`, `id`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status_key` (`status_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位投递关联表';

-- ── Phase 1: 投递状态变更审计表 ─────────────────────────────────────────
-- 记录每次状态变更的 actor、前后状态、原因和时间戳。

CREATE TABLE IF NOT EXISTS `application_status_transitions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `from_status` VARCHAR(64) NOT NULL COMMENT '变更前状态 key',
  `to_status` VARCHAR(64) NOT NULL COMMENT '变更后状态 key',
  `actor_user_id` BIGINT UNSIGNED NOT NULL COMMENT '操作人用户ID',
  `actor_account_type` VARCHAR(32) NOT NULL COMMENT '操作人账号类型：candidate / staff / service',
  `reason` VARCHAR(512) DEFAULT NULL COMMENT '变更原因（HR 操作时必填）',
  `metadata_json` TEXT DEFAULT NULL COMMENT '附加元数据 JSON',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_transition_app` (`application_id`),
  KEY `idx_transition_created` (`application_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='投递状态变更审计记录表';

CREATE TABLE IF NOT EXISTS `ai_chat_sessions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `owner_role` TINYINT NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR',
  `owner_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID',
  `title` VARCHAR(255) NOT NULL COMMENT '会话标题',
  `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '绑定的投递记录ID，0表示普通数据问答',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_hr_updated_at` (`hr_id`, `updated_at`),
  KEY `idx_hr_deleted_updated` (`hr_id`, `deleted_at`, `updated_at`),
  KEY `idx_owner_deleted_updated` (`owner_role`, `owner_id`, `deleted_at`, `updated_at`),
  KEY `idx_application_id` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 会话表';

CREATE TABLE IF NOT EXISTS `ai_chat_history` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'AI 会话ID',
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `owner_role` TINYINT NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR',
  `owner_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID',
  `role` VARCHAR(16) NOT NULL COMMENT '消息角色：user / assistant',
  `content` TEXT NOT NULL COMMENT '消息内容',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_session_id` (`session_id`),
  KEY `idx_session_created_id` (`session_id`, `created_at`, `id`),
  KEY `idx_hr_id_created` (`hr_id`, `created_at`),
  KEY `idx_owner_session_created` (`owner_role`, `owner_id`, `session_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 对话历史记录表';

CREATE TABLE IF NOT EXISTS `ai_session_summaries` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL COMMENT 'AI 会话ID',
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `summary` TEXT NOT NULL COMMENT '会话摘要',
  `covered_message_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '摘要覆盖到的最大消息ID',
  `message_count` INT NOT NULL DEFAULT 0 COMMENT '摘要覆盖消息数量',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session_id` (`session_id`),
  KEY `idx_hr_session` (`hr_id`, `session_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 会话滚动摘要表';

CREATE TABLE IF NOT EXISTS `ai_tool_traces` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `hr_id` BIGINT UNSIGNED NOT NULL,
  `tool_call_id` VARCHAR(128) NOT NULL DEFAULT '',
  `tool_name` VARCHAR(128) NOT NULL,
  `arguments_json` JSON NULL,
  `result_json` JSON NULL,
  `result_summary` TEXT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'success' COMMENT 'success / error',
  `error_message` TEXT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_session_created` (`session_id`, `created_at`),
  KEY `idx_hr_tool_created` (`hr_id`, `tool_name`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 工具调用轨迹表';

CREATE TABLE IF NOT EXISTS `ai_memories` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT '记忆归属 HR',
  `scope_type` VARCHAR(32) NOT NULL COMMENT 'hr / job / application / candidate',
  `scope_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '作用域ID',
  `memory_type` VARCHAR(32) NOT NULL COMMENT 'preference / fact / conclusion / warning',
  `content` TEXT NOT NULL COMMENT '记忆内容',
  `source` VARCHAR(32) NOT NULL DEFAULT 'agent' COMMENT 'user / tool / agent / system',
  `confidence` DECIMAL(4,3) NOT NULL DEFAULT 1.000,
  `expires_at` DATETIME NULL COMMENT '可选过期时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_scope` (`hr_id`, `scope_type`, `scope_id`),
  KEY `idx_hr_type` (`hr_id`, `memory_type`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 长期记忆表';

CREATE TABLE IF NOT EXISTS `notifications` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '通知ID',
  `event_id` VARCHAR(64) NULL COMMENT '来源 outbox 事件ID，用于 MQ 重复投递幂等',
  `receiver_id` BIGINT UNSIGNED NOT NULL COMMENT '接收用户ID',
  `receiver_role` TINYINT NOT NULL COMMENT '接收者角色（废弃，用 receiver_account_type）',
  `receiver_account_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '接收者账户类型：candidate/staff',
  `type` VARCHAR(64) NOT NULL COMMENT '通知类型',
  `title` VARCHAR(128) NOT NULL COMMENT '通知标题',
  `content` VARCHAR(512) NOT NULL COMMENT '通知内容',
  `link` VARCHAR(255) DEFAULT NULL COMMENT '点击跳转路径',
  `biz_type` VARCHAR(64) DEFAULT NULL COMMENT '业务对象类型，如 application/job',
  `biz_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '业务对象ID',
  `is_read` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读：0=未读 1=已读',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `read_at` DATETIME NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_notification_event_id` (`event_id`),
  UNIQUE KEY `uk_notification_once` (`receiver_id`, `receiver_account_type`, `biz_type`, `biz_id`, `type`),
  KEY `idx_receiver_read_created` (`receiver_id`, `receiver_account_type`, `is_read`, `created_at`),
  KEY `idx_receiver_read_created_id` (`receiver_id`, `receiver_account_type`, `is_read`, `created_at`, `id`),
  KEY `idx_receiver_created` (`receiver_id`, `receiver_account_type`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='站内通知表';

CREATE TABLE IF NOT EXISTS `event_outbox` (
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

CREATE TABLE IF NOT EXISTS `invite_codes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL COMMENT '邀请码（随机生成）',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建该邀请码的管理员用户ID',
  `expires_at` DATETIME NULL COMMENT '过期时间，NULL 表示永不过期',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '是否有效：1=有效 0=已撤销',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_created_by` (`created_by`),
  KEY `idx_code_active_expires` (`code`, `is_active`, `expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='HR 注册邀请码表';

CREATE TABLE IF NOT EXISTS `third_party_usage_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '用户ID，0表示匿名/系统',
  `role` TINYINT NOT NULL DEFAULT 0 COMMENT '用户角色：0未知 1候选人 2HR 3管理员',
  `service_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '服务类型：ai_chat/ai_analyze/oss_presign/oss_confirm',
  `endpoint` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '调用的接口路径',
  `provider` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '第三方服务商：dashscope/tencent_cos/aliyun_oss',
  `model` VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'AI模型名称',
  `request_chars` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '请求字符数',
  `response_chars` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '响应字符数',
  `estimated_tokens` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '估算token消耗',
  `object_key` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'OSS对象key',
  `object_size` BIGINT NOT NULL DEFAULT 0 COMMENT 'OSS对象大小(字节)',
  `status` VARCHAR(16) NOT NULL DEFAULT 'ok' COMMENT '调用结果：ok/error/timeout/rate_limited',
  `error_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '错误码',
  `cost_ms` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '调用耗时(毫秒)',
  `request_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求追踪ID',
  `ip` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_created` (`user_id`, `created_at`),
  KEY `idx_service_created` (`service_type`, `created_at`),
  KEY `idx_request_id` (`request_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='第三方服务调用审计日志';

-- ══════════════════════════════════════════════════════════════════════
-- RBAC 角色与权限系统 (Role-Based Access Control)
-- ══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS `roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_key` VARCHAR(64) NOT NULL COMMENT '角色唯一标识：candidate / recruiter / recruiting_admin / system_admin / interviewer',
  `name` VARCHAR(128) NOT NULL COMMENT '角色中文名称',
  `description` VARCHAR(512) DEFAULT NULL COMMENT '角色描述',
  `is_system` TINYINT NOT NULL DEFAULT 1 COMMENT '是否系统角色：1=系统预置 0=自定义',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_roles_role_key` (`role_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色定义表';

CREATE TABLE IF NOT EXISTS `permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `permission_key` VARCHAR(128) NOT NULL COMMENT '权限唯一标识，如 job.read / application.status.update',
  `resource` VARCHAR(64) NOT NULL COMMENT '资源域：job / application / admin / audit 等',
  `action` VARCHAR(64) NOT NULL COMMENT '操作：read / create / update / delete / manage / use',
  `description` VARCHAR(512) DEFAULT NULL COMMENT '权限说明',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_permissions_permission_key` (`permission_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='权限定义表';

CREATE TABLE IF NOT EXISTS `role_permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 roles.id',
  `permission_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 permissions.id',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_permission` (`role_id`, `permission_id`),
  KEY `idx_permission_id` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色-权限关联表';

CREATE TABLE IF NOT EXISTS `user_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 users.id',
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 roles.id',
  `assigned_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '分配人用户ID',
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '分配时间',
  `revoked_at` DATETIME DEFAULT NULL COMMENT '撤销时间，NULL 表示当前有效',
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED COMMENT '当前有效记录唯一约束键',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_role_active` (`user_id`, `role_id`, `active_key`),
  KEY `idx_user_roles_user` (`user_id`),
  KEY `idx_user_roles_role` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色分配表';

CREATE TABLE IF NOT EXISTS `user_data_scopes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 users.id',
  `scope_key` VARCHAR(64) NOT NULL COMMENT '数据范围：self / own_jobs / department / location / recruiting_all / system_all',
  `resource_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '限定资源类型，空=全局适用',
  `resource_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '限定资源ID，0=不限定具体资源',
  `assigned_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '分配人用户ID',
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '分配时间',
  `revoked_at` DATETIME DEFAULT NULL COMMENT '撤销时间，NULL 表示当前有效',
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED COMMENT '当前有效记录唯一约束键',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_scope_active` (`user_id`, `scope_key`, `resource_type`, `resource_id`, `active_key`),
  KEY `idx_user_scope` (`user_id`, `scope_key`, `revoked_at`),
  KEY `idx_scope_resource` (`scope_key`, `resource_type`, `resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户数据范围表';

CREATE TABLE IF NOT EXISTS `authorization_audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `actor_user_id` BIGINT UNSIGNED NOT NULL COMMENT '操作人用户ID',
  `actor_roles` VARCHAR(512) NOT NULL COMMENT '操作人当前角色，逗号分隔',
  `permission_key` VARCHAR(128) NOT NULL COMMENT '被检查的权限 key',
  `resource_type` VARCHAR(64) NOT NULL COMMENT '目标资源类型',
  `resource_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '目标资源ID',
  `decision` VARCHAR(16) NOT NULL COMMENT '授权决策：allowed | denied',
  `reason` VARCHAR(512) DEFAULT NULL COMMENT '拒绝原因或补充说明',
  `request_id` VARCHAR(64) DEFAULT NULL COMMENT '请求追踪ID',
  `client_ip` VARCHAR(64) DEFAULT NULL COMMENT '客户端IP',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_actor_created` (`actor_user_id`, `created_at`),
  KEY `idx_permission_created` (`permission_key`, `created_at`),
  KEY `idx_decision_created` (`decision`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='授权审计日志表';

-- ── 面试安排表 ────────────────────────────────────────────────────────
-- 用于面试官 (interviewer) 角色的 assigned_interviews 数据范围匹配。
-- 一个 application 可能对应多轮面试，每轮可指派不同面试官。

CREATE TABLE IF NOT EXISTS `interview_schedules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `interviewer_id` BIGINT UNSIGNED NOT NULL COMMENT '面试官用户ID（users.id）',
  `round_no` INT NOT NULL DEFAULT 1 COMMENT '面试轮次：1=初试 2=复试 ...',
  `title` VARCHAR(128) DEFAULT NULL COMMENT '面试标题，如 初试/复试/终面',
  `mode` VARCHAR(32) DEFAULT NULL COMMENT '面试模式：video / phone / onsite',
  `meeting_url` VARCHAR(512) DEFAULT NULL COMMENT '视频会议链接',
  `location` VARCHAR(256) DEFAULT NULL COMMENT '面试地点（线下）',
  `duration_minutes` INT DEFAULT NULL COMMENT '面试时长（分钟）',
  `candidate_note` VARCHAR(1024) DEFAULT NULL COMMENT '给候选人的注意事项',
  `internal_note` VARCHAR(1024) DEFAULT NULL COMMENT '内部备注（候选人不可见）',
  `cancel_reason` VARCHAR(512) DEFAULT NULL COMMENT '取消原因',
  `scheduled_at` DATETIME DEFAULT NULL COMMENT '计划面试时间',
  `status` VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '面试状态：pending / scheduled / completed / cancelled',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建人用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间，NULL 表示有效',
  PRIMARY KEY (`id`),
  KEY `idx_interviewer_deleted` (`interviewer_id`, `deleted_at`),
  KEY `idx_application_deleted` (`application_id`, `deleted_at`),
  KEY `idx_interviewer_app` (`interviewer_id`, `application_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='面试安排表';

-- ── 面试反馈表 ────────────────────────────────────────────────────────
-- 记录面试官对面试的反馈评价，提交后不可修改（有审核更正路径）。

CREATE TABLE IF NOT EXISTS `interview_feedback` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `interview_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 interview_schedules.id',
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `interviewer_id` BIGINT UNSIGNED NOT NULL COMMENT '面试官用户ID',
  `recommendation` VARCHAR(32) DEFAULT NULL COMMENT '推荐结论：positive / negative / pending',
  `score` INT DEFAULT NULL COMMENT '评分（0-10）',
  `dimension_scores_json` TEXT DEFAULT NULL COMMENT '维度评分 JSON，如 {"communication":4,"technical":5}',
  `comments` TEXT DEFAULT NULL COMMENT '面试评语',
  `submitted_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '提交时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_interview_feedback_once` (`interview_id`, `application_id`, `interviewer_id`),
  KEY `idx_feedback_interview` (`interview_id`),
  KEY `idx_feedback_interviewer` (`interviewer_id`),
  KEY `idx_feedback_application` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='面试反馈表';

-- ── Offer 表 ─────────────────────────────────────────────────────────────
-- 记录 Offer 的创建、发送、决策全过程。

CREATE TABLE IF NOT EXISTS `offers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Offer ID',
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `candidate_user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人用户ID',
  `job_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 jobs.id',
  `status` VARCHAR(32) NOT NULL DEFAULT 'draft' COMMENT 'Offer状态：draft / sent / accepted / rejected / withdrawn',
  `title` VARCHAR(128) NOT NULL COMMENT 'Offer职位名称',
  `salary_range` VARCHAR(64) DEFAULT NULL COMMENT '薪资范围',
  `level` VARCHAR(64) DEFAULT NULL COMMENT '职级',
  `work_location` VARCHAR(128) DEFAULT NULL COMMENT '工作地点',
  `start_date` VARCHAR(32) DEFAULT NULL COMMENT '预计入职日期',
  `expires_at` DATETIME DEFAULT NULL COMMENT 'Offer过期时间',
  `terms_json` TEXT DEFAULT NULL COMMENT 'Offer条款JSON（起草时填写）',
  `sent_snapshot_json` TEXT DEFAULT NULL COMMENT '发送时的快照JSON（发送时冻结）',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建人用户ID',
  `sent_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '发送人用户ID',
  `decided_at` DATETIME DEFAULT NULL COMMENT '候选人决策时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_offer_application` (`application_id`),
  KEY `idx_offer_candidate` (`candidate_user_id`),
  KEY `idx_offer_job` (`job_id`),
  KEY `idx_offer_status` (`status`),
  KEY `idx_offer_created_by` (`created_by`),
  KEY `idx_offer_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Offer表';

CREATE TABLE IF NOT EXISTS `offer_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `offer_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 offers.id',
  `event_type` VARCHAR(64) NOT NULL COMMENT '事件类型：created / updated / sent / withdrawn / accepted / rejected / expired',
  `actor_user_id` BIGINT UNSIGNED NOT NULL COMMENT '操作用户ID',
  `actor_account_type` VARCHAR(32) NOT NULL COMMENT '操作人账号类型：candidate / staff / service',
  `reason` VARCHAR(512) DEFAULT NULL COMMENT '操作原因说明',
  `metadata_json` TEXT DEFAULT NULL COMMENT '附加元数据JSON',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_offer_event_offer` (`offer_id`),
  KEY `idx_offer_event_type` (`event_type`),
  KEY `idx_offer_event_created` (`offer_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Offer事件审计表';

-- ══════════════════════════════════════════════════════════════════════
-- RBAC 种子数据
-- ══════════════════════════════════════════════════════════════════════

-- ── 角色 ──────────────────────────────────────────────────────────────

INSERT INTO `roles` (`role_key`, `name`, `description`, `is_system`) VALUES
  ('candidate',        '求职者',     '外部求职者，管理个人资料、简历、投递和AI会话', 1),
  ('recruiter',        '招聘专员',   '负责岗位发布、候选人流程、面试安排和HR AI使用', 1),
  ('recruiting_admin', '招聘管理员', '管理招聘配置、邀请码、部门、地点、用户角色分配', 1),
  ('system_admin',     '系统管理员', '管理平台安全配置、角色目录、权限目录、审计日志', 1),
  ('interviewer',      '面试官',     '查看被分配的面试并提交反馈', 1)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

-- ── 权限 ──────────────────────────────────────────────────────────────

INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`) VALUES
  ('auth.session.read',              'auth',        'read',   '查看自己的会话'),
  ('candidate.profile.manage',       'candidate',   'manage', '管理候选人个人信息'),
  ('candidate.resume.manage',        'candidate',   'manage', '管理个人简历'),
  ('candidate.application.manage',   'candidate',   'manage', '创建和查看个人投递'),
  ('job.read',                       'job',         'read',   '查看HR可见的岗位数据'),
  ('job.create',                     'job',         'create', '创建岗位'),
  ('job.update',                     'job',         'update', '编辑岗位'),
  ('job.publish',                    'job',         'publish','上下线岗位'),
  ('application.read',               'application', 'read',   '查看范围内的候选人台账'),
  ('application.status.update',      'application', 'update', '变更候选人状态'),
  ('interview.read',                 'interview',   'read',   '查看分配的面试'),
  ('interview.schedule',             'interview',   'manage', '安排/修改/取消面试'),
  ('interview.feedback.submit',      'interview',   'create', '提交面试反馈'),
  ('notification.read',              'notification','read',   '查看自己的通知'),
  ('ai.hr.use',                      'ai',          'use',    '使用HR AI助手'),
  ('ai.candidate.use',               'ai',          'use',    '使用候选人AI助手'),
  ('admin.invite.manage',            'admin',       'manage', '管理邀请码'),
  ('admin.department.manage',        'admin',       'manage', '管理部门及部门地点关联'),
  ('admin.location.manage',          'admin',       'manage', '管理工作地点'),
  ('admin.user.manage',              'admin',       'manage', '创建/修改/禁用员工账号'),
  ('admin.role.manage',              'admin',       'manage', '管理角色目录和权限分配'),
  ('audit.usage.read',               'audit',       'read',   '查看第三方/AI使用日志'),
  ('audit.security.read',            'audit',       'read',   '查看授权和安全审计事件'),
  ('system.config.manage',           'system',      'manage', '管理平台安全配置'),
  ('offer.read',                     'offer',       'read',   '查看Offer'),
  ('offer.manage',                   'offer',       'manage', '创建/编辑/撤回Offer'),
  ('offer.send',                     'offer',       'send',   '发送Offer（快照条款）'),
  ('offer.decision.manage',          'offer',       'manage', '候选人接受/拒绝Offer'),
  ('collaboration.note.read',        'collaboration','read',   '查看候选人内部备注'),
  ('collaboration.note.create',      'collaboration','create', '创建候选人内部备注'),
  ('collaboration.tag.manage',       'collaboration','manage', '管理候选人标签'),
  ('collaboration.task.manage',      'collaboration','manage', '管理跟进任务')
ON DUPLICATE KEY UPDATE `resource` = VALUES(`resource`), `action` = VALUES(`action`), `description` = VALUES(`description`);

-- ── 角色-权限映射 ──────────────────────────────────────────────────────

-- Candidate
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'candidate' AND p.permission_key IN (
    'auth.session.read', 'candidate.profile.manage', 'candidate.resume.manage',
    'candidate.application.manage', 'notification.read', 'ai.candidate.use',
    'offer.decision.manage'
  );

-- Recruiter
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'recruiter' AND p.permission_key IN (
    'auth.session.read', 'job.read', 'job.create', 'job.update', 'job.publish',
    'application.read', 'application.status.update',
    'interview.read', 'interview.schedule', 'interview.feedback.submit',
    'notification.read', 'ai.hr.use',
    'offer.read', 'offer.manage', 'offer.send',
    'collaboration.note.read', 'collaboration.note.create',
    'collaboration.tag.manage', 'collaboration.task.manage'
  );

-- Recruiting Admin (no recruiter workflow permissions by default)
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'recruiting_admin' AND p.permission_key IN (
    'auth.session.read', 'job.read', 'application.read',
    'notification.read', 'ai.hr.use',
    'admin.invite.manage', 'admin.department.manage', 'admin.location.manage',
    'admin.user.manage', 'admin.role.manage', 'audit.usage.read',
    'collaboration.note.read', 'collaboration.note.create',
    'collaboration.tag.manage', 'collaboration.task.manage'
  );

-- System Admin (platform-level only, no recruiting workflow)
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'system_admin' AND p.permission_key IN (
    'auth.session.read', 'admin.user.manage', 'admin.role.manage',
    'audit.usage.read', 'audit.security.read', 'system.config.manage'
  );

-- Interviewer
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'interviewer' AND p.permission_key IN (
    'auth.session.read', 'interview.read', 'interview.feedback.submit', 'notification.read'
  );

-- ── 默认管理员账号 (admin / 123456) ────────────────────────────────────
-- account_type=staff, role=3（兼容旧逻辑）, 分配 recruiting_admin + recruiter 角色

INSERT INTO `users` (`username`, `password`, `role`, `account_type`, `status`, `token_version`) VALUES
  ('admin', '$2b$10$qxetp5jT6U7U5dd/k1G/v.qJ.FDlqFLWO3LHKv8Kwt6c49VXhLhOy', 3, 'staff', 'active', 1)
ON DUPLICATE KEY UPDATE `account_type` = 'staff';

-- 给 admin 分配 recruiting_admin 角色
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`, `assigned_at`)
  SELECT u.id, r.id, NOW()
  FROM `users` u, `roles` r
  WHERE u.username = 'admin' AND r.role_key = 'recruiting_admin';

-- 给 admin 分配 recruiter 角色（显式授予，不依赖继承）
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`, `assigned_at`)
  SELECT u.id, r.id, NOW()
  FROM `users` u, `roles` r
  WHERE u.username = 'admin' AND r.role_key = 'recruiter';

-- 给 admin 分配 recruiting_all 数据范围
INSERT IGNORE INTO `user_data_scopes` (`user_id`, `scope_key`, `resource_type`, `resource_id`, `assigned_at`)
  SELECT u.id, 'recruiting_all', '', 0, NOW()
  FROM `users` u
  WHERE u.username = 'admin';

-- 给 admin 分配 system_admin 角色（系统管理权限，如审计日志查看、系统配置管理等）
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`, `assigned_at`)
  SELECT u.id, r.id, NOW()
  FROM `users` u, `roles` r
  WHERE u.username = 'admin' AND r.role_key = 'system_admin';

-- ── 部门基础数据表 ──────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS `departments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门ID',
  `parent_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父部门ID，0表示根部门',
  `name` VARCHAR(64) NOT NULL COMMENT '部门名称',
  `full_name` VARCHAR(255) NOT NULL COMMENT '完整部门路径，如 技术研发部/后端组',
  `path` VARCHAR(512) NOT NULL COMMENT 'ID路径，如 /1/8/13/',
  `depth` INT NOT NULL DEFAULT 1 COMMENT '层级深度，根节点为1',
  `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 0停用',
  `inherit_locations` TINYINT NOT NULL DEFAULT 1 COMMENT '是否继承上级部门地点配置：1=继承 0=自定义',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建管理员ID',
  `updated_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '最后更新管理员ID',
  `deleted_at` DATETIME DEFAULT NULL COMMENT '逻辑删除时间',
  `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_department_parent_name` (`parent_id`, `name`),
  KEY `idx_department_parent_sort` (`parent_id`, `sort_order`, `id`),
  KEY `idx_department_active` (`is_active`),
  KEY `idx_department_path` (`path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位部门基础数据表';

-- ── 岗位地点基础数据表 ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS `job_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '地点ID',
  `name` VARCHAR(128) NOT NULL COMMENT '地点名称',
  `code` VARCHAR(64) DEFAULT NULL COMMENT '地点编码，可选',
  `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 0停用',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建管理员ID',
  `updated_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '最后更新管理员ID',
  `deleted_at` DATETIME DEFAULT NULL COMMENT '逻辑删除时间',
  `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_job_location_name` (`name`),
  KEY `idx_job_location_active_sort` (`is_active`, `sort_order`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位地点基础数据表';

-- ── 部门-地点关联表 ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS `department_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门地点关联ID',
  `department_id` BIGINT UNSIGNED NOT NULL COMMENT '部门ID',
  `location_id` BIGINT UNSIGNED NOT NULL COMMENT '地点ID',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 0停用',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建管理员ID',
  `updated_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '最后更新管理员ID',
  `deleted_at` DATETIME DEFAULT NULL COMMENT '逻辑删除时间',
  `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_department_location` (`department_id`, `location_id`),
  KEY `idx_department_active` (`department_id`, `is_active`, `deleted_at`),
  KEY `idx_location_active` (`location_id`, `is_active`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门可用地点关联表';

-- ── 扩展 jobs 表，增加外键字段 ─────────────────────────────────────────

ALTER TABLE `jobs`
  ADD COLUMN `department_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 departments.id' AFTER `department`,
  ADD COLUMN `location_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 job_locations.id' AFTER `location`,
  ADD KEY `idx_department_id` (`department_id`),
  ADD KEY `idx_location_id` (`location_id`);

-- ── 初始化部门数据 ─────────────────────────────────────────────────────

INSERT INTO `departments` (`parent_id`, `name`, `full_name`, `path`, `depth`, `sort_order`, `is_active`, `inherit_locations`) VALUES
  (0, '技术研发部', '技术研发部', '/1/', 1, 1, 1, 0),
  (0, '产品部',     '产品部',     '/2/', 1, 2, 1, 0),
  (0, '设计部',     '设计部',     '/3/', 1, 3, 1, 0),
  (0, '市场部',     '市场部',     '/4/', 1, 4, 1, 0),
  (0, '销售部',     '销售部',     '/5/', 1, 5, 1, 0),
  (0, '运营部',     '运营部',     '/6/', 1, 6, 1, 0),
  (0, '人力资源部', '人力资源部', '/7/', 1, 7, 1, 0),
  (0, '财务部',     '财务部',     '/8/', 1, 8, 1, 0),
  (0, '客户成功部', '客户成功部', '/9/', 1, 9, 1, 0);

-- ── 初始化地点数据 ─────────────────────────────────────────────────────

INSERT INTO `job_locations` (`name`, `code`, `sort_order`, `is_active`) VALUES
  ('北京', 'beijing',  1, 1),
  ('上海', 'shanghai', 2, 1),
  ('广州', 'guangzhou',3, 1),
  ('深圳', 'shenzhen', 4, 1),
  ('杭州', 'hangzhou', 5, 1),
  ('成都', 'chengdu',  6, 1),
  ('武汉', 'wuhan',    7, 1),
  ('西安', 'xian',     8, 1),
  ('南京', 'nanjing',  9, 1),
  ('远程', 'remote',  10, 1);

-- ── 初始化部门可用地点数据 ─────────────────────────────────────────────
-- 使用 CTE 为每个根部门分配 3 个伪随机地点，与 v9 迁移逻辑一致。

INSERT INTO `department_locations` (`department_id`, `location_id`, `is_active`)
WITH active_locations AS (
  SELECT
    id,
    ROW_NUMBER() OVER (ORDER BY sort_order, id) AS rn,
    COUNT(*) OVER () AS total_count
  FROM job_locations
  WHERE is_active = 1
    AND deleted_at IS NULL
),
root_targets AS (
  SELECT d.id AS department_id, l.id AS location_id
  FROM departments d
  JOIN active_locations l ON (
    l.rn = MOD(d.id - 1, l.total_count) + 1
    OR l.rn = MOD(d.id + 2, l.total_count) + 1
    OR l.rn = MOD(d.id + 5, l.total_count) + 1
  )
  WHERE d.deleted_at IS NULL
    AND d.parent_id = 0
    AND NOT EXISTS (
      SELECT 1
      FROM department_locations dl
      WHERE dl.department_id = d.id
        AND dl.deleted_at IS NULL
  )
)
SELECT department_id, location_id, 1
FROM root_targets
ON DUPLICATE KEY UPDATE
  is_active = 1,
  deleted_at = NULL,
  deleted_by = NULL;

-- ══════════════════════════════════════════════════════════════════════
-- Phase 4: Candidate Collaboration
-- ══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS `candidate_notes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '备注ID',
  `candidate_user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人用户ID',
  `application_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联投递ID（可选）',
  `author_user_id` BIGINT UNSIGNED NOT NULL COMMENT '创建人用户ID',
  `content` TEXT NOT NULL COMMENT '备注内容',
  `visibility` VARCHAR(32) NOT NULL DEFAULT 'internal' COMMENT '可见性：internal(内部可见)',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_note_candidate` (`candidate_user_id`),
  KEY `idx_note_application` (`application_id`),
  KEY `idx_note_author` (`author_user_id`),
  KEY `idx_note_created` (`candidate_user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人内部备注表';

CREATE TABLE IF NOT EXISTS `candidate_tags` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '标签ID',
  `name` VARCHAR(64) NOT NULL COMMENT '标签名称',
  `color` VARCHAR(16) DEFAULT '#409eff' COMMENT '标签颜色',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建人用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tag_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人标签定义表';

CREATE TABLE IF NOT EXISTS `candidate_tag_assignments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '分配ID',
  `tag_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 candidate_tags.id',
  `candidate_user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人用户ID',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '分配人用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tag_candidate` (`tag_id`, `candidate_user_id`),
  KEY `idx_tag_assignment_candidate` (`candidate_user_id`),
  KEY `idx_tag_assignment_tag` (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人标签分配表';

CREATE TABLE IF NOT EXISTS `follow_up_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '任务ID',
  `candidate_user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联候选人用户ID',
  `application_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联投递ID（可选）',
  `assignee_user_id` BIGINT UNSIGNED NOT NULL COMMENT '负责人用户ID',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建人用户ID',
  `title` VARCHAR(256) NOT NULL COMMENT '任务标题',
  `description` TEXT DEFAULT NULL COMMENT '任务描述',
  `due_at` DATETIME DEFAULT NULL COMMENT '截止时间',
  `status` VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '任务状态：pending / completed',
  `completed_at` DATETIME DEFAULT NULL COMMENT '完成时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_task_candidate` (`candidate_user_id`),
  KEY `idx_task_assignee` (`assignee_user_id`),
  KEY `idx_task_application` (`application_id`),
  KEY `idx_task_status` (`status`),
  KEY `idx_task_due` (`due_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='跟进任务表';

-- ══════════════════════════════════════════════════════════════════════
-- Phase 6: AI Usage Auth Context
-- ══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS `ai_usage_auth_contexts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `usage_log_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 third_party_usage_logs.id',
  `actor_user_id` BIGINT UNSIGNED NOT NULL COMMENT '操作人用户ID',
  `account_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '账号类型：candidate / staff / service',
  `role_keys` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '逗号分隔的角色key列表',
  `permission_key` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '触发该次操作的权限key',
  `scope_keys` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '逗号分隔的数据范围key列表',
  `resource_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源类型，如 ai / application / job',
  `resource_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '资源ID，0表示全局',
  `decision` VARCHAR(32) NOT NULL DEFAULT 'allowed' COMMENT '授权决策：allowed / denied',
  `request_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求追踪ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_audit_context_actor` (`actor_user_id`, `created_at`),
  KEY `idx_audit_context_permission` (`permission_key`, `created_at`),
  KEY `idx_audit_context_usage_log` (`usage_log_id`),
  KEY `idx_audit_context_request` (`request_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI使用审计RBAC上下文表';
