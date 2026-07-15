import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const srcRoot = join(process.cwd(), 'web', 'src')

describe('frontend safety constraints', () => {
  it('keeps log rendering as text and avoids third-party resource URLs', () => {
    const files = [
      'components/AppShell.tsx',
      'components/DetailPanel.tsx',
      'components/LogTable.tsx',
      'styles/app.css',
    ]
    const source = files.map((file) => readFileSync(join(srcRoot, file), 'utf8')).join('\n')

    expect(source).not.toContain('dangerouslySetInnerHTML')
    expect(source).not.toMatch(/https?:\/\//)
    expect(source).not.toMatch(/fonts\.googleapis|cdn\./i)
  })

  it('keeps local sensitive-log warning text visible in the shell source', () => {
    const appShell = readFileSync(join(srcRoot, 'components', 'AppShell.tsx'), 'utf8')

    expect(appShell).toContain('Sensitive local logs')
    expect(appShell).toContain('Loopback + sensitive logs')
  })
})
