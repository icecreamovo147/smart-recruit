#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'
import { asList, globMatches, parseCli } from './plan_utils.mjs'

const gitPaths = () => execFileSync('git', ['status', '--porcelain=v1'], { encoding: 'utf8' }).trim()
  .split('\n').filter(Boolean)
  .flatMap((line) => {
    const raw = line.slice(3).trim()
    return raw.includes(' -> ') ? raw.split(' -> ').map((value) => value.trim()) : [raw]
  })

try {
  const args = parseCli(process.argv, new Set(['help', 'fromGit']))
  if (args.help || !args.state || !args.task || (!args.fromGit && !args.path)) {
    process.stdout.write('Usage: check_fix_scope.mjs --state <state.json> --task REM-001 (--from-git | --path <path>...)\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const state = JSON.parse(fs.readFileSync(path.resolve(args.state), 'utf8'))
    const task = asList(state.tasks).find((item) => item.id === args.task)
    if (!task) throw new Error(`task not found in state: ${args.task}`)
    let paths = args.fromGit
      ? gitPaths().filter((candidate) => !asList(state.repository.initial_worktree_paths).includes(candidate))
      : Array.isArray(args.path) ? args.path : [args.path]
    paths = [...new Set(paths.map((value) => value.replace(/^\.\//, '')))].sort()
    const excluded = paths.filter((candidate) => task.excluded_paths.some((pattern) => globMatches(pattern, candidate)))
    const unauthorized = paths.filter((candidate) => !task.allowed_paths.some((pattern) => globMatches(pattern, candidate)))
    const result = {
      task: task.id,
      checked_paths: paths,
      allowed_patterns: task.allowed_paths,
      excluded_patterns: task.excluded_paths,
      excluded_paths: excluded,
      unauthorized_paths: unauthorized,
      passed: excluded.length === 0 && unauthorized.length === 0,
    }
    process.stdout.write(`${JSON.stringify(result, null, 2)}\n`)
    if (!result.passed) process.exitCode = 1
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}

