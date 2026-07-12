-- Down migration: 000024_add_llm_providers
START TRANSACTION;

DROP TABLE IF EXISTS `llm_models`;
DROP TABLE IF EXISTS `llm_providers`;

COMMIT;
