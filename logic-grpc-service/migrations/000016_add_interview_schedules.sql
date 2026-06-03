-- Migration: 000016_add_interview_schedules
-- 用于面试官 (interviewer) 角色的 assigned_interviews 数据范围匹配。
-- 仓库代码（authz_repo.go IsInterviewerForJob / GetScopedInterviewJobIDs）
-- 依赖该表，否则面试官范围将无法生效。

CREATE TABLE IF NOT EXISTS `interview_schedules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `interviewer_id` BIGINT UNSIGNED NOT NULL COMMENT '面试官用户ID（users.id）',
  `round_no` INT NOT NULL DEFAULT 1 COMMENT '面试轮次：1=初试 2=复试 ...',
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='面试安排表（用于面试官数据范围匹配）';
