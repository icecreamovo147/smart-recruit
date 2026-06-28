-- 000032_add_agent_runs.sql
-- Description: Add HR AI agent run and run-step observability tables.

CREATE TABLE IF NOT EXISTS `agent_runs` (
    `id`             BIGINT       NOT NULL AUTO_INCREMENT,
    `session_id`     BIGINT       NOT NULL,
    `message_id`     BIGINT       NULL,
    `history_id`     BIGINT       NULL,
    `hr_id`          BIGINT       NOT NULL,
    `agent_type`     VARCHAR(64)  NOT NULL DEFAULT 'hr',
    `agent_id`       BIGINT       NULL,
    `agent_name`     VARCHAR(128) NOT NULL,
    `model_id`       BIGINT       NULL,
    `model_name`     VARCHAR(128) NOT NULL,
    `status`         VARCHAR(32)  NOT NULL DEFAULT 'planning',
    `plan_json`      JSON         NULL,
    `final_answer`   MEDIUMTEXT   NULL,
    `error_type`     VARCHAR(64)  NULL,
    `error_message`  TEXT         NULL,
    `started_at`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `completed_at`   TIMESTAMP    NULL,
    `created_at`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_agent_runs_session_created` (`hr_id`, `session_id`, `created_at`),
    KEY `idx_agent_runs_status` (`status`),
    KEY `idx_agent_runs_message` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='One observable run per HR AI user message';

CREATE TABLE IF NOT EXISTS `agent_run_steps` (
    `id`                BIGINT       NOT NULL AUTO_INCREMENT,
    `run_id`            BIGINT       NOT NULL,
    `step_index`        INT          NOT NULL,
    `step_type`         VARCHAR(32)  NOT NULL,
    `capability_source` VARCHAR(64)  NULL,
    `capability_key`    VARCHAR(128) NULL,
    `tool_name`         VARCHAR(128) NULL,
    `input_json`        JSON         NULL,
    `output_json`       JSON         NULL,
    `status`            VARCHAR(32)  NOT NULL DEFAULT 'running',
    `duration_ms`       BIGINT       NOT NULL DEFAULT 0,
    `error_message`     TEXT         NULL,
    `started_at`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `completed_at`      TIMESTAMP    NULL,
    `created_at`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_agent_run_steps_run_index` (`run_id`, `step_index`),
    KEY `idx_agent_run_steps_run` (`run_id`),
    KEY `idx_agent_run_steps_type` (`step_type`),
    CONSTRAINT `fk_agent_run_steps_run` FOREIGN KEY (`run_id`) REFERENCES `agent_runs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Observable steps within an agent run';

ALTER TABLE `ai_tool_traces`
    ADD COLUMN `agent_run_id` BIGINT NULL AFTER `hr_id`,
    ADD COLUMN `agent_run_step_id` BIGINT NULL AFTER `agent_run_id`,
    ADD COLUMN `duration_ms` BIGINT NOT NULL DEFAULT 0 AFTER `status`,
    ADD KEY `idx_ai_tool_traces_agent_run` (`agent_run_id`),
    ADD KEY `idx_ai_tool_traces_agent_run_step` (`agent_run_step_id`);
