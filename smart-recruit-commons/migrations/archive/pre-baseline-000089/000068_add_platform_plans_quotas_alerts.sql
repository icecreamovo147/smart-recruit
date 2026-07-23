-- Versioned plan catalogue, tenant subscriptions, entitlement overrides, usage snapshots and quota alerts.

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
SELECT `id`, 1, 'published', NOW(), 'initial catalogue'
FROM `platform_plans`
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
