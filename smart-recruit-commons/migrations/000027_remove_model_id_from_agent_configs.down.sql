-- Down migration: 000027_remove_model_id_from_agent_configs
-- Restore model_id column to agent_configs

START TRANSACTION;

ALTER TABLE `agent_configs`
    ADD COLUMN `model_id` BIGINT COMMENT 'FK to llm_models.id, NULL = use system default' AFTER `agent_type`,
    ADD INDEX `idx_model_id` (`model_id`),
    ADD CONSTRAINT `fk_agent_configs_model` FOREIGN KEY (`model_id`) REFERENCES `llm_models` (`id`) ON DELETE SET NULL;

COMMIT;
