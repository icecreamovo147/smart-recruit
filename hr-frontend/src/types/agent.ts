// ── Agent Configuration ─────────────────────────────────────────────────

export interface AgentToolBindingInfo {
  id: number
  agent_id: number
  tool_name: string
  is_enabled: boolean
  created_at: string
}

export interface AgentConfigInfo {
  id: number
  name: string
  display_name: string
  description: string
  agent_type: string        // hr_recruiting_agent / candidate_assistant / custom
  model_id: number
  prompt_template_id: number
  instruction: string
  max_iterations: number
  temperature_override: number
  is_default: boolean
  is_enabled: boolean
  created_at: string
  updated_at: string
  // Joined fields (read-only)
  model_name: string
  prompt_template_name: string
  tool_bindings: AgentToolBindingInfo[]
}

// ── Payloads ─────────────────────────────────────────────────────────────

export interface CreateAgentPayload {
  name: string
  display_name: string
  description?: string
  agent_type: string
  model_id?: number
  prompt_template_id?: number
  instruction?: string
  max_iterations?: number
  temperature_override?: number
  temperature_override_set?: boolean
  is_default?: boolean
  tool_names?: string[]
}

export interface UpdateAgentPayload {
  name?: string
  display_name?: string
  description?: string
  agent_type?: string
  model_id?: number
  model_id_set?: boolean
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
}
