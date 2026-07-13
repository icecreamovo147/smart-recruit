-- Migration: 000027_remove_model_id_from_agent_configs
-- Description: Remove model_id column and FK from agent_configs (model selection moved to runtime)

START TRANSACTION;

ALTER TABLE `agent_configs`
    DROP FOREIGN KEY `fk_agent_configs_model`,
    DROP INDEX `idx_model_id`,
    DROP COLUMN `model_id`;

COMMIT;
