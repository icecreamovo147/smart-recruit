#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { atomicWriteJson, effectiveConfirmationReasons, loadPlanFile, orderedTasks, parseCli } from './plan_utils.mjs'

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.input) {
    process.stdout.write('Usage: parse_remediation_plan.mjs --input <plan.md> [--output <plan.json>]\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const input = path.resolve(args.input)
    const loaded = loadPlanFile(input)
    const ordered = orderedTasks(loaded.manifest.tasks)
    const output = {
      schema_version: 1,
      plan_path: input,
      plan_sha256: loaded.sha256,
      execution_order: ordered.map((task) => task.id),
      effective_confirmation: Object.fromEntries(ordered.map((task) => [task.id, effectiveConfirmationReasons(task)])),
      manifest: loaded.manifest,
    }
    if (args.output) {
      const destination = path.resolve(args.output)
      if (fs.existsSync(destination)) throw new Error(`refusing to overwrite existing file: ${destination}`)
      atomicWriteJson(destination, output)
      process.stdout.write(`${destination}\n`)
    } else process.stdout.write(`${JSON.stringify(output, null, 2)}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
