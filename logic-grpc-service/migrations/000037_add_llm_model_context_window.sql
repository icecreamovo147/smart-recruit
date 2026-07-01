ALTER TABLE `llm_models`
  ADD COLUMN `context_window_tokens` INT NOT NULL DEFAULT 0
  COMMENT 'Total model context window tokens (input + output); 0 = unknown'
  AFTER `max_tokens`;
