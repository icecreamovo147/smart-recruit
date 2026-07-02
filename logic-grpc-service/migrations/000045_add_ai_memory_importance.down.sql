ALTER TABLE ai_memories
  DROP KEY idx_ai_memories_recall,
  DROP COLUMN importance;
