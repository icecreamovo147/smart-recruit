#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const failures = []

const generation = spawnSync(process.execPath, ['scripts/generate-i18n.mjs', '--check'], {
  cwd: root,
  encoding: 'utf8',
})
if (generation.status !== 0) {
  failures.push(generation.stderr.trim() || 'frontend i18n catalog generation check failed')
}

const catalogs = {}
for (const locale of ['zh-CN', 'en-US']) {
  const file = path.join(root, 'smart-recruit-platform-go', 'i18n', 'catalogs', `${locale}.json`)
  catalogs[locale] = JSON.parse(fs.readFileSync(file, 'utf8'))
}
const zhKeys = Object.keys(catalogs['zh-CN']).sort()
const enKeys = Object.keys(catalogs['en-US']).sort()
if (JSON.stringify(zhKeys) !== JSON.stringify(enKeys)) {
  failures.push('zh-CN and en-US catalog keys differ')
}

const placeholderPattern = /\{([a-zA-Z][a-zA-Z0-9_]*)\}/g
for (const key of zhKeys) {
  const placeholders = (message) => [...message.matchAll(placeholderPattern)].map((match) => match[1]).sort()
  if (JSON.stringify(placeholders(catalogs['zh-CN'][key])) !== JSON.stringify(placeholders(catalogs['en-US'][key]))) {
    failures.push(`placeholder mismatch for ${key}`)
  }
}

for (const relative of sourceFiles(['hr-frontend/src', 'user-frontend/src', 'platform-frontend/src'], /\.(?:ts|vue)$/)) {
  const source = fs.readFileSync(path.join(root, relative), 'utf8')
  const directMessage = /ElMessage\.(?:error|warning|success|info)\(\s*['"`]/
  if (directMessage.test(source)) {
    failures.push(`${relative} contains a direct user-visible ElMessage literal`)
  }
  if (!/\.test\.(?:ts|vue)$/.test(relative) && /new Error\(\s*['"`]/.test(source)) {
    failures.push(`${relative} contains a direct user-visible Error literal`)
  }
}

const serviceRoots = fs.readdirSync(root)
  .filter((name) => /^smart-recruit-.*-service$/.test(name))
  .map((name) => `${name}`)
for (const relative of sourceFiles(serviceRoots, /\.go$/, /_test\.go$/)) {
  const source = fs.readFileSync(path.join(root, relative), 'utf8')
  const responseMessage = /\bMsg:\s*([^,}\n]+)/g
  for (const match of source.matchAll(responseMessage)) {
    const expression = match[1].trim()
    if (relative.includes('/internal/domain/')) {
      continue
    }
    if (/^"[a-z][a-z0-9_]*(?:\.[a-z0-9_]+)+"$/.test(expression)) {
      ensureCatalogKey(expression.slice(1, -1), relative)
      continue
    }
    if (expression === 'messageKey' || /^hrContextMessageKey\(/.test(expression) || /^i18n\.KeyForCode\(/.test(expression)) {
      continue
    }
    if (relative.endsWith('/internal/domain/policy/recruitment.go') && expression === '"投递状态不合法"') {
      continue // Internal domain error; it is not a protobuf response field.
    }
    failures.push(`${relative} contains unmanaged protobuf response Msg expression: ${expression}`)
  }

  const eventMessage = /\bEventMessage:\s*"([^"]+)"/g
  for (const match of source.matchAll(eventMessage)) {
    if (/^[a-z][a-z0-9_]*(?:\.[a-z0-9_]+)+$/.test(match[1])) {
      ensureCatalogKey(match[1], relative)
    } else {
      failures.push(`${relative} contains unmanaged SSE EventMessage literal: ${match[1]}`)
    }
  }
}

for (const relative of sourceFiles(
  [...serviceRoots.map((service) => `${service}/cmd`), 'smart-recruit-gateway/cmd'],
  /\.go$/,
  /_test\.go$/,
)) {
  const source = fs.readFileSync(path.join(root, relative), 'utf8')
  if (/fmt\.Fprint(?:f|ln)?\(os\.(?:Stdout|Stderr)/.test(source)) {
    failures.push(`${relative} writes an unmanaged service message to stdout/stderr`)
  }
  const logLiteral = /\.(?:Info|Warn|Error|Debug|Fatal)\("([^"]+)"/g
  for (const match of source.matchAll(logLiteral)) {
    if (/^[a-z][a-z0-9_]*(?:\.[a-z0-9_]+)+$/.test(match[1])) {
      ensureCatalogKey(match[1], relative)
    } else {
      failures.push(`${relative} contains an unmanaged service log literal: ${match[1]}`)
    }
  }
}

for (const relative of sourceFiles(
  [...serviceRoots, 'smart-recruit-gateway', 'smart-recruit-platform-go'],
  /\.go$/,
  /_test\.go$/,
)) {
  const source = fs.readFileSync(path.join(root, relative), 'utf8')
  for (const match of source.matchAll(/\bi18n\.T\("([^"]+)"/g)) {
    ensureCatalogKey(match[1], relative)
  }
}

if (failures.length > 0) {
  process.stderr.write(`${failures.map((failure) => `ERROR: ${failure}`).join('\n')}\n`)
  process.exit(1)
}

process.stdout.write(`i18n governance OK (${zhKeys.length} synchronized keys)\n`)

function sourceFiles(relativeRoots, include, exclude = /$^/) {
  const result = []
  for (const relativeRoot of relativeRoots) {
    visit(relativeRoot)
  }
  return result

  function visit(relative) {
    const absolute = path.join(root, relative)
    const stat = fs.statSync(absolute)
    if (stat.isDirectory()) {
      for (const entry of fs.readdirSync(absolute)) {
        visit(path.join(relative, entry))
      }
      return
    }
    if (include.test(relative) && !exclude.test(relative)) {
      result.push(relative)
    }
  }
}

function ensureCatalogKey(key, relative) {
  if (!Object.prototype.hasOwnProperty.call(catalogs['zh-CN'], key)) {
    failures.push(`${relative} references missing i18n key: ${key}`)
  }
}
