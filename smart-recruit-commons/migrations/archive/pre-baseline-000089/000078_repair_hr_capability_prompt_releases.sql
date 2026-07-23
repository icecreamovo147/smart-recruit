-- Repair immutable HR capability releases seeded without their Agent-bound Prompt.
-- The original published rows remain untouched; corrected releases are appended and
-- entitlement pointers are advanced to the corrected immutable versions.

INSERT INTO `platform_ai_capability_versions`
  (`capability_id`, `version`, `status`, `snapshot_json`, `snapshot_hash`, `change_note`, `published_at`, `created_at`, `updated_at`)
SELECT
  source.capability_id,
  source.next_version,
  'published',
  source.corrected_snapshot,
  SHA2(source.corrected_snapshot, 256),
  'migration 000078: repair Agent-bound Prompt release references',
  NOW(3),
  NOW(3),
  NOW(3)
FROM (
  SELECT
    capability.id AS capability_id,
    (SELECT COALESCE(MAX(existing.version), 0) + 1
       FROM platform_ai_capability_versions existing
      WHERE existing.capability_id = capability.id) AS next_version,
    JSON_SET(
      current_version.snapshot_json,
      '$.configuration_refs.prompt_template_ids',
      COALESCE((
        SELECT JSON_ARRAYAGG(agent.prompt_template_id)
          FROM agent_configs agent
          JOIN prompt_templates prompt
            ON prompt.id = agent.prompt_template_id
           AND prompt.is_active = 1
           AND LOWER(TRIM(prompt.prompt_role)) = 'system'
         WHERE agent.is_enabled = 1
           AND agent.prompt_template_id IS NOT NULL
           AND JSON_CONTAINS(
             JSON_EXTRACT(current_version.snapshot_json, '$.configuration_refs.agent_ids'),
             CAST(agent.id AS JSON)
           )
      ), JSON_ARRAY())
    ) AS corrected_snapshot
  FROM platform_ai_capabilities capability
  JOIN platform_ai_capability_versions current_version
    ON current_version.id = capability.current_published_version_id
   AND current_version.status = 'published'
  WHERE capability.audience = 'tenant_hr'
    AND capability.capability_key IN ('ai.chat', 'ai.agent_run', 'ai.application_analysis')
    AND JSON_LENGTH(JSON_EXTRACT(current_version.snapshot_json, '$.configuration_refs.agent_ids')) > 0
    AND JSON_LENGTH(JSON_EXTRACT(current_version.snapshot_json, '$.configuration_refs.prompt_template_ids')) = 0
    AND NOT EXISTS (
      SELECT 1
        FROM platform_ai_capability_versions repaired
       WHERE repaired.capability_id = capability.id
         AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
    )
) source
WHERE JSON_LENGTH(JSON_EXTRACT(source.corrected_snapshot, '$.configuration_refs.prompt_template_ids')) > 0;

UPDATE platform_ai_capabilities capability
JOIN platform_ai_capability_versions repaired
  ON repaired.capability_id = capability.id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
SET capability.current_published_version_id = repaired.id,
    capability.updated_at = NOW(3);

UPDATE platform_plan_entitlements entitlement
JOIN platform_plan_versions plan_version ON plan_version.id = entitlement.plan_version_id
JOIN platform_ai_capabilities capability
  ON capability.audience = 'tenant_hr'
 AND entitlement.entitlement_key = CONCAT(capability.capability_key, '.release_version_id')
JOIN platform_ai_capability_versions repaired
  ON repaired.id = capability.current_published_version_id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
SET entitlement.value_type = 'integer',
    entitlement.value_json = CAST(repaired.id AS JSON),
    entitlement.enforcement_mode = 'hard',
    entitlement.updated_at = NOW(3);

UPDATE tenant_entitlement_overrides entitlement_override
JOIN platform_ai_capabilities capability
  ON capability.audience = 'tenant_hr'
 AND entitlement_override.entitlement_key = CONCAT(capability.capability_key, '.release_version_id')
JOIN platform_ai_capability_versions repaired
  ON repaired.id = capability.current_published_version_id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
SET entitlement_override.value_type = 'integer',
    entitlement_override.value_json = CAST(repaired.id AS JSON),
    entitlement_override.updated_at = NOW(3);

UPDATE billing_price_versions price
JOIN platform_ai_capabilities capability ON capability.audience = 'tenant_hr'
JOIN platform_ai_capability_versions repaired
  ON repaired.id = capability.current_published_version_id
 AND repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
SET price.entitlement_snapshot = JSON_SET(
      price.entitlement_snapshot,
      CONCAT('$."', capability.capability_key, '.release_version_id"'),
      repaired.id
    ),
    price.updated_at = NOW(3)
WHERE JSON_CONTAINS_PATH(
  price.entitlement_snapshot,
  'one',
  CONCAT('$."', capability.capability_key, '.release_version_id"')
);

INSERT INTO platform_ai_config_audit_logs
  (`action`, `resource_type`, `resource_id`, `capability_id`, `capability_version_id`, `after_snapshot`, `request_id`, `created_at`)
SELECT
  'capability.release.migration.repair',
  'platform_ai_capability_version',
  repaired.id,
  repaired.capability_id,
  repaired.id,
  repaired.snapshot_json,
  'migration-000078',
  NOW(3)
FROM platform_ai_capability_versions repaired
WHERE repaired.change_note = 'migration 000078: repair Agent-bound Prompt release references'
  AND NOT EXISTS (
    SELECT 1
      FROM platform_ai_config_audit_logs audit
     WHERE audit.capability_version_id = repaired.id
       AND audit.action = 'capability.release.migration.repair'
  );
