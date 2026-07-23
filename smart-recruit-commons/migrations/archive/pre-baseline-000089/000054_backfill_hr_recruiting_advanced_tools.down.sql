DELETE acb
FROM agent_capability_bindings acb
JOIN agent_configs a ON a.id = acb.agent_id
WHERE a.agent_type = 'hr_recruiting_agent'
  AND a.is_default = 1
  AND acb.capability_source = 'builtin'
  AND acb.capability_key IN (
    'parse_resume_profile',
    'get_resume_profile',
    'evaluate_candidate_match',
    'get_candidate_match_evaluation',
    'compare_candidates_for_job'
  );

DELETE atb
FROM agent_tool_bindings atb
JOIN agent_configs a ON a.id = atb.agent_id
WHERE a.agent_type = 'hr_recruiting_agent'
  AND a.is_default = 1
  AND atb.tool_name IN (
    'parse_resume_profile',
    'get_resume_profile',
    'evaluate_candidate_match',
    'get_candidate_match_evaluation',
    'compare_candidates_for_job'
  );
