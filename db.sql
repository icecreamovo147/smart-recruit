-- Create and select the target database before sourcing this baseline schema.
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS `tenants` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_key` CHAR(36) NOT NULL COMMENT 'Immutable external tenant identifier',
  `slug` VARCHAR(64) NOT NULL COMMENT 'Immutable public tenant slug',
  `name` VARCHAR(128) NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'provisioning' COMMENT 'provisioning/active/suspended/disabled',
  `timezone` VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
  `locale` VARCHAR(32) NOT NULL DEFAULT 'zh-CN',
  `is_default` TINYINT(1) NOT NULL DEFAULT 0,
  `default_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `is_default` = 1 THEN 1 ELSE NULL END) STORED,
  `created_by` BIGINT UNSIGNED NULL COMMENT 'Platform actor users.id',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenants_tenant_key` (`tenant_key`),
  UNIQUE KEY `uk_tenants_slug` (`slug`),
  UNIQUE KEY `uk_tenants_default` (`default_key`),
  KEY `idx_tenants_status` (`status`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Enterprise tenant directory';

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username` VARCHAR(64) NOT NULL COMMENT '用户名（唯一）',
  `password` VARCHAR(255) NOT NULL COMMENT 'bcrypt 哈希密码',
  `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色：1=候选人 2=HR 3=HR管理员（Deprecated: 保留用于兼容，新授权逻辑使用 RBAC 表）',
  `email` VARCHAR(128) DEFAULT NULL COMMENT '邮箱（可选）',
  `account_type` VARCHAR(32) NOT NULL DEFAULT 'candidate' COMMENT 'candidate | staff | platform | service',
  `status` VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT '账号状态：active | disabled | locked | pending',
  `token_version` INT NOT NULL DEFAULT 1 COMMENT '令牌版本号，权限变更时递增以失效旧令牌',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账号表';

CREATE TABLE IF NOT EXISTS `tenant_memberships` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT 'invited/active/suspended/left',
  `joined_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_membership_user` (`tenant_id`, `user_id`),
  KEY `idx_tenant_memberships_user_status` (`user_id`, `status`, `tenant_id`),
  KEY `idx_tenant_memberships_tenant_status` (`tenant_id`, `status`, `user_id`),
  CONSTRAINT `fk_tenant_memberships_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_tenant_memberships_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Global-user membership in an enterprise tenant';

CREATE TABLE IF NOT EXISTS `refresh_tokens` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '刷新令牌ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 users.id',
  `token_hash` CHAR(64) NOT NULL COMMENT '明文 refresh token 的 sha256 哈希',
  `family_id` VARCHAR(64) NOT NULL COMMENT '登录会话族ID，轮换时保持不变',
  `client_app` VARCHAR(32) NOT NULL DEFAULT '',
  `active_tenant_id` BIGINT UNSIGNED NULL,
  `membership_id` BIGINT UNSIGNED NULL,
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
  KEY `idx_refresh_tokens_active_tenant` (`active_tenant_id`, `user_id`),
  KEY `idx_refresh_tokens_membership` (`membership_id`),
  CONSTRAINT `fk_refresh_tokens_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_refresh_tokens_active_tenant` FOREIGN KEY (`active_tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_refresh_tokens_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='刷新令牌存储表';

CREATE TABLE IF NOT EXISTS `jobs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '岗位ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  UNIQUE KEY `uk_jobs_tenant_id` (`tenant_id`, `id`),
  KEY `idx_hr_id` (`hr_id`),
  KEY `idx_status` (`status`),
  KEY `idx_status_created_id` (`status`, `created_at`, `id`),
  KEY `idx_hr_created_id` (`hr_id`, `created_at`, `id`),
  KEY `idx_jobs_tenant_status_created` (`tenant_id`, `status`, `created_at`, `id`),
  CONSTRAINT `fk_jobs_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
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
  `city` VARCHAR(128) NULL COMMENT '所在/期望工作城市（省/市/区）',
  `years_of_experience` DECIMAL(4,1) NULL COMMENT '工作年限',
  `job_status` VARCHAR(32) NULL COMMENT '求职状态: employed/resigned/fresh_graduate/student',
  `expected_position` VARCHAR(128) NULL COMMENT '期望岗位',
  `expected_salary_min` INT NULL COMMENT '期望月薪下限（元）',
  `expected_salary_max` INT NULL COMMENT '期望月薪上限（元）',
  `available_from` DATE NULL COMMENT '可到岗日期',
  `summary` VARCHAR(500) NULL COMMENT '个人简介',
  `is_complete` TINYINT NOT NULL DEFAULT 0 COMMENT '档案是否完整：0=不完整 1=完整',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人结构化档案表';

CREATE TABLE IF NOT EXISTS `candidate_educations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人 users.id',
  `school` VARCHAR(128) NOT NULL,
  `degree` VARCHAR(64) NULL,
  `major` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `description` TEXT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_candidate_educations_user` (`user_id`),
  CONSTRAINT `fk_candidate_educations_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人自维护教育经历';

CREATE TABLE IF NOT EXISTS `candidate_experiences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人 users.id',
  `company` VARCHAR(128) NOT NULL,
  `title` VARCHAR(128) NULL,
  `location` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `is_current` TINYINT NOT NULL DEFAULT 0,
  `description` TEXT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_candidate_experiences_user` (`user_id`),
  CONSTRAINT `fk_candidate_experiences_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人自维护工作经历';

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

CREATE TABLE IF NOT EXISTS `resume_parse_runs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_id` BIGINT UNSIGNED NOT NULL COMMENT 'Source resumes.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT 'Candidate users.id',
  `agent_run_id` BIGINT NULL COMMENT 'Optional agent_runs.id that produced this parse',
  `requested_model_id` BIGINT NULL,
  `effective_model_id` BIGINT NULL,
  `model_fallback_reason` VARCHAR(64) NULL,
  `capability_version_id` BIGINT NULL,
  `capability_snapshot_hash` CHAR(64) NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'running' COMMENT 'running / succeeded / failed',
  `parser_version` VARCHAR(64) NULL COMMENT 'Parser or prompt version used',
  `input_hash` VARCHAR(128) NULL COMMENT 'Hash of parse input for idempotency/audit',
  `error_message` TEXT NULL COMMENT 'Parse failure details',
  `started_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_parse_runs_resume` (`resume_id`),
  KEY `idx_resume_parse_runs_user` (`user_id`),
  KEY `idx_resume_parse_runs_agent_run` (`agent_run_id`),
  KEY `idx_resume_parse_capability_version` (`capability_version_id`),
  KEY `idx_resume_parse_runs_status` (`status`),
  CONSTRAINT `fk_resume_parse_runs_resume` FOREIGN KEY (`resume_id`) REFERENCES `resumes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Resume structured parse run audit';

CREATE TABLE IF NOT EXISTS `resume_profiles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_id` BIGINT UNSIGNED NOT NULL COMMENT 'Source resumes.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT 'Candidate users.id',
  `parse_run_id` BIGINT UNSIGNED NOT NULL COMMENT 'Producing resume_parse_runs.id',
  `version` INT NOT NULL DEFAULT 1 COMMENT 'Monotonic version per resume',
  `is_current` TINYINT NOT NULL DEFAULT 1 COMMENT 'Whether this is the current structured profile for the resume',
  `current_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_current` = 1 THEN 1 ELSE NULL END
  ) STORED,
  `full_name` VARCHAR(128) NULL,
  `email` VARCHAR(128) NULL,
  `phone` VARCHAR(64) NULL,
  `location` VARCHAR(128) NULL,
  `headline` VARCHAR(256) NULL,
  `summary` TEXT NULL,
  `total_experience_years` DECIMAL(5,2) NULL,
  `highest_degree` VARCHAR(64) NULL,
  `raw_json` JSON NULL COMMENT 'Full structured parser output',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_resume_profile_version` (`resume_id`, `version`),
  UNIQUE KEY `uk_resume_profile_current` (`resume_id`, `current_key`),
  UNIQUE KEY `uk_resume_profiles_parse_run` (`parse_run_id`),
  KEY `idx_resume_profiles_resume` (`resume_id`),
  KEY `idx_resume_profiles_user` (`user_id`),
  KEY `idx_resume_profiles_current` (`is_current`),
  CONSTRAINT `fk_resume_profiles_resume` FOREIGN KEY (`resume_id`) REFERENCES `resumes` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_resume_profiles_parse_run` FOREIGN KEY (`parse_run_id`) REFERENCES `resume_parse_runs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Versioned structured resume profile';

CREATE TABLE IF NOT EXISTS `resume_educations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `school` VARCHAR(128) NOT NULL,
  `degree` VARCHAR(64) NULL,
  `major` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `description` TEXT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_educations_profile` (`resume_profile_id`),
  CONSTRAINT `fk_resume_educations_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Structured resume education entries';

CREATE TABLE IF NOT EXISTS `resume_experiences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `company` VARCHAR(128) NOT NULL,
  `title` VARCHAR(128) NULL,
  `location` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `is_current` TINYINT NOT NULL DEFAULT 0,
  `description` TEXT NULL,
  `achievements_json` JSON NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_experiences_profile` (`resume_profile_id`),
  CONSTRAINT `fk_resume_experiences_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Structured resume work experience entries';

CREATE TABLE IF NOT EXISTS `resume_projects` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `role` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `description` TEXT NULL,
  `technologies_json` JSON NULL,
  `highlights_json` JSON NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_projects_profile` (`resume_profile_id`),
  CONSTRAINT `fk_resume_projects_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Structured resume project entries';

CREATE TABLE IF NOT EXISTS `resume_skills` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `category` VARCHAR(64) NULL,
  `level` VARCHAR(32) NULL,
  `years` DECIMAL(5,2) NULL,
  `evidence` TEXT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_resume_skill_name` (`resume_profile_id`, `name`),
  KEY `idx_resume_skills_profile` (`resume_profile_id`),
  KEY `idx_resume_skills_category` (`category`),
  CONSTRAINT `fk_resume_skills_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Structured resume skill entries';

CREATE TABLE IF NOT EXISTS `applications` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  UNIQUE KEY `uk_applications_tenant_id` (`tenant_id`, `id`),
  UNIQUE KEY `uk_active_application` (`job_id`, `user_id`, `active_key`),
  KEY `idx_job_user_current_status` (`job_id`, `user_id`, `is_current`, `status`),
  KEY `idx_job_status_current` (`job_id`, `status`, `is_current`),
  KEY `idx_job_current_applied` (`job_id`, `is_current`, `applied_at`, `id`),
  KEY `idx_user_applied` (`user_id`, `applied_at`, `id`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status_key` (`status_key`),
  KEY `idx_applications_tenant_job_status` (`tenant_id`, `job_id`, `status_key`, `is_current`),
  CONSTRAINT `fk_applications_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_applications_tenant_job` FOREIGN KEY (`tenant_id`, `job_id`) REFERENCES `jobs` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位投递关联表';

-- ── Phase 1: 投递状态变更审计表 ─────────────────────────────────────────
-- 记录每次状态变更的 actor、前后状态、原因和时间戳。

CREATE TABLE IF NOT EXISTS `application_status_transitions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  KEY `idx_transition_created` (`application_id`, `created_at`),
  KEY `idx_app_transitions_tenant_app_created` (`tenant_id`, `application_id`, `created_at`),
  CONSTRAINT `fk_app_transitions_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_app_transitions_tenant_application` FOREIGN KEY (`tenant_id`, `application_id`) REFERENCES `applications` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='投递状态变更审计记录表';

CREATE TABLE IF NOT EXISTS `ai_chat_sessions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `owner_role` TINYINT NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR',
  `owner_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID',
  `title` VARCHAR(255) NOT NULL COMMENT '会话标题',
  `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '绑定的投递记录ID，0表示普通数据问答',
  `session_type` VARCHAR(32) NOT NULL DEFAULT 'general' COMMENT '会话类型：general/resume/job_match/interview/offer/progress',
  `source_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '来源类型：job/application/resume/interview/offer等',
  `source_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '来源业务ID',
  `source_title` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '来源标题',
  `summary` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '会话摘要',
  `last_message_preview` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '最近消息摘要',
  `message_count` INT NOT NULL DEFAULT 0 COMMENT '会话消息数',
  `latest_context_usage_json` TEXT NULL COMMENT '当前会话最近一次上下文占用快照(JSON)',
  `selected_model_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '下一轮请求选择的模型ID，0表示跟随默认模型',
  `active_run_id` BIGINT NULL COMMENT 'Current active agent_runs.id for this session',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_hr_updated_at` (`hr_id`, `updated_at`),
  KEY `idx_hr_deleted_updated` (`hr_id`, `deleted_at`, `updated_at`),
  KEY `idx_owner_deleted_updated` (`owner_role`, `owner_id`, `deleted_at`, `updated_at`),
  KEY `idx_owner_type_updated` (`owner_role`, `owner_id`, `session_type`, `updated_at`),
  KEY `idx_owner_source` (`owner_role`, `owner_id`, `source_type`, `source_id`),
  KEY `idx_application_id` (`application_id`),
  KEY `idx_ai_chat_sessions_active_run` (`active_run_id`),
  KEY `idx_ai_chat_sessions_selected_model` (`selected_model_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 会话表';

CREATE TABLE IF NOT EXISTS `ai_chat_history` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'AI 会话ID',
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `owner_role` TINYINT NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR',
  `owner_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID',
  `role` VARCHAR(16) NOT NULL COMMENT '消息角色：user / assistant',
  `content` TEXT NOT NULL COMMENT '消息内容',
  `process_content` TEXT NULL COMMENT 'assistant 前置执行过程说明',
  `context_usage_json` TEXT NULL COMMENT '本条消息对应的上下文占用快照(JSON)',
  `model_id` BIGINT NULL COMMENT 'assistant 实际使用的模型ID',
  `model_name` VARCHAR(128) NULL COMMENT 'assistant 实际使用的模型名称',
  `agent_skill_ids_json` TEXT NULL COMMENT '本条用户消息选择的 Agent Skill ID 快照(JSON数组)',
  `agent_skill_names_json` TEXT NULL COMMENT '本条用户消息选择的 Agent Skill 名称快照(JSON数组)',
  `agent_run_id` BIGINT NULL COMMENT 'Optional agent_runs.id that produced this history message',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_session_id` (`session_id`),
  KEY `idx_session_created_id` (`session_id`, `created_at`, `id`),
  KEY `idx_hr_id_created` (`hr_id`, `created_at`),
  KEY `idx_owner_session_created` (`owner_role`, `owner_id`, `session_id`, `created_at`),
  KEY `idx_ai_chat_history_agent_run` (`agent_run_id`)
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
  `agent_run_id` BIGINT NULL,
  `agent_run_step_id` BIGINT NULL,
  `tool_call_id` VARCHAR(128) NOT NULL DEFAULT '',
  `tool_name` VARCHAR(128) NOT NULL,
  `arguments_json` JSON NULL,
  `result_json` JSON NULL,
  `result_summary` TEXT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'success' COMMENT 'success / error',
  `duration_ms` BIGINT NOT NULL DEFAULT 0,
  `error_message` TEXT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_session_created` (`session_id`, `created_at`),
  KEY `idx_hr_tool_created` (`hr_id`, `tool_name`, `created_at`),
  KEY `idx_ai_tool_traces_agent_run` (`agent_run_id`),
  KEY `idx_ai_tool_traces_agent_run_step` (`agent_run_step_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 工具调用轨迹表';

CREATE TABLE IF NOT EXISTS `agent_runs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT NOT NULL,
  `message_id` BIGINT NULL,
  `history_id` BIGINT NULL,
  `hr_id` BIGINT NOT NULL,
  `client_request_id` VARCHAR(128) NULL COMMENT 'Client idempotency key for create-run',
  `agent_type` VARCHAR(64) NOT NULL DEFAULT 'hr',
  `agent_id` BIGINT NULL,
  `agent_name` VARCHAR(128) NOT NULL,
  `model_id` BIGINT NULL,
  `model_name` VARCHAR(128) NOT NULL,
  `requested_model_id` BIGINT NULL,
  `effective_model_id` BIGINT NULL,
  `model_fallback_reason` VARCHAR(64) NULL,
  `capability_version_id` BIGINT UNSIGNED NULL,
  `capability_snapshot_hash` CHAR(64) NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'planning',
  `plan_json` JSON NULL,
  `final_answer` MEDIUMTEXT NULL,
  `assistant_text` MEDIUMTEXT NULL COMMENT 'Latest assistant answer snapshot for refresh restore',
  `process_text` MEDIUMTEXT NULL COMMENT 'Latest process trace snapshot for refresh restore',
  `result_metadata_json` JSON NULL COMMENT 'Result metadata snapshot',
  `confirmation_request_json` JSON NULL COMMENT 'Pending skill confirmation request snapshot',
  `option_context_json` JSON NULL COMMENT 'Active candidate/action option context snapshot',
  `last_event_seq` BIGINT NOT NULL DEFAULT 0 COMMENT 'Last persisted event sequence for this run',
  `cancel_requested_at` TIMESTAMP NULL COMMENT 'When cancel was requested',
  `canceled_at` TIMESTAMP NULL COMMENT 'When run reached canceled terminal state',
  `error_type` VARCHAR(64) NULL,
  `error_message` TEXT NULL,
  `started_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` TIMESTAMP NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_runs_client_request` (`hr_id`, `session_id`, `client_request_id`),
  KEY `idx_agent_runs_session_created` (`hr_id`, `session_id`, `created_at`),
  KEY `idx_agent_runs_status` (`status`),
  KEY `idx_agent_runs_message` (`message_id`),
  KEY `idx_agent_runs_session_status` (`session_id`, `status`),
  KEY `idx_agent_runs_capability_version` (`capability_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='One observable run per HR AI user message';

CREATE TABLE IF NOT EXISTS `agent_run_events` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `run_id` BIGINT NOT NULL,
  `seq` BIGINT NOT NULL,
  `event_type` VARCHAR(64) NOT NULL,
  `payload_json` JSON NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_run_events_run_seq` (`run_id`, `seq`),
  KEY `idx_agent_run_events_run` (`run_id`),
  CONSTRAINT `fk_agent_run_events_run` FOREIGN KEY (`run_id`) REFERENCES `agent_runs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Ordered durable events for resumable agent runs';

CREATE TABLE IF NOT EXISTS `agent_run_steps` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `run_id` BIGINT NOT NULL,
  `step_index` INT NOT NULL,
  `step_type` VARCHAR(32) NOT NULL,
  `capability_source` VARCHAR(64) NULL,
  `capability_key` VARCHAR(128) NULL,
  `tool_name` VARCHAR(128) NULL,
  `input_json` JSON NULL,
  `output_json` JSON NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'running',
  `duration_ms` BIGINT NOT NULL DEFAULT 0,
  `error_message` TEXT NULL,
  `started_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` TIMESTAMP NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_run_steps_run_index` (`run_id`, `step_index`),
  KEY `idx_agent_run_steps_run` (`run_id`),
  KEY `idx_agent_run_steps_type` (`step_type`),
  CONSTRAINT `fk_agent_run_steps_run` FOREIGN KEY (`run_id`) REFERENCES `agent_runs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Observable steps within an agent run';

ALTER TABLE `resume_parse_runs`
  ADD CONSTRAINT `fk_resume_parse_runs_agent_run` FOREIGN KEY (`agent_run_id`) REFERENCES `agent_runs` (`id`) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS `candidate_match_evaluations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT 'applications.id',
  `job_id` BIGINT UNSIGNED NOT NULL COMMENT 'jobs.id',
  `candidate_user_id` BIGINT UNSIGNED NOT NULL COMMENT 'users.id for candidate',
  `resume_profile_id` BIGINT UNSIGNED NOT NULL COMMENT 'resume_profiles.id used for matching',
  `agent_run_id` BIGINT NULL COMMENT 'Optional agent_runs.id that produced this evaluation',
  `requested_model_id` BIGINT NULL,
  `effective_model_id` BIGINT NULL,
  `model_fallback_reason` VARCHAR(64) NULL,
  `capability_version_id` BIGINT NULL,
  `capability_snapshot_hash` CHAR(64) NULL,
  `evaluation_version` INT NOT NULL DEFAULT 1 COMMENT 'Monotonic version per application',
  `is_latest` TINYINT NOT NULL DEFAULT 1 COMMENT 'Whether this is the latest match evaluation for the application',
  `latest_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_latest` = 1 THEN 1 ELSE NULL END
  ) STORED,
  `overall_score` DECIMAL(6,2) NULL,
  `recommendation` VARCHAR(32) NULL COMMENT 'strong_match / possible_match / weak_match / reject',
  `summary` TEXT NULL,
  `strengths_json` JSON NULL,
  `risks_json` JSON NULL,
  `score_breakdown_json` JSON NULL,
  `model_name` VARCHAR(128) NULL,
  `evaluated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_match_eval_tenant_id` (`tenant_id`, `id`),
  UNIQUE KEY `uk_candidate_match_version` (`application_id`, `evaluation_version`),
  UNIQUE KEY `uk_candidate_match_latest` (`application_id`, `latest_key`),
  KEY `idx_candidate_match_application` (`application_id`),
  KEY `idx_candidate_match_job` (`job_id`),
  KEY `idx_candidate_match_candidate` (`candidate_user_id`),
  KEY `idx_candidate_match_resume_profile` (`resume_profile_id`),
  KEY `idx_candidate_match_agent_run` (`agent_run_id`),
  KEY `idx_candidate_match_capability_version` (`capability_version_id`),
  KEY `idx_candidate_match_latest` (`is_latest`),
  CONSTRAINT `fk_candidate_match_application` FOREIGN KEY (`application_id`) REFERENCES `applications` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_candidate_match_resume_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE RESTRICT,
  KEY `idx_match_eval_tenant_app` (`tenant_id`, `application_id`),
  CONSTRAINT `fk_candidate_match_agent_run` FOREIGN KEY (`agent_run_id`) REFERENCES `agent_runs` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_match_eval_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_match_eval_tenant_application` FOREIGN KEY (`tenant_id`, `application_id`) REFERENCES `applications` (`tenant_id`, `id`),
  CONSTRAINT `fk_match_eval_tenant_job` FOREIGN KEY (`tenant_id`, `job_id`) REFERENCES `jobs` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Versioned candidate-job match evaluation';

CREATE TABLE IF NOT EXISTS `candidate_match_evidence` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `evaluation_id` BIGINT UNSIGNED NOT NULL,
  `evidence_type` VARCHAR(64) NOT NULL,
  `dimension` VARCHAR(64) NULL,
  `source_table` VARCHAR(64) NULL,
  `source_id` BIGINT UNSIGNED NULL,
  `snippet` TEXT NULL,
  `weight` DECIMAL(6,3) NULL,
  `score_impact` DECIMAL(6,2) NULL,
  `metadata_json` JSON NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_candidate_match_evidence_eval` (`evaluation_id`),
  KEY `idx_candidate_match_evidence_type` (`evidence_type`),
  KEY `idx_match_evidence_tenant_eval` (`tenant_id`, `evaluation_id`),
  CONSTRAINT `fk_candidate_match_evidence_eval` FOREIGN KEY (`evaluation_id`) REFERENCES `candidate_match_evaluations` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_match_evidence_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_match_evidence_tenant_evaluation` FOREIGN KEY (`tenant_id`, `evaluation_id`) REFERENCES `candidate_match_evaluations` (`tenant_id`, `id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Evidence supporting candidate match evaluations';

CREATE TABLE IF NOT EXISTS `ai_memories` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT '记忆归属 HR',
  `scope_type` VARCHAR(32) NOT NULL COMMENT 'hr / job / application / candidate',
  `scope_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '作用域ID',
  `memory_type` VARCHAR(32) NOT NULL COMMENT 'preference / fact / conclusion / warning',
  `content` TEXT NOT NULL COMMENT '记忆内容',
  `source` VARCHAR(32) NOT NULL DEFAULT 'agent' COMMENT 'user / tool / agent / system',
  `confidence` DECIMAL(4,3) NOT NULL DEFAULT 1.000,
  `importance` DECIMAL(4,3) NOT NULL DEFAULT 1.000,
  `expires_at` DATETIME NULL COMMENT '可选过期时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_scope` (`hr_id`, `scope_type`, `scope_id`),
  KEY `idx_hr_type` (`hr_id`, `memory_type`),
  KEY `idx_expires_at` (`expires_at`),
  KEY `idx_ai_memories_recall` (`hr_id`, `scope_type`, `scope_id`, `importance`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 长期记忆表';

CREATE TABLE IF NOT EXISTS `ai_embeddings` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `object_type` VARCHAR(64) NOT NULL,
  `object_id` BIGINT UNSIGNED NOT NULL,
  `scope_type` VARCHAR(32) NOT NULL DEFAULT '',
  `scope_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `text_hash` CHAR(64) NOT NULL,
  `embedding_model` VARCHAR(128) NOT NULL,
  `embedding_dim` INT NOT NULL DEFAULT 0,
  `vector_json` JSON NULL,
  `metadata_json` JSON NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'ready',
  `last_error` TEXT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_embeddings_object_model_hash` (`object_type`, `object_id`, `embedding_model`, `text_hash`),
  KEY `idx_ai_embeddings_object` (`object_type`, `object_id`),
  KEY `idx_ai_embeddings_scope` (`scope_type`, `scope_id`),
  KEY `idx_ai_embeddings_query` (`object_type`, `embedding_model`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI object embeddings for semantic retrieval';

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
  `schema_version` VARCHAR(16) NOT NULL DEFAULT '1.0' COMMENT '领域事件信封版本',
  `event_type` VARCHAR(64) NOT NULL COMMENT 'notification.create / resume.parse',
  `aggregate_type` VARCHAR(64) NOT NULL COMMENT 'application / resume / notification',
  `aggregate_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `routing_key` VARCHAR(128) NOT NULL,
  `producer` VARCHAR(128) NOT NULL DEFAULT 'smart-recruit-domain-go.outbox' COMMENT '事件生产者',
  `idempotency_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '消费者幂等键',
  `correlation_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '请求/流程关联ID',
  `causation_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '触发当前事件的命令或事件ID',
  `trace_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '链路追踪ID',
  `payload` JSON NOT NULL,
  `metadata` JSON NULL COMMENT '安全诊断元数据',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=pending 1=published 2=dead 3=processing',
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME NULL,
  `last_error` TEXT NULL,
  `locked_at` DATETIME NULL,
  `locked_by` VARCHAR(128) NOT NULL DEFAULT '',
  `published_at` DATETIME NULL COMMENT '成功发布到消息队列时间',
  `dead_lettered_at` DATETIME NULL COMMENT '进入死信状态时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_event_id` (`event_id`),
  KEY `idx_outbox_idempotency_key` (`idempotency_key`),
  KEY `idx_status_next_retry` (`status`, `next_retry_at`, `locked_at`, `id`),
  KEY `idx_aggregate` (`aggregate_type`, `aggregate_id`),
  KEY `idx_outbox_published_at` (`status`, `published_at`),
  KEY `idx_outbox_dead_lettered_at` (`status`, `dead_lettered_at`),
  KEY `idx_outbox_status_created` (`status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事务消息 outbox 表';

CREATE TABLE IF NOT EXISTS `email_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(64) NOT NULL COMMENT '来源 outbox 事件ID，用于幂等去重',
  `user_id` BIGINT NOT NULL COMMENT '接收用户ID',
  `email` VARCHAR(128) NOT NULL COMMENT '接收邮箱地址',
  `type` VARCHAR(64) NOT NULL COMMENT '通知类型：interview_scheduled / offer_sent 等',
  `subject` VARCHAR(256) NOT NULL COMMENT '邮件标题',
  `status` VARCHAR(32) NOT NULL DEFAULT 'sent' COMMENT '状态：sent / failed / skipped',
  `error_msg` TEXT NULL COMMENT '发送失败原因',
  `sent_at` DATETIME NOT NULL COMMENT '发送时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_email_event_id` (`event_id`),
  INDEX `idx_email_user_id` (`user_id`),
  INDEX `idx_email_type_status` (`type`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮件发送记录表';

CREATE TABLE IF NOT EXISTS `event_inbox` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(128) NOT NULL COMMENT '事件ID或无事件ID消息的稳定哈希',
  `event_type` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '事件类型',
  `consumer_name` VARCHAR(128) NOT NULL COMMENT '消费者名称',
  `idempotency_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '消费者幂等键',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=processing 1=processed 2=failed 3=dead',
  `attempt_count` INT NOT NULL DEFAULT 0,
  `last_error` TEXT NULL,
  `received_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `processing_at` DATETIME NULL,
  `processed_at` DATETIME NULL,
  `dead_lettered_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_event_inbox_consumer_event` (`consumer_name`, `event_id`),
  KEY `idx_event_inbox_consumer_status` (`consumer_name`, `status`),
  KEY `idx_event_inbox_idempotency_key` (`idempotency_key`),
  KEY `idx_event_inbox_processed_at` (`status`, `processed_at`),
  KEY `idx_event_inbox_dead_lettered_at` (`status`, `dead_lettered_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='事件消费者 Inbox 幂等表';

CREATE TABLE IF NOT EXISTS `analytics_projection_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `projection_name` VARCHAR(64) NOT NULL COMMENT 'Analytics projection/read-model name',
  `source` VARCHAR(32) NOT NULL DEFAULT 'domain_event' COMMENT 'domain_event / replay / backfill',
  `event_id` VARCHAR(128) NOT NULL COMMENT 'Domain event id',
  `event_type` VARCHAR(128) NOT NULL COMMENT 'Source-domain event type',
  `aggregate_type` VARCHAR(64) NOT NULL COMMENT 'Source aggregate type',
  `aggregate_id` VARCHAR(128) NOT NULL COMMENT 'Source aggregate id',
  `producer` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'Event producer',
  `idempotency_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Projection idempotency key',
  `correlation_id` VARCHAR(128) NOT NULL DEFAULT '',
  `causation_id` VARCHAR(128) NOT NULL DEFAULT '',
  `trace_id` VARCHAR(128) NOT NULL DEFAULT '',
  `payload` JSON NOT NULL,
  `metadata` JSON NULL,
  `occurred_at` DATETIME NOT NULL COMMENT 'Source event occurrence time',
  `projected_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Projection write time',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_analytics_projection_event` (`event_id`),
  KEY `idx_analytics_projection_name_occurred` (`projection_name`, `occurred_at`),
  KEY `idx_analytics_projection_event_type` (`event_type`),
  KEY `idx_analytics_projection_aggregate` (`aggregate_type`, `aggregate_id`),
  KEY `idx_analytics_projection_idempotency_key` (`idempotency_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Analytics event-projection read model input';

CREATE TABLE IF NOT EXISTS `analytics_projection_checkpoints` (
  `projection_name` VARCHAR(64) NOT NULL,
  `cursor` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Last processed event cursor',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`projection_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Analytics projection checkpoint store';

CREATE TABLE IF NOT EXISTS `invite_codes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `code` VARCHAR(64) NOT NULL COMMENT '邀请码（随机生成）',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建该邀请码的管理员用户ID',
  `expires_at` DATETIME NULL COMMENT '过期时间，NULL 表示永不过期',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '是否有效：1=有效 0=已撤销',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_created_by` (`created_by`),
  KEY `idx_code_active_expires` (`code`, `is_active`, `expires_at`),
  KEY `idx_invite_codes_tenant_active` (`tenant_id`, `is_active`, `expires_at`),
  CONSTRAINT `fk_invite_codes_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='HR 注册邀请码表';

CREATE TABLE IF NOT EXISTS `tenant_invitations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `email` VARCHAR(128) NOT NULL,
  `token_hash` CHAR(64) NOT NULL,
  `role_keys_json` JSON NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT 'pending/accepted/revoked/expired',
  `expires_at` DATETIME NOT NULL,
  `created_by` BIGINT UNSIGNED NOT NULL,
  `accepted_by` BIGINT UNSIGNED NULL,
  `accepted_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_invitations_token_hash` (`token_hash`),
  KEY `idx_tenant_invitations_tenant_status` (`tenant_id`, `status`, `expires_at`),
  KEY `idx_tenant_invitations_email_status` (`email`, `status`, `expires_at`),
  CONSTRAINT `fk_tenant_invitations_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_tenant_invitations_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_tenant_invitations_acceptor` FOREIGN KEY (`accepted_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Single-use tenant staff invitations';

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
  `role_key` VARCHAR(64) NOT NULL COMMENT 'candidate/recruiter/recruiting_admin/interviewer/platform_admin/system_admin',
  `name` VARCHAR(128) NOT NULL COMMENT '角色中文名称',
  `description` VARCHAR(512) DEFAULT NULL COMMENT '角色描述',
  `scope_type` VARCHAR(32) NOT NULL DEFAULT 'identity' COMMENT 'identity/tenant/platform',
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

CREATE TABLE IF NOT EXISTS `tenant_membership_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `membership_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `assigned_by` BIGINT UNSIGNED NULL,
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` DATETIME NULL,
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_membership_role_active` (`membership_id`, `role_id`, `active_key`),
  KEY `idx_tenant_membership_roles_role` (`role_id`, `revoked_at`),
  CONSTRAINT `fk_tenant_membership_roles_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_tenant_membership_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tenant-scoped roles assigned to a membership';

CREATE TABLE IF NOT EXISTS `tenant_membership_data_scopes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `membership_id` BIGINT UNSIGNED NOT NULL,
  `scope_key` VARCHAR(64) NOT NULL,
  `resource_type` VARCHAR(64) NOT NULL DEFAULT '',
  `resource_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `assigned_by` BIGINT UNSIGNED NULL,
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` DATETIME NULL,
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_membership_scope_active` (`membership_id`, `scope_key`, `resource_type`, `resource_id`, `active_key`),
  KEY `idx_tenant_membership_scopes_scope` (`scope_key`, `resource_type`, `resource_id`),
  CONSTRAINT `fk_tenant_membership_scopes_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tenant-scoped data grants assigned to a membership';

CREATE TABLE IF NOT EXISTS `platform_user_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `assigned_by` BIGINT UNSIGNED NULL,
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` DATETIME NULL,
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_user_role_active` (`user_id`, `role_id`, `active_key`),
  KEY `idx_platform_user_roles_role` (`role_id`, `revoked_at`),
  CONSTRAINT `fk_platform_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_platform_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Platform-scoped role assignments';

CREATE TABLE IF NOT EXISTS `authorization_audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NULL,
  `membership_id` BIGINT UNSIGNED NULL,
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
  KEY `idx_decision_created` (`decision`, `created_at`),
  KEY `idx_authorization_audit_tenant_created` (`tenant_id`, `created_at`),
  CONSTRAINT `fk_authorization_audit_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_authorization_audit_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='授权审计日志表';

-- ── 面试安排表 ────────────────────────────────────────────────────────
-- 用于面试官 (interviewer) 角色的 assigned_interviews 数据范围匹配。
-- 一个 application 可能对应多轮面试，每轮可指派不同面试官。

CREATE TABLE IF NOT EXISTS `interview_schedules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  UNIQUE KEY `uk_interviews_tenant_id` (`tenant_id`, `id`),
  KEY `idx_interviewer_deleted` (`interviewer_id`, `deleted_at`),
  KEY `idx_application_deleted` (`application_id`, `deleted_at`),
  KEY `idx_interviewer_app` (`interviewer_id`, `application_id`, `deleted_at`),
  KEY `idx_interviews_tenant_interviewer` (`tenant_id`, `interviewer_id`, `deleted_at`),
  CONSTRAINT `fk_interviews_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_interviews_tenant_application` FOREIGN KEY (`tenant_id`, `application_id`) REFERENCES `applications` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='面试安排表';

-- ── 面试反馈表 ────────────────────────────────────────────────────────
-- 记录面试官对面试的反馈评价，提交后不可修改（有审核更正路径）。

CREATE TABLE IF NOT EXISTS `interview_feedback` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `interview_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 interview_schedules.id',
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `interviewer_id` BIGINT UNSIGNED NOT NULL COMMENT '面试官用户ID',
  `recommendation` VARCHAR(32) DEFAULT NULL COMMENT '推荐结论：positive / negative / pending',
  `score` INT DEFAULT NULL COMMENT '综合评分（0-100）',
  `dimension_scores_json` TEXT DEFAULT NULL COMMENT '维度评分 JSON，如 {"communication":4,"technical":5}',
  `comments` TEXT DEFAULT NULL COMMENT '面试评语',
  `submitted_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '提交时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_interview_feedback_once` (`interview_id`, `application_id`, `interviewer_id`),
  KEY `idx_feedback_interview` (`interview_id`),
  KEY `idx_feedback_interviewer` (`interviewer_id`),
  KEY `idx_feedback_application` (`application_id`),
  KEY `idx_feedback_tenant_interview` (`tenant_id`, `interview_id`),
  CONSTRAINT `chk_interview_feedback_score` CHECK (`score` IS NULL OR (`score` BETWEEN 0 AND 100)),
  CONSTRAINT `fk_feedback_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_feedback_tenant_interview` FOREIGN KEY (`tenant_id`, `interview_id`) REFERENCES `interview_schedules` (`tenant_id`, `id`),
  CONSTRAINT `fk_feedback_tenant_application` FOREIGN KEY (`tenant_id`, `application_id`) REFERENCES `applications` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='面试反馈表';

-- ── Offer 表 ─────────────────────────────────────────────────────────────
-- 记录 Offer 的创建、发送、决策全过程。

CREATE TABLE IF NOT EXISTS `offers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Offer ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  UNIQUE KEY `uk_offers_tenant_id` (`tenant_id`, `id`),
  KEY `idx_offer_application` (`application_id`),
  KEY `idx_offer_candidate` (`candidate_user_id`),
  KEY `idx_offer_job` (`job_id`),
  KEY `idx_offer_status` (`status`),
  KEY `idx_offer_created_by` (`created_by`),
  KEY `idx_offer_created` (`created_at`),
  KEY `idx_offers_tenant_status_created` (`tenant_id`, `status`, `created_at`),
  CONSTRAINT `fk_offers_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_offers_tenant_application` FOREIGN KEY (`tenant_id`, `application_id`) REFERENCES `applications` (`tenant_id`, `id`),
  CONSTRAINT `fk_offers_tenant_job` FOREIGN KEY (`tenant_id`, `job_id`) REFERENCES `jobs` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Offer表';

CREATE TABLE IF NOT EXISTS `offer_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  KEY `idx_offer_event_created` (`offer_id`, `created_at`),
  KEY `idx_offer_events_tenant_offer` (`tenant_id`, `offer_id`, `created_at`),
  CONSTRAINT `fk_offer_events_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_offer_events_tenant_offer` FOREIGN KEY (`tenant_id`, `offer_id`) REFERENCES `offers` (`tenant_id`, `id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Offer事件审计表';

-- ══════════════════════════════════════════════════════════════════════
-- RBAC 种子数据
-- ══════════════════════════════════════════════════════════════════════

-- ── 角色 ──────────────────────────────────────────────────────────────

INSERT INTO `roles` (`role_key`, `name`, `description`, `scope_type`, `is_system`) VALUES
  ('candidate',        '求职者',     '外部求职者，管理个人资料、简历、投递和AI会话', 'identity', 1),
  ('recruiter',        '招聘专员',   '负责岗位发布、候选人流程、面试安排和HR AI使用', 'tenant', 1),
  ('recruiting_admin', '招聘管理员', '管理招聘配置、邀请码、部门、地点、用户角色分配', 'tenant', 1),
  ('system_admin',     '系统管理员', '平台管理员兼容角色，后续使用 platform_admin', 'platform', 1),
  ('platform_admin',   '平台管理员', '管理租户、平台安全、全局目录与跨租户运营', 'platform', 1),
  ('platform_operator','平台运营管理员', '管理租户运营、订阅、用量和告警，不管理平台账号或发布套餐', 'platform', 1),
  ('platform_auditor', '平台审计员', '只读查看平台运营、租户、用量和审计数据', 'platform', 1),
  ('interviewer',      '面试官',     '查看被分配的面试并提交反馈', 'tenant', 1)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`),
  `scope_type` = VALUES(`scope_type`);

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
  ('ai.prompt.manage',               'ai',          'manage', '管理招聘 Prompt 模板'),
  ('ai.agent.manage',                'ai',          'manage', '管理招聘 Agent 配置'),
  ('ai.agent_skill.manage',          'ai',          'manage', '管理招聘 Agent Skill'),
  ('admin.invite.manage',            'admin',       'manage', '管理邀请码'),
  ('admin.department.manage',        'admin',       'manage', '管理部门及部门地点关联'),
  ('admin.location.manage',          'admin',       'manage', '管理工作地点'),
  ('admin.user.manage',              'admin',       'manage', '创建/修改/禁用员工账号'),
  ('admin.role.manage',              'admin',       'manage', '管理角色目录和权限分配'),
  ('audit.usage.read',               'audit',       'read',   '查看第三方/AI使用日志'),
  ('audit.security.read',            'audit',       'read',   '查看授权和安全审计事件'),
  ('system.config.manage',           'system',      'manage', '管理平台安全配置'),
  ('platform.dashboard.read',        'platform_dashboard', 'read', '查看平台运营总览'),
  ('platform.tenant.read',           'platform_tenant', 'read', '查看平台租户和聚合诊断信息'),
  ('platform.tenant.manage',         'platform_tenant', 'manage', '创建和变更平台租户生命周期'),
  ('platform.member.manage',         'platform_member', 'manage', '管理租户成员状态和主管理员'),
  ('platform.plan.read',             'platform_plan', 'read', '查看平台套餐和权益'),
  ('platform.plan.manage',           'platform_plan', 'manage', '维护套餐草稿和租户覆盖'),
  ('platform.plan.publish',          'platform_plan', 'publish', '发布或退役套餐版本'),
  ('platform.billing.refund.review', 'platform_billing_refund', 'review', '查看并审批平台退款'),
  ('platform.subscription.manage',   'platform_subscription', 'manage', '管理租户套餐订阅'),
  ('platform.usage.read',            'platform_usage', 'read', '查看跨租户用量和配额'),
  ('platform.alert.read',            'platform_alert', 'read', '查看平台运营告警'),
  ('platform.alert.manage',          'platform_alert', 'manage', '认领和处理平台运营告警'),
  ('platform.audit.read',            'platform_audit', 'read', '查看平台控制面操作审计'),
  ('platform.user.manage',           'platform_user', 'manage', '管理平台账号和平台角色'),
  ('platform.ai.config.read',        'platform_ai_config', 'read', '查看平台 AI 技术配置'),
  ('platform.ai.config.manage',      'platform_ai_config', 'manage', '维护平台 AI 技术配置'),
  ('platform.ai.release.read',       'platform_ai_release', 'read', '查看平台 AI 能力版本'),
  ('platform.ai.release.manage',     'platform_ai_release', 'manage', '维护平台 AI 能力草稿'),
  ('platform.ai.release.publish',    'platform_ai_release', 'publish', '发布或退役平台 AI 能力版本'),
  ('platform.ai.diagnostics.read',   'platform_ai_diagnostics', 'read', '查看平台 AI 诊断信息'),
  ('platform.ai.diagnostics.execute','platform_ai_diagnostics', 'execute', '执行平台 AI 诊断'),
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
    'collaboration.tag.manage', 'collaboration.task.manage',
    'ai.prompt.manage', 'ai.agent.manage', 'ai.agent_skill.manage'
  );

-- System Admin (platform-level only, no recruiting workflow)
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'system_admin' AND p.permission_key IN (
    'auth.session.read', 'admin.user.manage', 'admin.role.manage',
    'audit.usage.read', 'audit.security.read', 'system.config.manage',
    'ai.prompt.manage', 'ai.agent.manage', 'ai.agent_skill.manage'
  );

-- Interviewer
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'interviewer' AND p.permission_key IN (
    'auth.session.read', 'interview.read', 'interview.feedback.submit', 'notification.read'
  );

-- Platform Admin receives the same platform permissions as the compatibility system_admin role.
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT platform_role.id, rp.permission_id
  FROM `roles` platform_role
  JOIN `roles` legacy_role ON legacy_role.role_key = 'system_admin'
  JOIN `role_permissions` rp ON rp.role_id = legacy_role.id
  WHERE platform_role.role_key = 'platform_admin';

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'platform_admin' AND p.permission_key LIKE 'platform.%';

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'platform_operator' AND p.permission_key IN (
    'platform.dashboard.read', 'platform.tenant.read', 'platform.tenant.manage',
    'platform.member.manage', 'platform.plan.read', 'platform.plan.manage',
    'platform.subscription.manage', 'platform.usage.read', 'platform.alert.read',
    'platform.alert.manage', 'platform.audit.read', 'platform.billing.refund.review',
    'platform.ai.config.read', 'platform.ai.release.read',
    'platform.ai.diagnostics.read', 'platform.ai.diagnostics.execute'
  );

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'platform_auditor' AND p.permission_key IN (
    'platform.dashboard.read', 'platform.tenant.read', 'platform.plan.read',
    'platform.usage.read', 'platform.alert.read', 'platform.audit.read',
    'platform.ai.config.read', 'platform.ai.release.read', 'platform.ai.diagnostics.read'
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

-- ── 默认租户与兼容数据 ────────────────────────────────────────────────

INSERT INTO `tenants`
  (`tenant_key`, `slug`, `name`, `status`, `timezone`, `locale`, `is_default`)
VALUES
  ('00000000-0000-4000-8000-000000000001', 'default', '默认企业', 'active', 'Asia/Shanghai', 'zh-CN', 1)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `status` = 'active', `is_default` = 1;

INSERT IGNORE INTO `tenant_memberships` (`tenant_id`, `user_id`, `status`, `joined_at`)
  SELECT tenant.id, users.id, 'active', NOW()
  FROM `tenants` tenant
  JOIN `users` users ON users.account_type = 'staff'
  WHERE tenant.is_default = 1;

INSERT IGNORE INTO `tenant_membership_roles` (`membership_id`, `role_id`, `assigned_by`, `assigned_at`)
  SELECT membership.id, user_role.role_id, user_role.assigned_by, user_role.assigned_at
  FROM `tenant_memberships` membership
  JOIN `user_roles` user_role ON user_role.user_id = membership.user_id AND user_role.revoked_at IS NULL
  JOIN `roles` role ON role.id = user_role.role_id AND role.scope_type = 'tenant'
  JOIN `tenants` tenant ON tenant.id = membership.tenant_id AND tenant.is_default = 1;

INSERT IGNORE INTO `tenant_membership_data_scopes`
  (`membership_id`, `scope_key`, `resource_type`, `resource_id`, `assigned_by`, `assigned_at`)
  SELECT membership.id, scope.scope_key, scope.resource_type, scope.resource_id, scope.assigned_by, scope.assigned_at
  FROM `tenant_memberships` membership
  JOIN `user_data_scopes` scope ON scope.user_id = membership.user_id AND scope.revoked_at IS NULL
  JOIN `tenants` tenant ON tenant.id = membership.tenant_id AND tenant.is_default = 1;

INSERT IGNORE INTO `platform_user_roles` (`user_id`, `role_id`, `assigned_by`, `assigned_at`)
  SELECT user_role.user_id, platform_role.id, user_role.assigned_by, user_role.assigned_at
  FROM `user_roles` user_role
  JOIN `roles` legacy_role ON legacy_role.id = user_role.role_id AND legacy_role.role_key = 'system_admin'
  JOIN `roles` platform_role ON platform_role.role_key = 'platform_admin'
  WHERE user_role.revoked_at IS NULL;

-- ── 部门基础数据表 ──────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS `departments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  UNIQUE KEY `uk_departments_tenant_id` (`tenant_id`, `id`),
  UNIQUE KEY `uk_department_tenant_parent_name` (`tenant_id`, `parent_id`, `name`),
  KEY `idx_department_parent_sort` (`parent_id`, `sort_order`, `id`),
  KEY `idx_department_active` (`is_active`),
  KEY `idx_department_path` (`path`),
  KEY `idx_departments_tenant_active` (`tenant_id`, `is_active`, `deleted_at`),
  CONSTRAINT `fk_departments_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位部门基础数据表';

-- ── 岗位地点基础数据表 ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS `job_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '地点ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  UNIQUE KEY `uk_job_locations_tenant_id` (`tenant_id`, `id`),
  UNIQUE KEY `uk_job_location_tenant_name` (`tenant_id`, `name`),
  KEY `idx_job_location_active_sort` (`is_active`, `sort_order`, `id`),
  KEY `idx_job_locations_tenant_active` (`tenant_id`, `is_active`, `deleted_at`),
  CONSTRAINT `fk_job_locations_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位地点基础数据表';

-- ── 部门-地点关联表 ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS `department_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门地点关联ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  KEY `idx_location_active` (`location_id`, `is_active`, `deleted_at`),
  KEY `idx_department_locations_tenant` (`tenant_id`, `department_id`, `location_id`),
  CONSTRAINT `fk_department_locations_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_department_locations_tenant_department` FOREIGN KEY (`tenant_id`, `department_id`) REFERENCES `departments` (`tenant_id`, `id`),
  CONSTRAINT `fk_department_locations_tenant_location` FOREIGN KEY (`tenant_id`, `location_id`) REFERENCES `job_locations` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门可用地点关联表';

-- ── 扩展 jobs 表，增加外键字段 ─────────────────────────────────────────

ALTER TABLE `jobs`
  ADD COLUMN `department_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 departments.id' AFTER `department`,
  ADD COLUMN `location_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 job_locations.id' AFTER `location`,
  ADD KEY `idx_department_id` (`department_id`),
  ADD KEY `idx_location_id` (`location_id`);

-- ── 初始化部门数据 ─────────────────────────────────────────────────────

INSERT INTO `departments` (`tenant_id`, `parent_id`, `name`, `full_name`, `path`, `depth`, `sort_order`, `is_active`, `inherit_locations`) VALUES
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '技术研发部', '技术研发部', '/1/', 1, 1, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '产品部',     '产品部',     '/2/', 1, 2, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '设计部',     '设计部',     '/3/', 1, 3, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '市场部',     '市场部',     '/4/', 1, 4, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '销售部',     '销售部',     '/5/', 1, 5, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '运营部',     '运营部',     '/6/', 1, 6, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '人力资源部', '人力资源部', '/7/', 1, 7, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '财务部',     '财务部',     '/8/', 1, 8, 1, 0),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), 0, '客户成功部', '客户成功部', '/9/', 1, 9, 1, 0);

-- ── 初始化地点数据 ─────────────────────────────────────────────────────

INSERT INTO `job_locations` (`tenant_id`, `name`, `code`, `sort_order`, `is_active`) VALUES
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '北京', 'beijing',  1, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '上海', 'shanghai', 2, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '广州', 'guangzhou',3, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '深圳', 'shenzhen', 4, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '杭州', 'hangzhou', 5, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '成都', 'chengdu',  6, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '武汉', 'wuhan',    7, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '西安', 'xian',     8, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '南京', 'nanjing',  9, 1),
  ((SELECT id FROM tenants WHERE is_default = 1 LIMIT 1), '远程', 'remote',  10, 1);

-- ── 初始化部门可用地点数据 ─────────────────────────────────────────────
-- 使用 CTE 为每个根部门分配 3 个伪随机地点，与 v9 迁移逻辑一致。

INSERT INTO `department_locations` (`tenant_id`, `department_id`, `location_id`, `is_active`)
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
SELECT d.tenant_id, root_targets.department_id, root_targets.location_id, 1
FROM root_targets
JOIN departments d ON d.id = root_targets.department_id
ON DUPLICATE KEY UPDATE
  is_active = 1,
  deleted_at = NULL,
  deleted_by = NULL;

-- ══════════════════════════════════════════════════════════════════════
-- Phase 4: Candidate Collaboration
-- ══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS `candidate_notes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '备注ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  KEY `idx_note_created` (`candidate_user_id`, `created_at`),
  KEY `idx_candidate_notes_tenant_candidate` (`tenant_id`, `candidate_user_id`, `created_at`),
  CONSTRAINT `fk_candidate_notes_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_candidate_notes_tenant_application` FOREIGN KEY (`tenant_id`, `application_id`) REFERENCES `applications` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人内部备注表';

CREATE TABLE IF NOT EXISTS `candidate_tags` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '标签ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(64) NOT NULL COMMENT '标签名称',
  `color` VARCHAR(16) DEFAULT '#409eff' COMMENT '标签颜色',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建人用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_candidate_tags_tenant_id` (`tenant_id`, `id`),
  UNIQUE KEY `uk_candidate_tag_tenant_name` (`tenant_id`, `name`),
  CONSTRAINT `fk_candidate_tags_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人标签定义表';

CREATE TABLE IF NOT EXISTS `candidate_tag_assignments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '分配ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `tag_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 candidate_tags.id',
  `candidate_user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人用户ID',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '分配人用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tag_candidate` (`tag_id`, `candidate_user_id`),
  KEY `idx_tag_assignment_candidate` (`candidate_user_id`),
  KEY `idx_tag_assignment_tag` (`tag_id`),
  KEY `idx_tag_assignments_tenant_candidate` (`tenant_id`, `candidate_user_id`),
  CONSTRAINT `fk_tag_assignments_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_tag_assignments_tenant_tag` FOREIGN KEY (`tenant_id`, `tag_id`) REFERENCES `candidate_tags` (`tenant_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人标签分配表';

CREATE TABLE IF NOT EXISTS `follow_up_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '任务ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL,
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
  KEY `idx_task_due` (`due_at`),
  KEY `idx_follow_up_tenant_assignee` (`tenant_id`, `assignee_user_id`, `status`, `due_at`),
  CONSTRAINT `fk_follow_up_tasks_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_follow_up_tenant_application` FOREIGN KEY (`tenant_id`, `application_id`) REFERENCES `applications` (`tenant_id`, `id`)
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

-- ══════════════════════════════════════════════════════════════════════
-- Phase 7: LLM, Prompt, Agent, and MCP Configuration
-- ══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS `llm_providers` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT 'Provider display name',
  `base_url` VARCHAR(512) NOT NULL COMMENT 'API base URL',
  `api_key_encrypted` VARCHAR(512) NOT NULL COMMENT 'AES-256-GCM encrypted API key',
  `provider_type` VARCHAR(64) NOT NULL COMMENT 'openai/anthropic/azure_openai/google_gemini/ollama/openai_compatible/custom',
  `protocol_type` VARCHAR(64) NOT NULL DEFAULT 'openai_chat_completions' COMMENT 'Runtime protocol adapter',
  `auth_type` VARCHAR(64) NOT NULL DEFAULT 'bearer' COMMENT 'Authentication strategy',
  `api_version` VARCHAR(64) NULL COMMENT 'Provider API version, primarily Azure',
  `discovery_url` VARCHAR(512) NULL COMMENT 'Optional explicit model discovery endpoint',
  `extra_headers` JSON COMMENT 'Extra HTTP headers as JSON object',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether provider is enabled',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_provider_type` (`provider_type`),
  KEY `idx_llm_provider_protocol_type` (`protocol_type`),
  KEY `idx_is_enabled` (`is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='LLM provider configurations';

CREATE TABLE IF NOT EXISTS `llm_models` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `provider_id` BIGINT NOT NULL COMMENT 'FK to llm_providers.id',
  `model_name` VARCHAR(128) NOT NULL COMMENT 'Model name used in API calls',
  `catalog_model_name` VARCHAR(256) NULL COMMENT 'Provider catalog model id when runtime name differs, e.g. Azure deployment',
  `display_name` VARCHAR(128) COMMENT 'Human-readable display name',
  `temperature` DOUBLE NOT NULL DEFAULT 0.7 COMMENT 'LLM temperature parameter',
  `top_p` DOUBLE NOT NULL DEFAULT 1.0 COMMENT 'LLM top_p parameter',
  `max_tokens` INT NOT NULL DEFAULT 4096 COMMENT 'Max tokens for generation (output budget)',
  `context_window_tokens` INT NOT NULL DEFAULT 0 COMMENT 'Total model context window tokens (input + output); 0 = unknown',
  `provider_max_input_tokens` INT NULL COMMENT 'Provider-advertised input token limit',
  `provider_max_output_tokens` INT NULL COMMENT 'Provider-advertised output token limit',
  `capabilities` JSON NULL COMMENT 'Provider-advertised capabilities',
  `metadata_source` VARCHAR(32) NOT NULL DEFAULT 'manual' COMMENT 'manual/provider/merged/mixed',
  `metadata_sources` JSON NULL COMMENT 'Field-level metadata provenance captured when the user saved the model',
  `metadata_synced_at` DATETIME NULL COMMENT 'Last provider metadata synchronization time',
  `temperature_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether temperature is sent to provider',
  `top_p_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether top_p is sent to provider',
  `max_concurrency` INT NOT NULL DEFAULT 10 COMMENT 'Max concurrent LLM calls',
  `timeout_seconds` INT NOT NULL DEFAULT 90 COMMENT 'Request timeout in seconds',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether model is enabled',
  `is_default` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Whether this is the default model',
  `global_default_key` VARCHAR(16) GENERATED ALWAYS AS (CASE WHEN `is_default` = 1 AND `is_enabled` = 1 THEN 'global' ELSE NULL END) STORED,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_provider_id` (`provider_id`),
  KEY `idx_model_enabled` (`is_enabled`),
  KEY `idx_model_default` (`is_default`),
  UNIQUE KEY `uk_llm_global_default` (`global_default_key`),
  CONSTRAINT `fk_llm_models_provider` FOREIGN KEY (`provider_id`) REFERENCES `llm_providers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='LLM model configurations';

CREATE TABLE IF NOT EXISTS `llm_model_catalog` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `provider_family` VARCHAR(64) NOT NULL COMMENT 'Stable provider family, e.g. deepseek/openai/anthropic',
  `model_name` VARCHAR(256) NOT NULL COMMENT 'Provider catalog model identifier',
  `display_name` VARCHAR(256) NULL COMMENT 'Human-readable model name',
  `context_window_tokens` INT NULL COMMENT 'Total context window; NULL = unknown',
  `max_input_tokens` INT NULL COMMENT 'Provider-advertised input limit; NULL = unknown',
  `max_output_tokens` INT NULL COMMENT 'Provider-advertised output limit; NULL = unknown',
  `temperature` DOUBLE NULL COMMENT 'Provider request default; NULL = unknown',
  `top_p` DOUBLE NULL COMMENT 'Provider request default; NULL = unknown',
  `capabilities` JSON NULL COMMENT 'Verified model capabilities',
  `field_sources` JSON NULL COMMENT 'Per-field provenance overrides',
  `source_type` VARCHAR(32) NOT NULL COMMENT 'official_document/provider_api/admin',
  `source_url` VARCHAR(1024) NULL COMMENT 'Evidence URL without credentials',
  `source_revision` VARCHAR(128) NOT NULL COMMENT 'Version of the imported source data',
  `content_hash` CHAR(64) NOT NULL COMMENT 'SHA-256 of normalized catalog content',
  `status` VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT 'active/inactive',
  `managed_by` VARCHAR(32) NOT NULL DEFAULT 'bundled' COMMENT 'bundled/admin/remote_sync',
  `verified_at` DATETIME NULL,
  `expires_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_llm_model_catalog_family_model` (`provider_family`, `model_name`),
  KEY `idx_llm_model_catalog_status` (`status`, `expires_at`),
  KEY `idx_llm_model_catalog_hash` (`content_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Verified reusable LLM model metadata catalog';

CREATE TABLE IF NOT EXISTS `llm_model_metadata_observations` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `provider_id` BIGINT NOT NULL COMMENT 'Provider that produced the observation',
  `model_name` VARCHAR(256) NOT NULL,
  `field_name` VARCHAR(64) NOT NULL COMMENT 'Observed metadata field',
  `value_json` JSON NOT NULL COMMENT 'Typed observed value encoded as JSON',
  `source_type` VARCHAR(32) NOT NULL COMMENT 'provider_api/provider_detail/official_document',
  `source_ref` VARCHAR(1024) NULL COMMENT 'Sanitized endpoint or evidence URL',
  `confidence` DECIMAL(5,4) NOT NULL DEFAULT 1.0000,
  `content_hash` CHAR(64) NOT NULL COMMENT 'Deduplication hash excluding observation time',
  `observed_at` DATETIME NOT NULL,
  `expires_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_llm_model_observation_hash` (`content_hash`),
  KEY `idx_llm_model_observation_lookup` (`provider_id`, `model_name`, `field_name`, `observed_at`),
  KEY `idx_llm_model_observation_expiry` (`expires_at`),
  CONSTRAINT `fk_llm_model_observations_provider` FOREIGN KEY (`provider_id`) REFERENCES `llm_providers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Field-level model metadata observations and provenance';

CREATE TABLE IF NOT EXISTS `embedding_providers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `provider_type` VARCHAR(64) NOT NULL,
  `endpoint` VARCHAR(512) NOT NULL,
  `api_key_encrypted` TEXT NOT NULL,
  `extra_headers` JSON NULL,
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_embedding_providers_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Embedding provider configuration';

CREATE TABLE IF NOT EXISTS `embedding_models` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `provider_id` BIGINT UNSIGNED NOT NULL,
  `model_name` VARCHAR(128) NOT NULL,
  `display_name` VARCHAR(256) NOT NULL DEFAULT '',
  `embedding_dim` INT NOT NULL DEFAULT 0,
  `input_token_limit` INT NOT NULL DEFAULT 0,
  `batch_size` INT NOT NULL DEFAULT 1,
  `timeout_seconds` INT NOT NULL DEFAULT 30,
  `max_retries` INT NOT NULL DEFAULT 2,
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `is_default` TINYINT(1) NOT NULL DEFAULT 0,
  `last_test_status` VARCHAR(32) NOT NULL DEFAULT 'untested',
  `last_test_error` TEXT NULL,
  `last_test_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_embedding_models_provider_model` (`provider_id`, `model_name`),
  KEY `idx_embedding_models_default` (`is_default`),
  KEY `idx_embedding_models_enabled_default` (`is_enabled`, `is_default`),
  CONSTRAINT `fk_embedding_models_provider` FOREIGN KEY (`provider_id`) REFERENCES `embedding_providers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Embedding model configuration';

CREATE TABLE IF NOT EXISTS `prompt_templates` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(256) NOT NULL COMMENT 'Template name',
  `content` TEXT NOT NULL COMMENT 'Prompt template content with {{variable}} placeholders',
  `variables` JSON COMMENT 'JSON array of variable names, e.g. ["hr_id","session_id"]',
  `version` INT NOT NULL DEFAULT 1 COMMENT 'Current version number',
  `is_active` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether template is active',
  `agent_type` VARCHAR(64) NOT NULL COMMENT 'hr_agent / candidate_assistant',
  `prompt_role` VARCHAR(32) NOT NULL DEFAULT 'system' COMMENT 'system / user',
  `created_by` BIGINT COMMENT 'Creator user ID',
  `updated_by` BIGINT COMMENT 'Last updater user ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_agent_type` (`agent_type`),
  KEY `idx_is_active` (`is_active`),
  KEY `idx_agent_type_active` (`agent_type`, `is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Prompt template definitions';

CREATE TABLE IF NOT EXISTS `prompt_versions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `template_id` BIGINT NOT NULL COMMENT 'FK to prompt_templates.id',
  `version` INT NOT NULL COMMENT 'Version number',
  `content` TEXT NOT NULL COMMENT 'Snapshot of prompt content at this version',
  `changed_by` BIGINT COMMENT 'User ID who made the change',
  `change_note` VARCHAR(512) COMMENT 'Description of what changed',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_template_version` (`template_id`, `version`),
  CONSTRAINT `fk_prompt_versions_template` FOREIGN KEY (`template_id`) REFERENCES `prompt_templates` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Prompt version history for audit and rollback';

CREATE TABLE IF NOT EXISTS `agent_configs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT 'Agent internal name (unique)',
  `display_name` VARCHAR(256) NOT NULL COMMENT 'Agent display name for UI',
  `description` TEXT COMMENT 'Agent description',
  `agent_type` VARCHAR(64) NOT NULL COMMENT 'hr_recruiting_agent / candidate_assistant / custom',
  `prompt_template_id` BIGINT COMMENT 'FK to prompt_templates.id, NULL = use system default',
  `instruction` TEXT COMMENT 'Extra instruction appended after Prompt',
  `max_iterations` INT NOT NULL DEFAULT 5 COMMENT 'Max tool call iterations',
  `temperature_override` DOUBLE COMMENT 'Overrides model default temperature, NULL = use model default',
  `is_default` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Whether this is the default agent for its agent_type',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether this agent is enabled',
  `default_key` VARCHAR(64) GENERATED ALWAYS AS (
    CASE WHEN `is_default` = 1 AND `is_enabled` = 1 THEN `agent_type` ELSE NULL END
  ) STORED COMMENT 'Enforces one enabled default per agent_type via unique index',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  UNIQUE KEY `uk_agent_default_type` (`default_key`),
  KEY `idx_agent_type` (`agent_type`),
  KEY `idx_agent_type_default` (`agent_type`, `is_default`),
  KEY `idx_prompt_template_id` (`prompt_template_id`),
  CONSTRAINT `fk_agent_configs_prompt` FOREIGN KEY (`prompt_template_id`) REFERENCES `prompt_templates` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent configuration definitions';

CREATE TABLE IF NOT EXISTS `agent_tool_bindings` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `agent_id` BIGINT NOT NULL COMMENT 'FK to agent_configs.id',
  `tool_name` VARCHAR(128) NOT NULL COMMENT 'Tool name identifier',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether the tool is enabled for this agent',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_tool` (`agent_id`, `tool_name`),
  KEY `idx_tool_name` (`tool_name`),
  CONSTRAINT `fk_agent_tool_bindings_agent` FOREIGN KEY (`agent_id`) REFERENCES `agent_configs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent to tool binding assignments';

CREATE TABLE IF NOT EXISTS `mcp_servers` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT 'MCP server display name',
  `description` TEXT NULL COMMENT 'Human-readable server description',
  `transport` VARCHAR(16) NOT NULL COMMENT 'Transport type: stdio / sse / http',
  `command_or_url` TEXT NOT NULL COMMENT 'Command (stdio) or URL (sse/http)',
  `args` JSON COMMENT 'Command arguments (JSON array string)',
  `env_vars` JSON COMMENT 'Environment variables (JSON object)',
  `timeout_seconds` INT NOT NULL DEFAULT 30 COMMENT 'Connection timeout',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Default disabled for safety',
  `status` VARCHAR(32) NOT NULL DEFAULT 'disconnected' COMMENT 'connected / disconnected / error',
  `tool_count` INT NOT NULL DEFAULT 0 COMMENT 'Number of tools discovered',
  `last_error` VARCHAR(512) NULL COMMENT 'Last connection or operation error message',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_mcp_servers_enabled` (`is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MCP server registry';

CREATE TABLE IF NOT EXISTS `mcp_tool_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `server_id` BIGINT NOT NULL COMMENT 'FK to mcp_servers.id',
  `tool_name` VARCHAR(128) NOT NULL COMMENT 'Called tool name',
  `args_json` JSON COMMENT 'Desensitized tool arguments',
  `result_content` TEXT COMMENT 'Desensitized/truncated tool result',
  `duration_ms` INT NOT NULL DEFAULT 0 COMMENT 'Execution time in ms',
  `error_msg` VARCHAR(512) COMMENT 'Error message if any',
  `called_by_hr_id` BIGINT COMMENT 'HR user who triggered the call',
  `session_id` BIGINT COMMENT 'AI chat session ID',
  `policy_id` BIGINT COMMENT 'MCP tool policy ID evaluated for this call',
  `policy_decision` VARCHAR(32) NOT NULL DEFAULT 'allow' COMMENT 'allow / deny / confirmation_required / rate_limited',
  `policy_reason` VARCHAR(512) COMMENT 'Policy decision reason',
  `policy_snapshot_json` JSON COMMENT 'Non-secret snapshot of evaluated policy',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_mcp_tool_logs_server` (`server_id`),
  KEY `idx_mcp_tool_logs_session` (`session_id`),
  KEY `idx_mcp_tool_logs_server_tool_created` (`server_id`, `tool_name`, `created_at`),
  CONSTRAINT `fk_mcp_tool_logs_server` FOREIGN KEY (`server_id`) REFERENCES `mcp_servers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MCP tool call audit logs';

CREATE TABLE IF NOT EXISTS `mcp_tool_policies` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `server_id` BIGINT NOT NULL COMMENT 'FK to mcp_servers.id',
  `tool_name` VARCHAR(128) NOT NULL COMMENT 'Governed MCP tool name',
  `effect` VARCHAR(32) NOT NULL DEFAULT 'allow' COMMENT 'allow / deny',
  `risk_level` VARCHAR(32) DEFAULT 'medium' COMMENT 'low / medium / high / critical',
  `require_confirmation` TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Whether caller confirmation is required',
  `allowed_roles_json` JSON COMMENT 'Allowed caller roles JSON array',
  `allowed_scopes_json` JSON COMMENT 'Allowed caller scopes JSON array',
  `required_args_json` JSON COMMENT 'Required argument names JSON array',
  `denied_args_json` JSON COMMENT 'Forbidden argument names JSON array',
  `arg_rules_json` JSON COMMENT 'Per-argument validation rules JSON object',
  `redact_fields_json` JSON COMMENT 'Additional fields to redact in logs JSON array',
  `rate_limit_window_seconds` INT NOT NULL DEFAULT 0 COMMENT 'Rolling rate limit window, 0 disables',
  `rate_limit_max_calls` INT NOT NULL DEFAULT 0 COMMENT 'Max calls in window, 0 disables',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether this policy is active',
  `created_by_hr_id` BIGINT COMMENT 'Creator HR ID',
  `updated_by_hr_id` BIGINT COMMENT 'Last updater HR ID',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_mcp_tool_policy_server_tool` (`server_id`, `tool_name`),
  KEY `idx_mcp_tool_policies_enabled` (`is_enabled`),
  CONSTRAINT `fk_mcp_tool_policies_server` FOREIGN KEY (`server_id`) REFERENCES `mcp_servers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MCP tool governance policies';

CREATE TABLE IF NOT EXISTS `agent_capability_bindings` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `agent_id` BIGINT NOT NULL COMMENT 'FK to agent_configs.id',
  `capability_source` VARCHAR(32) NOT NULL COMMENT 'builtin / mcp',
  `capability_key` VARCHAR(256) NOT NULL COMMENT 'builtin: tool_name; mcp: server_id:tool_name',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether the capability is enabled for this agent',
  `priority` INT NOT NULL DEFAULT 0 COMMENT 'Capability ordering hint',
  `policy_json` JSON COMMENT 'Optional capability policy payload',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_capability` (`agent_id`, `capability_source`, `capability_key`),
  KEY `idx_agent_capability_source` (`capability_source`),
  KEY `idx_agent_capability_key` (`capability_key`),
  CONSTRAINT `fk_agent_capability_bindings_agent` FOREIGN KEY (`agent_id`) REFERENCES `agent_configs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent to unified capability binding assignments';

CREATE TABLE IF NOT EXISTS `agent_skills` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `display_name` VARCHAR(128) NOT NULL,
  `description` TEXT,
  `current_version_id` BIGINT NULL,
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `is_manual_invocable` TINYINT(1) NOT NULL DEFAULT 1,
  `trigger_keywords` JSON NULL,
  `agent_type` VARCHAR(64) NOT NULL DEFAULT 'hr_recruiting_agent',
  `category` VARCHAR(64) NOT NULL DEFAULT 'general',
  `scenario` VARCHAR(128) NOT NULL DEFAULT '',
  `priority` INT NOT NULL DEFAULT 0,
  `risk_level` VARCHAR(32) NOT NULL DEFAULT 'medium',
  `required_capabilities` JSON NULL,
  `output_schema` JSON NULL,
  `evaluation_criteria` JSON NULL,
  `semantic_tags` JSON NULL,
  `created_by` BIGINT NULL,
  `updated_by` BIGINT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skills_name` (`name`),
  KEY `idx_agent_skills_enabled` (`is_enabled`),
  KEY `idx_agent_skills_governance` (`agent_type`, `category`, `priority`),
  KEY `idx_agent_skills_current_version` (`current_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Governed Agent SKILL.md registry';

CREATE TABLE IF NOT EXISTS `agent_skill_versions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `skill_id` BIGINT NOT NULL,
  `version` VARCHAR(64) NOT NULL,
  `flow_json` JSON NULL,
  `skill_md` MEDIUMTEXT NOT NULL,
  `frontmatter_json` JSON NULL,
  `body_markdown` MEDIUMTEXT,
  `change_note` TEXT,
  `created_by` BIGINT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skill_versions_skill_version` (`skill_id`, `version`),
  KEY `idx_agent_skill_versions_skill` (`skill_id`),
  CONSTRAINT `fk_agent_skill_versions_skill` FOREIGN KEY (`skill_id`) REFERENCES `agent_skills` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Immutable Agent SKILL.md versions';

CREATE TABLE IF NOT EXISTS `platform_ai_capabilities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `capability_key` VARCHAR(96) NOT NULL,
  `audience` VARCHAR(32) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `description` VARCHAR(500) DEFAULT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `current_published_version_id` BIGINT UNSIGNED DEFAULT NULL,
  `created_by` BIGINT UNSIGNED DEFAULT NULL,
  `updated_by` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_ai_capability_audience_key` (`audience`, `capability_key`),
  KEY `idx_platform_ai_capabilities_status` (`status`, `audience`),
  CONSTRAINT `chk_platform_ai_capabilities_audience` CHECK (`audience` IN ('tenant_hr', 'candidate')),
  CONSTRAINT `chk_platform_ai_capabilities_status` CHECK (`status` IN ('active', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Platform-owned AI capability catalogue';

CREATE TABLE IF NOT EXISTS `platform_ai_capability_versions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `capability_id` BIGINT UNSIGNED NOT NULL,
  `version` INT UNSIGNED NOT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'draft',
  `snapshot_json` JSON NOT NULL,
  `snapshot_hash` CHAR(64) NOT NULL,
  `change_note` VARCHAR(500) DEFAULT NULL,
  `created_by` BIGINT UNSIGNED DEFAULT NULL,
  `published_by` BIGINT UNSIGNED DEFAULT NULL,
  `published_at` DATETIME(3) DEFAULT NULL,
  `retired_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_ai_capability_versions_number` (`capability_id`, `version`),
  KEY `idx_platform_ai_capability_versions_hash` (`capability_id`, `snapshot_hash`),
  KEY `idx_platform_ai_capability_versions_status` (`status`, `published_at`),
  CONSTRAINT `fk_platform_ai_capability_versions_capability` FOREIGN KEY (`capability_id`) REFERENCES `platform_ai_capabilities` (`id`),
  CONSTRAINT `chk_platform_ai_capability_versions_status` CHECK (`status` IN ('draft', 'published', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable snapshots for published AI capability releases';

ALTER TABLE `platform_ai_capabilities`
  ADD CONSTRAINT `fk_platform_ai_capabilities_current_version`
  FOREIGN KEY (`current_published_version_id`) REFERENCES `platform_ai_capability_versions` (`id`);

CREATE TABLE IF NOT EXISTS `platform_ai_config_audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `actor_user_id` BIGINT UNSIGNED DEFAULT NULL,
  `action` VARCHAR(64) NOT NULL,
  `resource_type` VARCHAR(64) NOT NULL,
  `resource_id` BIGINT UNSIGNED DEFAULT NULL,
  `capability_id` BIGINT UNSIGNED DEFAULT NULL,
  `capability_version_id` BIGINT UNSIGNED DEFAULT NULL,
  `before_snapshot` JSON DEFAULT NULL,
  `after_snapshot` JSON DEFAULT NULL,
  `request_id` VARCHAR(128) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_platform_ai_config_audit_actor` (`actor_user_id`, `created_at`),
  KEY `idx_platform_ai_config_audit_resource` (`resource_type`, `resource_id`, `created_at`),
  KEY `idx_platform_ai_config_audit_capability` (`capability_id`, `capability_version_id`, `created_at`),
  CONSTRAINT `fk_platform_ai_config_audit_capability` FOREIGN KEY (`capability_id`) REFERENCES `platform_ai_capabilities` (`id`),
  CONSTRAINT `fk_platform_ai_config_audit_version` FOREIGN KEY (`capability_version_id`) REFERENCES `platform_ai_capability_versions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Atomic audit trail for platform AI configuration and releases';

-- Runtime and business evidence remains tenant-scoped. Technical AI
-- configuration tables are platform-global and intentionally have no tenant_id.
ALTER TABLE ai_chat_sessions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_chat_sessions_tenant_owner (tenant_id, owner_role, owner_id, updated_at), ADD CONSTRAINT fk_ai_chat_sessions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_chat_history ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_chat_history_tenant_session (tenant_id, session_id, created_at), ADD CONSTRAINT fk_ai_chat_history_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_session_summaries ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_session_summaries_tenant_session (tenant_id, session_id), ADD CONSTRAINT fk_ai_session_summaries_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_tool_traces ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_tool_traces_tenant_session (tenant_id, session_id, created_at), ADD CONSTRAINT fk_ai_tool_traces_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_runs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_runs_tenant_session (tenant_id, session_id, created_at), ADD CONSTRAINT fk_agent_runs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_run_events ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_run_events_tenant_run (tenant_id, run_id, seq), ADD CONSTRAINT fk_agent_run_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_run_steps ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_run_steps_tenant_run (tenant_id, run_id, step_index), ADD CONSTRAINT fk_agent_run_steps_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_memories ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_memories_tenant_owner (tenant_id, hr_id, scope_type, scope_id), ADD CONSTRAINT fk_ai_memories_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_embeddings ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_embeddings_tenant_object (tenant_id, object_type, object_id), ADD CONSTRAINT fk_ai_embeddings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE notifications ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_notifications_tenant_receiver (tenant_id, receiver_id, receiver_account_type, created_at), ADD CONSTRAINT fk_notifications_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE event_outbox ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_event_outbox_tenant_status (tenant_id, status, created_at), ADD CONSTRAINT fk_event_outbox_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE email_logs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_email_logs_tenant_created (tenant_id, created_at), ADD CONSTRAINT fk_email_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE event_inbox ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_event_inbox_tenant_consumer (tenant_id, consumer_name, received_at), ADD CONSTRAINT fk_event_inbox_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE analytics_projection_events ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_projection_events_tenant_projection (tenant_id, projection_name, created_at), ADD CONSTRAINT fk_projection_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE third_party_usage_logs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_usage_logs_tenant_created (tenant_id, created_at), ADD CONSTRAINT fk_usage_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_usage_auth_contexts ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_usage_auth_tenant_actor (tenant_id, actor_user_id, created_at), ADD CONSTRAINT fk_ai_usage_auth_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE mcp_tool_logs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_mcp_tool_logs_tenant_created (tenant_id, created_at), ADD CONSTRAINT fk_mcp_tool_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

CREATE TABLE IF NOT EXISTS `platform_audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `actor_user_id` BIGINT UNSIGNED NOT NULL,
  `action` VARCHAR(128) NOT NULL,
  `resource_type` VARCHAR(64) NOT NULL,
  `resource_id` BIGINT UNSIGNED NULL,
  `target_tenant_id` BIGINT UNSIGNED NULL,
  `before_json` JSON NULL,
  `after_json` JSON NULL,
  `request_id` VARCHAR(128) NOT NULL DEFAULT '',
  `client_ip` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_platform_audit_actor_created` (`actor_user_id`, `created_at`),
  KEY `idx_platform_audit_tenant_created` (`target_tenant_id`, `created_at`),
  KEY `idx_platform_audit_action_created` (`action`, `created_at`),
  CONSTRAINT `fk_platform_audit_actor` FOREIGN KEY (`actor_user_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_platform_audit_target_tenant` FOREIGN KEY (`target_tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Immutable platform control-plane audit trail';

CREATE TABLE IF NOT EXISTS `platform_plans` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `plan_key` VARCHAR(64) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `description` VARCHAR(500) DEFAULT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active',
  `created_by` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_plans_key` (`plan_key`),
  KEY `idx_platform_plans_status` (`status`),
  CONSTRAINT `chk_platform_plans_status` CHECK (`status` IN ('active', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Platform plan catalogue';

CREATE TABLE IF NOT EXISTS `platform_plan_versions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `plan_id` BIGINT UNSIGNED NOT NULL,
  `version` INT UNSIGNED NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'draft',
  `effective_at` DATETIME DEFAULT NULL,
  `retired_at` DATETIME DEFAULT NULL,
  `change_note` VARCHAR(500) DEFAULT NULL,
  `created_by` BIGINT UNSIGNED DEFAULT NULL,
  `published_by` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_plan_versions_plan_version` (`plan_id`, `version`),
  KEY `idx_platform_plan_versions_status_effective` (`status`, `effective_at`),
  CONSTRAINT `fk_platform_plan_versions_plan` FOREIGN KEY (`plan_id`) REFERENCES `platform_plans` (`id`),
  CONSTRAINT `chk_platform_plan_versions_status` CHECK (`status` IN ('draft', 'published', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable published plan versions';

CREATE TABLE IF NOT EXISTS `platform_plan_entitlements` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `plan_version_id` BIGINT UNSIGNED NOT NULL,
  `entitlement_key` VARCHAR(96) NOT NULL,
  `value_type` VARCHAR(16) NOT NULL DEFAULT 'integer',
  `value_json` JSON NOT NULL,
  `enforcement_mode` VARCHAR(16) NOT NULL DEFAULT 'hard',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_plan_entitlements_version_key` (`plan_version_id`, `entitlement_key`),
  CONSTRAINT `fk_platform_plan_entitlements_version` FOREIGN KEY (`plan_version_id`) REFERENCES `platform_plan_versions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_platform_plan_entitlements_type` CHECK (`value_type` IN ('integer', 'boolean', 'string')),
  CONSTRAINT `chk_platform_plan_entitlements_mode` CHECK (`enforcement_mode` IN ('hard', 'soft', 'observe'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Entitlements attached to a plan version';

CREATE TABLE IF NOT EXISTS `tenant_subscriptions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `plan_version_id` BIGINT UNSIGNED NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active',
  `starts_at` DATETIME NOT NULL,
  `ends_at` DATETIME DEFAULT NULL,
  `reason` VARCHAR(500) NOT NULL,
  `created_by` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_tenant_subscriptions_tenant_status` (`tenant_id`, `status`, `starts_at`),
  KEY `idx_tenant_subscriptions_plan_version` (`plan_version_id`),
  CONSTRAINT `fk_tenant_subscriptions_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_tenant_subscriptions_plan_version` FOREIGN KEY (`plan_version_id`) REFERENCES `platform_plan_versions` (`id`),
  CONSTRAINT `chk_tenant_subscriptions_status` CHECK (`status` IN ('scheduled', 'active', 'expired', 'cancelled'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tenant plan subscription history';

CREATE TABLE IF NOT EXISTS `tenant_entitlement_overrides` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `entitlement_key` VARCHAR(96) NOT NULL,
  `value_type` VARCHAR(16) NOT NULL DEFAULT 'integer',
  `value_json` JSON NOT NULL,
  `reason` VARCHAR(500) NOT NULL,
  `expires_at` DATETIME DEFAULT NULL,
  `created_by` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_entitlement_overrides_key` (`tenant_id`, `entitlement_key`),
  KEY `idx_tenant_entitlement_overrides_expiry` (`expires_at`),
  CONSTRAINT `fk_tenant_entitlement_overrides_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Temporary tenant-specific entitlement overrides';

CREATE TABLE IF NOT EXISTS `tenant_usage_snapshots` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `metric_key` VARCHAR(96) NOT NULL,
  `metric_value` BIGINT NOT NULL DEFAULT 0,
  `window_start` DATETIME NOT NULL,
  `window_end` DATETIME NOT NULL,
  `measured_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_usage_snapshots_window` (`tenant_id`, `metric_key`, `window_start`, `window_end`),
  KEY `idx_tenant_usage_snapshots_metric_measured` (`metric_key`, `measured_at`),
  CONSTRAINT `fk_tenant_usage_snapshots_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Auditable tenant usage snapshots';

CREATE TABLE IF NOT EXISTS `platform_quota_alerts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `metric_key` VARCHAR(96) NOT NULL,
  `threshold_percent` INT NOT NULL,
  `usage_value` BIGINT NOT NULL,
  `quota_value` BIGINT NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'open',
  `assignee_user_id` BIGINT UNSIGNED DEFAULT NULL,
  `acknowledged_at` DATETIME DEFAULT NULL,
  `resolved_at` DATETIME DEFAULT NULL,
  `resolution_note` VARCHAR(500) DEFAULT NULL,
  `first_triggered_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_triggered_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_platform_quota_alerts_tenant_metric_status` (`tenant_id`, `metric_key`, `status`),
  KEY `idx_platform_quota_alerts_status_triggered` (`status`, `last_triggered_at`),
  CONSTRAINT `fk_platform_quota_alerts_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_platform_quota_alerts_threshold` CHECK (`threshold_percent` IN (80, 90, 100)),
  CONSTRAINT `chk_platform_quota_alerts_status` CHECK (`status` IN ('open', 'acknowledged', 'resolved'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Quota threshold operational alerts';

INSERT INTO `platform_plans` (`plan_key`, `name`, `description`, `status`) VALUES
  ('starter', '基础版', '适合小型招聘团队的基础套餐', 'active'),
  ('growth', '成长版', '适合持续招聘和协作的成长套餐', 'active'),
  ('enterprise', '企业版', '适合大型组织的企业治理套餐', 'active')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

INSERT INTO `platform_plan_versions` (`plan_id`, `version`, `status`, `effective_at`, `change_note`)
SELECT `id`, 1, 'published', NOW(), 'initial catalogue' FROM `platform_plans`
ON DUPLICATE KEY UPDATE `plan_id` = VALUES(`plan_id`);

INSERT INTO `platform_plan_entitlements` (`plan_version_id`, `entitlement_key`, `value_type`, `value_json`, `enforcement_mode`)
SELECT version.id, entitlement.entitlement_key, 'integer', entitlement.value_json, 'hard'
FROM `platform_plan_versions` version
JOIN `platform_plans` plan ON plan.id = version.plan_id AND version.version = 1
JOIN (
  SELECT 'starter' plan_key, 'members.max' entitlement_key, CAST(10 AS JSON) value_json UNION ALL
  SELECT 'starter', 'jobs.published.max', CAST(20 AS JSON) UNION ALL
  SELECT 'starter', 'applications.monthly.max', CAST(500 AS JSON) UNION ALL
  SELECT 'starter', 'resumes.storage.max', CAST(1000 AS JSON) UNION ALL
  SELECT 'growth', 'members.max', CAST(50 AS JSON) UNION ALL
  SELECT 'growth', 'jobs.published.max', CAST(100 AS JSON) UNION ALL
  SELECT 'growth', 'applications.monthly.max', CAST(5000 AS JSON) UNION ALL
  SELECT 'growth', 'resumes.storage.max', CAST(10000 AS JSON) UNION ALL
  SELECT 'enterprise', 'members.max', CAST(500 AS JSON) UNION ALL
  SELECT 'enterprise', 'jobs.published.max', CAST(1000 AS JSON) UNION ALL
  SELECT 'enterprise', 'applications.monthly.max', CAST(100000 AS JSON) UNION ALL
  SELECT 'enterprise', 'resumes.storage.max', CAST(500000 AS JSON)
) entitlement ON entitlement.plan_key = plan.plan_key
ON DUPLICATE KEY UPDATE `value_json` = VALUES(`value_json`), `enforcement_mode` = VALUES(`enforcement_mode`);

-- Migration 000069: AI billing foundation.
-- AI billing foundation for tenant and candidate owners, including sandbox-isolated payments.

CREATE TABLE IF NOT EXISTS `billing_products` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `product_key` VARCHAR(64) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `description` VARCHAR(500) DEFAULT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `product_type` VARCHAR(24) NOT NULL DEFAULT 'subscription',
  `status` VARCHAR(16) NOT NULL DEFAULT 'draft',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_products_key` (`product_key`),
  KEY `idx_billing_products_owner_status` (`owner_type`, `status`),
  CONSTRAINT `chk_billing_products_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_billing_products_type` CHECK (`product_type` IN ('subscription', 'credit_pack')),
  CONSTRAINT `chk_billing_products_status` CHECK (`status` IN ('draft', 'active', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Billing product catalogue';

CREATE TABLE IF NOT EXISTS `billing_price_versions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `product_id` BIGINT UNSIGNED NOT NULL,
  `version` INT UNSIGNED NOT NULL,
  `billing_term` VARCHAR(16) NOT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `included_credits` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `entitlement_snapshot` JSON DEFAULT NULL,
  `platform_plan_version_id` BIGINT UNSIGNED DEFAULT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'draft',
  `effective_at` DATETIME(3) DEFAULT NULL,
  `retired_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_price_versions_product_version` (`product_id`, `version`),
  KEY `idx_billing_price_versions_status_effective` (`status`, `effective_at`),
  KEY `idx_billing_price_versions_platform_plan` (`platform_plan_version_id`),
  CONSTRAINT `fk_billing_price_versions_product` FOREIGN KEY (`product_id`) REFERENCES `billing_products` (`id`),
  CONSTRAINT `fk_billing_price_versions_platform_plan` FOREIGN KEY (`platform_plan_version_id`) REFERENCES `platform_plan_versions` (`id`),
  CONSTRAINT `chk_billing_price_versions_term` CHECK (`billing_term` IN ('monthly', 'yearly', 'one_time')),
  CONSTRAINT `chk_billing_price_versions_currency` CHECK (`currency` = 'CNY'),
  CONSTRAINT `chk_billing_price_versions_status` CHECK (`status` IN ('draft', 'published', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable billing price versions';

CREATE TABLE IF NOT EXISTS `billing_subscriptions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `product_id` BIGINT UNSIGNED NOT NULL,
  `price_version_id` BIGINT UNSIGNED NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending',
  `active_slot` TINYINT UNSIGNED DEFAULT NULL,
  `term` VARCHAR(16) NOT NULL,
  `current_period_start` DATETIME(3) NOT NULL,
  `current_period_end` DATETIME(3) NOT NULL,
  `next_price_version_id` BIGINT UNSIGNED DEFAULT NULL,
  `next_term` VARCHAR(16) DEFAULT NULL,
  `cancel_at_period_end` TINYINT(1) NOT NULL DEFAULT 0,
  `activated_by_order_id` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_billing_subscriptions_owner_status` (`owner_type`, `owner_id`, `status`),
  KEY `idx_billing_subscriptions_period_end` (`status`, `current_period_end`),
  CONSTRAINT `fk_billing_subscriptions_product` FOREIGN KEY (`product_id`) REFERENCES `billing_products` (`id`),
  CONSTRAINT `fk_billing_subscriptions_price` FOREIGN KEY (`price_version_id`) REFERENCES `billing_price_versions` (`id`),
  CONSTRAINT `fk_billing_subscriptions_next_price` FOREIGN KEY (`next_price_version_id`) REFERENCES `billing_price_versions` (`id`),
  CONSTRAINT `chk_billing_subscriptions_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_billing_subscriptions_status` CHECK (`status` IN ('pending', 'active', 'past_due', 'expired', 'cancelled')),
  CONSTRAINT `chk_billing_subscriptions_term` CHECK (`term` IN ('monthly', 'yearly')),
  CONSTRAINT `chk_billing_subscriptions_next_term` CHECK (`next_term` IS NULL OR `next_term` IN ('monthly', 'yearly')),
  CONSTRAINT `chk_billing_subscriptions_period` CHECK (`current_period_end` > `current_period_start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tenant and candidate billing subscriptions';

CREATE TABLE IF NOT EXISTS `billing_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_no` VARCHAR(64) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `order_type` VARCHAR(24) NOT NULL,
  `product_id` BIGINT UNSIGNED NOT NULL,
  `price_version_id` BIGINT UNSIGNED NOT NULL,
  `subscription_id` BIGINT UNSIGNED DEFAULT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending',
  `payment_environment` VARCHAR(16) NOT NULL DEFAULT 'sandbox',
  `idempotency_key` VARCHAR(128) NOT NULL,
  `expires_at` DATETIME(3) NOT NULL,
  `paid_at` DATETIME(3) DEFAULT NULL,
  `closed_at` DATETIME(3) DEFAULT NULL,
  `metadata` JSON DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_orders_no` (`order_no`),
  UNIQUE KEY `uk_billing_orders_owner_idempotency` (`owner_type`, `owner_id`, `idempotency_key`),
  UNIQUE KEY `uk_billing_orders_owner_active` (`owner_type`, `owner_id`, `active_slot`),
  KEY `idx_billing_orders_owner_created` (`owner_type`, `owner_id`, `created_at`),
  KEY `idx_billing_orders_status_expiry` (`status`, `expires_at`),
  CONSTRAINT `fk_billing_orders_product` FOREIGN KEY (`product_id`) REFERENCES `billing_products` (`id`),
  CONSTRAINT `fk_billing_orders_price` FOREIGN KEY (`price_version_id`) REFERENCES `billing_price_versions` (`id`),
  CONSTRAINT `fk_billing_orders_subscription` FOREIGN KEY (`subscription_id`) REFERENCES `billing_subscriptions` (`id`),
  CONSTRAINT `chk_billing_orders_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_billing_orders_type` CHECK (`order_type` IN ('subscribe', 'renew', 'upgrade', 'credit_pack')),
  CONSTRAINT `chk_billing_orders_status` CHECK (`status` IN ('pending', 'paying', 'paid', 'closed', 'refunding', 'refunded')),
  CONSTRAINT `chk_billing_orders_environment` CHECK (`payment_environment` IN ('sandbox', 'production')),
  CONSTRAINT `chk_billing_orders_currency` CHECK (`currency` = 'CNY'),
  CONSTRAINT `chk_billing_orders_active_slot` CHECK (`active_slot` IS NULL OR `active_slot` = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Billing orders with payment-environment isolation';

ALTER TABLE `billing_subscriptions`
  ADD CONSTRAINT `fk_billing_subscriptions_activation_order`
  FOREIGN KEY (`activated_by_order_id`) REFERENCES `billing_orders` (`id`);

CREATE TABLE IF NOT EXISTS `billing_payments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `payment_no` VARCHAR(64) NOT NULL,
  `merchant_order_no` VARCHAR(64) NOT NULL,
  `channel` VARCHAR(24) NOT NULL DEFAULT 'alipay',
  `scene` VARCHAR(16) NOT NULL,
  `source_app` VARCHAR(16) NOT NULL DEFAULT 'hr',
  `payment_environment` VARCHAR(16) NOT NULL DEFAULT 'sandbox',
  `channel_trade_no` VARCHAR(128) DEFAULT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `status` VARCHAR(24) NOT NULL DEFAULT 'created',
  `active_slot` TINYINT UNSIGNED DEFAULT NULL,
  `pay_payload` MEDIUMTEXT DEFAULT NULL,
  `return_token_hash` CHAR(64) DEFAULT NULL,
  `expires_at` DATETIME(3) DEFAULT NULL,
  `next_reconcile_at` DATETIME(3) DEFAULT NULL,
  `reconcile_attempts` INT UNSIGNED NOT NULL DEFAULT 0,
  `last_reconcile_error` VARCHAR(500) DEFAULT NULL,
  `last_queried_at` DATETIME(3) DEFAULT NULL,
  `paid_at` DATETIME(3) DEFAULT NULL,
  `closed_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_payments_no` (`payment_no`),
  UNIQUE KEY `uk_billing_payments_merchant_order` (`payment_environment`, `merchant_order_no`),
  UNIQUE KEY `uk_billing_payments_channel_trade` (`channel`, `payment_environment`, `channel_trade_no`),
  UNIQUE KEY `uk_billing_payments_order_active` (`order_id`, `active_slot`),
  UNIQUE KEY `uk_billing_payments_return_token` (`return_token_hash`),
  KEY `idx_billing_payments_order_status` (`order_id`, `status`),
  KEY `idx_billing_payments_reconcile` (`payment_environment`, `next_reconcile_at`, `status`),
  CONSTRAINT `fk_billing_payments_order` FOREIGN KEY (`order_id`) REFERENCES `billing_orders` (`id`),
  CONSTRAINT `chk_billing_payments_channel` CHECK (`channel` = 'alipay'),
  CONSTRAINT `chk_billing_payments_scene` CHECK (`scene` IN ('desktop', 'wap')),
  CONSTRAINT `chk_billing_payments_environment` CHECK (`payment_environment` IN ('sandbox', 'production')),
  CONSTRAINT `chk_billing_payments_currency` CHECK (`currency` = 'CNY'),
  CONSTRAINT `chk_billing_payments_status` CHECK (`status` IN ('created', 'pending', 'closing', 'unknown', 'succeeded', 'failed', 'closed', 'refunded')),
  CONSTRAINT `chk_billing_payments_source_app` CHECK (`source_app` IN ('hr', 'candidate')),
  CONSTRAINT `chk_billing_payments_active_slot` CHECK (`active_slot` IS NULL OR `active_slot` = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Payment attempts; sandbox and production never mix';

CREATE TABLE IF NOT EXISTS `billing_refunds` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `refund_no` VARCHAR(64) NOT NULL,
  `idempotency_key` VARCHAR(128) NOT NULL,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `payment_id` BIGINT UNSIGNED NOT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `reason` VARCHAR(500) NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'requested',
  `active_slot` TINYINT UNSIGNED DEFAULT NULL,
  `review_mode` VARCHAR(16) NOT NULL DEFAULT 'automatic',
  `channel_refund_no` VARCHAR(128) DEFAULT NULL,
  `requested_by` BIGINT UNSIGNED NOT NULL,
  `reviewed_by` BIGINT UNSIGNED DEFAULT NULL,
  `reviewed_at` DATETIME(3) DEFAULT NULL,
  `succeeded_at` DATETIME(3) DEFAULT NULL,
  `next_reconcile_at` DATETIME(3) DEFAULT NULL,
  `reconcile_attempts` INT UNSIGNED NOT NULL DEFAULT 0,
  `last_error` VARCHAR(500) DEFAULT NULL,
  `rejected_reason` VARCHAR(500) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_refunds_no` (`refund_no`),
  UNIQUE KEY `uk_billing_refunds_order_idempotency` (`order_id`, `idempotency_key`),
  UNIQUE KEY `uk_billing_refunds_payment_active` (`payment_id`, `active_slot`),
  KEY `idx_billing_refunds_order_status` (`order_id`, `status`),
  KEY `idx_billing_refunds_reconcile` (`next_reconcile_at`, `status`),
  CONSTRAINT `fk_billing_refunds_order` FOREIGN KEY (`order_id`) REFERENCES `billing_orders` (`id`),
  CONSTRAINT `fk_billing_refunds_payment` FOREIGN KEY (`payment_id`) REFERENCES `billing_payments` (`id`),
  CONSTRAINT `chk_billing_refunds_status` CHECK (`status` IN ('requested', 'reviewing', 'approved', 'processing', 'unknown', 'succeeded', 'failed', 'rejected')),
  CONSTRAINT `chk_billing_refunds_review` CHECK (`review_mode` IN ('automatic', 'manual')),
  CONSTRAINT `chk_billing_refunds_active_slot` CHECK (`active_slot` IS NULL OR `active_slot` = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Refund requests and channel results';

CREATE TABLE IF NOT EXISTS `billing_webhook_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `channel` VARCHAR(24) NOT NULL DEFAULT 'alipay',
  `payment_environment` VARCHAR(16) NOT NULL DEFAULT 'sandbox',
  `event_key` VARCHAR(191) NOT NULL,
  `event_type` VARCHAR(64) NOT NULL,
  `signature_verified` TINYINT(1) NOT NULL DEFAULT 0,
  `payload_sha256` CHAR(64) NOT NULL,
  `payload` JSON NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'received',
  `failure_reason` VARCHAR(500) DEFAULT NULL,
  `retry_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `last_attempt_at` DATETIME(3) DEFAULT NULL,
  `processed_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_webhook_events_key` (`channel`, `payment_environment`, `event_key`),
  KEY `idx_billing_webhook_events_status_created` (`status`, `created_at`),
  CONSTRAINT `chk_billing_webhook_events_channel` CHECK (`channel` = 'alipay'),
  CONSTRAINT `chk_billing_webhook_events_environment` CHECK (`payment_environment` IN ('sandbox', 'production')),
  CONSTRAINT `chk_billing_webhook_events_status` CHECK (`status` IN ('received', 'verified', 'processed', 'ignored', 'failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Idempotent payment callback inbox';

CREATE TABLE IF NOT EXISTS `ai_rate_cards` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `version` INT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `input_micros_per_1k_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `output_micros_per_1k_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `cached_input_micros_per_1k_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `credit_micros` BIGINT UNSIGNED NOT NULL DEFAULT 1000000,
  `status` VARCHAR(16) NOT NULL DEFAULT 'draft',
  `effective_at` DATETIME(3) DEFAULT NULL,
  `retired_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_rate_cards_model_version` (`provider_key`, `model_key`, `version`),
  KEY `idx_ai_rate_cards_effective` (`provider_key`, `model_key`, `status`, `effective_at`),
  CONSTRAINT `chk_ai_rate_cards_status` CHECK (`status` IN ('draft', 'published', 'retired')),
  CONSTRAINT `chk_ai_rate_cards_currency` CHECK (`currency` = 'CNY'),
  CONSTRAINT `chk_ai_rate_cards_credit_micros` CHECK (`credit_micros` > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable AI supplier cost and credit conversion rates';

CREATE TABLE IF NOT EXISTS `ai_usage_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `tenant_id` BIGINT UNSIGNED DEFAULT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `reservation_id` BIGINT UNSIGNED DEFAULT NULL,
  `operation` VARCHAR(64) NOT NULL,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `provider_call_seq` INT UNSIGNED NOT NULL DEFAULT 1,
  `input_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `output_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `cached_input_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `supplier_cost_micros` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `credits_charged` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `usage_source` VARCHAR(16) NOT NULL DEFAULT 'provider',
  `provider_request_id` VARCHAR(191) DEFAULT NULL,
  `occurred_at` DATETIME(3) NOT NULL,
  `metadata` JSON DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_usage_events_event_id` (`event_id`),
  UNIQUE KEY `uk_ai_usage_events_provider_call` (`reservation_id`, `provider_call_seq`),
  KEY `idx_ai_usage_events_owner_occurred` (`owner_type`, `owner_id`, `occurred_at`),
  KEY `idx_ai_usage_events_tenant_occurred` (`tenant_id`, `occurred_at`),
  KEY `idx_ai_usage_events_provider_model` (`provider_key`, `model_key`, `occurred_at`),
  CONSTRAINT `chk_ai_usage_events_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_usage_events_owner_tenant` CHECK ((`owner_type` = 'tenant' AND `tenant_id` = `owner_id`) OR (`owner_type` = 'user')),
  CONSTRAINT `chk_ai_usage_events_source` CHECK (`usage_source` IN ('provider', 'estimated'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable billable AI usage facts';

CREATE TABLE IF NOT EXISTS `ai_credit_grants` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `grant_type` VARCHAR(24) NOT NULL,
  `source_type` VARCHAR(24) NOT NULL,
  `source_id` BIGINT UNSIGNED DEFAULT NULL,
  `total_credits` BIGINT UNSIGNED NOT NULL,
  `remaining_credits` BIGINT UNSIGNED NOT NULL,
  `valid_from` DATETIME(3) NOT NULL,
  `expires_at` DATETIME(3) DEFAULT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_credit_grants_source` (`owner_type`, `owner_id`, `source_type`, `source_id`, `valid_from`),
  KEY `idx_ai_credit_grants_consume` (`owner_type`, `owner_id`, `status`, `expires_at`, `valid_from`),
  CONSTRAINT `chk_ai_credit_grants_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_credit_grants_type` CHECK (`grant_type` IN ('free_monthly', 'subscription_monthly', 'credit_pack', 'manual_adjustment')),
  CONSTRAINT `chk_ai_credit_grants_source` CHECK (`source_type` IN ('subscription', 'order', 'manual', 'system')),
  CONSTRAINT `chk_ai_credit_grants_balance` CHECK (`remaining_credits` <= `total_credits`),
  CONSTRAINT `chk_ai_credit_grants_status` CHECK (`status` IN ('active', 'exhausted', 'expired', 'revoked'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Expiring AI credit buckets consumed earliest-expiry first';

CREATE TABLE IF NOT EXISTS `ai_credit_reservations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `reservation_no` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `capability` VARCHAR(96) NOT NULL,
  `operation` VARCHAR(64) NOT NULL,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `idempotency_key` VARCHAR(191) NOT NULL,
  `reserved_credits` BIGINT UNSIGNED NOT NULL,
  `settled_credits` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `enforcement_mode` VARCHAR(16) NOT NULL DEFAULT 'shadow',
  `expires_at` DATETIME(3) NOT NULL,
  `settled_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_credit_reservations_no` (`reservation_no`),
  UNIQUE KEY `uk_ai_credit_reservations_owner_idempotency` (`owner_type`, `owner_id`, `idempotency_key`),
  KEY `idx_ai_credit_reservations_expiry` (`status`, `expires_at`),
  CONSTRAINT `chk_ai_credit_reservations_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_credit_reservations_status` CHECK (`status` IN ('active', 'settled', 'cancelled', 'expired')),
  CONSTRAINT `chk_ai_credit_reservations_mode` CHECK (`enforcement_mode` IN ('shadow', 'enforce'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Idempotent pre-provider AI credit reservations';

ALTER TABLE `ai_usage_events`
  ADD CONSTRAINT `fk_ai_usage_events_reservation`
  FOREIGN KEY (`reservation_id`) REFERENCES `ai_credit_reservations` (`id`);

CREATE TABLE IF NOT EXISTS `ai_credit_ledger` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `entry_id` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `grant_id` BIGINT UNSIGNED DEFAULT NULL,
  `reservation_id` BIGINT UNSIGNED DEFAULT NULL,
  `usage_event_id` BIGINT UNSIGNED DEFAULT NULL,
  `entry_type` VARCHAR(24) NOT NULL,
  `credits_delta` BIGINT NOT NULL,
  `balance_after` BIGINT NOT NULL,
  `idempotency_key` VARCHAR(191) NOT NULL,
  `description` VARCHAR(500) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_credit_ledger_entry_id` (`entry_id`),
  UNIQUE KEY `uk_ai_credit_ledger_owner_idempotency` (`owner_type`, `owner_id`, `idempotency_key`),
  KEY `idx_ai_credit_ledger_owner_created` (`owner_type`, `owner_id`, `created_at`),
  CONSTRAINT `fk_ai_credit_ledger_grant` FOREIGN KEY (`grant_id`) REFERENCES `ai_credit_grants` (`id`),
  CONSTRAINT `fk_ai_credit_ledger_reservation` FOREIGN KEY (`reservation_id`) REFERENCES `ai_credit_reservations` (`id`),
  CONSTRAINT `fk_ai_credit_ledger_usage` FOREIGN KEY (`usage_event_id`) REFERENCES `ai_usage_events` (`id`),
  CONSTRAINT `chk_ai_credit_ledger_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_credit_ledger_type` CHECK (`entry_type` IN ('grant', 'reserve', 'release', 'consume', 'expire', 'refund', 'adjustment'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Append-only AI credit accounting ledger';

CREATE TABLE IF NOT EXISTS `ai_billing_settlement_outbox` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `reservation_no` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `capability` VARCHAR(96) NOT NULL,
  `operation` VARCHAR(64) NOT NULL,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'reserved',
  `request_payload` JSON DEFAULT NULL,
  `idempotency_key` VARCHAR(191) DEFAULT NULL,
  `retry_count` INT UNSIGNED NOT NULL DEFAULT 0,
  `next_attempt_at` DATETIME(3) DEFAULT NULL,
  `locked_at` DATETIME(3) DEFAULT NULL,
  `last_error` VARCHAR(500) DEFAULT NULL,
  `completed_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_billing_settlement_reservation` (`reservation_no`),
  KEY `idx_ai_billing_settlement_due` (`status`, `next_attempt_at`),
  KEY `idx_ai_billing_settlement_owner` (`owner_type`, `owner_id`, `created_at`),
  CONSTRAINT `fk_ai_billing_settlement_reservation` FOREIGN KEY (`reservation_no`) REFERENCES `ai_credit_reservations` (`reservation_no`),
  CONSTRAINT `chk_ai_billing_settlement_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_billing_settlement_status` CHECK (`status` IN ('reserved', 'pending_settle', 'processing_settle', 'pending_cancel', 'processing_cancel', 'settled', 'cancelled', 'dead'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI Agent durable Billing settlement delivery outbox';

-- Existing tenants stay grandfathered on V1 until explicitly migrated. V2 adds
-- observable AI entitlements and can be assigned during the paid pilot.
INSERT INTO `platform_plan_versions` (`plan_id`, `version`, `status`, `effective_at`, `change_note`)
SELECT `id`, 2, 'published', NOW(3), 'AI billing shadow defaults; review before enforcement'
FROM `platform_plans`
WHERE `plan_key` IN ('starter', 'growth', 'enterprise')
ON DUPLICATE KEY UPDATE `plan_id` = VALUES(`plan_id`);

INSERT INTO `platform_plan_entitlements` (`plan_version_id`, `entitlement_key`, `value_type`, `value_json`, `enforcement_mode`)
SELECT v2.id, v1e.entitlement_key, v1e.value_type, v1e.value_json, v1e.enforcement_mode
FROM `platform_plan_versions` v2
JOIN `platform_plan_versions` v1 ON v1.plan_id = v2.plan_id AND v1.version = 1
JOIN `platform_plan_entitlements` v1e ON v1e.plan_version_id = v1.id
WHERE v2.version = 2
ON DUPLICATE KEY UPDATE `plan_version_id` = VALUES(`plan_version_id`);

INSERT INTO `platform_plan_entitlements` (`plan_version_id`, `entitlement_key`, `value_type`, `value_json`, `enforcement_mode`)
SELECT version.id, entitlement.entitlement_key, entitlement.value_type, entitlement.value_json, 'observe'
FROM `platform_plan_versions` version
JOIN `platform_plans` plan ON plan.id = version.plan_id AND version.version = 2
JOIN (
  SELECT 'starter' plan_key, 'ai.hr.enabled' entitlement_key, 'boolean' value_type, CAST('true' AS JSON) value_json UNION ALL
  SELECT 'starter', 'ai.chat.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.resume_parse.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.match_evaluation.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.application_analysis.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.agent_run.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.credits.monthly', 'integer', CAST(100 AS JSON) UNION ALL
  SELECT 'starter', 'ai.concurrent_runs.max', 'integer', CAST(1 AS JSON) UNION ALL
  SELECT 'starter', 'ai.single_run.max_credits', 'integer', CAST(20 AS JSON) UNION ALL
  SELECT 'growth', 'ai.hr.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.chat.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.resume_parse.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.match_evaluation.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.application_analysis.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.agent_run.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.credits.monthly', 'integer', CAST(1000 AS JSON) UNION ALL
  SELECT 'growth', 'ai.concurrent_runs.max', 'integer', CAST(3 AS JSON) UNION ALL
  SELECT 'growth', 'ai.single_run.max_credits', 'integer', CAST(100 AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.hr.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.chat.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.resume_parse.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.match_evaluation.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.application_analysis.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.agent_run.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.credits.monthly', 'integer', CAST(10000 AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.concurrent_runs.max', 'integer', CAST(10 AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.single_run.max_credits', 'integer', CAST(500 AS JSON)
) entitlement ON entitlement.plan_key = plan.plan_key
ON DUPLICATE KEY UPDATE `plan_version_id` = VALUES(`plan_version_id`);

INSERT INTO `billing_products` (`product_key`, `name`, `description`, `owner_type`, `product_type`, `status`) VALUES
  ('tenant_starter', '企业基础版', '包含基础 AI 权益的企业套餐', 'tenant', 'subscription', 'draft'),
  ('tenant_growth', '企业成长版', '包含成长 AI 权益的企业套餐', 'tenant', 'subscription', 'draft'),
  ('tenant_enterprise', '企业版', '包含企业级 AI 权益的企业套餐', 'tenant', 'subscription', 'draft'),
  ('candidate_free', '求职者免费版', '每月提供少量持续可用的 AI 额度', 'user', 'subscription', 'active'),
  ('candidate_pro', '求职者 Pro', '面向高频求职场景的 AI 套餐', 'user', 'subscription', 'draft'),
  ('tenant_credit_pack', '企业 AI 加量包', '购买后十二个月内有效', 'tenant', 'credit_pack', 'draft'),
  ('candidate_credit_pack', '求职者 AI 加量包', '购买后十二个月内有效', 'user', 'credit_pack', 'draft')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

INSERT INTO `billing_price_versions` (`product_id`, `version`, `billing_term`, `amount_fen`, `currency`, `included_credits`, `entitlement_snapshot`, `status`, `effective_at`)
SELECT `id`, 1, 'monthly', 0, 'CNY', 100, JSON_OBJECT('ai.chat.enabled', true, 'ai.credits.monthly', 100), 'published', NOW(3)
FROM `billing_products`
WHERE `product_key` = 'candidate_free'
ON DUPLICATE KEY UPDATE `product_id` = VALUES(`product_id`);
INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`, `created_at`, `updated_at`)
VALUES ('billing.manage', 'billing', 'manage', '管理当前租户的 AI 套餐、订单、支付与退款', NOW(), NOW())
ON DUPLICATE KEY UPDATE `resource` = VALUES(`resource`), `action` = VALUES(`action`), `description` = VALUES(`description`), `updated_at` = NOW();

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key = 'billing.manage'
WHERE role.role_key IN ('recruiting_admin', 'system_admin');

-- Migration 000070: platform AI control plane baseline and immutable releases.
INSERT INTO `platform_ai_capabilities` (`capability_key`, `audience`, `name`, `description`, `status`) VALUES
  ('ai.chat', 'tenant_hr', '企业招聘 AI 助手', '企业招聘工作台中的对话与工具调用能力', 'active'),
  ('ai.chat', 'candidate', '候选人 AI 助手', '候选人门户中的对话辅助能力', 'active'),
  ('ai.agent_run', 'tenant_hr', '企业 Agent Run', '企业招聘 Agent 的异步执行能力', 'active'),
  ('ai.application_analysis', 'tenant_hr', '申请分析', '对职位申请进行结构化 AI 分析', 'active'),
  ('ai.resume_parse', 'tenant_hr', '简历解析', '将候选人简历解析为结构化资料', 'active'),
  ('ai.match_evaluation', 'tenant_hr', '人岗匹配评估', '基于职位与候选人资料进行匹配评估', 'active')
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`),
  `status` = 'active';

INSERT INTO `platform_ai_capability_versions`
  (`capability_id`, `version`, `status`, `snapshot_json`, `snapshot_hash`, `change_note`, `published_at`)
SELECT
  capability.id,
  1,
  'published',
  snapshot.snapshot_json,
  SHA2(CAST(snapshot.snapshot_json AS CHAR), 256),
  '默认企业配置提升为平台全局基线',
  NOW(3)
FROM `platform_ai_capabilities` capability
JOIN LATERAL (
  SELECT JSON_OBJECT(
    'schema_version', 1,
    'capability_key', capability.capability_key,
    'audience', capability.audience,
    'model_policy', JSON_OBJECT(
      'allowed_llm_model_ids', COALESCE((SELECT JSON_ARRAYAGG(model.id) FROM llm_models model WHERE model.is_enabled = 1), JSON_ARRAY()),
      'default_llm_model_id', (SELECT model.id FROM llm_models model WHERE model.is_enabled = 1 AND model.is_default = 1 ORDER BY model.id DESC LIMIT 1),
      'allowed_embedding_model_ids', CASE WHEN capability.capability_key = 'ai.match_evaluation' THEN COALESCE((SELECT JSON_ARRAYAGG(model.id) FROM embedding_models model WHERE model.is_enabled = 1), JSON_ARRAY()) ELSE JSON_ARRAY() END,
      'default_embedding_model_id', CASE WHEN capability.capability_key = 'ai.match_evaluation' THEN (SELECT model.id FROM embedding_models model WHERE model.is_enabled = 1 AND model.is_default = 1 ORDER BY model.id DESC LIMIT 1) ELSE NULL END
    ),
    'configuration_refs', JSON_OBJECT(
      'agent_ids', COALESCE((SELECT JSON_ARRAYAGG(agent.id) FROM agent_configs agent WHERE agent.is_enabled = 1 AND agent.agent_type = CASE WHEN capability.audience = 'candidate' THEN 'candidate_assistant' ELSE 'hr_recruiting_agent' END), JSON_ARRAY()),
      'prompt_template_ids', COALESCE((SELECT JSON_ARRAYAGG(prompt.id) FROM prompt_templates prompt WHERE prompt.is_active = 1 AND (
        (capability.capability_key IN ('ai.chat', 'ai.agent_run', 'ai.application_analysis') AND prompt.id IN (
          SELECT agent.prompt_template_id
          FROM agent_configs agent
          WHERE agent.is_enabled = 1
            AND agent.prompt_template_id IS NOT NULL
            AND agent.agent_type = CASE WHEN capability.audience = 'candidate' THEN 'candidate_assistant' ELSE 'hr_recruiting_agent' END
        ))
        OR (capability.capability_key = 'ai.resume_parse' AND prompt.agent_type = 'resume_profile_extractor')
        OR (capability.capability_key = 'ai.match_evaluation' AND prompt.agent_type IN ('job_requirement_extractor', 'candidate_match_evaluator'))
      )), JSON_ARRAY()),
      'agent_skill_version_ids', CASE WHEN capability.audience = 'tenant_hr' AND capability.capability_key IN ('ai.chat', 'ai.agent_run') THEN COALESCE((SELECT JSON_ARRAYAGG(skill.current_version_id) FROM agent_skills skill WHERE skill.is_enabled = 1 AND skill.current_version_id IS NOT NULL), JSON_ARRAY()) ELSE JSON_ARRAY() END,
      'mcp_policy_ids', CASE WHEN capability.audience = 'tenant_hr' AND capability.capability_key IN ('ai.chat', 'ai.agent_run') THEN COALESCE((SELECT JSON_ARRAYAGG(policy.id) FROM mcp_tool_policies policy WHERE policy.is_enabled = 1), JSON_ARRAY()) ELSE JSON_ARRAY() END
    )
  ) AS snapshot_json
) snapshot ON TRUE
ON DUPLICATE KEY UPDATE `capability_id` = VALUES(`capability_id`);

UPDATE `platform_ai_capabilities` capability
JOIN `platform_ai_capability_versions` version
  ON version.capability_id = capability.id AND version.version = 1 AND version.status = 'published'
SET capability.current_published_version_id = version.id;

INSERT INTO `platform_plan_entitlements`
  (`plan_version_id`, `entitlement_key`, `value_type`, `value_json`, `enforcement_mode`)
SELECT
  plan_version.id,
  CONCAT(capability.capability_key, '.release_version_id'),
  'integer',
  CAST(capability_version.id AS JSON),
  'hard'
FROM `platform_plan_versions` plan_version
JOIN `platform_plan_entitlements` enabled ON enabled.plan_version_id = plan_version.id
JOIN `platform_ai_capabilities` capability
  ON capability.audience = 'tenant_hr'
 AND enabled.entitlement_key = CONCAT(capability.capability_key, '.enabled')
JOIN `platform_ai_capability_versions` capability_version
  ON capability_version.id = capability.current_published_version_id
WHERE plan_version.version = 2
ON DUPLICATE KEY UPDATE
  `value_type` = VALUES(`value_type`),
  `value_json` = VALUES(`value_json`),
  `enforcement_mode` = VALUES(`enforcement_mode`);

UPDATE `billing_price_versions` price
JOIN `billing_products` product ON product.id = price.product_id AND product.owner_type = 'user'
JOIN `platform_ai_capabilities` capability ON capability.capability_key = 'ai.chat' AND capability.audience = 'candidate'
SET price.entitlement_snapshot = JSON_SET(
  COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
  '$."ai.chat.release_version_id"',
  capability.current_published_version_id
)
WHERE product.product_type = 'subscription';

-- Migration 000072: give legacy active tenants without any subscription
-- history a published Starter v2 plan, which pins their AI capability releases.
INSERT INTO `tenant_subscriptions`
  (`tenant_id`, `plan_version_id`, `status`, `starts_at`, `ends_at`, `reason`, `created_by`)
SELECT
  tenant.id,
  version.id,
  'active',
  NOW(3),
  NULL,
  'legacy tenant starter plan bootstrap',
  NULL
FROM `tenants` tenant
JOIN `platform_plans` plan
  ON plan.plan_key = 'starter'
 AND plan.status = 'active'
JOIN `platform_plan_versions` version
  ON version.plan_id = plan.id
 AND version.version = 2
 AND version.status = 'published'
 AND version.effective_at <= NOW(3)
WHERE tenant.status = 'active'
  AND NOT EXISTS (
    SELECT 1
    FROM `tenant_subscriptions` existing
    WHERE existing.tenant_id = tenant.id
  );

-- Migration 000073 final state: immediate entitlement seed data uses the
-- platform's Asia/Shanghai database session clock.
UPDATE `platform_plan_versions`
SET `effective_at` = NOW(3)
WHERE `version` = 2
  AND `change_note` = 'AI billing shadow defaults; review before enforcement'
  AND `effective_at` > NOW(3);

UPDATE `billing_price_versions` price
JOIN `billing_products` product
  ON product.id = price.product_id
 AND product.product_key = 'candidate_free'
SET price.effective_at = NOW(3)
WHERE price.version = 1
  AND price.status = 'published'
  AND price.effective_at > NOW(3);

UPDATE `tenant_subscriptions`
SET `starts_at` = NOW(3)
WHERE `reason` = 'legacy tenant starter plan bootstrap'
  AND `created_by` IS NULL
  AND `starts_at` > NOW(3);

-- Migration 000074: publish the candidate Pro offer for Alipay sandbox checkout.
INSERT IGNORE INTO `billing_price_versions`
  (`product_id`, `version`, `billing_term`, `amount_fen`, `currency`, `included_credits`,
   `entitlement_snapshot`, `status`, `effective_at`, `created_at`, `updated_at`)
SELECT
  product.id,
  1,
  'monthly',
  990,
  'CNY',
  200,
  JSON_OBJECT(
    'ai.chat.enabled', true,
    'ai.chat.release_version_id', capability.current_published_version_id,
    'ai.credits.monthly', 200
  ),
  'draft',
  NULL,
  NOW(3),
  NOW(3)
FROM `billing_products` product
JOIN `platform_ai_capabilities` capability
  ON capability.capability_key = 'ai.chat'
 AND capability.audience = 'candidate'
 AND capability.current_published_version_id IS NOT NULL
WHERE product.product_key = 'candidate_pro'
  AND product.owner_type = 'user'
  AND product.product_type = 'subscription';

UPDATE `billing_price_versions` price
JOIN `billing_products` product
  ON product.id = price.product_id
 AND product.product_key = 'candidate_pro'
 AND product.owner_type = 'user'
JOIN `platform_ai_capabilities` capability
  ON capability.capability_key = 'ai.chat'
 AND capability.audience = 'candidate'
 AND capability.current_published_version_id IS NOT NULL
SET price.entitlement_snapshot = JSON_SET(
      COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
      '$."ai.chat.enabled"',
      true,
      '$."ai.chat.release_version_id"',
      capability.current_published_version_id,
      '$."ai.credits.monthly"',
      price.included_credits
    ),
    price.effective_at = NOW(3),
    price.status = 'published',
    price.updated_at = NOW(3)
WHERE price.version = 1
  AND price.status = 'draft';

UPDATE `billing_products`
SET `status` = 'active',
    `updated_at` = NOW(3)
WHERE `product_key` = 'candidate_pro'
  AND `owner_type` = 'user'
  AND `product_type` = 'subscription'
  AND `status` = 'draft';

-- Migration 000080: keep candidate subscriptions bound to candidate releases.
UPDATE `billing_price_versions` price
JOIN `billing_products` product
  ON product.id = price.product_id
 AND product.owner_type = 'user'
 AND product.product_type = 'subscription'
JOIN `platform_ai_capabilities` capability
  ON capability.capability_key = 'ai.chat'
 AND capability.audience = 'candidate'
 AND capability.status = 'active'
 AND capability.current_published_version_id IS NOT NULL
SET price.entitlement_snapshot = JSON_SET(
      COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
      '$."ai.chat.release_version_id"',
      capability.current_published_version_id
    ),
    price.updated_at = NOW(3)
WHERE JSON_CONTAINS_PATH(
  COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
  'one',
  '$."ai.chat.enabled"'
);

-- Migration 000081 final state under the UTC+8 connection contract.
UPDATE `billing_price_versions`
SET `effective_at` = NOW(3),
    `updated_at` = NOW(3)
WHERE `status` = 'published'
  AND `effective_at` > NOW(3)
  AND ABS(TIMESTAMPDIFF(SECOND, `created_at`, `effective_at`)) <= 5;

UPDATE `ai_rate_cards`
SET `effective_at` = NOW(3)
WHERE `status` = 'published'
  AND `effective_at` > NOW(3)
  AND ABS(TIMESTAMPDIFF(SECOND, `created_at`, `effective_at`)) <= 5;

-- Migration 000082: retain cell-level evidence for production conversions.
-- A cold-start database is already UTC+8-native and therefore has no rows.
CREATE TABLE IF NOT EXISTS `utc8_time_conversion_audit` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `batch_id` VARCHAR(96) NOT NULL,
  `table_name` VARCHAR(96) NOT NULL,
  `row_pk` VARCHAR(191) NOT NULL,
  `column_name` VARCHAR(96) NOT NULL,
  `old_value` DATETIME(6) NOT NULL,
  `new_value` DATETIME(6) NOT NULL,
  `reason` VARCHAR(500) NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_utc8_time_conversion_cell` (`batch_id`, `table_name`, `row_pk`, `column_name`),
  KEY `idx_utc8_time_conversion_lookup` (`table_name`, `row_pk`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Auditable UTC wall-clock to Asia/Shanghai conversions';
