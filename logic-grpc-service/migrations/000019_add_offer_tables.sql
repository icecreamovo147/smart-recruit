-- 000019_add_offer_tables.sql
-- Adds offer management tables for Phase 3: Offer And Onboarding.

-- ── Offers table ────────────────────────────────────────────────────────

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

-- ── Offer Events table ─────────────────────────────────────────────────

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

-- ── Offer permissions (idempotent seed) ─────────────────────────────────

INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`) VALUES
  ('offer.read',            'offer', 'read',   '查看Offer'),
  ('offer.manage',          'offer', 'manage', '创建/编辑/撤回Offer'),
  ('offer.send',            'offer', 'send',   '发送Offer（快照条款）'),
  ('offer.decision.manage', 'offer', 'manage', '候选人接受/拒绝Offer')
ON DUPLICATE KEY UPDATE `resource` = VALUES(`resource`), `action` = VALUES(`action`), `description` = VALUES(`description`);

-- ── Map offer permissions to roles ──────────────────────────────────────

-- Recruiter: offer.read, offer.manage, offer.send
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'recruiter' AND p.permission_key IN (
    'offer.read', 'offer.manage', 'offer.send'
  );

-- Candidate: offer.decision.manage
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`)
  SELECT r.id, p.id FROM `roles` r, `permissions` p
  WHERE r.role_key = 'candidate' AND p.permission_key IN (
    'offer.decision.manage'
  );
