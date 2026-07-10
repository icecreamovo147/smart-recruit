export interface ResumeParseRunInfo {
  id: number
  resume_id: number
  user_id: number
  agent_run_id: number
  status: string
  parser_version: string
  input_hash: string
  error_message: string
  started_at: string
  completed_at: string
  created_at: string
  updated_at: string
}

export interface ResumeProfileInfo {
  id: number
  resume_id: number
  user_id: number
  parse_run_id: number
  version: number
  is_current: number
  full_name: string
  email: string
  phone: string
  location: string
  headline: string
  summary: string
  total_experience_years: number
  highest_degree: string
  raw_json: string
  created_at: string
  updated_at: string
}

export interface ResumeEducationInfo {
  id: number
  school: string
  degree: string
  major: string
  start_date: string
  end_date: string
  description: string
  sort_order: number
}

export interface ResumeExperienceInfo {
  id: number
  company: string
  title: string
  location: string
  start_date: string
  end_date: string
  is_current: number
  description: string
  achievements_json: string
  sort_order: number
}

export interface ResumeProjectInfo {
  id: number
  name: string
  role: string
  start_date: string
  end_date: string
  description: string
  technologies_json: string
  highlights_json: string
  sort_order: number
}

export interface ResumeSkillInfo {
  id: number
  name: string
  category: string
  level: string
  years: number
  evidence: string
  sort_order: number
}

export interface ResumeProfileSnapshotInfo {
  parse_run?: ResumeParseRunInfo
  profile?: ResumeProfileInfo
  educations: ResumeEducationInfo[]
  experiences: ResumeExperienceInfo[]
  projects: ResumeProjectInfo[]
  skills: ResumeSkillInfo[]
}

export interface CandidateMatchEvaluationInfo {
  id: number
  application_id: number
  job_id: number
  candidate_user_id: number
  resume_profile_id: number
  agent_run_id: number
  evaluation_version: number
  is_latest: number
  overall_score: number
  recommendation: string
  summary: string
  strengths_json: string
  risks_json: string
  missing_requirements_json: string
  score_breakdown_json: string
  model_name: string
  evaluated_at: string
  created_at: string
  updated_at: string
  dimensions_json: string
}

export interface CandidateMatchEvidenceInfo {
  id: number
  evidence_type: string
  dimension: string
  source_table: string
  source_id: number
  snippet: string
  weight: number
  score_impact: number
  metadata_json: string
  created_at: string
}

export interface CandidateMatchEvaluationSnapshotInfo {
  evaluation?: CandidateMatchEvaluationInfo
  evidence: CandidateMatchEvidenceInfo[]
}

export interface CandidateComparisonItem {
  application_id: number
  candidate_user_id: number
  candidate_name: string
  resume_id: number
  evaluation_id: number
  evaluation_version: number
  overall_score: number
  recommendation: string
  summary: string
  evaluated_at: string
  has_evaluation: boolean
}
