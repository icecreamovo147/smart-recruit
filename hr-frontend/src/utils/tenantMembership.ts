import type { TenantMembership, User } from '@/types/domain'

type UnknownRecord = Record<string, unknown>

const asRecord = (value: unknown): UnknownRecord | null => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  return value as UnknownRecord
}

const asPositiveNumber = (value: unknown): number | undefined => {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

const asBoolean = (value: unknown): boolean => value === true || value === 1 || value === '1' || value === 'true'

export const normalizeTenantMemberships = (value: unknown): TenantMembership[] => {
  if (!Array.isArray(value)) return []

  return value.flatMap((item) => {
    const record = asRecord(item)
    if (!record) return []

    const membershipId = asPositiveNumber(record.membership_id)
    const tenantId = asPositiveNumber(record.tenant_id)
    if (!membershipId || !tenantId) return []

    return [{
      membership_id: membershipId,
      tenant_id: tenantId,
      tenant_key: String(record.tenant_key || ''),
      slug: String(record.slug || ''),
      name: String(record.name || '').trim() || `企业 ${tenantId}`,
      tenant_status: String(record.tenant_status || ''),
      membership_status: String(record.membership_status || ''),
      roles: Array.isArray(record.roles) ? record.roles.map(String) : [],
      is_default: asBoolean(record.is_default),
    }]
  })
}

export const normalizeTenantUser = (user: User | null): User | null => {
  if (!user) return null

  return {
    ...user,
    tenant_id: asPositiveNumber(user.tenant_id),
    membership_id: asPositiveNumber(user.membership_id),
    memberships: normalizeTenantMemberships(user.memberships),
  }
}
