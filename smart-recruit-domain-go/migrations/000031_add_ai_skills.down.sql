-- 000031_add_ai_skills.down.sql

ALTER TABLE `ai_skills` DROP FOREIGN KEY `fk_ai_skills_current_version`;
DROP TABLE IF EXISTS `ai_skill_tools`;
DROP TABLE IF EXISTS `ai_skill_versions`;
DROP TABLE IF EXISTS `ai_skills`;
