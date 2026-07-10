ALTER TABLE llm_models
  DROP INDEX uk_llm_global_default,
  DROP COLUMN global_default_key;
