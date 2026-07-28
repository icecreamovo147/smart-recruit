import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { scanEntries } from './check-agent-skill-v2-cutover.mjs'

const scriptPath = fileURLToPath(new URL('./check-agent-skill-v2-cutover.mjs', import.meta.url))
const repositoryRoot = path.resolve(path.dirname(scriptPath), '..')

test('rejects legacy identifiers and the retired HTTP route in active source', () => {
  const findings = scanEntries([
    {
      file: 'smart-recruit-gateway/handler/hr/example.go',
      source: 'var agent_skill_ids []int64\nconst route = "/api/v1/hr/ai/skill-capabilities"\n',
    },
    {
      file: 'hr-frontend/src/example.ts',
      source: 'const selectedSkillKeys = []\nconst body = { skill_capability_keys: [] }\n',
    },
  ])
  assert.deepEqual(
    findings.map((finding) => finding.rule),
    [
      'misnamed-skill-capability-contract',
      'misnamed-skill-capability-contract',
      'legacy-agent-skill-id-selection',
      'misnamed-skill-capability-route',
    ],
  )
})

test('allows only reserved/generated, migration history, and explicit negative-test occurrences', () => {
  const findings = scanEntries([
    {
      file: 'smart-recruit-proto/proto/recruitment.proto',
      source: 'reserved "agent_skill_ids", "skill_md";\n',
    },
    {
      file: 'smart-recruit-proto/recruitment/pb/recruitment.pb.go',
      source: 'var file_recruitment_proto_rawDesc = []byte("agent_skill_ids skill_md")\n',
    },
    {
      file: 'smart-recruit-commons/migrations/000090_agent_skill_package_v2.down.sql',
      source: 'CHANGE COLUMN agent_skill_version_ids_json agent_skill_ids_json;\n',
    },
    {
      file: 'smart-recruit-gateway/handler/hr/ai_agent_run_test.go',
      source: 'body := `{"agent_skill_ids":[1],"agent_skill_selection_confirmed":true}`\n',
    },
  ])
  assert.deepEqual(findings, [])
})

test('does not allow executable legacy identifiers hidden in a negative-test path', () => {
  const findings = scanEntries([
    {
      file: 'smart-recruit-gateway/handler/hr/ai_agent_run_test.go',
      source: 'func productionAdapter() { var agent_skill_ids []int64 }\n',
    },
  ])
  assert.deepEqual(findings, [
    {
      file: 'smart-recruit-gateway/handler/hr/ai_agent_run_test.go',
      line: 1,
      rule: 'legacy-agent-skill-id-selection',
    },
  ])
})

test('rejects legacy fields and routes in active repository scripts', () => {
  const findings = scanEntries([
    {
      file: 'scripts/runtime-check.sh',
      source: 'curl /api/v1/hr/ai/skill-capabilities\n',
    },
    {
      file: 'tools/legacy_adapter.py',
      source: 'request = {"agent_skill_ids": [1]}\n',
    },
    {
      file: 'tools/legacy_adapter.bash',
      source: 'field=skill_capability_keys\n',
    },
    {
      file: 'tools/legacy_adapter.zsh',
      source: 'field=agent_skill_selection_confirmed\n',
    },
  ])
  assert.deepEqual(findings.map((finding) => finding.rule), [
    'misnamed-skill-capability-route',
    'misnamed-skill-capability-contract',
    'legacy-agent-skill-id-selection',
    'legacy-agent-skill-boolean-confirmation',
  ])
})

test('real repository scan accepts only the current narrow legacy occurrences', () => {
  assert.doesNotThrow(() => execFileSync(
    process.execPath,
    [scriptPath],
    {
      cwd: repositoryRoot,
      encoding: 'utf8',
      stdio: 'pipe',
    },
  ))
})

test('does not flag Package v2 registry and exact-version concepts', () => {
  const findings = scanEntries([
    {
      file: 'smart-recruit-ai-agent-service/internal/example.go',
      source: 'current_version_id := exactVersionID\nis_manual_invocable := true\ntrigger_keywords := []string{"screen"}\n',
    },
    {
      file: 'hr-frontend/src/example.ts',
      source: 'const capabilityKeys = []\nconst selectedAgentSkillVersionIds = []\n',
    },
  ])
  assert.deepEqual(findings, [])
})
