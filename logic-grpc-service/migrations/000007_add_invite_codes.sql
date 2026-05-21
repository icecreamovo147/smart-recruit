-- 000007_add_invite_codes.sql
-- Add invite_codes table for HR admin management and update users.role comment.

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

ALTER TABLE `users`
  MODIFY COLUMN `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色：1=候选人 2=HR 3=HR管理员';
