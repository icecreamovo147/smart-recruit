// ── Agent Configuration ─────────────────────────────────────────────────

export interface AgentToolBindingInfo {
  id: number
  agent_id: number
  tool_name: string
  is_enabled: boolean
  created_at: string
}

export interface AgentCapabilityBindingInfo {
  id?: number
  agent_id?: number
  capability_source: 'builtin' | 'mcp' | 'skill'
  capability_key: string
  is_enabled?: boolean
  priority?: number
  policy_json?: string
  created_at?: string
  updated_at?: string
}

export interface CapabilityInfo {
  source: 'builtin' | 'mcp' | 'skill'
  key: string
  name: string
  display_name: string
  description: string
  mcp_server_id: number
  mcp_server_name: string
  is_available: boolean
  skill_id?: number
  skill_name?: string
  skill_version?: string
  runtime_type?: 'prompt' | 'tool' | 'workflow' | 'http' | string
}

export interface AgentConfigInfo {
  id: number
  name: string
  display_name: string
  description: string
  agent_type: string        // hr_recruiting_agent / candidate_assistant / custom
  prompt_template_id: number
  instruction: string
  max_iterations: number
  temperature_override: number
  is_default: boolean
  is_enabled: boolean
  created_at: string
  updated_at: string
  // Joined fields (read-only)
  prompt_template_name: string
  tool_bindings: AgentToolBindingInfo[]
  capability_bindings: AgentCapabilityBindingInfo[]
}

// ── Payloads ─────────────────────────────────────────────────────────────

export interface CreateAgentPayload {
  name: string
  display_name: string
  description?: string
  agent_type: string
  prompt_template_id?: number
  instruction?: string
  max_iterations?: number
  temperature_override?: number
  temperature_override_set?: boolean
  is_default?: boolean
  tool_names?: string[]
  capability_bindings?: AgentCapabilityBindingInfo[]
}

export interface UpdateAgentPayload {
  name?: string
  display_name?: string
  description?: string
  agent_type?: string
  prompt_template_id?: number
  prompt_template_id_set?: boolean
  instruction?: string
  max_iterations?: number
  max_iterations_set?: boolean
  temperature_override?: number
  temperature_override_set?: boolean
  is_default?: boolean
  is_default_set?: boolean
  is_enabled?: boolean
  is_enabled_set?: boolean
  tool_names?: string[]
  tool_names_set?: boolean
  capability_bindings?: AgentCapabilityBindingInfo[]
  capability_bindings_set?: boolean
}
