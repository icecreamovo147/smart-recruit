-- 000050_add_resumable_agent_runs.sql
-- Description: Add durable resumable HR Agent run fields, event log, and session/history associations.

ALTER TABLE `agent_runs`
    ADD COLUMN `client_request_id` VARCHAR(128) NULL COMMENT 'Client idempotency key for create-run' AFTER `hr_id`,
    ADD COLUMN `assistant_text` MEDIUMTEXT NULL COMMENT 'Latest assistant answer snapshot for refresh restore' AFTER `final_answer`,
    ADD COLUMN `process_text` MEDIUMTEXT NULL COMMENT 'Latest process trace snapshot for refresh restore' AFTER `assistant_text`,
    ADD COLUMN `result_metadata_json` JSON NULL COMMENT 'Result metadata snapshot' AFTER `process_text`,
    ADD COLUMN `confirmation_request_json` JSON NULL COMMENT 'Pending skill confirmation request snapshot' AFTER `result_metadata_json`,
    ADD COLUMN `option_context_json` JSON NULL COMMENT 'Active candidate/action option context snapshot' AFTER `confirmation_request_json`,
    ADD COLUMN `last_event_seq` BIGINT NOT NULL DEFAULT 0 COMMENT 'Last persisted event sequence for this run' AFTER `option_context_json`,
    ADD COLUMN `cancel_requested_at` TIMESTAMP NULL COMMENT 'When cancel was requested' AFTER `last_event_seq`,
    ADD COLUMN `canceled_at` TIMESTAMP NULL COMMENT 'When run reached canceled terminal state' AFTER `cancel_requested_at`,
    ADD UNIQUE KEY `uk_agent_runs_client_request` (`hr_id`, `session_id`, `client_request_id`),
    ADD KEY `idx_agent_runs_session_status` (`session_id`, `status`);

CREATE TABLE IF NOT EXISTS `agent_run_events` (
    `id`           BIGINT       NOT NULL AUTO_INCREMENT,
    `run_id`       BIGINT       NOT NULL,
    `seq`          BIGINT       NOT NULL,
    `event_type`   VARCHAR(64)  NOT NULL,
    `payload_json` JSON         NULL,
    `created_at`   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_agent_run_events_run_seq` (`run_id`, `seq`),
    KEY `idx_agent_run_events_run` (`run_id`),
    CONSTRAINT `fk_agent_run_events_run` FOREIGN KEY (`run_id`) REFERENCES `agent_runs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Ordered durable events for resumable agent runs';

ALTER TABLE `ai_chat_sessions`
    ADD COLUMN `active_run_id` BIGINT NULL COMMENT 'Current active agent_runs.id for this session' AFTER `latest_context_usage_json`,
    ADD KEY `idx_ai_chat_sessions_active_run` (`active_run_id`);

ALTER TABLE `ai_chat_history`
    ADD COLUMN `agent_run_id` BIGINT NULL COMMENT 'Optional agent_runs.id that produced this history message' AFTER `agent_skill_names_json`,
    ADD KEY `idx_ai_chat_history_agent_run` (`agent_run_id`);
