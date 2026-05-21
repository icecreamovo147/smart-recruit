-- 000001_init_schema.sql
-- Baseline schema snapshot. Do not modify after creation.
-- MySQL DDL performs implicit commits, so this baseline is intentionally
-- written as an ordered sequence rather than a transactional script.

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username` VARCHAR(64) NOT NULL COMMENT '用户名（唯一）',
  `password` VARCHAR(255) NOT NULL COMMENT 'bcrypt 哈希密码',
  `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色：1=候选人 2=HR',
  `email` VARCHAR(128) DEFAULT NULL COMMENT '邮箱（可选）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账号表';

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
  KEY `idx_status` (`status`)
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
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_is_valid` (`is_valid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='简历 OSS 存储记录表';

CREATE TABLE IF NOT EXISTS `applications` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id` BIGINT UNSIGNED NOT NULL COMMENT '投递的岗位ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '投递的候选人用户ID',
  `resume_id` BIGINT UNSIGNED NOT NULL COMMENT '投递时使用的简历ID',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '投递状态：0=待查看 1=已查看 2=通过 3=淘汰',
  `round_no` INT NOT NULL DEFAULT 1 COMMENT '同一候选人同一岗位的第几次投递',
  `is_current` TINYINT NOT NULL DEFAULT 1 COMMENT '是否当前有效投递：1=当前流程 0=历史流程',
  `applied_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_job_user_current_status` (`job_id`, `user_id`, `is_current`, `status`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位投递关联表';

CREATE TABLE IF NOT EXISTS `ai_chat_sessions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `title` VARCHAR(255) NOT NULL COMMENT '会话标题',
  `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '绑定的投递记录ID，0表示普通数据问答',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_hr_updated_at` (`hr_id`, `updated_at`),
  KEY `idx_hr_deleted_updated` (`hr_id`, `deleted_at`, `updated_at`),
  KEY `idx_application_id` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI 会话表';

CREATE TABLE IF NOT EXISTS `ai_chat_history` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'AI 会话ID',
  `hr_id` BIGINT UNSIGNED NOT NULL COMMENT 'HR 用户ID',
  `role` VARCHAR(16) NOT NULL COMMENT '消息角色：user / assistant',
  `content` TEXT NOT NULL COMMENT '消息内容',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_session_id` (`session_id`),
  KEY `idx_hr_id_created` (`hr_id`, `created_at`)
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
  `receiver_id` BIGINT UNSIGNED NOT NULL COMMENT '接收用户ID',
  `receiver_role` TINYINT NOT NULL COMMENT '接收者角色：1=候选人 2=HR',
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
  KEY `idx_receiver_read_created` (`receiver_id`, `receiver_role`, `is_read`, `created_at`),
  KEY `idx_receiver_created` (`receiver_id`, `receiver_role`, `created_at`),
  KEY `idx_receiver_biz_type` (`receiver_id`, `receiver_role`, `biz_type`, `biz_id`, `type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='站内通知表';
