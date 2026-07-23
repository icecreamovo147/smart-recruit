#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { parseCli } from './plan_utils.mjs'
import { validateState } from './state_utils.mjs'

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.state) {
    process.stdout.write('Usage: validate_fix_state.mjs --state <fix-state.json>\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const statePath = path.resolve(args.state)
    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
    const schedule = validateState(state, statePath)
    process.stdout.write(`${JSON.stringify({ valid: true, run_id: state.run_id, run_status: state.run_status, ...schedule }, null, 2)}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
