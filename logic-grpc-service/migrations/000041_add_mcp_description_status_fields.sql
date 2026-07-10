ALTER TABLE mcp_servers
  ADD COLUMN description TEXT NULL COMMENT 'Human-readable server description' AFTER name,
  ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'disconnected' COMMENT 'connected / disconnected / error' AFTER is_enabled,
  ADD COLUMN tool_count INT NOT NULL DEFAULT 0 COMMENT 'Number of tools discovered' AFTER status,
  ADD COLUMN last_error VARCHAR(512) NULL COMMENT 'Last connection or operation error message' AFTER tool_count;
