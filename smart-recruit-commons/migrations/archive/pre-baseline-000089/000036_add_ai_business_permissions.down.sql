-- 000036_add_ai_business_permissions.down.sql
-- Reverses the addition of AI business management permissions.

DELETE rp FROM role_permissions rp
  INNER JOIN permissions p ON p.id = rp.permission_id
WHERE p.permission_key IN ('ai.prompt.manage', 'ai.agent.manage', 'ai.agent_skill.manage');

DELETE FROM permissions
WHERE permission_key IN ('ai.prompt.manage', 'ai.agent.manage', 'ai.agent_skill.manage');
