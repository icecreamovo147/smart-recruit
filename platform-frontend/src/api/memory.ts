import request from './http'
import type {
  CreateMemoryPayload,
  MemoryListParams,
  MemoryListResult,
  MemoryInfo,
  RevokeMemoryPayload,
} from '@shared/types/memory'

export const listPlatformMemories = async (
  params: MemoryListParams,
): Promise<MemoryListResult> => {
  const res: any = await request.get('/api/v1/platform/ai/memories', {
    params: {
      ...(params.tenant_id ? { tenant_id: params.tenant_id } : {}),
      owner_role: params.owner_role,
      owner_id: params.owner_id,
      page: params.page ?? 1,
      page_size: params.page_size ?? 50,
      ...(params.scope_type ? { scope_type: params.scope_type } : {}),
      ...(params.scope_id ? { scope_id: params.scope_id } : {}),
      ...(params.memory_type ? { memory_type: params.memory_type } : {}),
      ...(params.status ? { status: params.status } : {}),
    },
  })
  return res
}

export const createPlatformMemory = async (
  data: CreateMemoryPayload,
): Promise<{ memory: MemoryInfo }> => {
  const res: any = await request.post('/api/v1/platform/ai/memories', data)
  return res
}

export const revokePlatformMemory = async (
  id: number,
  data: RevokeMemoryPayload,
): Promise<{ memory?: MemoryInfo }> => {
  const res: any = await request.post(`/api/v1/platform/ai/memories/${id}/revoke`, data)
  return res
}
