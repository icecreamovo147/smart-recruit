-- Down migration: 000026_add_agent_configs
START TRANSACTION;

DROP TABLE IF EXISTS `agent_tool_bindings`;
DROP TABLE IF EXISTS `agent_configs`;

COMMIT;
