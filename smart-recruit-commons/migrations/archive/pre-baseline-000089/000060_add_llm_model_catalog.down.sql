ALTER TABLE `llm_models`
    DROP COLUMN `metadata_sources`,
    MODIFY COLUMN `metadata_source` VARCHAR(32) NOT NULL DEFAULT 'manual' COMMENT 'manual or provider';

DROP TABLE IF EXISTS `llm_model_metadata_observations`;
DROP TABLE IF EXISTS `llm_model_catalog`;
