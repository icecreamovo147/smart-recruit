-- 000032_add_agent_runs.down.sql

ALTER TABLE `ai_tool_traces`
    DROP KEY `idx_ai_tool_traces_agent_run_step`,
    DROP KEY `idx_ai_tool_traces_agent_run`,
    DROP COLUMN `duration_ms`,
    DROP COLUMN `agent_run_step_id`,
    DROP COLUMN `agent_run_id`;

DROP TABLE IF EXISTS `agent_run_steps`;
DROP TABLE IF EXISTS `agent_runs`;
