import http from './http'
import type { PlatformRequestConfig } from './http'

export const PLATFORM_AI_CAPABILITY_SCHEMA_VERSION = 2 as const
export const PLATFORM_AI_SKILL_POLICY_VERSION = 'skill-package-v2.1' as const
export const PLATFORM_AI_MAX_SKILL_TOKENS = 3000
export const PLATFORM_AI_MAX_INPUT_RATIO = 0.15
export const PLATFORM_AI_MAX_SKILLS = 2 as const

export interface PlatformAIModelPolicy {
  allowed_llm_model_ids: number[]
  default_llm_model_id: number
  allowed_embedding_model_ids: number[]
  default_embedding_model_id: number
}

export interface PlatformAIConfigurationRefs {
  agent_ids: number[]
  prompt_template_ids: number[]
  agent_skill_version_ids: number[]
  mcp_policy_ids: number[]
}

export interface PlatformAISkillRuntimePolicy {
  policy_version: typeof PLATFORM_AI_SKILL_POLICY_VERSION
  max_skill_tokens: number
  max_input_ratio: number
  max_skills: typeof PLATFORM_AI_MAX_SKILLS
  evaluation_suite_hash: string
  evaluation_result_hash: string
}

export interface PlatformAICapabilitySnapshotV2 {
  schema_version: typeof PLATFORM_AI_CAPABILITY_SCHEMA_VERSION
  capability_key: string
  audience: PlatformAICapability['audience']
  model_policy: PlatformAIModelPolicy
  configuration_refs: PlatformAIConfigurationRefs
  skill_runtime_policy: PlatformAISkillRuntimePolicy
}

export interface PlatformAICapabilityDraftPayload {
  snapshot_json: string
  change_note: string
}

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

export const createPlatformAICapabilityDraft = (
  capabilityId: number,
  payload: PlatformAICapabilityDraftPayload,
  config?: PlatformRequestConfig,
) => http.post<{ version: PlatformAICapabilityVersion }>(
  `/api/v1/platform/ai/capabilities/${capabilityId}/versions`,
  payload,
  config,
)

export const updatePlatformAICapabilityDraft = (
  versionId: number,
  payload: PlatformAICapabilityDraftPayload,
  config?: PlatformRequestConfig,
) => http.put<{ version: PlatformAICapabilityVersion }>(
  `/api/v1/platform/ai/capability-versions/${versionId}`,
  payload,
  config,
)

export const deletePlatformAICapabilityDraft = (versionId: number) =>
  http.delete(`/api/v1/platform/ai/capability-versions/${versionId}`)

export const publishPlatformAICapabilityVersion = (versionId: number, config?: PlatformRequestConfig) =>
  http.post<{ version: PlatformAICapabilityVersion }>(
    `/api/v1/platform/ai/capability-versions/${versionId}/publish`,
    undefined,
    config,
  )
