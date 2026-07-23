-- Before adding the unique constraint, clean existing duplicate defaults.
-- For each agent_type, only the lowest id remains as default.
UPDATE agent_configs a
JOIN (
  SELECT agent_type, MIN(id) AS keep_id
  FROM agent_configs
  WHERE is_default = 1 AND is_enabled = 1
  GROUP BY agent_type
  HAVING COUNT(*) > 1
) x ON a.agent_type = x.agent_type
SET a.is_default = 0
WHERE a.is_default = 1
  AND a.is_enabled = 1
  AND a.id <> x.keep_id;

-- Generated column that is non-NULL only when is_default=1 AND is_enabled=1.
-- The unique index enforces at most one enabled default per agent_type.
ALTER TABLE agent_configs
  ADD COLUMN default_key VARCHAR(64)
  GENERATED ALWAYS AS (
    CASE WHEN is_default = 1 AND is_enabled = 1 THEN agent_type ELSE NULL END
  ) STORED,
  ADD UNIQUE KEY uk_agent_default_type (default_key);
