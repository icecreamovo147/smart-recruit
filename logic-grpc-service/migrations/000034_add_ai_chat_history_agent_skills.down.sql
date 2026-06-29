-- Down migration: 000034_add_ai_chat_history_agent_skills

ALTER TABLE ai_chat_history
    DROP COLUMN `agent_skill_names_json`,
    DROP COLUMN `agent_skill_ids_json`;
