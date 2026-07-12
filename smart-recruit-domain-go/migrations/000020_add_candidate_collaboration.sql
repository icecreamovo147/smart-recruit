-- 000020_add_candidate_collaboration.sql
-- Adds collaboration tables and permissions for Phase 4: Candidate Collaboration.

-- ── Candidate Notes table ─────────────────────────────────────────────────

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

-- ── Candidate Tags table ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS `candidate_tags` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '标签ID',
  `name` VARCHAR(64) NOT NULL COMMENT '标签名称',
  `color` VARCHAR(16) DEFAULT '#409eff' COMMENT '标签颜色',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建人用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tag_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人标签定义表';

-- ── Candidate Tag Assignments table ───────────────────────────────────────

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

-- ── Follow-up Tasks table ─────────────────────────────────────────────────

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

-- ── Collaboration permissions (idempotent seed) ───────────────────────────

INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`) VALUES
  ('collaboration.note.read',   'collaboration', 'read',   '查看候选人内部备注'),
  ('collaboration.note.create', 'collaboration', 'create', '创建候选人内部备注'),
  ('collaboration.tag.manage',  'collaboration', 'manage', '管理候选人标签'),
  ('collaboration.task.manage', 'collaboration', 'manage', '管理跟进任务')
ON DUPLICATE KEY UPDATE `resource` = VALUES(`resource`), `action` = VALUES(`action`), `description` = VALUES(`description`);

-- ── Map collaboration permissions to roles ────────────────────────────────

-- Recruiter: full collaboration access
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'recruiter' AND p.permission_key IN (
    'collaboration.note.read', 'collaboration.note.create',
    'collaboration.tag.manage', 'collaboration.task.manage'
  );

-- Recruiting Admin: full collaboration access
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'recruiting_admin' AND p.permission_key IN (
    'collaboration.note.read', 'collaboration.note.create',
    'collaboration.tag.manage', 'collaboration.task.manage'
  );
