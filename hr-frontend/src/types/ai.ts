// ---- AI Chat Types ----

export interface ChatMessageSkill {
  id?: string | number
  name: string
  command?: string
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  created_at?: string
  model_id?: number
  model_name?: string
  skill?: ChatMessageSkill
  skills?: ChatMessageSkill[]
  skill_id?: string | number
  skill_name?: string
  skill_command?: string
  skillId?: string | number
  skillName?: string
  skillCommand?: string
  agent_skill_ids?: number[]
  agent_skill_names?: string[]
  agentSkillIds?: number[]
  agentSkillNames?: string[]
  pending?: boolean
  failed?: boolean
  waitingText?: string
  process_content?: string
  processContent?: string
  suggested_questions?: string[] | string
  suggestedQuestions?: string[] | string
  context_usage?: ContextUsageInfo
  contextUsage?: ContextUsageInfo
  candidateOptions?: CandidateOption[]
}

export interface CandidateOption {
  application_id: number
  candidate_name: string
  job_title: string
  masked_phone: string
  round_no: number
  status_text?: string
  is_current?: number
  applied_at?: string
}

export interface Session {
  id: number
  title: string
  application_id: number
  created_at?: string
  updated_at?: string
  latest_context_usage?: ContextUsageInfo
  latestContextUsage?: ContextUsageInfo
  selected_model_id?: number
  selectedModelId?: number
}

export interface ContextUsageBreakdown {
  system_prompt_tokens: number
  recent_message_tokens: number
  summary_tokens: number
  memory_tokens: number
  current_message_tokens: number
  skill_tokens: number
  tool_result_tokens: number
  tool_schema_tokens?: number
  protocol_overhead_tokens?: number
}

export interface ContextUsageInfo {
  model_id: number
  model_name: string
  requested_model_id?: number
  effective_model_id?: number
  model_fallback_reason?: string
  capability_version_id?: number
  capability_snapshot_hash?: string
  context_window_tokens: number
  max_output_tokens: number
  prompt_tokens_estimated: number
  prompt_tokens_actual: number
  completion_tokens_actual: number
  total_tokens_actual: number
  remaining_tokens_estimated: number
  usage_ratio: number
  estimated: boolean
  source: string
  stage: string
  input_budget_tokens?: number
  safety_margin_tokens?: number
  budget_usage_ratio?: number
  budget_status?: string
  included_message_count?: number
  omitted_message_count?: number
  summary_applied?: boolean
  memory_applied?: boolean
  breakdown?: ContextUsageBreakdown
}

export interface StreamPayload {
  delta?: string
  done?: boolean
  code?: number
  msg?: string
  request_id?: string
  session_id?: number
  candidate_options?: string
  action?: string
  action_status?: number
  application_id?: number
  candidate_name?: string
  job_title?: string
  status?: number
  created_at?: string
  // Phase 4: streaming UX status events
  event_type?: string // thinking | process_delta | process_clear | tool_calling | tool_done | generating | timeout_warning | partial_done | done | error | model_info | agent_run_started | model_selected | planning | capability_selected | fallback | agent_run_done | context_usage
  event_message?: string
  error_type?: string
  tool_name?: string
  model_name?: string
  agent_skill_ids?: number[]
  agent_skill_selection?: AgentSkillSelectionPayload
  context_usage?: ContextUsageInfo
  suggested_questions?: string[]
}

export interface AgentSkillSelectionCandidate {
  id: number
  name: string
  display_name: string
  reason: string
  score: number
  priority: number
  category: string
  scenario: string
  risk_level: string
  recommended: boolean
  vector_score: number
  lexical_score: number
  metadata_score: number
  relevance_score: number
  business_boost: number
  final_rank_score: number
  relevance_mode: string
  pool_rank: number
  ranking_confidence: string
}

export interface AgentSkillSelectionPayload {
  required: boolean
  reason: string
  candidates: AgentSkillSelectionCandidate[]
  recommended_agent_skill_ids: number[]
  user_message_id?: number
}

export interface StreamHandlers {
  onDelta?: (delta: string, payload: StreamPayload) => void
  onDone?: (payload: StreamPayload) => void
  onStatus?: (eventType: string, eventMessage: string, payload: StreamPayload) => void
  onError?: (errorType: string, errorMessage: string, payload: StreamPayload) => void
}

export interface StreamErrorPayload {
  code: number
  msg: string
  request_id?: string
}

export interface ChatSessionListItem {
  session_id: number
  title: string
  application_id: number
  created_at: string
  updated_at: string
  latest_context_usage?: ContextUsageInfo
  latestContextUsage?: ContextUsageInfo
  selected_model_id?: number
  selectedModelId?: number
}

// ---- Agent Tool Trace Types ----

export interface ToolTraceItem {
  id: number
  session_id: number
  tool_name: string
  args_json: string
  result_content: string
  duration_ms: number
  error_msg: string
  created_at: string
}

export interface AgentRunStepItem {
  id: number
  run_id: number
  step_index: number
  step_type: string
  capability_source: string
  capability_key: string
  tool_name: string
  input_json: string
  output_json: string
  status: string
  duration_ms: number
  error_message: string
  started_at: string
  completed_at: string
  created_at: string
}

export interface AgentRunConfirmationRequirement {
  required?: boolean
  reason?: string
}

export interface AgentRunRecruitingPlan {
  intent?: string
  required_tools?: string[]
  required_data?: string[]
  selected_skills?: string[]
  selected_memories?: string[]
  output_schema?: Record<string, unknown>
  confirmation_requirement?: AgentRunConfirmationRequirement
  risk_checks?: string[]
  [key: string]: unknown
}

export interface AgentRunDecision {
  intent?: string
  confirmation_required?: boolean
  confirmation_reason?: string
  required_tool_count?: number
  required_data_count?: number
  risk_flag_count?: number
  unavailable_tool_risk?: boolean
  requires_human_confirm?: boolean
  requires_evidence_citation?: boolean
  status?: string
  partial?: boolean
  failed?: boolean
  risk_flag_hit?: boolean
  [key: string]: unknown
}

export interface AgentRunPlanJSON {
  agent?: string
  agent_type?: string
  model?: string
  model_id?: number
  capabilities?: unknown
  user_message_summary?: string
  application_bound?: boolean
  application_id?: number
  runtime?: string
  recruiting_plan?: AgentRunRecruitingPlan
  planner_json?: string | AgentRunRecruitingPlan
  risk_flags?: string[]
  decision?: AgentRunDecision
  selected_agent_skill_ids?: number[]
  selected_memory_ids?: number[]
  status?: string
  partial?: boolean
  failed?: boolean
  risk_flag_hit?: boolean
  [key: string]: unknown
}

export interface AgentRunItem {
  id: number
  session_id: number
  message_id: number
  history_id: number
  hr_id: number
  agent_type: string
  agent_id: number
  agent_name: string
  model_id: number
  model_name: string
  status: string
  plan_json: string
  final_answer: string
  error_type: string
  error_message: string
  started_at: string
  completed_at: string
  created_at: string
  steps: AgentRunStepItem[]
}
