-- 000003_add_idempotency_constraints.sql
-- Adds generated stored columns + partial unique indexes for data integrity.
-- MySQL 8: NULLs in unique indexes are not counted as duplicates,
-- so a generated column that is NULL when the condition is false
-- creates an effective partial unique index.
-- MySQL DDL performs implicit commits, so this migration is intentionally
-- written as an ordered sequence rather than a transactional script.
-- Uses IF NOT EXISTS / IF EXISTS for idempotency when db.sql is pre-imported.

-- Archive historical duplicate active applications before adding the unique
-- key. Keep the newest active row per (job_id, user_id), and make older rows
-- historical by clearing is_current.
UPDATE applications a
JOIN (
  SELECT id
  FROM (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY job_id, user_id
        ORDER BY updated_at DESC, applied_at DESC, id DESC
      ) AS rn
    FROM applications
    WHERE is_current = 1 AND status <> 3
  ) ranked
  WHERE ranked.rn > 1
) dup ON dup.id = a.id
SET a.is_current = 0;

-- Invalidate historical duplicate valid resumes before adding the unique key.
-- Keep the newest valid resume per user.
UPDATE resumes r
JOIN (
  SELECT id
  FROM (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY user_id
        ORDER BY uploaded_at DESC, id DESC
      ) AS rn
    FROM resumes
    WHERE is_valid = 1
  ) ranked
  WHERE ranked.rn > 1
) dup ON dup.id = r.id
SET r.is_valid = 0;

-- Applications: one active (is_current=1, status<>3) application per user per job
ALTER TABLE applications
  ADD COLUMN IF NOT EXISTS active_key TINYINT
    GENERATED ALWAYS AS (
      CASE WHEN is_current = 1 AND status <> 3 THEN 1 ELSE NULL END
    ) STORED;

ALTER TABLE applications
  DROP INDEX IF EXISTS uk_active_application,
  DROP INDEX IF EXISTS idx_job_current_applied,
  DROP INDEX IF EXISTS idx_user_applied;

ALTER TABLE applications
  ADD UNIQUE KEY uk_active_application (job_id, user_id, active_key),
  ADD KEY idx_job_current_applied (job_id, is_current, applied_at, id),
  ADD KEY idx_user_applied (user_id, applied_at, id);

-- Resumes: one valid (is_valid=1) resume per user
ALTER TABLE resumes
  ADD COLUMN IF NOT EXISTS valid_key TINYINT
    GENERATED ALWAYS AS (
      CASE WHEN is_valid = 1 THEN 1 ELSE NULL END
  ) STORED;

ALTER TABLE resumes
  DROP INDEX IF EXISTS uk_user_valid_resume,
  DROP INDEX IF EXISTS idx_user_uploaded;

ALTER TABLE resumes
  ADD UNIQUE KEY uk_user_valid_resume (user_id, valid_key),
  ADD KEY idx_user_uploaded (user_id, uploaded_at, id);
