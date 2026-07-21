import { beforeEach, describe, expect, it, vi } from 'vitest'

const { messageError } = vi.hoisted(() => ({ messageError: vi.fn() }))

vi.mock('element-plus', () => ({ ElMessage: { error: messageError } }))
vi.mock('@/router', () => ({ default: { push: vi.fn() } }))
vi.mock('@/utils/token', () => ({ clearLocalAuthCache: vi.fn() }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ $reset: vi.fn() }) }))
vi.mock('./request', () => ({ default: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() } }))
vi.mock('./authRefresh', () => ({ silentRefresh: vi.fn() }))

import { sendMessageStream } from './ai'

function sseResponse(events: string[]): Response {
  const encoder = new TextEncoder()
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const event of events) controller.enqueue(encoder.encode(`data: ${event}\n\n`))
      controller.close()
    },
  })
  return new Response(body, { headers: { 'content-type': 'text/event-stream' } })
}

describe('candidate sendMessageStream', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    messageError.mockClear()
  })

  it('reports an abnormal EOF when the stream closes without done', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(sseResponse([
      JSON.stringify({ delta: 'partial' }),
    ])))
    const onError = vi.fn()

    await expect(sendMessageStream({ message: 'hello' }, { onError })).rejects.toMatchObject({ code: 502 })

    expect(onError).toHaveBeenCalledWith(
      '502',
      'AI 服务连接已中断，请稍后重试',
      expect.objectContaining({ code: 502 }),
    )
    expect(messageError).toHaveBeenCalledOnce()
  })

  it('accepts a stream that ends with a done event', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(sseResponse([
      JSON.stringify({ delta: 'ok' }),
      JSON.stringify({ done: true, session_id: 1 }),
    ])))
    const onDone = vi.fn()
    const onError = vi.fn()

    await expect(sendMessageStream({ message: 'hello' }, { onDone, onError })).resolves.toBeUndefined()

    expect(onDone).toHaveBeenCalledWith(expect.objectContaining({ done: true }))
    expect(onError).not.toHaveBeenCalled()
  })
})
