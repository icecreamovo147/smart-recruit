export interface SkillInfo {
  id: number
  name: string
  display_name: string
  description: string
  source_type: 'local' | 'git' | 'http' | 'mcp' | 'builtin' | string
  source_uri: string
  current_version_id: number
  is_enabled: boolean
  created_at: string
  updated_at: string
}

export interface SkillVersionInfo {
  id: number
  skill_id: number
  version: string
  manifest_json: string
  instruction: string
  input_schema_json: string
  output_schema_json: string
  runtime_type: 'prompt' | 'tool' | 'workflow' | 'http' | string
  created_at: string
}

export interface SkillToolInfo {
  id: number
  skill_version_id: number
  tool_name: string
  description: string
  input_schema_json: string
  runtime_config_json: string
  is_enabled: boolean
  capability_key: string
  runtime_tool_name: string
  created_at: string
  updated_at: string
}

export interface CreateSkillPayload {
  name: string
  display_name?: string
  description?: string
  source_type?: string
  source_uri?: string
  is_enabled?: boolean
  is_enabled_set?: boolean
}

export interface UpdateSkillPayload {
  display_name?: string
  description?: string
  source_type?: string
  source_uri?: string
  is_enabled?: boolean
  is_enabled_set?: boolean
}

export interface CreateSkillVersionPayload {
  manifest_json: string
  activate?: boolean
}

export interface UpdateSkillToolPayload {
  description?: string
  runtime_config_json?: string
  is_enabled?: boolean
  is_enabled_set?: boolean
}
