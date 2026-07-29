export type AgentSkillRiskLevel = 'low' | 'medium' | 'high' | 'critical'
export type AgentSkillActivationPolicy = 'auto' | 'confirm' | 'manual_only'
export type AgentSkillCompositionRole = 'primary' | 'supporting'
export type AgentSkillOutputMode = 'none' | 'advisory' | 'strict'

export interface AgentSkillComposition {
  role: AgentSkillCompositionRole
}

export interface AgentSkillOutputContract {
  mode: AgentSkillOutputMode
  schema_id: string
  schema_json: string
}

export interface AgentSkillManifestDraft {
  schema_version: 2
  skill_name: string
  display_name: string
  description?: string
  agent_type: string
  category?: string
  scenario?: string
  priority?: number
  risk: AgentSkillRiskLevel
  /** Derived from risk by the server; clients may echo the read-only value. */
  activation_policy?: AgentSkillActivationPolicy
  required_capabilities?: string[]
  trigger_keywords?: string[]
  semantic_tags?: string[]
  composition: AgentSkillComposition
  output_contract?: AgentSkillOutputContract
  evaluation_criteria?: string[]
}

export interface AgentSkillManifest extends AgentSkillManifestDraft {
  description: string
  category: string
  scenario: string
  priority: number
  activation_policy: AgentSkillActivationPolicy
  required_capabilities: string[]
  trigger_keywords: string[]
  semantic_tags: string[]
  output_contract: AgentSkillOutputContract
  evaluation_criteria: string[]
}

export interface AgentSkillSectionDraft {
  section_key: string
  title: string
  description?: string
  content_markdown: string
  trigger_terms?: string[]
  semantic_tags?: string[]
  planner_intents?: string[]
  priority?: number
  ordinal?: number
}

export interface AgentSkillSectionInfo {
  id: number
  section_key: string
  title: string
  description: string
  content_markdown: string
  trigger_terms: string[]
  semantic_tags: string[]
  planner_intents: string[]
  priority: number
  ordinal: number
  estimated_tokens: number
  content_hash: string
}

export interface AgentSkillPackageDraft {
  manifest: AgentSkillManifestDraft
  core_markdown: string
  sections?: AgentSkillSectionDraft[]
  /** Editor-only state. Runtime never interprets this value. */
  authoring_json?: string
}

export interface AgentSkillPackageInfo {
  manifest: AgentSkillManifest
  core_markdown: string
  sections: AgentSkillSectionInfo[]
  compiled_markdown: string
  authoring_json: string
  compiled_hash: string
  core_estimated_tokens: number
  package_estimated_tokens: number
}

export interface AgentSkillVersionSummary {
  version_id: number
  version: string
  compiled_hash: string
  agent_type: string
  category: string
  scenario: string
  priority: number
  risk: AgentSkillRiskLevel
  activation_policy: AgentSkillActivationPolicy
  composition_role: AgentSkillCompositionRole
  core_estimated_tokens: number
  package_estimated_tokens: number
}

export interface AgentSkillInfo {
  id: number
  name: string
  display_name: string
  description: string
  current_version_id: number
  is_enabled: boolean
  is_manual_invocable: boolean
  created_at: string
  updated_at: string
  current_version: AgentSkillVersionSummary | null
}

export interface AgentSkillVersionInfo {
  id: number
  skill_id: number
  version: string
  change_note: string
  created_at: string
  package: AgentSkillPackageInfo
}

export interface AgentSkillDetailResponse {
  skill: AgentSkillInfo
}

export interface AgentSkillResponse {
  skill: AgentSkillInfo
}

export interface AgentSkillVersionResponse {
  version: AgentSkillVersionInfo
}

export interface AgentSkillVersionListResponse {
  list: AgentSkillVersionInfo[]
}

export interface AgentSkillListParams {
  page?: number
  page_size?: number
  keyword?: string
  enabled_only?: boolean
}

export interface AgentSkillPreviewPayload {
  package: AgentSkillPackageDraft
}

export interface AgentSkillPreviewResult {
  package: AgentSkillPackageInfo
}

export interface CreateAgentSkillPayload {
  version: string
  package: AgentSkillPackageDraft
  is_enabled?: boolean
  is_enabled_set?: boolean
  is_manual_invocable?: boolean
  is_manual_invocable_set?: boolean
  change_note?: string
  activate?: boolean
}

export interface UpdateAgentSkillPayload {
  display_name?: string
  display_name_set?: boolean
  description?: string
  description_set?: boolean
  is_enabled?: boolean
  is_enabled_set?: boolean
  is_manual_invocable?: boolean
  is_manual_invocable_set?: boolean
}

export interface CreateAgentSkillVersionPayload {
  version: string
  package: AgentSkillPackageDraft
  change_note?: string
  activate?: boolean
}

export interface UpdateAgentSkillStatusPayload {
  is_enabled: boolean
}

export type AvailableAgentSkill = AgentSkillInfo

export interface SemanticSkillDebugItem {
  skill_id: number
  version_id: number
  version: string
  compiled_hash: string
  name: string
  display_name: string
  category?: string
  scenario?: string
  priority?: number
  risk?: AgentSkillRiskLevel
  composition_role?: AgentSkillCompositionRole
  score: number
  reason: string
  semantic_tags?: string[]
  vector_score?: number
  lexical_score?: number
  metadata_score?: number
  relevance_score?: number
  business_boost?: number
  final_rank_score?: number
  relevance_mode?: string
  pool_rank?: number
}

export interface SemanticMemoryDebugItem {
  id: number
  scope_type: string
  scope_id: number
  memory_type: string
  content: string
  source: string
  confidence: number
  importance: number
  score: number
  reason: string
  created_at?: string
  vector_score?: number
  lexical_score?: number
  metadata_score?: number
  relevance_score?: number
  business_boost?: number
  final_rank_score?: number
  relevance_mode?: string
  pool_rank?: number
}

export interface SemanticRetrievalDebugParams {
  query: string
  agent_type?: string
  tenant_id?: number
  job_id?: number
  application_id?: number
  limit?: number
  owner_role?: number
  owner_id?: number
}

export interface SemanticRetrievalDebugResult {
  embedding_available: boolean
  fallback_reason?: string
  embedding_provider?: string
  embedding_model?: string
  embedding_dim?: number
  candidate_count?: number
  query_embedding_latency_ms?: number
  skills: SemanticSkillDebugItem[]
  memories: SemanticMemoryDebugItem[]
  skill_pool_confidence?: string
  memory_pool_confidence?: string
}
