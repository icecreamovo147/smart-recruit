import type {
  PlatformAICapability,
  PlatformAICapabilityVersion,
} from '@/api/platformAI'
import type { PlatformEntitlement } from '@/types'

export type CapabilityReleaseVersions = Record<string, PlatformAICapabilityVersion[]>
export type CapabilityReleaseSelections = Record<string, number>

const releaseEntitlementKey = (capabilityKey: string) =>
  `${capabilityKey}.release_version_id`

export const initializeCapabilityReleaseSelections = (
  capabilities: PlatformAICapability[],
  versions: CapabilityReleaseVersions,
  entitlements: PlatformEntitlement[] = [],
): CapabilityReleaseSelections => {
  const entitlementValues = new Map(
    entitlements.map((item) => [item.key, Number(item.value_json)]),
  )
  return Object.fromEntries(capabilities.map((capability) => {
    const publishedIDs = new Set(
      (versions[capability.capability_key] || [])
        .filter((version) => version.status === 'published')
        .map((version) => version.id),
    )
    const configuredID = entitlementValues.get(
      releaseEntitlementKey(capability.capability_key),
    ) || 0
    const selectedID = publishedIDs.has(configuredID)
      ? configuredID
      : capability.current_published_version_id
    return [capability.capability_key, publishedIDs.has(selectedID) ? selectedID : 0]
  }))
}

export const capabilityReleaseEntitlements = (
  capabilities: PlatformAICapability[],
  selections: CapabilityReleaseSelections,
): PlatformEntitlement[] => capabilities
  .map((capability) => ({
    key: releaseEntitlementKey(capability.capability_key),
    value_type: 'integer' as const,
    value_json: String(selections[capability.capability_key] || 0),
    enforcement_mode: 'hard' as const,
  }))
  .filter((entitlement) => Number(entitlement.value_json) > 0)
