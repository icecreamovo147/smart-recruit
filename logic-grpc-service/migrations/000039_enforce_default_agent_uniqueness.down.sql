ALTER TABLE agent_configs
  DROP INDEX uk_agent_default_type,
  DROP COLUMN default_key;
