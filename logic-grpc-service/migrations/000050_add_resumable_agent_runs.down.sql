-- 000050_add_resumable_agent_runs_down.sql
-- Description: Remove only resumable HR Agent run schema changes from 000050.
-- Runner expects `000050_add_resumable_agent_runs.down.sql`; this allowlisted name must be renamed before Down().

ALTER TABLE `ai_chat_history`
    DROP KEY `idx_ai_chat_history_agent_run`,
    DROP COLUMN `agent_run_id`;

ALTER TABLE `ai_chat_sessions`
    DROP KEY `idx_ai_chat_sessions_active_run`,
    DROP COLUMN `active_run_id`;

DROP TABLE IF EXISTS `agent_run_events`;

ALTER TABLE `agent_runs`
    DROP KEY `idx_agent_runs_session_status`,
    DROP KEY `uk_agent_runs_client_request`,
    DROP COLUMN `canceled_at`,
    DROP COLUMN `cancel_requested_at`,
    DROP COLUMN `last_event_seq`,
    DROP COLUMN `option_context_json`,
    DROP COLUMN `confirmation_request_json`,
    DROP COLUMN `result_metadata_json`,
    DROP COLUMN `process_text`,
    DROP COLUMN `assistant_text`,
    DROP COLUMN `client_request_id`;
