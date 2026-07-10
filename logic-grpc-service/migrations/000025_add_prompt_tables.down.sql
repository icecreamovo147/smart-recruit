-- Down migration: 000025_add_prompt_tables
START TRANSACTION;

DROP TABLE IF EXISTS `prompt_versions`;
DROP TABLE IF EXISTS `prompt_templates`;

COMMIT;
