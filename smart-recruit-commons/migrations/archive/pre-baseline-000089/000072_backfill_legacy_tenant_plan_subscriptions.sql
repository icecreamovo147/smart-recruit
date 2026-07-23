-- The platform plan tables were introduced after the original tenant seed and
-- onboarding flows. Backfill only active tenants that have never had a plan so
-- the AI release gate can resolve the immutable release IDs carried by the
-- published Starter v2 entitlements. Tenants with any subscription history are
-- deliberately left untouched.
INSERT INTO `tenant_subscriptions`
  (`tenant_id`, `plan_version_id`, `status`, `starts_at`, `ends_at`, `reason`, `created_by`)
SELECT
  tenant.id,
  version.id,
  'active',
  NOW(3),
  NULL,
  'legacy tenant starter plan bootstrap',
  NULL
FROM `tenants` tenant
JOIN `platform_plans` plan
  ON plan.plan_key = 'starter'
 AND plan.status = 'active'
JOIN `platform_plan_versions` version
  ON version.plan_id = plan.id
 AND version.version = 2
 AND version.status = 'published'
 AND version.effective_at <= NOW(3)
WHERE tenant.status = 'active'
  AND NOT EXISTS (
    SELECT 1
    FROM `tenant_subscriptions` existing
    WHERE existing.tenant_id = tenant.id
  );
