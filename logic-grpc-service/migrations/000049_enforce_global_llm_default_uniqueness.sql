-- Before adding the unique constraint, clean existing duplicate enabled defaults.
-- Keep the lowest id as the single global default.
UPDATE llm_models a
JOIN (
  SELECT MIN(id) AS keep_id
  FROM llm_models
  WHERE is_default = 1 AND is_enabled = 1
  HAVING COUNT(*) > 1
) x ON 1 = 1
SET a.is_default = 0
WHERE a.is_default = 1
  AND a.is_enabled = 1
  AND a.id <> x.keep_id;

-- Generated column that is non-NULL only when is_default=1 AND is_enabled=1.
-- The unique index enforces at most one enabled global default.
ALTER TABLE llm_models
  ADD COLUMN global_default_key VARCHAR(16)
  GENERATED ALWAYS AS (
    CASE WHEN is_default = 1 AND is_enabled = 1 THEN 'global' ELSE NULL END
  ) STORED,
  ADD UNIQUE KEY uk_llm_global_default (global_default_key);
