ALTER TABLE ai_memories
  ADD COLUMN importance DECIMAL(4,3) NOT NULL DEFAULT 1.000 AFTER confidence,
  ADD KEY idx_ai_memories_recall (hr_id, scope_type, scope_id, importance, created_at);
