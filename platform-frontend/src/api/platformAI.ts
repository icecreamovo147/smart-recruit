import http from './http'

export interface PlatformAICapability {
  id: number
  capability_key: string
  audience: 'tenant_hr' | 'candidate'
  name: string
  description: string
  status: string
  current_published_version_id: number
}

export interface PlatformAICapabilityVersion {
  id: number
  capability_id: number
  version: number
  status: 'draft' | 'published' | 'retired'
  snapshot_json: string
  snapshot_hash: string
  change_note: string
  published_at: string
}

export const listPlatformAICapabilities = () =>
  http.get<{ list: PlatformAICapability[] }>('/api/v1/platform/ai/capabilities')

export const listPlatformAICapabilityVersions = (capabilityId: number) =>
  http.get<{ list: PlatformAICapabilityVersion[] }>(`/api/v1/platform/ai/capabilities/${capabilityId}/versions`)

export const createPlatformAICapabilityDraft = (capabilityId: number, payload: { snapshot_json: string; change_note: string }) =>
  http.post<{ version: PlatformAICapabilityVersion }>(`/api/v1/platform/ai/capabilities/${capabilityId}/versions`, payload)

export const updatePlatformAICapabilityDraft = (versionId: number, payload: { snapshot_json: string; change_note: string }) =>
  http.put<{ version: PlatformAICapabilityVersion }>(`/api/v1/platform/ai/capability-versions/${versionId}`, payload)

export const deletePlatformAICapabilityDraft = (versionId: number) =>
  http.delete(`/api/v1/platform/ai/capability-versions/${versionId}`)

export const publishPlatformAICapabilityVersion = (versionId: number) =>
  http.post<{ version: PlatformAICapabilityVersion }>(`/api/v1/platform/ai/capability-versions/${versionId}/publish`)
