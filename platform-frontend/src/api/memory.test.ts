import { beforeEach, describe, expect, it, vi } from 'vitest'

const request = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('./http', () => ({ default: request }))

import {
  createPlatformMemory,
  listPlatformMemories,
  revokePlatformMemory,
} from './memory'

describe('platform Memory API owner context', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('keeps tenant and HR owner fields on list, create, and revoke requests', async () => {
    request.get.mockResolvedValueOnce({ list: [], total: 0 })
    request.post.mockResolvedValue({})

    await listPlatformMemories({
      tenant_id: 7,
      owner_role: 2,
      owner_id: 41,
      page: 1,
      page_size: 50,
    })
    await createPlatformMemory({
      tenant_id: 7,
      owner_role: 2,
      owner_id: 41,
      scope_type: 'hr',
      memory_type: 'preference',
      content: 'Prefer evidence-based screening.',
    })
    await revokePlatformMemory(9, {
      tenant_id: 7,
      owner_role: 2,
      owner_id: 41,
      revoke_reason: 'platform_correction',
    })

    expect(request.get).toHaveBeenCalledWith('/api/v1/platform/ai/memories', {
      params: {
        tenant_id: 7,
        owner_role: 2,
        owner_id: 41,
        page: 1,
        page_size: 50,
      },
    })
    expect(request.post).toHaveBeenNthCalledWith(1, '/api/v1/platform/ai/memories', {
      tenant_id: 7,
      owner_role: 2,
      owner_id: 41,
      scope_type: 'hr',
      memory_type: 'preference',
      content: 'Prefer evidence-based screening.',
    })
    expect(request.post).toHaveBeenNthCalledWith(2, '/api/v1/platform/ai/memories/9/revoke', {
      tenant_id: 7,
      owner_role: 2,
      owner_id: 41,
      revoke_reason: 'platform_correction',
    })
  })
})
