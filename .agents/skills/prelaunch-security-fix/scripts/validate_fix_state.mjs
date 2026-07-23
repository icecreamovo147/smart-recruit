#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { parseCli } from './plan_utils.mjs'
import { validateState } from './state_utils.mjs'

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.state) {
    process.stdout.write('Usage: validate_fix_state.mjs --state <fix-state.json> [--plan <plan.md>]\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const state = JSON.parse(fs.readFileSync(path.resolve(args.state), 'utf8'))
    const summary = validateState(state, args.plan && path.resolve(args.plan))
    process.stdout.write(`${JSON.stringify({ valid: true, run_status: state.run_status, ...summary }, null, 2)}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
