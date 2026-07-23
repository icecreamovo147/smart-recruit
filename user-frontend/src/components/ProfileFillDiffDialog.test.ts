import { describe, expect, it } from 'vitest'
import type { ProfileFillFieldDiff } from '@/types/domain'

function recomputeDiffs(diffs: ProfileFillFieldDiff[], overwrite: boolean): ProfileFillFieldDiff[] {
  return diffs.map((item) => {
    if (item.action === 'unsupported') return item
    const before = (item.before || '').trim()
    const after = (item.after || '').trim()
    if (!after) return { ...item, action: 'skip' }
    if (!before) return { ...item, action: 'fill' }
    if (overwrite && before !== after) return { ...item, action: 'overwrite' }
    return { ...item, action: 'skip' }
  })
}

describe('profile fill diff local recompute', () => {
  const base: ProfileFillFieldDiff[] = [
    { field: 'phone', label: '电话', action: 'fill', before: '', after: '13800138000' },
    { field: 'real_name', label: '姓名', action: 'skip', before: '张三', after: '李四' },
    { field: 'projects', label: '项目', action: 'unsupported', before: '', after: '简历含 1 个项目' },
  ]

  it('keeps fill for empty before', () => {
    const next = recomputeDiffs(base, false)
    expect(next.find((item) => item.field === 'phone')?.action).toBe('fill')
    expect(next.find((item) => item.field === 'real_name')?.action).toBe('skip')
  })

  it('marks overwrite when enabled', () => {
    const next = recomputeDiffs(base, true)
    expect(next.find((item) => item.field === 'real_name')?.action).toBe('overwrite')
    expect(next.find((item) => item.field === 'projects')?.action).toBe('unsupported')
  })
})
