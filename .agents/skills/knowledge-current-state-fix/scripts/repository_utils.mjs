import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'
import { sha256 } from './plan_utils.mjs'

export const git = (args, cwd = process.cwd()) => execFileSync('git', args, { cwd, encoding: 'utf8' }).trim()

export const repositoryRoot = (cwd = process.cwd()) => git(['rev-parse', '--show-toplevel'], cwd)

export const worktreePaths = (root) => {
  const raw = execFileSync('git', ['status', '--porcelain=v1', '-z', '--untracked-files=all'], { cwd: root })
  const fields = raw.toString('utf8').split('\0')
  const paths = []
  for (let index = 0; index < fields.length; index += 1) {
    const entry = fields[index]
    if (!entry) continue
    const status = entry.slice(0, 2)
    paths.push(entry.slice(3))
    if (/[RC]/.test(status) && fields[index + 1]) paths.push(fields[++index])
  }
  return [...new Set(paths)].sort()
}

export const repositoryPaths = (root) => {
  const raw = execFileSync('git', ['ls-files', '-co', '--exclude-standard', '-z'], { cwd: root })
  return [...new Set(raw.toString('utf8').split('\0').filter(Boolean))].sort()
}

const fingerprintPath = (root, relativePath) => {
  const absolute = path.join(root, relativePath)
  try {
    const stat = fs.lstatSync(absolute)
    if (stat.isSymbolicLink()) return `symlink:${sha256(fs.readlinkSync(absolute))}`
    if (stat.isFile()) return `file:${sha256(fs.readFileSync(absolute))}`
    return `other:${stat.mode}:${stat.size}`
  } catch (error) {
    if (error?.code === 'ENOENT') return 'missing'
    throw error
  }
}

export const snapshotRepository = (root) => Object.fromEntries(
  repositoryPaths(root).map((relativePath) => [relativePath, fingerprintPath(root, relativePath)]),
)

export const changedPaths = (before, after) => [...new Set([...Object.keys(before || {}), ...Object.keys(after || {})])]
  .filter((relativePath) => before?.[relativePath] !== after?.[relativePath])
  .sort()
