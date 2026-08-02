import { beforeEach, describe, expect, it, vi } from 'vitest'

const httpMocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
}))

vi.mock('./http', () => ({ default: httpMocks }))

import {
  getTenantSubscription,
  listPlans,
  updateTenantSubscription,
} from './control'

describe('platform commercial API numeric IDs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('normalizes protobuf int64 strings in plans and subscriptions', async () => {
    httpMocks.get
      .mockResolvedValueOnce({
        list: [{
          id: '1',
          versions: [{
            id: '7',
            plan_id: '1',
            version: 3,
            entitlements: [],
          }],
        }],
      })
      .mockResolvedValueOnce({
        subscription: {
          id: '9',
          tenant_id: '1',
          plan_version_id: '7',
          plan_version: 3,
          effective_entitlements: [],
        },
      })

    const plans = await listPlans()
    const subscription = await getTenantSubscription(1)

    expect(plans.list[0]?.id).toBe(1)
    expect(plans.list[0]?.versions[0]?.id).toBe(7)
    expect(plans.list[0]?.versions[0]?.plan_id).toBe(1)
    expect(subscription.subscription?.id).toBe(9)
    expect(subscription.subscription?.tenant_id).toBe(1)
    expect(subscription.subscription?.plan_version_id).toBe(7)
  })

  it('always sends plan_version_id as a JSON number', async () => {
    httpMocks.put.mockResolvedValue({ subscription: {} })

    await updateTenantSubscription(1, {
      plan_version_id: '7' as unknown as number,
      starts_at: '2026-07-29T11:20:44+08:00',
      reason: 'upgrade plan',
    })

    expect(httpMocks.put).toHaveBeenCalledWith(
      '/api/v1/platform/tenants/1/subscription',
      expect.objectContaining({ plan_version_id: 7 }),
    )
  })
})
