-- Migration: 000059_add_llm_model_discovery_metadata
-- Description: Preserve provider-sourced model metadata separately from editable runtime settings.

ALTER TABLE `llm_models`
    ADD COLUMN `catalog_model_name` VARCHAR(256) NULL COMMENT 'Provider catalog model id when runtime name differs, e.g. Azure deployment' AFTER `model_name`,
    ADD COLUMN `provider_max_input_tokens` INT NULL COMMENT 'Provider-advertised input token limit' AFTER `context_window_tokens`,
    ADD COLUMN `provider_max_output_tokens` INT NULL COMMENT 'Provider-advertised output token limit' AFTER `provider_max_input_tokens`,
    ADD COLUMN `capabilities` JSON NULL COMMENT 'Provider-advertised capabilities' AFTER `provider_max_output_tokens`,
    ADD COLUMN `metadata_source` VARCHAR(32) NOT NULL DEFAULT 'manual' COMMENT 'manual or provider' AFTER `capabilities`,
    ADD COLUMN `metadata_synced_at` DATETIME NULL COMMENT 'Last provider metadata synchronization time' AFTER `metadata_source`,
    ADD COLUMN `temperature_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether temperature is sent to provider' AFTER `metadata_synced_at`,
    ADD COLUMN `top_p_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether top_p is sent to provider' AFTER `temperature_enabled`;

