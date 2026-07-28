import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptPath = path.join(path.dirname(fileURLToPath(import.meta.url)), 'parse_code_review_sdd.mjs')

const baseFinding = {
  id: 'F-001',
  title: 'example finding',
  priority: 'P1',
  risk: 'high',
  status: 'OPEN',
  file: 'src/a.ts',
  line: 1,
}

const baseTask = {
  id: 'CR-001',
  finding_ids: ['F-001'],
  priority: 'P1',
  title: 'repair example',
  status: 'PENDING',
  dependencies: [],
  scope: {
    allowed_paths: ['src/a.ts'],
    excluded_paths: [],
  },
  steps: ['repair'],
  acceptance_criteria: ['passes'],
  tests: [],
}

const baseManifest = () => ({
  schema_version: 1,
  review: {},
  findings: [{ ...baseFinding }],
  tasks: [{
    ...baseTask,
    finding_ids: [...baseTask.finding_ids],
    dependencies: [...baseTask.dependencies],
    scope: {
      allowed_paths: [...baseTask.scope.allowed_paths],
      excluded_paths: [...baseTask.scope.excluded_paths],
    },
    steps: [...baseTask.steps],
    acceptance_criteria: [...baseTask.acceptance_criteria],
    tests: [],
  }],
})

const parseManifest = (manifest, extraArgs = []) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'code-review-sdd-parser-'))
  const input = path.join(directory, 'plan.md')
  fs.writeFileSync(input, `\`\`\`code-review-sdd-manifest\n${JSON.stringify(manifest)}\n\`\`\`\n`)
  const result = spawnSync(process.execPath, [scriptPath, '--input', input, ...extraArgs], { encoding: 'utf8' })
  fs.rmSync(directory, { recursive: true, force: true })
  return result
}

const assertStructuredFailure = (result, expected) => {
  assert.equal(result.status, 1)
  assert.match(result.stderr, expected)
  assert.doesNotMatch(result.stderr, /file:\/\/|parse_code_review_sdd\.mjs:\d+|\n\s+at\s/)
}

test('accepts a valid focused repair plan', () => {
  const result = parseManifest(baseManifest())
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stderr, /code_review_sdd_parse: PASS/)
})

test('accepts an explicit no-action canonical manifest with zero tasks', () => {
  const manifest = baseManifest()
  manifest.no_action = true
  manifest.findings = []
  manifest.tasks = []
  const result = parseManifest(manifest)
  assert.equal(result.status, 0, result.stderr)
  const parsed = JSON.parse(result.stdout)
  assert.equal(parsed.no_action, true)
  assert.equal(parsed.task_count, 0)
  assert.deepEqual(parsed.tasks, [])
  assert.match(result.stderr, /0 tasks; no_action/)
})

test('rejects findings or tasks inside a no-action manifest', async (t) => {
  for (const field of ['findings', 'tasks']) {
    await t.test(field, () => {
      const manifest = baseManifest()
      manifest.no_action = true
      if (field === 'findings') manifest.tasks = []
      else manifest.findings = []
      const result = parseManifest(manifest)
      assert.equal(result.status, 1)
      assert.match(result.stderr, new RegExp(`no_action manifest must not contain ${field}`))
    })
  }
})

test('rejects task or priority selection from a no-action manifest without a stack', async (t) => {
  for (const args of [['--task', 'CR-001'], ['--priority', 'P1']]) {
    await t.test(args.join(' '), () => {
      const manifest = baseManifest()
      manifest.no_action = true
      manifest.findings = []
      manifest.tasks = []
      assertStructuredFailure(
        parseManifest(manifest, args),
        /error: no tasks can be selected from a no_action manifest/,
      )
    })
  }
})

test('rejects null and primitive findings without a stack', async (t) => {
  for (const invalidFinding of [null, 42, 'F-001', []]) {
    await t.test(JSON.stringify(invalidFinding), () => {
      const manifest = baseManifest()
      manifest.findings = [invalidFinding]
      assertStructuredFailure(
        parseManifest(manifest),
        /error: findings\[0\] must be an object/,
      )
    })
  }
})

test('rejects an unknown task selector without a stack', () => {
  assertStructuredFailure(
    parseManifest(baseManifest(), ['--task', 'CR-999']),
    /error: unknown task selection: CR-999/,
  )
})

test('rejects repository root, cross-platform absolute, traversal, home, and protected paths', async (t) => {
  const unsafePaths = [
    '.',
    './',
    '..',
    '../outside',
    'src/../../outside',
    '/tmp/outside',
    '~/outside',
    '~other/outside',
    'C:\\temp\\outside',
    'D:/temp/outside',
    '\\\\server\\share\\outside',
    '.git/config',
    '.GIT/config',
    '.code-review-sdd/code-review-sdd.md',
  ]

  for (const unsafePath of unsafePaths) {
    await t.test(unsafePath, () => {
      const manifest = baseManifest()
      manifest.tasks[0].scope.allowed_paths = [unsafePath]
      const result = parseManifest(manifest)
      assert.equal(result.status, 1)
      assert.match(result.stderr, /contains unsafe path/)
    })
  }
})

test('rejects non-string scope entries in allowed and excluded paths', async (t) => {
  const invalidValues = [
    ['object', {}],
    ['boolean', true],
    ['number', 42],
    ['array', ['src/a.ts']],
  ]
  for (const field of ['allowed_paths', 'excluded_paths']) {
    for (const [name, value] of invalidValues) {
      await t.test(`${field}: ${name}`, () => {
        const manifest = baseManifest()
        manifest.tasks[0].scope[field] = [value]
        const result = parseManifest(manifest)
        assert.equal(result.status, 1)
        assert.match(result.stderr, new RegExp(`${field} contains unsafe path`))
      })
    }
  }
})

test('rejects URI schemes and control characters in allowed and excluded paths', async (t) => {
  const unsafePaths = [
    ['file URI', 'file:///etc/passwd'],
    ['https URI', 'https://example.com/repo'],
    ['custom URI', 'ssh+git://example.com/repo'],
    ['newline', 'src/a\n.md'],
    ['tab', 'src/a\t.md'],
    ['carriage return', 'src/a\r.md'],
    ['DEL', 'src/a\u007F.md'],
  ]
  for (const field of ['allowed_paths', 'excluded_paths']) {
    for (const [name, value] of unsafePaths) {
      await t.test(`${field}: ${name}`, () => {
        const manifest = baseManifest()
        manifest.tasks[0].scope[field] = [value]
        const result = parseManifest(manifest)
        assert.equal(result.status, 1)
        assert.match(result.stderr, new RegExp(`${field} contains unsafe path`))
        if (name === 'newline') assert.match(result.stderr, /src\/a\\n\.md/)
      })
    }
  }
})

test('rejects repository-wide wildcard-only scopes', async (t) => {
  const unsafeGlobs = ['*', '**', '*/*', '**/*', '**/**']

  for (const unsafeGlob of unsafeGlobs) {
    await t.test(unsafeGlob, () => {
      const manifest = baseManifest()
      manifest.tasks[0].scope.allowed_paths = [unsafeGlob]
      const result = parseManifest(manifest)
      assert.equal(result.status, 1)
      assert.match(result.stderr, /contains unsafe path/)
    })
  }
})

test('rejects protected-path-equivalent and ambiguous root globs', async (t) => {
  const unsafeGlobs = [
    '.git*',
    '.g?t/**',
    '[.]git/**',
    '.code-review-sdd*/**',
    '{.git,src}/**',
    '**/.git/**',
    '@(.git|src)/**',
  ]
  for (const field of ['allowed_paths', 'excluded_paths']) {
    for (const unsafeGlob of unsafeGlobs) {
      await t.test(`${field}: ${unsafeGlob}`, () => {
        const manifest = baseManifest()
        manifest.tasks[0].scope[field] = [unsafeGlob]
        const result = parseManifest(manifest)
        assert.equal(result.status, 1)
        assert.match(result.stderr, new RegExp(`${field} contains unsafe path`))
      })
    }
  }
})

test('accepts focused module globs and static hidden directories', async (t) => {
  const focusedPaths = ['src/**/*.ts', 'smart-recruit-*/**', '.agents/**', '.knowledge/**']
  for (const focusedPath of focusedPaths) {
    await t.test(focusedPath, () => {
      const manifest = baseManifest()
      manifest.tasks[0].scope.allowed_paths = [focusedPath]
      const result = parseManifest(manifest)
      assert.equal(result.status, 0, result.stderr)
    })
  }
})

test('rejects an open finding that no task references', () => {
  const manifest = baseManifest()
  manifest.findings.push({ ...baseFinding, id: 'F-002' })
  const result = parseManifest(manifest)
  assert.equal(result.status, 1)
  assert.match(result.stderr, /open finding F-002 is not referenced by any task/)
})

test('normalizes finding status before enforcing open-finding coverage', () => {
  const manifest = baseManifest()
  manifest.findings.push({ ...baseFinding, id: 'F-002', status: ' OPEN ' })
  const result = parseManifest(manifest)
  assert.equal(result.status, 1)
  assert.match(result.stderr, /open finding F-002 is not referenced by any task/)
})

test('rejects unknown finding status', () => {
  const manifest = baseManifest()
  manifest.findings[0].status = 'PENDING'
  const result = parseManifest(manifest)
  assert.equal(result.status, 1)
  assert.match(result.stderr, /findings\[0\]\.status must be OPEN or CLOSED/)
})

test('allows an unreferenced closed finding as historical context', () => {
  const manifest = baseManifest()
  manifest.findings.push({ ...baseFinding, id: 'F-002', status: 'CLOSED' })
  const result = parseManifest(manifest)
  assert.equal(result.status, 0, result.stderr)
})
