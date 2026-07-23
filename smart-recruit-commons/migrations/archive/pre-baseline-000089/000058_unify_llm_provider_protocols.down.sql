ALTER TABLE `llm_providers`
    DROP INDEX `idx_llm_provider_protocol_type`,
    DROP COLUMN `discovery_url`,
    DROP COLUMN `api_version`,
    DROP COLUMN `auth_type`,
    DROP COLUMN `protocol_type`;

