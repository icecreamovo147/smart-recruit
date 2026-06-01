CREATE DATABASE IF NOT EXISTS `recruitment`
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `recruitment`;

SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username` VARCHAR(64) NOT NULL COMMENT '用户名（唯一）',
  `password` VARCHAR(255) NOT NULL COMMENT 'bcrypt 哈希密码',
  `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色：1=候选人 2=HR 3=HR管理员',
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
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '投递状态：0=待查看 1=已查看 2=通过 3=淘汰',
  `round_no` INT NOT NULL DEFAULT 1 COMMENT '同一候选人同一岗位的第几次投递',
  `is_current` TINYINT NOT NULL DEFAULT 1 COMMENT '是否当前有效投递：1=当前流程 0=历史流程',
  `applied_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `active_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_current` = 1 AND `status` <> 3 THEN 1 ELSE NULL END
  ) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_active_application` (`job_id`, `user_id`, `active_key`),
  KEY `idx_job_user_current_status` (`job_id`, `user_id`, `is_current`, `status`),
  KEY `idx_job_status_current` (`job_id`, `status`, `is_current`),
  KEY `idx_job_current_applied` (`job_id`, `is_current`, `applied_at`, `id`),
  KEY `idx_user_applied` (`user_id`, `applied_at`, `id`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位投递关联表';

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
  UNIQUE KEY `uk_notification_event_id` (`event_id`),
  UNIQUE KEY `uk_notification_once` (`receiver_id`, `receiver_role`, `biz_type`, `biz_id`, `type`),
  KEY `idx_receiver_read_created` (`receiver_id`, `receiver_role`, `is_read`, `created_at`),
  KEY `idx_receiver_read_created_id` (`receiver_id`, `receiver_role`, `is_read`, `created_at`, `id`),
  KEY `idx_receiver_created` (`receiver_id`, `receiver_role`, `created_at`)
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

-- ── 插入默认 HR 管理员账号 (admin / 123456，角色=3 表示 HR 管理员) ────

INSERT IGNORE INTO `users` (`username`, `password`, `role`) VALUES
  ('admin', '$2b$10$qxetp5jT6U7U5dd/k1G/v.qJ.FDlqFLWO3LHKv8Kwt6c49VXhLhOy', 3);

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
-- 每个默认部门至少配置两个可用地点；当前默认部门均为根部门，后续子部门配置需保持为父级地点子集。

INSERT INTO `department_locations` (`department_id`, `location_id`, `is_active`)
SELECT d.id, l.id, 1
FROM `departments` d
JOIN `job_locations` l ON (
  (d.name = '技术研发部' AND l.name IN ('北京', '上海', '深圳', '杭州')) OR
  (d.name = '产品部'     AND l.name IN ('北京', '上海', '杭州')) OR
  (d.name = '设计部'     AND l.name IN ('上海', '深圳', '杭州')) OR
  (d.name = '市场部'     AND l.name IN ('北京', '广州', '成都')) OR
  (d.name = '销售部'     AND l.name IN ('广州', '深圳', '成都', '武汉')) OR
  (d.name = '运营部'     AND l.name IN ('上海', '成都', '武汉')) OR
  (d.name = '人力资源部' AND l.name IN ('北京', '上海', '南京')) OR
  (d.name = '财务部'     AND l.name IN ('北京', '上海')) OR
  (d.name = '客户成功部' AND l.name IN ('深圳', '成都', '远程'))
)
WHERE d.parent_id = 0
  AND d.deleted_at IS NULL
  AND l.deleted_at IS NULL
ON DUPLICATE KEY UPDATE
  is_active = 1,
  deleted_at = NULL,
  deleted_by = NULL;
