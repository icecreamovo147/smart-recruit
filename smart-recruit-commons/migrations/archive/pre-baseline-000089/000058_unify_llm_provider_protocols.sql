-- Migration: 000058_unify_llm_provider_protocols
-- Description: Separate provider identity from runtime protocol, authentication, and discovery settings.

ALTER TABLE `llm_providers`
    ADD COLUMN `protocol_type` VARCHAR(64) NOT NULL DEFAULT 'openai_chat_completions' COMMENT 'Runtime protocol adapter' AFTER `provider_type`,
    ADD COLUMN `auth_type` VARCHAR(64) NOT NULL DEFAULT 'bearer' COMMENT 'Authentication strategy' AFTER `protocol_type`,
    ADD COLUMN `api_version` VARCHAR(64) NULL COMMENT 'Provider API version, primarily Azure' AFTER `auth_type`,
    ADD COLUMN `discovery_url` VARCHAR(512) NULL COMMENT 'Optional explicit model discovery endpoint' AFTER `api_version`,
    ADD INDEX `idx_llm_provider_protocol_type` (`protocol_type`);

-- Normalize historical UI/runtime aliases before deriving protocol and auth.
UPDATE `llm_providers`
SET `provider_type` = CASE `provider_type`
    WHEN 'azure' THEN 'azure_openai'
    WHEN 'google' THEN 'google_gemini'
    WHEN 'other' THEN 'custom'
    WHEN 'deepseek' THEN 'openai_compatible'
    WHEN '' THEN 'openai_compatible'
    ELSE `provider_type`
END;

UPDATE `llm_providers`
SET
    `protocol_type` = CASE `provider_type`
        WHEN 'anthropic' THEN 'anthropic_messages'
        WHEN 'google_gemini' THEN 'gemini_generate_content'
        WHEN 'ollama' THEN 'ollama_chat'
        ELSE 'openai_chat_completions'
    END,
    `auth_type` = CASE `provider_type`
        WHEN 'anthropic' THEN 'x_api_key'
        WHEN 'azure_openai' THEN 'azure_api_key'
        WHEN 'google_gemini' THEN 'google_api_key'
        WHEN 'ollama' THEN 'none'
        ELSE 'bearer'
    END;

