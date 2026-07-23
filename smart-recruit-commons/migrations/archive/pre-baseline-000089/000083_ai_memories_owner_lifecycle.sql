-- Extend ai_memories with owner lifecycle, provenance, deduplication, and recall indexes.
-- Existing HR-owned rows are backfilled to owner_role=2 with hr_id preserved as a
-- compatibility mirror. MySQL DDL performs implicit commits, so this migration is
-- intentionally written as an ordered sequence rather than a transactional script.

ALTER TABLE ai_memories
  ADD COLUMN owner_role TINYINT NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR' AFTER tenant_id,
  ADD COLUMN owner_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID' AFTER owner_role,
  ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active / archived / revoked' AFTER importance,
  ADD COLUMN deleted_at DATETIME NULL COMMENT '软删除时间' AFTER status,
  ADD COLUMN content_hash CHAR(64) NULL COMMENT 'SHA-256 of normalized content' AFTER deleted_at,
  ADD COLUMN pii_level VARCHAR(16) NOT NULL DEFAULT 'none' COMMENT 'none / low / high' AFTER content_hash,
  ADD COLUMN source_session_id BIGINT UNSIGNED NULL COMMENT '来源会话ID' AFTER pii_level,
  ADD COLUMN source_message_id BIGINT UNSIGNED NULL COMMENT '来源消息ID' AFTER source_session_id,
  ADD COLUMN source_run_id BIGINT UNSIGNED NULL COMMENT '来源 Agent Run ID' AFTER source_message_id,
  ADD COLUMN created_by BIGINT UNSIGNED NULL COMMENT '创建者用户ID' AFTER source_run_id,
  ADD COLUMN revoked_by BIGINT UNSIGNED NULL COMMENT '撤销者用户ID' AFTER created_by,
  ADD COLUMN revoke_reason VARCHAR(512) NULL COMMENT '撤销原因' AFTER revoked_by,
  ADD KEY idx_ai_memories_tenant_owner_status_scope (tenant_id, owner_role, owner_id, status, scope_type, scope_id),
  ADD KEY idx_ai_memories_expires_status (expires_at, status),
  ADD KEY idx_ai_memories_content_hash_owner (content_hash, owner_role, owner_id);

-- Backfill existing HR-owned memories.
UPDATE ai_memories
SET owner_role = 2,
    owner_id = hr_id,
    status = 'active'
WHERE owner_id = 0 AND hr_id > 0;

UPDATE ai_memories
SET owner_role = 2,
    owner_id = hr_id,
    status = 'active'
WHERE owner_role = 2 AND owner_id = 0 AND hr_id > 0;
