DELETE subscription
FROM `tenant_subscriptions` subscription
JOIN `platform_plan_versions` version
  ON version.id = subscription.plan_version_id
 AND version.version = 2
JOIN `platform_plans` plan
  ON plan.id = version.plan_id
 AND plan.plan_key = 'starter'
WHERE subscription.reason = 'legacy tenant starter plan bootstrap'
  AND subscription.created_by IS NULL;
