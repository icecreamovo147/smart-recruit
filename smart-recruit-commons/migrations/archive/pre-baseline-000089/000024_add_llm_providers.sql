-- Migration: 000024_add_llm_providers
-- Description: Add llm_providers and llm_models tables for LLM provider/model config management

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `llm_providers` (
    `id`              BIGINT       NOT NULL AUTO_INCREMENT,
    `name`            VARCHAR(128) NOT NULL COMMENT 'Provider display name',
    `base_url`        VARCHAR(512) NOT NULL COMMENT 'API base URL',
    `api_key_encrypted` VARCHAR(512) NOT NULL COMMENT 'AES-256-GCM encrypted API key',
    `provider_type`   VARCHAR(64)  NOT NULL COMMENT 'openai_compatible/anthropic/deepseek/ollama',
    `extra_headers`   JSON         COMMENT 'Extra HTTP headers as JSON object',
    `is_enabled`      TINYINT(1)   NOT NULL DEFAULT 1 COMMENT 'Whether provider is enabled',
    `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_provider_type` (`provider_type`),
    INDEX `idx_is_enabled` (`is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='LLM provider configurations';

CREATE TABLE IF NOT EXISTS `llm_models` (
    `id`              BIGINT       NOT NULL AUTO_INCREMENT,
    `provider_id`     BIGINT       NOT NULL COMMENT 'FK to llm_providers.id',
    `model_name`      VARCHAR(128) NOT NULL COMMENT 'Model name used in API calls',
    `display_name`    VARCHAR(128) COMMENT 'Human-readable display name',
    `temperature`     DOUBLE       NOT NULL DEFAULT 0.7 COMMENT 'LLM temperature parameter',
    `top_p`           DOUBLE       NOT NULL DEFAULT 1.0 COMMENT 'LLM top_p parameter',
    `max_tokens`      INT          NOT NULL DEFAULT 4096 COMMENT 'Max tokens for generation',
    `max_concurrency` INT          NOT NULL DEFAULT 10 COMMENT 'Max concurrent LLM calls',
    `timeout_seconds` INT          NOT NULL DEFAULT 90 COMMENT 'Request timeout in seconds',
    `is_enabled`      TINYINT(1)   NOT NULL DEFAULT 1 COMMENT 'Whether model is enabled',
    `is_default`      TINYINT(1)   NOT NULL DEFAULT 0 COMMENT 'Whether this is the default model',
    `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_provider_id` (`provider_id`),
    INDEX `idx_model_enabled` (`is_enabled`),
    INDEX `idx_model_default` (`is_default`),
    CONSTRAINT `fk_llm_models_provider` FOREIGN KEY (`provider_id`) REFERENCES `llm_providers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='LLM model configurations';

COMMIT;
