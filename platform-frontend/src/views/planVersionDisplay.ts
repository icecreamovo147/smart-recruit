import type {
  PlatformEntitlement,
  PlatformPlan,
  PlatformPlanVersion,
} from '@/types'

export interface PlanEntitlementSection {
  key: 'base' | 'ai_features' | 'ai_quota' | 'capability_versions'
  title: string
  items: PlatformEntitlement[]
}

const aiQuotaKeys = new Set([
  'ai.credits.monthly',
  'ai.concurrent_runs.max',
  'ai.single_run.max_credits',
])

export const planEntitlementSections = (
  entitlements: PlatformEntitlement[],
): PlanEntitlementSection[] => {
  const base: PlatformEntitlement[] = []
  const aiFeatures: PlatformEntitlement[] = []
  const aiQuota: PlatformEntitlement[] = []
  const capabilityVersions: PlatformEntitlement[] = []

  for (const entitlement of entitlements) {
    if (!entitlement.key.startsWith('ai.')) {
      base.push(entitlement)
    } else if (entitlement.key.endsWith('.release_version_id')) {
      capabilityVersions.push(entitlement)
    } else if (aiQuotaKeys.has(entitlement.key)) {
      aiQuota.push(entitlement)
    } else {
      aiFeatures.push(entitlement)
    }
  }

  return [
    { key: 'base', title: '基础容量', items: base },
    { key: 'ai_features', title: 'AI 功能', items: aiFeatures },
    { key: 'ai_quota', title: 'AI 额度', items: aiQuota },
    { key: 'capability_versions', title: '能力版本', items: capabilityVersions },
  ].filter((section) => section.items.length > 0) as PlanEntitlementSection[]
}

export const resolveDisplayedPlanID = (
  plans: PlatformPlan[],
  requestedPlanID: number | null,
): number | null => {
  if (requestedPlanID != null && plans.some((plan) => plan.id === requestedPlanID)) {
    return requestedPlanID
  }
  return plans[0]?.id ?? null
}

export const displayedPlans = (
  plans: PlatformPlan[],
  requestedPlanID: number | null,
): PlatformPlan[] => {
  const planID = resolveDisplayedPlanID(plans, requestedPlanID)
  if (planID == null) return []
  const plan = plans.find((item) => item.id === planID)
  return plan ? [plan] : []
}

export const currentPlanVersions = (plan: PlatformPlan): PlatformPlanVersion[] => {
  const current = (plan.versions || []).reduce<PlatformPlanVersion | undefined>((latest, version) => {
    if (!latest) return version
    if (version.version !== latest.version) {
      return version.version > latest.version ? version : latest
    }
    return version.id > latest.id ? version : latest
  }, undefined)

  return current ? [current] : []
}
