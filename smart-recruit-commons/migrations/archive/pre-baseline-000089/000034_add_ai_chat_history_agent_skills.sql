-- Migration: 000034_add_ai_chat_history_agent_skills
-- Description: Persist per-message Agent Skill metadata for chat history display.

ALTER TABLE ai_chat_history
    ADD COLUMN `agent_skill_ids_json` TEXT NULL COMMENT '本条用户消息选择的 Agent Skill ID 快照(JSON数组)' AFTER `model_name`,
    ADD COLUMN `agent_skill_names_json` TEXT NULL COMMENT '本条用户消息选择的 Agent Skill 名称快照(JSON数组)' AFTER `agent_skill_ids_json`;
