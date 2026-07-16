import { describe, expect, it } from 'vitest'
import type { AgentRunItem, AgentRunStepItem, ToolTraceItem } from '@/types/ai'
import {
  applyTraceFilters,
  buildLegacyVM,
  buildRunVM,
  buildTraceSessionVM,
  classifyStatus,
  formatJsonContent,
  formatToolTitle,
  nextTraceVisibleCount,
  paginateTraceItems,
  policyDecisionFromJson,
  resetTraceFilters,
} from './agentTraceViewModel'

function makeStep(partial: Partial<AgentRunStepItem> = {}): AgentRunStepItem {
  return {
    id: partial.id ?? 1,
    run_id: partial.run_id ?? 10,
    step_index: partial.step_index ?? 0,
    step_type: partial.step_type ?? 'tool',
    capability_source: partial.capability_source ?? '',
    capability_key: partial.capability_key ?? '',
    tool_name: partial.tool_name ?? 'search_candidates',
    input_json: partial.input_json ?? '{"query":"alice"}',
    output_json: partial.output_json ?? '{"count":1}',
    status: partial.status ?? 'succeeded',
    duration_ms: partial.duration_ms ?? 12,
    error_message: partial.error_message ?? '',
    started_at: partial.started_at ?? '2026-07-11T10:00:00Z',
    completed_at: partial.completed_at ?? '2026-07-11T10:00:01Z',
    created_at: partial.created_at ?? '2026-07-11T10:00:00Z',
  }
}

function makeRun(partial: Partial<AgentRunItem> = {}): AgentRunItem {
  const plan = partial.plan_json ?? JSON.stringify({
    agent: 'hr_recruiting_agent',
    runtime: 'adk',
    risk_flags: ['verify_candidate_identity'],
    decision: {
      intent: 'candidate_match_evaluation',
      requires_human_confirm: true,
      confirmation_reason: 'sensitive action',
    },
    recruiting_plan: {
      intent: 'candidate_match_evaluation',
      risk_checks: ['verify_candidate_identity'],
      confirmation_requirement: { required: true, reason: 'sensitive action' },
    },
  })

  return {
    id: partial.id ?? 10,
    session_id: partial.session_id ?? 1,
    message_id: partial.message_id ?? 2,
    history_id: partial.history_id ?? 3,
    hr_id: partial.hr_id ?? 4,
    agent_type: partial.agent_type ?? 'hr_recruiting_agent',
    agent_id: partial.agent_id ?? 5,
    agent_name: partial.agent_name ?? 'hr_recruiting_agent',
    model_id: partial.model_id ?? 6,
    model_name: partial.model_name ?? 'qwen-plus',
    status: partial.status ?? 'succeeded',
    plan_json: plan,
    final_answer: partial.final_answer ?? 'ok',
    error_type: partial.error_type ?? '',
    error_message: partial.error_message ?? '',
    started_at: partial.started_at ?? '2026-07-11T10:00:00Z',
    completed_at: partial.completed_at ?? '2026-07-11T10:00:05Z',
    created_at: partial.created_at ?? '2026-07-11T10:00:00Z',
    steps: partial.steps ?? [makeStep()],
  }
}

function makeTrace(partial: Partial<ToolTraceItem> = {}): ToolTraceItem {
  return {
    id: partial.id ?? 100,
    session_id: partial.session_id ?? 1,
    tool_name: partial.tool_name ?? 'get_job_detail',
    args_json: partial.args_json ?? '{"job_id":1}',
    result_content: partial.result_content ?? '{"title":"Engineer"}',
    duration_ms: partial.duration_ms ?? 20,
    error_msg: partial.error_msg ?? '',
    created_at: partial.created_at ?? '2026-07-11T09:00:00Z',
  }
}

describe('agentTraceViewModel classification helpers', () => {
  it('classifies failed, warning, success, and active statuses', () => {
    expect(classifyStatus('failed')).toBe('failed')
    expect(classifyStatus('error')).toBe('failed')
    expect(classifyStatus('partial')).toBe('warning')
    expect(classifyStatus('fallback')).toBe('warning')
    expect(classifyStatus('succeeded')).toBe('succeeded')
    expect(classifyStatus('success')).toBe('succeeded')
    expect(classifyStatus('running')).toBe('active')
    expect(classifyStatus('queued')).toBe('active')
  })

  it('formats tool titles with Chinese labels while keeping english names', () => {
    expect(formatToolTitle('search_candidates')).toBe('搜索候选人（search_candidates）')
    expect(formatToolTitle('unknown_custom_tool')).toBe('unknown_custom_tool')
    expect(formatToolTitle('')).toBe('')
  })

  it('parses MCP policy decisions and risk flags from step/output JSON', () => {
    const deny = policyDecisionFromJson(JSON.stringify({
      policy_decision: 'deny',
      policy_reason: 'tool blocked',
      policy_id: 9,
    }))
    expect(deny).toMatchObject({
      decision: 'deny',
      reason: 'tool blocked',
      policyId: 9,
      isIssue: true,
      severity: 'error',
    })

    const confirm = policyDecisionFromJson(JSON.stringify({
      policy_decision: 'confirmation_required',
      policy_reason: 'needs review',
    }))
    expect(confirm?.severity).toBe('warning')
    expect(confirm?.isIssue).toBe(true)
  })

  it('formats valid JSON and preserves invalid JSON as raw text', () => {
    const valid = formatJsonContent('{"a":1}')
    expect(valid.isValidJson).toBe(true)
    expect(valid.formatted).toContain('\n')
    expect(valid.formatted).toContain('"a": 1')
    expect(valid.nestedExpanded).toBe(false)

    const invalid = formatJsonContent('not-json {')
    expect(invalid.isValidJson).toBe(false)
    expect(invalid.formatted).toBe('not-json {')

    const empty = formatJsonContent('')
    expect(empty.isEmpty).toBe(true)
  })

  it('expands double-encoded JSON string fields for display without mutating raw', () => {
    const nestedJobs = {
      total: 3,
      page: 1,
      page_size: 10,
      jobs: [{ job_id: 3, title: '软件开发实习生——后台开发方向' }],
    }
    const raw = JSON.stringify({
      result: JSON.stringify(nestedJobs),
      tool_call_id: 'call_01_1d6x868aUDt89hSeUEa45789',
    })

    const meta = formatJsonContent(raw)
    expect(meta.isValidJson).toBe(true)
    expect(meta.nestedExpanded).toBe(true)
    expect(meta.raw).toBe(raw)
    // Nested object fields become real structure, not escaped string
    expect(meta.formatted).toContain('"total": 3')
    expect(meta.formatted).toContain('"job_id": 3')
    expect(meta.formatted).toContain('"tool_call_id": "call_01_1d6x868aUDt89hSeUEa45789"')
    expect(meta.formatted).not.toContain('\\"total\\"')

    // Plain non-JSON strings are left alone
    const plain = formatJsonContent(JSON.stringify({ note: 'hello {world}', count: 1 }))
    expect(plain.nestedExpanded).toBe(false)
    expect(plain.formatted).toContain('"hello {world}"')
  })
})

describe('agentTraceViewModel durable runs', () => {
  it('derives overview metrics from durable runs', () => {
    const failedStep = makeStep({
      id: 2,
      status: 'failed',
      tool_name: 'evaluate_candidate_match',
      error_message: 'timeout',
      output_json: JSON.stringify({
        policy_decision: 'rate_limited',
        policy_reason: 'too many calls',
      }),
    })
    const evidenceStep = makeStep({
      id: 3,
      step_type: 'tool',
      tool_name: 'get_resume_profile',
      output_json: JSON.stringify({ evidence: [{ id: 1 }, { id: 2 }] }),
    })
    const run = makeRun({
      status: 'partial',
      steps: [makeStep({ id: 1 }), failedStep, evidenceStep],
    })

    const session = buildTraceSessionVM([run], [])
    expect(session.overview.hasData).toBe(true)
    expect(session.overview.runCount).toBe(1)
    expect(session.overview.stepCount).toBe(3)
    expect(session.overview.toolStepCount).toBeGreaterThan(0)
    expect(session.overview.failureCount).toBeGreaterThanOrEqual(1)
    expect(session.overview.riskCount).toBe(1)
    expect(session.overview.policyIssueCount).toBeGreaterThanOrEqual(1)
    expect(session.overview.modelName).toBe('qwen-plus')
    expect(session.overview.intentLabel).toContain('匹配')
    expect(session.overview.confirmationRequired).toBe(true)
    expect(session.overview.durationMs).toBe(5000)

    const runVM = session.runs[0]
    expect(runVM.statusCategory).toBe('warning')
    expect(runVM.riskFlags).toContain('verify_candidate_identity')
    expect(runVM.steps.some((step) => step.isEvidence)).toBe(true)
    expect(runVM.steps.some((step) => step.statusCategory === 'failed')).toBe(true)
    expect(runVM.issues.some((issue) => issue.severity === 'error')).toBe(true)
    expect(runVM.issues.some((issue) => issue.label.includes('MCP'))).toBe(true)
    // Issue summary should show localized risk text, not raw English keys
    expect(runVM.issues.some((issue) => issue.detail === '核验候选人身份')).toBe(true)
    expect(runVM.issues.every((issue) => !String(issue.detail).includes('verify_candidate_identity'))).toBe(true)
  })

  it('uses runtime planner metadata when run model and labels are not denormalized', () => {
    const run = makeRun({
      model_id: 0,
      model_name: '',
      agent_type: 'hr_recruiting_agent',
      plan_json: JSON.stringify({
        runtime: 'native-hr-runtime',
        model: '默认模型',
        recruiting_plan: {
          intent: 'candidate_match_evaluation',
          risk_checks: ['verify_candidate_identity', 'cite_tool_returned_evidence'],
          confirmation_requirement: { required: false },
        },
        risk_flags: ['verify_candidate_identity', 'cite_tool_returned_evidence'],
        decision: {
          intent: 'candidate_match_evaluation',
          risk_flag_count: 2,
          runtime_warning: true,
          warning_count: 1,
          warning_messages: ['AI 响应较慢，请稍候...'],
        },
      }),
      steps: [],
    })

    const session = buildTraceSessionVM([run], [])

    expect(session.overview.modelName).toBe('默认模型')
    expect(session.overview.runtimeLabel).toBe('HR 招聘运行时')
    expect(session.overview.intentLabel).toBe('候选人匹配评估')
    expect(session.overview.riskCount).toBe(2)
    expect(session.overview.warningCount).toBe(1)
    expect(session.issues.some((issue) => issue.label === '运行告警')).toBe(true)
  })

  it('localizes known risk flags including do_not_claim_unavailable_tools', () => {
    const run = makeRun({
      plan_json: JSON.stringify({
        risk_flags: ['do_not_claim_unavailable_tools', 'no_builtin_recruiting_tools_available'],
        decision: { intent: 'unknown' },
        recruiting_plan: { intent: 'unknown' },
      }),
      steps: [],
    })
    const runVM = buildRunVM(run)
    const riskIssues = runVM.issues.filter((issue) => issue.label === '风险检查')
    expect(riskIssues.map((issue) => issue.detail)).toEqual([
      '不得声称使用了不可用工具',
      '内置招聘工具不可用',
    ])
  })

  it('does not mutate source run or step objects while building searchable text', () => {
    const step = makeStep({
      input_json: '{"secret":"***"}',
      output_json: '{"ok":true}',
    })
    const run = makeRun({ steps: [step] })
    const originalPlan = run.plan_json
    const originalInput = step.input_json

    const runVM = buildRunVM(run)
    expect(run.plan_json).toBe(originalPlan)
    expect(step.input_json).toBe(originalInput)
    expect(runVM.searchableText).toContain('search_candidates')
    expect(runVM.searchableText).toContain('***')
    expect(runVM.steps[0].searchableText).toContain('secret')
  })
})

describe('agentTraceViewModel legacy traces', () => {
  it('builds a useful view model for legacy-only data', () => {
    const traces = [
      makeTrace({
        id: 1,
        tool_name: 'search_jobs',
        result_content: JSON.stringify({
          policy_decision: 'deny',
          policy_reason: 'mcp deny',
          policy_id: 3,
        }),
      }),
      makeTrace({
        id: 2,
        tool_name: 'broken_tool',
        error_msg: 'network down',
        result_content: 'plain failure text',
      }),
      makeTrace({
        id: 3,
        args_json: 'not-json',
        result_content: '{"ok": true}',
      }),
    ]

    const session = buildTraceSessionVM([], traces)
    expect(session.overview.hasData).toBe(true)
    expect(session.overview.runCount).toBe(0)
    expect(session.overview.legacyTraceCount).toBe(3)
    expect(session.overview.failureCount).toBeGreaterThanOrEqual(2)
    expect(session.legacyTraces[0].typeCategory).toBe('legacy')
    expect(session.legacyTraces[0].policyDecision?.decision).toBe('deny')
    expect(session.legacyTraces[1].statusCategory).toBe('failed')
    expect(session.legacyTraces[2].argsDisplay.isValidJson).toBe(false)
    expect(session.legacyTraces[2].resultDisplay.isValidJson).toBe(true)
    expect(session.issues.length).toBeGreaterThan(0)
  })

  it('classifies policy warning on legacy result without mutating source', () => {
    const trace = makeTrace({
      result_content: JSON.stringify({
        policy_decision: 'confirmation_required',
        policy_reason: 'confirm me',
      }),
    })
    const original = trace.result_content
    const vm = buildLegacyVM(trace)
    expect(trace.result_content).toBe(original)
    expect(vm.statusCategory).toBe('warning')
    expect(vm.issues.some((issue) => issue.severity === 'warning')).toBe(true)
  })
})

describe('agentTraceViewModel frontend lazy pagination', () => {
  it('paginates without mutating source and reports truncation', () => {
    const source = Array.from({ length: 45 }, (_, i) => ({ id: i + 1 }))
    const snapshot = JSON.stringify(source)
    const page = paginateTraceItems(source, 20, 20)
    expect(page.items).toHaveLength(20)
    expect(page.total).toBe(45)
    expect(page.hasMore).toBe(true)
    expect(page.truncated).toBe(true)
    expect(JSON.stringify(source)).toBe(snapshot)
    expect(nextTraceVisibleCount(20, 45, 20)).toBe(40)
    expect(nextTraceVisibleCount(40, 45, 20)).toBe(45)
    const full = paginateTraceItems(source, 45, 20)
    expect(full.hasMore).toBe(false)
    expect(full.items).toHaveLength(45)
  })
})

describe('agentTraceViewModel filters', () => {
  it('filters by keyword, status, type, and issue-only without mutating source arrays', () => {
    const runs: AgentRunItem[] = [
      makeRun({
        id: 1,
        status: 'failed',
        error_message: 'boom',
        steps: [
          makeStep({
            id: 11,
            status: 'failed',
            tool_name: 'search_candidates',
            error_message: 'tool boom',
          }),
          makeStep({
            id: 12,
            step_type: 'memory',
            tool_name: '',
            capability_key: 'memory_recall',
            status: 'succeeded',
          }),
        ],
      }),
      makeRun({
        id: 2,
        status: 'succeeded',
        model_name: 'gpt-test',
        steps: [
          makeStep({
            id: 21,
            step_type: 'plan',
            tool_name: '',
            capability_key: 'planner',
            status: 'succeeded',
          }),
        ],
      }),
    ]
    const traces = [makeTrace({ id: 99, tool_name: 'legacy_only_tool' })]
    const sourceRunsSnapshot = JSON.stringify(runs)
    const sourceTracesSnapshot = JSON.stringify(traces)

    const session = buildTraceSessionVM(runs, traces)

    const keyword = applyTraceFilters(session, {
      keyword: 'search_candidates',
      statusFilter: 'all',
      typeFilter: 'all',
      issueOnly: false,
    })
    expect(keyword.isFilterEmpty).toBe(false)
    expect(keyword.runs.some((run) => run.steps.some((step) => step.title === '搜索候选人（search_candidates）'))).toBe(true)

    const failedOnly = applyTraceFilters(session, {
      keyword: '',
      statusFilter: 'failed',
      typeFilter: 'all',
      issueOnly: false,
    })
    expect(failedOnly.runs.every((run) => (
      run.statusCategory === 'failed'
      || run.steps.some((step) => step.statusCategory === 'failed')
    ))).toBe(true)

    const typeOnly = applyTraceFilters(session, {
      keyword: '',
      statusFilter: 'all',
      typeFilter: 'memory',
      issueOnly: false,
    })
    expect(typeOnly.runs.length).toBe(1)
    expect(typeOnly.runs[0].steps.every((step) => step.typeCategory === 'memory')).toBe(true)

    const legacyOnly = applyTraceFilters(session, {
      keyword: '',
      statusFilter: 'all',
      typeFilter: 'legacy',
      issueOnly: false,
    })
    expect(legacyOnly.runs.length).toBe(0)
    expect(legacyOnly.legacyTraces.length).toBe(1)

    const issueOnly = applyTraceFilters(session, {
      keyword: '',
      statusFilter: 'all',
      typeFilter: 'all',
      issueOnly: true,
    })
    expect(issueOnly.runs.every((run) => run.issues.length > 0)).toBe(true)

    const empty = applyTraceFilters(session, {
      keyword: 'definitely-not-present-xyz',
      statusFilter: 'all',
      typeFilter: 'all',
      issueOnly: false,
    })
    expect(empty.isFilterEmpty).toBe(true)
    expect(empty.hasActiveFilters).toBe(true)

    expect(JSON.stringify(runs)).toBe(sourceRunsSnapshot)
    expect(JSON.stringify(traces)).toBe(sourceTracesSnapshot)
    expect(resetTraceFilters()).toEqual({
      keyword: '',
      statusFilter: 'all',
      typeFilter: 'all',
      issueOnly: false,
    })
  })
})
