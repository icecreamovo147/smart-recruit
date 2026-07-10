export type AgentSkillNodeType =
  | 'trigger'
  | 'context'
  | 'instruction'
  | 'condition'
  | 'output'
  | 'constraint'

export const AGENT_SKILL_NODE_TYPES = [
  'trigger',
  'context',
  'instruction',
  'condition',
  'output',
  'constraint',
] as const satisfies readonly AgentSkillNodeType[]

export interface AgentSkillCanvasNodeData {
  title: string
  content: string
}

export interface AgentSkillCanvasPosition {
  x: number
  y: number
}

export type AgentSkillCanvasHandle = 'top' | 'right' | 'bottom' | 'left'

export interface AgentSkillCanvasEdge {
  id: string
  source: string
  target: string
  sourceHandle?: AgentSkillCanvasHandle
  targetHandle?: AgentSkillCanvasHandle
  curvature?: number
  label?: string
}

export interface AgentSkillCanvasViewport {
  x: number
  y: number
  zoom: number
}

export interface AgentSkillNodeTypeOption {
  type: AgentSkillNodeType
  label: string
  description: string
  placeholder?: string
}

export interface AgentSkillNode {
  id: string
  type: AgentSkillNodeType
  title: string
  content: string
  order?: number
}

export interface AgentSkillCanvasNode {
  id: string
  type: AgentSkillNodeType
  position: AgentSkillCanvasPosition
  data: AgentSkillCanvasNodeData
}

export interface AgentSkillCanvasFlow {
  format: 'canvas.v1'
  version: string
  type: 'agent-skill'
  nodes: AgentSkillCanvasNode[]
  edges: AgentSkillCanvasEdge[]
  viewport?: AgentSkillCanvasViewport
}

export interface AgentSkillInfo {
  id: number
  name: string
  display_name: string
  description: string
  agent_type?: string
  category?: string
  scenario?: string
  priority?: number
  risk_level?: string
  required_capabilities?: string[]
  output_schema?: string
  evaluation_criteria?: string[]
  semantic_tags?: string[]
  unavailable_capabilities?: string[]
  validation_warnings?: string[]
  current_version_id?: number
  is_enabled: boolean
  is_manual_invocable?: boolean
  trigger_keywords?: string[]
  node_schema?: AgentSkillNode[]
  flow_json?: string
  skill_md?: string
  created_at?: string
  updated_at?: string
}

export interface AgentSkillVersionInfo {
  id: number
  skill_id: number
  version: string
  flow_json: string
  skill_md: string
  frontmatter_json: string
  body_markdown: string
  change_note: string
  is_current?: boolean
  created_at?: string
}

export interface AgentSkillDetail extends AgentSkillInfo {
  current_version?: AgentSkillVersionInfo
}

export interface AgentSkillDetailResponse {
  skill: AgentSkillDetail
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
  name: string
  display_name: string
  description?: string
  agent_type?: string
  category?: string
  scenario?: string
  priority?: number
  risk_level?: string
  required_capabilities?: string[]
  output_schema?: string
  evaluation_criteria?: string[]
  semantic_tags?: string[]
  version?: string
  flow_json?: string
  nodes?: AgentSkillNode[]
}

export interface AgentSkillPreviewResult {
  skill_md: string
  frontmatter_json: string
  body_markdown: string
}

export interface AgentSkillValidation {
  valid: boolean
  errors: string[]
  warnings: string[]
}

export interface CreateAgentSkillPayload extends AgentSkillPreviewPayload {
  is_enabled?: boolean
  is_enabled_set?: boolean
  is_manual_invocable?: boolean
  is_manual_invocable_set?: boolean
  trigger_keywords?: string[]
  change_note?: string
  activate?: boolean
  skill_md?: string
}

export interface UpdateAgentSkillPayload {
  display_name?: string
  display_name_set?: boolean
  description?: string
  description_set?: boolean
  agent_type?: string
  agent_type_set?: boolean
  category?: string
  category_set?: boolean
  scenario?: string
  scenario_set?: boolean
  priority?: number
  priority_set?: boolean
  risk_level?: string
  risk_level_set?: boolean
  required_capabilities?: string[]
  required_capabilities_set?: boolean
  output_schema?: string
  output_schema_set?: boolean
  evaluation_criteria?: string[]
  evaluation_criteria_set?: boolean
  semantic_tags?: string[]
  semantic_tags_set?: boolean
  is_enabled?: boolean
  is_enabled_set?: boolean
  is_manual_invocable?: boolean
  is_manual_invocable_set?: boolean
  trigger_keywords?: string[]
  trigger_keywords_set?: boolean
}

export interface CreateAgentSkillVersionPayload {
  version: string
  flow_json?: string
  nodes?: AgentSkillNode[]
  skill_md?: string
  change_note?: string
  activate?: boolean
}

export interface UpdateAgentSkillStatusPayload {
  is_enabled: boolean
}

export interface AvailableAgentSkill {
  id: number
  name: string
  display_name: string
  description: string
  agent_type?: string
  category?: string
  scenario?: string
  priority?: number
  risk_level?: string
  required_capabilities?: string[]
  output_schema?: string
  semantic_tags?: string[]
  unavailable_capabilities?: string[]
  validation_warnings?: string[]
  current_version_id?: number
  trigger_keywords?: string[]
}

export interface SemanticSkillDebugItem {
  id: number
  name: string
  display_name: string
  category?: string
  scenario?: string
  priority?: number
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
  job_id?: number
  application_id?: number
  limit?: number
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
