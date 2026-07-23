ALTER TABLE `agent_runs`
  DROP KEY `idx_agent_runs_capability_version`,
  DROP COLUMN `capability_snapshot_hash`,
  DROP COLUMN `capability_version_id`,
  DROP COLUMN `model_fallback_reason`,
  DROP COLUMN `effective_model_id`,
  DROP COLUMN `requested_model_id`;

UPDATE `billing_price_versions` price
JOIN `billing_products` product ON product.id = price.product_id AND product.owner_type = 'user'
SET price.entitlement_snapshot = JSON_REMOVE(
  COALESCE(price.entitlement_snapshot, JSON_OBJECT()),
  '$."ai.chat.release_version_id"'
)
WHERE product.product_type = 'subscription';

DELETE FROM `platform_plan_entitlements`
WHERE `entitlement_key` IN (
  'ai.chat.release_version_id',
  'ai.resume_parse.release_version_id',
  'ai.match_evaluation.release_version_id',
  'ai.application_analysis.release_version_id',
  'ai.agent_run.release_version_id'
);

DELETE role_permission
FROM `role_permissions` role_permission
JOIN `permissions` permission ON permission.id = role_permission.permission_id
WHERE permission.permission_key LIKE 'platform.ai.%';

DELETE FROM `permissions` WHERE `permission_key` LIKE 'platform.ai.%';

ALTER TABLE `platform_ai_capabilities` DROP FOREIGN KEY `fk_platform_ai_capabilities_current_version`;
DROP TABLE IF EXISTS `platform_ai_config_audit_logs`;
DROP TABLE IF EXISTS `platform_ai_capability_versions`;
DROP TABLE IF EXISTS `platform_ai_capabilities`;

ALTER TABLE llm_providers ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_llm_providers_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_llm_providers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE llm_models ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_llm_models_tenant_provider (tenant_id, provider_id), ADD CONSTRAINT fk_llm_models_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE embedding_providers ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_embedding_providers_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_embedding_providers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE embedding_models ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_embedding_models_tenant_provider (tenant_id, provider_id), ADD CONSTRAINT fk_embedding_models_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE prompt_templates ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_prompt_templates_tenant_agent (tenant_id, agent_type, is_active), ADD CONSTRAINT fk_prompt_templates_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE prompt_versions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_prompt_versions_tenant_template (tenant_id, template_id, version), ADD CONSTRAINT fk_prompt_versions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_configs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_configs_tenant_type (tenant_id, agent_type, is_enabled), ADD CONSTRAINT fk_agent_configs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_tool_bindings ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_tool_bindings_tenant_agent (tenant_id, agent_id), ADD CONSTRAINT fk_agent_tool_bindings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE mcp_servers ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_mcp_servers_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_mcp_servers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE mcp_tool_policies ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_mcp_policies_tenant_server (tenant_id, server_id, tool_name), ADD CONSTRAINT fk_mcp_policies_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_capability_bindings ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_capabilities_tenant_agent (tenant_id, agent_id), ADD CONSTRAINT fk_agent_capabilities_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_skills ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_skills_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_ai_skills_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_skill_versions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_skill_versions_tenant_skill (tenant_id, skill_id), ADD CONSTRAINT fk_ai_skill_versions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_skill_tools ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_skill_tools_tenant_version (tenant_id, skill_version_id), ADD CONSTRAINT fk_ai_skill_tools_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_skills ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_skills_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_agent_skills_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_skill_versions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_skill_versions_tenant_skill (tenant_id, skill_id), ADD CONSTRAINT fk_agent_skill_versions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
