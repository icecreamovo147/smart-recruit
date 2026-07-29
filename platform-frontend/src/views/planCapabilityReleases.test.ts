import { describe, expect, it } from 'vitest'
import type {
  PlatformAICapability,
  PlatformAICapabilityVersion,
} from '@/api/platformAI'
import {
  capabilityReleaseEntitlements,
  initializeCapabilityReleaseSelections,
} from './planCapabilityReleases'

const capability = (
  capabilityKey: string,
  currentPublishedVersionID: number,
): PlatformAICapability => ({
  id: currentPublishedVersionID,
  capability_key: capabilityKey,
  audience: 'tenant_hr',
  name: capabilityKey,
  description: '',
  status: 'active',
  current_published_version_id: currentPublishedVersionID,
})

const version = (
  id: number,
  capabilityID: number,
  status: PlatformAICapabilityVersion['status'] = 'published',
): PlatformAICapabilityVersion => ({
  id,
  capability_id: capabilityID,
  version: id,
  status,
  snapshot_json: '{}',
  snapshot_hash: '',
  change_note: '',
  published_at: '',
})

describe('plan capability release selections', () => {
  it('preserves an explicitly pinned published version instead of silently upgrading it', () => {
    const agentRun = capability('ai.agent_run', 12)
    const selections = initializeCapabilityReleaseSelections(
      [agentRun],
      { 'ai.agent_run': [version(10, agentRun.id), version(12, agentRun.id)] },
      [{
        key: 'ai.agent_run.release_version_id',
        value_type: 'integer',
        value_json: '10',
        enforcement_mode: 'hard',
      }],
    )

    expect(selections).toEqual({ 'ai.agent_run': 10 })
  })

  it('defaults new or stale selections to the current published release', () => {
    const agentRun = capability('ai.agent_run', 12)
    const releases = {
      'ai.agent_run': [
        version(9, agentRun.id, 'retired'),
        version(12, agentRun.id),
      ],
    }

    expect(initializeCapabilityReleaseSelections([agentRun], releases)).toEqual({
      'ai.agent_run': 12,
    })
    expect(initializeCapabilityReleaseSelections([agentRun], releases, [{
      key: 'ai.agent_run.release_version_id',
      value_type: 'integer',
      value_json: '9',
      enforcement_mode: 'hard',
    }])).toEqual({ 'ai.agent_run': 12 })
  })

  it('builds entitlements from the administrator selection', () => {
    const agentRun = capability('ai.agent_run', 12)
    expect(capabilityReleaseEntitlements(
      [agentRun],
      { 'ai.agent_run': 10 },
    )).toEqual([{
      key: 'ai.agent_run.release_version_id',
      value_type: 'integer',
      value_json: '10',
      enforcement_mode: 'hard',
    }])
  })
})
