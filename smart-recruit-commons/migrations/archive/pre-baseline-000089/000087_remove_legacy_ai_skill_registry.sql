-- Retire the unused AI Skill Registry after the application cutover.
-- Published release snapshots are intentionally rewritten under the approved
-- maintenance-window exception; the before/after payloads remain auditable.

INSERT INTO `platform_ai_config_audit_logs` (
  `action`, `resource_type`, `resource_id`, `capability_id`,
  `capability_version_id`, `before_snapshot`, `after_snapshot`, `request_id`
)
SELECT
  'legacy.ai_skill_registry.remove',
  'platform_ai_capability_version',
  version.`id`,
  version.`capability_id`,
  version.`id`,
  version.`snapshot_json`,
  JSON_REMOVE(version.`snapshot_json`, '$.configuration_refs.ai_skill_version_ids'),
  'migration:000087'
FROM `platform_ai_capability_versions` version
WHERE JSON_CONTAINS_PATH(version.`snapshot_json`, 'one', '$.configuration_refs.ai_skill_version_ids');

-- Two historical releases can become byte-identical after their only
-- difference (the retired registry reference) is removed. Keep the hash as a
-- content hash, rather than adding artificial entropy to satisfy the former
-- uniqueness constraint.
ALTER TABLE `platform_ai_capability_versions`
  DROP KEY `uk_platform_ai_capability_versions_hash`,
  ADD KEY `idx_platform_ai_capability_versions_hash` (`capability_id`, `snapshot_hash`);

UPDATE `platform_ai_capability_versions`
SET
  `snapshot_json` = JSON_REMOVE(`snapshot_json`, '$.configuration_refs.ai_skill_version_ids'),
  `snapshot_hash` = SHA2(CAST(JSON_REMOVE(`snapshot_json`, '$.configuration_refs.ai_skill_version_ids') AS CHAR), 256)
WHERE JSON_CONTAINS_PATH(`snapshot_json`, 'one', '$.configuration_refs.ai_skill_version_ids');

DELETE FROM `agent_capability_bindings`
WHERE `capability_source` = 'skill';

ALTER TABLE `ai_skills` DROP FOREIGN KEY `fk_ai_skills_current_version`;
DROP TABLE IF EXISTS `ai_skill_tools`;
DROP TABLE IF EXISTS `ai_skill_versions`;
DROP TABLE IF EXISTS `ai_skills`;
