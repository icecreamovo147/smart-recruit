-- 000036_add_ai_business_permissions.sql
-- Adds AI business management permissions for Prompt, Agent, and Agent Skill.
-- Grants them to recruiting_admin and system_admin only.

-- ── Insert new permissions (idempotent) ─────────────────────────────────

INSERT INTO permissions (permission_key, resource, action, description, created_at, updated_at) VALUES
('ai.prompt.manage', 'ai', 'manage', '管理招聘 Prompt 模板', NOW(), NOW()),
('ai.agent.manage', 'ai', 'manage', '管理招聘 Agent 配置', NOW(), NOW()),
('ai.agent_skill.manage', 'ai', 'manage', '管理招聘 Agent Skill', NOW(), NOW())
ON DUPLICATE KEY UPDATE
  resource = VALUES(resource),
  action = VALUES(action),
  description = VALUES(description),
  updated_at = VALUES(updated_at);

-- ── Map to roles ────────────────────────────────────────────────────────

INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW()
FROM roles r, permissions p
WHERE r.role_key IN ('recruiting_admin', 'system_admin')
  AND p.permission_key IN ('ai.prompt.manage', 'ai.agent.manage', 'ai.agent_skill.manage');
