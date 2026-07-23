-- Migration: 000060_add_llm_model_catalog
-- Description: Add verified model catalog data and field-level provider observations.

CREATE TABLE `llm_model_catalog` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `provider_family` VARCHAR(64) NOT NULL COMMENT 'Stable provider family, e.g. deepseek/openai/anthropic',
    `model_name` VARCHAR(256) NOT NULL COMMENT 'Provider catalog model identifier',
    `display_name` VARCHAR(256) NULL COMMENT 'Human-readable model name',
    `context_window_tokens` INT NULL COMMENT 'Total context window; NULL = unknown',
    `max_input_tokens` INT NULL COMMENT 'Provider-advertised input limit; NULL = unknown',
    `max_output_tokens` INT NULL COMMENT 'Provider-advertised output limit; NULL = unknown',
    `temperature` DOUBLE NULL COMMENT 'Provider request default; NULL = unknown',
    `top_p` DOUBLE NULL COMMENT 'Provider request default; NULL = unknown',
    `capabilities` JSON NULL COMMENT 'Verified model capabilities',
    `field_sources` JSON NULL COMMENT 'Per-field provenance overrides',
    `source_type` VARCHAR(32) NOT NULL COMMENT 'official_document/provider_api/admin',
    `source_url` VARCHAR(1024) NULL COMMENT 'Evidence URL without credentials',
    `source_revision` VARCHAR(128) NOT NULL COMMENT 'Version of the imported source data',
    `content_hash` CHAR(64) NOT NULL COMMENT 'SHA-256 of normalized catalog content',
    `status` VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT 'active/inactive',
    `managed_by` VARCHAR(32) NOT NULL DEFAULT 'bundled' COMMENT 'bundled/admin/remote_sync',
    `verified_at` DATETIME NULL,
    `expires_at` DATETIME NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_llm_model_catalog_family_model` (`provider_family`, `model_name`),
    KEY `idx_llm_model_catalog_status` (`status`, `expires_at`),
    KEY `idx_llm_model_catalog_hash` (`content_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Verified reusable LLM model metadata catalog';

CREATE TABLE `llm_model_metadata_observations` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `provider_id` BIGINT NOT NULL COMMENT 'Provider that produced the observation',
    `model_name` VARCHAR(256) NOT NULL,
    `field_name` VARCHAR(64) NOT NULL COMMENT 'Observed metadata field',
    `value_json` JSON NOT NULL COMMENT 'Typed observed value encoded as JSON',
    `source_type` VARCHAR(32) NOT NULL COMMENT 'provider_api/provider_detail/official_document',
    `source_ref` VARCHAR(1024) NULL COMMENT 'Sanitized endpoint or evidence URL',
    `confidence` DECIMAL(5,4) NOT NULL DEFAULT 1.0000,
    `content_hash` CHAR(64) NOT NULL COMMENT 'Deduplication hash excluding observation time',
    `observed_at` DATETIME NOT NULL,
    `expires_at` DATETIME NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_llm_model_observation_hash` (`content_hash`),
    KEY `idx_llm_model_observation_lookup` (`provider_id`, `model_name`, `field_name`, `observed_at`),
    KEY `idx_llm_model_observation_expiry` (`expires_at`),
    CONSTRAINT `fk_llm_model_observations_provider` FOREIGN KEY (`provider_id`) REFERENCES `llm_providers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Field-level model metadata observations and provenance';

ALTER TABLE `llm_models`
    ADD COLUMN `metadata_sources` JSON NULL COMMENT 'Field-level metadata provenance captured when the user saved the model' AFTER `metadata_source`,
    MODIFY COLUMN `metadata_source` VARCHAR(32) NOT NULL DEFAULT 'manual' COMMENT 'manual/provider/merged/mixed';
