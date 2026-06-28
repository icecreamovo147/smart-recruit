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

export interface AgentSkillCanvasEdge {
  id: string
  source: string
  target: string
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
  category?: string
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
  category?: string
  version?: string
  flow_json?: string
  nodes?: AgentSkillNode[]
}

export interface AgentSkillPreviewResult {
  skill_md: string
  validation: AgentSkillValidation
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
  category?: string
  current_version_id?: number
  trigger_keywords?: string[]
}
