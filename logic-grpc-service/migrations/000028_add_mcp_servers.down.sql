-- 000028_add_mcp_servers.down.sql
-- Rollback P1-006: Remove MCP tool call logs and server tables

DROP TABLE IF EXISTS `mcp_tool_logs`;
DROP TABLE IF EXISTS `mcp_servers`;
