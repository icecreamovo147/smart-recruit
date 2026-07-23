import { describe, expect, it } from 'vitest'
import { normalizeTenantMemberships, normalizeTenantUser } from './tenantMembership'

describe('tenant membership normalization', () => {
  it('normalizes protobuf-style string ids so select values match the active tenant', () => {
    const memberships = normalizeTenantMemberships([{
      membership_id: '11',
      tenant_id: '1',
      tenant_key: 'default',
      slug: 'default',
      name: '默认企业',
      tenant_status: 'active',
      membership_status: 'active',
      roles: ['recruiter'],
      is_default: 'true',
    }])

    expect(memberships).toEqual([expect.objectContaining({
      membership_id: 11,
      tenant_id: 1,
      name: '默认企业',
      is_default: true,
    })])
  })

  it('normalizes cached user tenant ids and drops malformed memberships', () => {
    const user = normalizeTenantUser({
      user_id: 7,
      role: 2,
      username: 'admin',
      tenant_id: '2' as unknown as number,
      membership_id: '22' as unknown as number,
      memberships: [null, { tenant_id: 2 }] as unknown as NonNullable<ReturnType<typeof normalizeTenantUser>>['memberships'],
    })

    expect(user?.tenant_id).toBe(2)
    expect(user?.membership_id).toBe(22)
    expect(user?.memberships).toEqual([])
  })
})
