#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptPath = fileURLToPath(import.meta.url)
const repositoryRoot = path.resolve(path.dirname(scriptPath), '..')

const rules = [
  {
    id: 'legacy-agent-skill-id-selection',
    pattern: /\b(?:agent_skill_ids|agentSkillIds|recommended_agent_skill_ids|recommendedAgentSkillIds)\b/,
  },
  {
    id: 'legacy-agent-skill-boolean-confirmation',
    pattern: /\b(?:agent_skill_selection_confirmed|agentSkillSelectionConfirmed)\b/,
  },
  {
    id: 'misnamed-skill-capability-contract',
    pattern: /\b(?:skill_capability_keys|skillCapabilityKeys|selectedSkillKeys|SkillCapabilityKeys|ListSkillCapabilities|availableSkillCapabilities)\b/,
  },
  {
    id: 'misnamed-skill-capability-route',
    pattern: /\/api\/v1\/hr\/ai\/skill-capabilities\b/,
  },
  {
    id: 'legacy-agent-skill-document-shape',
    pattern: /\b(?:skill_md|flow_json|frontmatter_json|body_markdown)\b/,
  },
  {
    id: 'legacy-agent-skill-semantic-selection',
    pattern: /\b(?:SemanticScores|semanticScores|selected_skills|SelectedSkills)\b/,
  },
]

const allowedNegativeTestOccurrences = [
  {
    file: 'hr-frontend/src/components/hr/ai/agentRunChatFlow.test.ts',
    rule: 'legacy-agent-skill-id-selection',
    pattern: /^\s*expect\(.+\)\.not\.toHaveProperty\(['"](?:recommended_agent_skill_ids|agent_skill_ids)['"]\)\s*$/,
  },
  {
    file: 'platform-frontend/src/api/agentSkill.test.ts',
    rule: 'legacy-agent-skill-document-shape',
    pattern: /^\s*expect\(payload\)\.not\.toHaveProperty\(['"](?:skill_md|flow_json)['"]\)\s*$/,
  },
  {
    file: 'platform-frontend/src/components/agent-skill/packageEditor.test.ts',
    rule: 'legacy-agent-skill-document-shape',
    pattern: /^\s*expect\(draft\)\.not\.toHaveProperty\(['"](?:skill_md|flow_json)['"]\)\s*$/,
  },
  {
    file: 'smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_run_test.go',
    rule: 'misnamed-skill-capability-contract',
    pattern: /^\s*if strings\.Contains\(planJSON, "skill_capability_keys"\) \{\s*$/,
  },
  {
    file: 'smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go',
    rule: 'legacy-agent-skill-id-selection',
    pattern: /^\s*"(?:agent_skill_ids)",?\s*$/,
  },
  {
    file: 'smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go',
    rule: 'legacy-agent-skill-boolean-confirmation',
    pattern: /^\s*"(?:agent_skill_selection_confirmed)",?\s*$/,
  },
  {
    file: 'smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_runtime_test.go',
    rule: 'misnamed-skill-capability-contract',
    pattern: /^\s*"(?:skill_capability_keys)",?\s*$/,
  },
  {
    file: 'smart-recruit-gateway/handler/hr/agent_skill_test.go',
    rule: 'legacy-agent-skill-document-shape',
    pattern: /^\s*if strings\.Contains\(w\.Body\.String\(\), "skill_md"\) \|\| strings\.Contains\(w\.Body\.String\(\), "flow_json"\) \{\s*$/,
  },
  {
    file: 'smart-recruit-gateway/handler/hr/agent_skill_test.go',
    rule: 'legacy-agent-skill-document-shape',
    pattern: /^\s*\{"legacy field", `\{"name":"legacy","version":"1","skill_md":"old","package":` \+ packageJSON\(.+\) \+ `\}`\},\s*$/,
  },
  {
    file: 'smart-recruit-gateway/handler/hr/ai_agent_run_test.go',
    rule: 'misnamed-skill-capability-contract',
    pattern: /^\s*\{method: http\.Method(?:Post|Put), path: "\/api\/v1\/hr\/ai\/.+", body: `\{.+,"skill_capability_keys":\["search"\]\}`\},\s*$/,
  },
  {
    file: 'smart-recruit-gateway/handler/hr/ai_agent_run_test.go',
    rule: 'legacy-agent-skill-id-selection',
    pattern: /^\s*(?:body := )?`\{[^`]*"agent_skill_ids":\[[0-9,]+\][^`]*\}`,?\s*$/,
  },
  {
    file: 'smart-recruit-gateway/handler/hr/ai_agent_run_test.go',
    rule: 'legacy-agent-skill-boolean-confirmation',
    pattern: /^\s*(?:body := )?`\{"agent_skill_ids":\[[0-9,]+\],"agent_skill_selection_confirmed":true\}`,?\s*$/,
  },
  {
    file: 'smart-recruit-gateway/router/contract_baseline_test.go',
    rule: 'misnamed-skill-capability-route',
    pattern: /^\s*"GET \/api\/v1\/hr\/ai\/skill-capabilities",\s*$/,
  },
]

const isMigrationHistory = (file) =>
  file === 'smart-recruit-commons/migrations/000089_schema_baseline.sql'
  || file === 'smart-recruit-commons/migrations/000090_agent_skill_package_v2.sql'
  || file === 'smart-recruit-commons/migrations/000090_agent_skill_package_v2.down.sql'
  || file.startsWith('smart-recruit-commons/migrations/archive/pre-baseline-000089/')

const isActiveSource = (file) => {
  if (file === 'scripts/check-agent-skill-v2-cutover.mjs'
    || file === 'scripts/check-agent-skill-v2-cutover.test.mjs') return false
  if (file.startsWith('.knowledge/')
    || file.startsWith('.ai-guides/')
    || file.startsWith('.code-review-sdd/')
    || file.startsWith('.spec/')
    || file.includes('/node_modules/')
    || file.includes('/dist/')
    || file.includes('/coverage/')) return false
  return /\.(?:go|proto|ts|vue|js|mjs|cjs|json|ya?ml|sql|sh|bash|zsh|py)$/.test(file)
}

const isGeneratedProtoDescriptorOccurrence = (file, rule, line) =>
  file === 'smart-recruit-proto/recruitment/pb/recruitment.pb.go'
  && /^legacy-agent-skill-|^misnamed-skill-capability-/.test(rule)
  && /^\s*var file_recruitment_proto_rawDesc(?:Data)?\s*=.*\b(?:agent_skill_ids|agent_skill_selection_confirmed|skill_capability_keys|skill_md|flow_json|frontmatter_json|body_markdown)\b.*$/.test(line)

const isAllowedNegativeTestOccurrence = (file, rule, line) =>
  allowedNegativeTestOccurrences.some((occurrence) =>
    occurrence.file === file
    && occurrence.rule === rule
    && occurrence.pattern.test(line))

const isAllowedOccurrence = (file, rule, line) => {
  if (isMigrationHistory(file)) return true
  if (file === 'smart-recruit-proto/proto/recruitment.proto'
    && /^\s*reserved\b/.test(line)) return true
  if (isGeneratedProtoDescriptorOccurrence(file, rule, line)) return true
  if (isAllowedNegativeTestOccurrence(file, rule, line)) return true
  return false
}

export const scanEntries = (entries) => {
  const findings = []
  for (const entry of [...entries].sort((left, right) => left.file.localeCompare(right.file))) {
    if (!isActiveSource(entry.file)) continue
    const lines = entry.source.split('\n')
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index]
      for (const rule of rules) {
        if (rule.pattern.test(line) && !isAllowedOccurrence(entry.file, rule.id, line)) {
          findings.push({
            file: entry.file,
            line: index + 1,
            rule: rule.id,
          })
        }
      }
    }
  }
  return findings.sort((left, right) =>
    left.file.localeCompare(right.file)
    || left.line - right.line
    || left.rule.localeCompare(right.rule))
}

const trackedAndUntrackedFiles = (root) => execFileSync(
  'git',
  ['ls-files', '--cached', '--others', '--exclude-standard'],
  { cwd: root, encoding: 'utf8' },
)
  .split('\n')
  .filter(Boolean)

const main = () => {
  const entries = trackedAndUntrackedFiles(repositoryRoot)
    .filter(isActiveSource)
    .map((file) => ({
      file,
      source: readFileSync(path.join(repositoryRoot, file), 'utf8'),
    }))
  const findings = scanEntries(entries)
  if (findings.length > 0) {
    console.error('Agent Skill Package v2 cutover violations:')
    for (const finding of findings) {
      console.error(`- ${finding.file}:${finding.line}: ${finding.rule}`)
    }
    process.exitCode = 1
    return
  }
  console.log('Agent Skill Package v2 cutover check passed')
}

if (process.argv[1] && path.resolve(process.argv[1]) === scriptPath) {
  main()
}
