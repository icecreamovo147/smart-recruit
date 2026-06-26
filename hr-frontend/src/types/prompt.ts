// ── Prompt Template ─────────────────────────────────────────────────────────

export interface PromptTemplate {
  id: number
  name: string
  content: string
  variables_json: string     // JSON array string, e.g. '["name","job_title"]'
  version: number
  is_active: boolean
  agent_type: string         // hr_agent / candidate_assistant
  prompt_role: string        // system / user
  created_by: number
  updated_by: number
  created_at: string
  updated_at: string
}

export interface CreatePromptPayload {
  name: string
  content: string
  variables_json?: string
  agent_type: string
  prompt_role: string
  created_by: number
}

export interface UpdatePromptPayload {
  name?: string
  content?: string
  variables_json?: string
  is_active?: boolean
  is_active_set?: boolean
  updated_by: number
  change_note?: string
}

// ── Version History ─────────────────────────────────────────────────────────

export interface PromptVersion {
  id: number
  template_id: number
  version: number
  content: string
  changed_by: number
  change_note: string
  created_at: string
}

// ── Rollback ────────────────────────────────────────────────────────────────

export interface RollbackPayload {
  version: number
  updated_by: number
  change_note?: string
}
