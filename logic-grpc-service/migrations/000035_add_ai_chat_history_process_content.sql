-- Migration: 000035_add_ai_chat_history_process_content
-- Description: Persist assistant pre-answer process narration separately from final reply.

ALTER TABLE ai_chat_history
    ADD COLUMN `process_content` TEXT NULL COMMENT 'assistant 前置执行过程说明' AFTER `content`;
