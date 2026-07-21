#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const exceptions = JSON.parse(readFileSync(path.join(root, 'scripts/timezone-exceptions.json'), 'utf8'))
const files = execFileSync('git', ['ls-files', '--cached', '--others', '--exclude-standard'], { cwd: root, encoding: 'utf8' })
  .split('\n')
  .filter(Boolean)

const findings = []
const isHistoricalMigration = (file) => /^smart-recruit-commons\/migrations\/0000(?:73|74|76|80|81)_/.test(file)
const allowed = (kind, file) => Boolean(exceptions[kind]?.[file])

for (const file of files) {
  if (file.startsWith('.git/') || file.includes('/node_modules/') || file.endsWith('.sum')) continue
  if (file === 'scripts/check-timezone-contract.mjs' || file === 'scripts/timezone-exceptions.json') continue
  let source
  try {
    source = readFileSync(path.join(root, file), 'utf8')
  } catch {
    continue
  }
  const lines = source.split('\n')
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]
    if (/\.(?:go|sql|sh|ya?ml)$/.test(file) && /UTC_TIMESTAMP\s*\(/.test(line) && !isHistoricalMigration(file) && !allowed('utc_timestamp', file)) {
      findings.push(`${file}:${index + 1}: UTC_TIMESTAMP is forbidden outside immutable historical migrations`)
    }
    if (/\.(?:go|sh|ya?ml)$/.test(file) && /loc=Local/.test(line) && !isHistoricalMigration(file) && !allowed('loc_local', file)) {
      findings.push(`${file}:${index + 1}: loc=Local is forbidden; use Asia%2FShanghai`)
    }
    if (file.endsWith('.go') && /Truncate\(\s*24\s*\*\s*time\.Hour\s*\)/.test(line)) {
      findings.push(`${file}:${index + 1}: Truncate(24*time.Hour) is not a civil-day boundary`)
    }
    if (file.startsWith('smart-recruit-') && file.endsWith('.go') && !file.endsWith('_test.go') && /\.UTC\(\)/.test(line) && !allowed('utc_calls', file)) {
      findings.push(`${file}:${index + 1}: .UTC() requires a documented protocol/absolute-time exception`)
    }
  }
}

if (findings.length > 0) {
  console.error('UTC+8 timezone contract violations:\n' + findings.map((item) => `- ${item}`).join('\n'))
  process.exit(1)
}
console.log('UTC+8 timezone contract check passed')
