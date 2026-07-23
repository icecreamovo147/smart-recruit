-- Promote technical AI configuration to the platform control plane and create
-- immutable capability releases. This migration intentionally fails when AI
-- configuration owned by a non-default tenant exists: the product has not
-- launched yet, so only the seeded default tenant is a supported baseline.

SET @default_tenant_id := (SELECT id FROM tenants WHERE is_default = 1 LIMIT 1);

CREATE TEMPORARY TABLE `_platform_ai_config_scope_guard` (
  `ok` TINYINT NOT NULL,
  CONSTRAINT `chk_platform_ai_config_scope_guard` CHECK (`ok` = 1)
);

INSERT INTO `_platform_ai_config_scope_guard` (`ok`)
SELECT IF(
  @default_tenant_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM (
      SELECT tenant_id FROM llm_providers WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM llm_models WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM embedding_providers WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM embedding_models WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM prompt_templates WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM prompt_versions WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM agent_configs WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM agent_tool_bindings WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM mcp_servers WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM mcp_tool_policies WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM agent_capability_bindings WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM ai_skills WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM ai_skill_versions WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM ai_skill_tools WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM agent_skills WHERE tenant_id IS NOT NULL
      UNION ALL SELECT tenant_id FROM agent_skill_versions WHERE tenant_id IS NOT NULL
    ) scoped
    WHERE scoped.tenant_id <> @default_tenant_id
  ),
  1,
  0
);

DROP TEMPORARY TABLE `_platform_ai_config_scope_guard`;

ALTER TABLE `agent_runs`
  ADD COLUMN `requested_model_id` BIGINT NULL AFTER `model_name`,
  ADD COLUMN `effective_model_id` BIGINT NULL AFTER `requested_model_id`,
  ADD COLUMN `model_fallback_reason` VARCHAR(64) NULL AFTER `effective_model_id`,
  ADD COLUMN `capability_version_id` BIGINT UNSIGNED NULL AFTER `model_fallback_reason`,
  ADD COLUMN `capability_snapshot_hash` CHAR(64) NULL AFTER `capability_version_id`,
  ADD KEY `idx_agent_runs_capability_version` (`capability_version_id`);

UPDATE llm_providers SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE llm_models SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE embedding_providers SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE embedding_models SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE prompt_templates SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE prompt_versions SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE agent_configs SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE agent_tool_bindings SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE mcp_servers SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE mcp_tool_policies SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE agent_capability_bindings SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE ai_skills SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE ai_skill_versions SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE ai_skill_tools SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE agent_skills SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;
UPDATE agent_skill_versions SET tenant_id = NULL WHERE tenant_id = @default_tenant_id;

ALTER TABLE agent_skill_versions DROP FOREIGN KEY fk_agent_skill_versions_tenant, DROP INDEX idx_agent_skill_versions_tenant_skill, DROP COLUMN tenant_id;
ALTER TABLE agent_skills DROP FOREIGN KEY fk_agent_skills_tenant, DROP INDEX idx_agent_skills_tenant_enabled, DROP COLUMN tenant_id;
ALTER TABLE ai_skill_tools DROP FOREIGN KEY fk_ai_skill_tools_tenant, DROP INDEX idx_ai_skill_tools_tenant_version, DROP COLUMN tenant_id;
ALTER TABLE ai_skill_versions DROP FOREIGN KEY fk_ai_skill_versions_tenant, DROP INDEX idx_ai_skill_versions_tenant_skill, DROP COLUMN tenant_id;
ALTER TABLE ai_skills DROP FOREIGN KEY fk_ai_skills_tenant, DROP INDEX idx_ai_skills_tenant_enabled, DROP COLUMN tenant_id;
ALTER TABLE agent_capability_bindings DROP FOREIGN KEY fk_agent_capabilities_tenant, DROP INDEX idx_agent_capabilities_tenant_agent, DROP COLUMN tenant_id;
ALTER TABLE mcp_tool_policies DROP FOREIGN KEY fk_mcp_policies_tenant, DROP INDEX idx_mcp_policies_tenant_server, DROP COLUMN tenant_id;
ALTER TABLE mcp_servers DROP FOREIGN KEY fk_mcp_servers_tenant, DROP INDEX idx_mcp_servers_tenant_enabled, DROP COLUMN tenant_id;
ALTER TABLE agent_tool_bindings DROP FOREIGN KEY fk_agent_tool_bindings_tenant, DROP INDEX idx_agent_tool_bindings_tenant_agent, DROP COLUMN tenant_id;
ALTER TABLE agent_configs DROP FOREIGN KEY fk_agent_configs_tenant, DROP INDEX idx_agent_configs_tenant_type, DROP COLUMN tenant_id;
ALTER TABLE prompt_versions DROP FOREIGN KEY fk_prompt_versions_tenant, DROP INDEX idx_prompt_versions_tenant_template, DROP COLUMN tenant_id;
ALTER TABLE prompt_templates DROP FOREIGN KEY fk_prompt_templates_tenant, DROP INDEX idx_prompt_templates_tenant_agent, DROP COLUMN tenant_id;
ALTER TABLE embedding_models DROP FOREIGN KEY fk_embedding_models_tenant, DROP INDEX idx_embedding_models_tenant_provider, DROP COLUMN tenant_id;
ALTER TABLE embedding_providers DROP FOREIGN KEY fk_embedding_providers_tenant, DROP INDEX idx_embedding_providers_tenant_enabled, DROP COLUMN tenant_id;
ALTER TABLE llm_models DROP FOREIGN KEY fk_llm_models_tenant, DROP INDEX idx_llm_models_tenant_provider, DROP COLUMN tenant_id;
ALTER TABLE llm_providers DROP FOREIGN KEY fk_llm_providers_tenant, DROP INDEX idx_llm_providers_tenant_enabled, DROP COLUMN tenant_id;

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
  `snapshot_hash` CHAR(64) DEFAULT NULL,
  `change_note` VARCHAR(500) DEFAULT NULL,
  `created_by` BIGINT UNSIGNED DEFAULT NULL,
  `published_by` BIGINT UNSIGNED DEFAULT NULL,
  `published_at` DATETIME(3) DEFAULT NULL,
  `retired_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_ai_capability_versions_number` (`capability_id`, `version`),
  UNIQUE KEY `uk_platform_ai_capability_versions_hash` (`capability_id`, `snapshot_hash`),
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

INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`, `created_at`, `updated_at`) VALUES
  ('platform.ai.config.read', 'platform_ai_config', 'read', '查看平台 AI 技术配置', NOW(), NOW()),
  ('platform.ai.config.manage', 'platform_ai_config', 'manage', '维护平台 AI 技术配置', NOW(), NOW()),
  ('platform.ai.release.read', 'platform_ai_release', 'read', '查看平台 AI 能力版本', NOW(), NOW()),
  ('platform.ai.release.manage', 'platform_ai_release', 'manage', '维护平台 AI 能力草稿', NOW(), NOW()),
  ('platform.ai.release.publish', 'platform_ai_release', 'publish', '发布或退役平台 AI 能力版本', NOW(), NOW()),
  ('platform.ai.diagnostics.read', 'platform_ai_diagnostics', 'read', '查看平台 AI 诊断信息', NOW(), NOW()),
  ('platform.ai.diagnostics.execute', 'platform_ai_diagnostics', 'execute', '执行平台 AI 诊断', NOW(), NOW())
ON DUPLICATE KEY UPDATE
  `resource` = VALUES(`resource`),
  `action` = VALUES(`action`),
  `description` = VALUES(`description`),
  `updated_at` = NOW();

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key LIKE 'platform.ai.%'
WHERE role.role_key = 'platform_admin';

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key IN (
  'platform.ai.config.read', 'platform.ai.release.read',
  'platform.ai.diagnostics.read', 'platform.ai.diagnostics.execute'
)
WHERE role.role_key = 'platform_operator';

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key IN (
  'platform.ai.config.read', 'platform.ai.release.read', 'platform.ai.diagnostics.read'
)
WHERE role.role_key = 'platform_auditor';

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
  (`capability_id`, `version`, `status`, `snapshot_json`, `change_note`, `published_at`)
SELECT
  capability.id,
  1,
  'published',
  JSON_OBJECT(
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
        (capability.capability_key IN ('ai.chat', 'ai.agent_run') AND prompt.agent_type = CASE WHEN capability.audience = 'candidate' THEN 'candidate_assistant' ELSE 'hr_recruiting_agent' END)
        OR (capability.capability_key = 'ai.resume_parse' AND prompt.agent_type = 'resume_profile_extractor')
        OR (capability.capability_key = 'ai.match_evaluation' AND prompt.agent_type IN ('job_requirement_extractor', 'candidate_match_evaluator'))
      )), JSON_ARRAY()),
      'agent_skill_version_ids', CASE WHEN capability.audience = 'tenant_hr' AND capability.capability_key IN ('ai.chat', 'ai.agent_run') THEN COALESCE((SELECT JSON_ARRAYAGG(skill.current_version_id) FROM agent_skills skill WHERE skill.is_enabled = 1 AND skill.current_version_id IS NOT NULL), JSON_ARRAY()) ELSE JSON_ARRAY() END,
      'ai_skill_version_ids', CASE WHEN capability.audience = 'tenant_hr' AND capability.capability_key IN ('ai.chat', 'ai.agent_run') THEN COALESCE((SELECT JSON_ARRAYAGG(skill.current_version_id) FROM ai_skills skill WHERE skill.is_enabled = 1 AND skill.current_version_id IS NOT NULL), JSON_ARRAY()) ELSE JSON_ARRAY() END,
      'mcp_policy_ids', CASE WHEN capability.audience = 'tenant_hr' AND capability.capability_key IN ('ai.chat', 'ai.agent_run') THEN COALESCE((SELECT JSON_ARRAYAGG(policy.id) FROM mcp_tool_policies policy WHERE policy.is_enabled = 1), JSON_ARRAY()) ELSE JSON_ARRAY() END
    )
  ),
  '默认企业配置提升为平台全局基线',
  NOW(3)
FROM `platform_ai_capabilities` capability
ON DUPLICATE KEY UPDATE `capability_id` = VALUES(`capability_id`);

UPDATE `platform_ai_capability_versions`
SET `snapshot_hash` = SHA2(CAST(`snapshot_json` AS CHAR), 256)
WHERE `snapshot_hash` IS NULL;

ALTER TABLE `platform_ai_capability_versions`
  MODIFY COLUMN `snapshot_hash` CHAR(64) NOT NULL;

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
JOIN `platform_plan_entitlements` enabled
  ON enabled.plan_version_id = plan_version.id
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
