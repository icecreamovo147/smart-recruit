import { describe, expect, it } from 'vitest'
import type { PlatformPlan, PlatformPlanVersion } from '@/types'
import {
  currentPlanVersions,
  displayedPlans,
  planEntitlementSections,
  resolveDisplayedPlanID,
} from './planVersionDisplay'

const version = (id: number, versionNumber: number): PlatformPlanVersion => ({
  id,
  plan_id: 1,
  version: versionNumber,
  status: 'published',
  effective_at: '',
  retired_at: '',
  change_note: '',
  entitlements: [],
  created_at: '',
  updated_at: '',
})

const plan = (
  versions: PlatformPlanVersion[],
  id = 1,
  planKey = 'starter',
): PlatformPlan => ({
  id,
  plan_key: planKey,
  name: planKey,
  description: '',
  status: 'active',
  versions,
  created_at: '',
  updated_at: '',
})

describe('plan version display', () => {
  it('groups read-only form fields by base, AI features, quotas, and capability versions', () => {
    const sections = planEntitlementSections([
      {
        key: 'ai.chat.enabled',
        value_type: 'boolean',
        value_json: 'true',
        enforcement_mode: 'hard',
      },
      {
        key: 'members.max',
        value_type: 'integer',
        value_json: '20',
        enforcement_mode: 'hard',
      },
      {
        key: 'ai.credits.monthly',
        value_type: 'integer',
        value_json: '100',
        enforcement_mode: 'hard',
      },
      {
        key: 'ai.chat.release_version_id',
        value_type: 'integer',
        value_json: '7',
        enforcement_mode: 'hard',
      },
    ])

    expect(sections.map((section) => ({
      key: section.key,
      items: section.items.map((item) => item.key),
    }))).toEqual([
      { key: 'base', items: ['members.max'] },
      { key: 'ai_features', items: ['ai.chat.enabled'] },
      { key: 'ai_quota', items: ['ai.credits.monthly'] },
      { key: 'capability_versions', items: ['ai.chat.release_version_id'] },
    ])
  })

  it('defaults to the first plan and displays only the selected plan', () => {
    const plans = [
      plan([], 1, 'starter'),
      plan([], 2, 'growth'),
      plan([], 3, 'enterprise'),
    ]

    expect(resolveDisplayedPlanID(plans, null)).toBe(1)
    expect(displayedPlans(plans, 2).map((item) => item.id)).toEqual([2])
  })

  it('falls back to the first plan when a refreshed list no longer contains the selection', () => {
    const plans = [
      plan([], 2, 'growth'),
      plan([], 3, 'enterprise'),
    ]

    expect(resolveDisplayedPlanID(plans, 1)).toBe(2)
    expect(displayedPlans(plans, 1).map((item) => item.id)).toEqual([2])
  })

  it('shows only the highest numbered current version regardless of API ordering', () => {
    expect(currentPlanVersions(plan([
      version(3, 1),
      version(9, 3),
      version(6, 2),
    ])).map((item) => item.id)).toEqual([9])
  })

  it('returns no display version when a plan has no versions', () => {
    expect(currentPlanVersions(plan([]))).toEqual([])
  })
})
