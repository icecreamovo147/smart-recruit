-- Migration: 000054_backfill_hr_recruiting_advanced_tools
-- Description: Ensure the default HR recruiting Agent has the complete native HR tool catalog.

INSERT IGNORE INTO agent_tool_bindings (agent_id, tool_name, is_enabled)
SELECT a.id, tools.tool_name, 1
FROM agent_configs a
JOIN (
  SELECT 'query_total_applications' AS tool_name UNION ALL
  SELECT 'query_today_applications' UNION ALL
  SELECT 'get_job_heat_ranking' UNION ALL
  SELECT 'search_candidates' UNION ALL
  SELECT 'get_job_detail' UNION ALL
  SELECT 'search_jobs' UNION ALL
  SELECT 'get_candidate_detail' UNION ALL
  SELECT 'propose_application_status_update' UNION ALL
  SELECT 'list_all_applications' UNION ALL
  SELECT 'list_applications_by_job' UNION ALL
  SELECT 'list_applications_by_status' UNION ALL
  SELECT 'get_application_status_summary' UNION ALL
  SELECT 'get_application_trend' UNION ALL
  SELECT 'get_job_list' UNION ALL
  SELECT 'parse_resume_profile' UNION ALL
  SELECT 'get_resume_profile' UNION ALL
  SELECT 'evaluate_candidate_match' UNION ALL
  SELECT 'get_candidate_match_evaluation' UNION ALL
  SELECT 'compare_candidates_for_job'
) tools
WHERE a.agent_type = 'hr_recruiting_agent'
  AND a.is_default = 1
  AND a.is_enabled = 1;

INSERT IGNORE INTO agent_capability_bindings (agent_id, capability_source, capability_key, is_enabled, priority, policy_json)
SELECT a.id, 'builtin', tools.tool_name, 1, 0, NULL
FROM agent_configs a
JOIN (
  SELECT 'query_total_applications' AS tool_name UNION ALL
  SELECT 'query_today_applications' UNION ALL
  SELECT 'get_job_heat_ranking' UNION ALL
  SELECT 'search_candidates' UNION ALL
  SELECT 'get_job_detail' UNION ALL
  SELECT 'search_jobs' UNION ALL
  SELECT 'get_candidate_detail' UNION ALL
  SELECT 'propose_application_status_update' UNION ALL
  SELECT 'list_all_applications' UNION ALL
  SELECT 'list_applications_by_job' UNION ALL
  SELECT 'list_applications_by_status' UNION ALL
  SELECT 'get_application_status_summary' UNION ALL
  SELECT 'get_application_trend' UNION ALL
  SELECT 'get_job_list' UNION ALL
  SELECT 'parse_resume_profile' UNION ALL
  SELECT 'get_resume_profile' UNION ALL
  SELECT 'evaluate_candidate_match' UNION ALL
  SELECT 'get_candidate_match_evaluation' UNION ALL
  SELECT 'compare_candidates_for_job'
) tools
WHERE a.agent_type = 'hr_recruiting_agent'
  AND a.is_default = 1
  AND a.is_enabled = 1;
