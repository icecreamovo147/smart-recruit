import request from './request'
import type { CapabilityDetailResponse, CapabilityListResponse, CreateCapabilityFromTemplatePayload } from '@/types/capability'

export const listCapabilities = (): Promise<CapabilityListResponse> =>
  request.get('/api/v1/hr/capabilities')

export const getCapability = (id: number): Promise<CapabilityDetailResponse> =>
  request.get(`/api/v1/hr/capabilities/${id}`)

export const createCapabilityFromTemplate = (
  data: CreateCapabilityFromTemplatePayload,
): Promise<CapabilityDetailResponse> =>
  request.post('/api/v1/hr/capabilities/from-template', data)
