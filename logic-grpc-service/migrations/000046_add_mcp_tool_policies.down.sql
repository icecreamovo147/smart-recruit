ALTER TABLE `mcp_tool_logs`
    DROP INDEX `idx_mcp_tool_logs_server_tool_created`,
    DROP COLUMN `policy_snapshot_json`,
    DROP COLUMN `policy_reason`,
    DROP COLUMN `policy_decision`,
    DROP COLUMN `policy_id`;

DROP TABLE IF EXISTS `mcp_tool_policies`;
