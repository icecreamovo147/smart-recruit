import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { AgentRunEvent, AgentRunSnapshot } from '@shared/types/agentRun'

const {
  createAgentRun,
  getActiveAgentRun,
  getAgentRun,
  cancelAgentRun,
  confirmAgentRun,
  subscribeAgentRunEvents,
} = vi.hoisted(() => ({
  createAgentRun: vi.fn(),
  getActiveAgentRun: vi.fn(),
  getAgentRun: vi.fn(),
  cancelAgentRun: vi.fn(),
  confirmAgentRun: vi.fn(),
  subscribeAgentRunEvents: vi.fn(),
}))

vi.mock('@/api/agentRun', () => ({
  createAgentRun,
  getActiveAgentRun,
  getAgentRun,
  cancelAgentRun,
  confirmAgentRun,
  subscribeAgentRunEvents,
}))

import { useHrAgentRun } from './useHrAgentRun'

const snapshot = (partial: Partial<AgentRunSnapshot> = {}): AgentRunSnapshot => ({
  run_id: 100,
  session_id: 9,
  status: 'queued',
  assistant_text: '',
  process_text: '',
  last_event_seq: 0,
  ...partial,
})

function mountComposable(): {
  api: ReturnType<typeof useHrAgentRun>
  wrapper: ReturnType<typeof mount>
} {
  let api: ReturnType<typeof useHrAgentRun> | undefined
  const Comp = defineComponent({
    setup() {
      api = useHrAgentRun()
      return () => null
    },
  })
  const wrapper = mount(Comp)
  if (!api) throw new Error('composable not initialized')
  return { api, wrapper }
}

describe('useHrAgentRun', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    subscribeAgentRunEvents.mockImplementation(async () => {
      // default: no events, resolves when aborted or immediately
    })
  })

  it('startRun hydrates state and auto-subscribes from last_event_seq', async () => {
    createAgentRun.mockResolvedValue({
      run: snapshot({ run_id: 55, session_id: 3, status: 'planning', last_event_seq: 1 }),
      idempotent_replay: false,
    })
    subscribeAgentRunEvents.mockImplementation(async () => {})

    const { api, wrapper } = mountComposable()
    const next = await api.startRun({ session_id: 3, message: 'hi' })

    expect(next.runId).toBe(55)
    expect(api.state.value.status).toBe('planning')
    expect(createAgentRun).toHaveBeenCalledWith({ session_id: 3, message: 'hi' })

    // allow microtask for void subscribe
    await nextTick()
    await Promise.resolve()

    expect(subscribeAgentRunEvents).toHaveBeenCalled()
    const [runId, afterSeq] = subscribeAgentRunEvents.mock.calls[0]
    expect(runId).toBe(55)
    expect(afterSeq).toBe(1)

    wrapper.unmount()
  })

  it('dispose/abort subscription does not call cancelAgentRun', async () => {
    let capturedSignal: AbortSignal | undefined
    subscribeAgentRunEvents.mockImplementation(
      async (
        _runId: number,
        _after: number,
        handlers: { onDone?: () => void },
        options?: { signal?: AbortSignal },
      ) => {
        capturedSignal = options?.signal
        await new Promise<void>((resolve) => {
          if (options?.signal?.aborted) {
            resolve()
            return
          }
          options?.signal?.addEventListener(
            'abort',
            () => {
              handlers.onDone?.()
              resolve()
            },
            { once: true },
          )
        })
      },
    )

    const { api, wrapper } = mountComposable()
    api.state.value = {
      ...api.state.value,
      runId: 77,
      lastEventSeq: 4,
      status: 'running',
    }

    const subPromise = api.subscribe(77, 4)
    await nextTick()
    expect(capturedSignal).toBeDefined()
    expect(capturedSignal?.aborted).toBe(false)

    api.dispose()
    await subPromise

    expect(capturedSignal?.aborted).toBe(true)
    expect(cancelAgentRun).not.toHaveBeenCalled()

    wrapper.unmount()
    // unmount dispose is also subscription-only
    expect(cancelAgentRun).not.toHaveBeenCalled()
  })

  it('explicit cancel calls cancelAgentRun', async () => {
    cancelAgentRun.mockResolvedValue({
      run: snapshot({ run_id: 12, status: 'cancel_requested', last_event_seq: 2 }),
    })
    const { api, wrapper } = mountComposable()
    api.state.value = {
      ...api.state.value,
      runId: 12,
      status: 'running',
      assistantText: 'partial',
      lastEventSeq: 2,
    }

    await api.cancel({ client_request_id: 'c1' })
    expect(cancelAgentRun).toHaveBeenCalledWith(12, { client_request_id: 'c1' })
    expect(api.state.value.status).toBe('cancel_requested')
    expect(api.state.value.assistantText).toBe('partial')

    wrapper.unmount()
  })

  it('hydrateFromActive restores snapshot and resumes subscribe', async () => {
    getActiveAgentRun.mockResolvedValue({
      has_active_run: true,
      run: snapshot({
        run_id: 200,
        session_id: 8,
        status: 'running',
        assistant_text: 'restored',
        process_text: 'trace',
        last_event_seq: 9,
      }),
    })
    subscribeAgentRunEvents.mockImplementation(async () => {})

    const { api, wrapper } = mountComposable()
    const next = await api.hydrateFromActive(8)
    expect(next?.assistantText).toBe('restored')
    expect(api.state.value.processText).toBe('trace')
    expect(api.state.value.lastEventSeq).toBe(9)

    await nextTick()
    await Promise.resolve()
    expect(subscribeAgentRunEvents).toHaveBeenCalledWith(
      200,
      9,
      expect.any(Object),
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    )

    wrapper.unmount()
  })

  it('hydrateFromActive returns null when no active run', async () => {
    getActiveAgentRun.mockResolvedValue({ has_active_run: false, run: null })
    const { api, wrapper } = mountComposable()
    const next = await api.hydrateFromActive(1)
    expect(next).toBeNull()
    expect(subscribeAgentRunEvents).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('applyEvent reduces into local state (happy path event wiring)', async () => {
    const { api, wrapper } = mountComposable()
    api.state.value = {
      ...api.state.value,
      runId: 5,
      status: 'running',
    }
    const event: AgentRunEvent = {
      run_id: 5,
      seq: 1,
      event_type: 'assistant.delta',
      delta: 'hi',
    }
    api.applyEvent(event)
    expect(api.state.value.assistantText).toBe('hi')
    expect(api.state.value.lastEventSeq).toBe(1)
    wrapper.unmount()
  })

  it('confirm posts confirmation and resubscribes', async () => {
    confirmAgentRun.mockResolvedValue({
      run: snapshot({ run_id: 33, status: 'running', last_event_seq: 3 }),
    })
    subscribeAgentRunEvents.mockImplementation(async () => {})

    const { api, wrapper } = mountComposable()
    api.state.value = {
      ...api.state.value,
      runId: 33,
      status: 'waiting_confirmation',
      lastEventSeq: 3,
    }

    await api.confirm({
      agent_skill_ids: [1, 2],
      agent_skill_selection_confirmed: true,
    })
    expect(confirmAgentRun).toHaveBeenCalledWith(33, {
      agent_skill_ids: [1, 2],
      agent_skill_selection_confirmed: true,
    })
    await nextTick()
    await Promise.resolve()
    expect(subscribeAgentRunEvents).toHaveBeenCalled()
    wrapper.unmount()
  })

  it('resubscribes with last_event_seq after unexpected stream drop while run is active', async () => {
    vi.useFakeTimers()
    try {
      let call = 0
      subscribeAgentRunEvents.mockImplementation(
        async (
          _runId: number,
          _after: number,
          handlers: { onDone?: () => void; onEvent?: (e: AgentRunEvent) => void },
        ) => {
          call += 1
          if (call === 1) {
            // First subscription ends unexpectedly without terminal status.
            handlers.onDone?.()
            return
          }
          // Second subscription stays open until dispose.
          await new Promise<void>(() => {
            /* never resolves */
          })
        },
      )

      let api: ReturnType<typeof useHrAgentRun> | undefined
      const Comp = defineComponent({
        setup() {
          api = useHrAgentRun({
            reconnectBaseMs: 50,
            reconnectMaxMs: 50,
            maxReconnectAttempts: 3,
          })
          return () => null
        },
      })
      const wrapper = mount(Comp)
      if (!api) throw new Error('composable not initialized')

      api.state.value = {
        ...api.state.value,
        runId: 88,
        sessionId: 2,
        status: 'running',
        lastEventSeq: 12,
        isTerminal: false,
      }

      const subPromise = api.subscribe(88, 12)
      await subPromise
      expect(subscribeAgentRunEvents).toHaveBeenCalledTimes(1)
      expect(subscribeAgentRunEvents.mock.calls[0][1]).toBe(12)

      await vi.advanceTimersByTimeAsync(60)
      await Promise.resolve()
      await nextTick()

      expect(subscribeAgentRunEvents.mock.calls.length).toBeGreaterThanOrEqual(2)
      const [, afterSeq] = subscribeAgentRunEvents.mock.calls[1]
      expect(afterSeq).toBe(12)
      expect(cancelAgentRun).not.toHaveBeenCalled()

      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not reconnect after dispose, and dispose aborts waitUntilSettled', async () => {
    vi.useFakeTimers()
    try {
      subscribeAgentRunEvents.mockImplementation(
        async (
          _runId: number,
          _after: number,
          handlers: { onDone?: () => void },
          options?: { signal?: AbortSignal },
        ) => {
          await new Promise<void>((resolve) => {
            if (options?.signal?.aborted) {
              handlers.onDone?.()
              resolve()
              return
            }
            options?.signal?.addEventListener(
              'abort',
              () => {
                handlers.onDone?.()
                resolve()
              },
              { once: true },
            )
          })
        },
      )

      let api: ReturnType<typeof useHrAgentRun> | undefined
      const Comp = defineComponent({
        setup() {
          api = useHrAgentRun({ reconnectBaseMs: 10, maxReconnectAttempts: 5 })
          return () => null
        },
      })
      const wrapper = mount(Comp)
      if (!api) throw new Error('composable not initialized')

      api.state.value = {
        ...api.state.value,
        runId: 9,
        status: 'running',
        lastEventSeq: 3,
        isTerminal: false,
      }

      void api.subscribe(9, 3)
      await nextTick()

      const settled = api.waitUntilSettled({ runId: 9 })
      api.dispose()
      await expect(settled).resolves.toBe('aborted')
      expect(cancelAgentRun).not.toHaveBeenCalled()

      const callsAfterDispose = subscribeAgentRunEvents.mock.calls.length
      await vi.advanceTimersByTimeAsync(500)
      expect(subscribeAgentRunEvents.mock.calls.length).toBe(callsAfterDispose)

      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('unmount marks waitUntilSettled as aborted without canceling the run', async () => {
    subscribeAgentRunEvents.mockImplementation(async () => {})

    let api: ReturnType<typeof useHrAgentRun> | undefined
    const Comp = defineComponent({
      setup() {
        api = useHrAgentRun()
        return () => null
      },
    })
    const wrapper = mount(Comp)
    if (!api) throw new Error('composable not initialized')

    api.state.value = {
      ...api.state.value,
      runId: 44,
      status: 'running',
      isTerminal: false,
    }

    const settled = api.waitUntilSettled({ runId: 44 })
    wrapper.unmount()
    await expect(settled).resolves.toBe('aborted')
    expect(cancelAgentRun).not.toHaveBeenCalled()
  })

  it('refreshes the snapshot once and resolves timed_out when a run never settles', async () => {
    vi.useFakeTimers()
    try {
      getAgentRun.mockResolvedValue({
        run: snapshot({ run_id: 45, status: 'running', last_event_seq: 2 }),
      })
      const { api, wrapper } = mountComposable()
      api.state.value = {
        ...api.state.value,
        runId: 45,
        status: 'running',
        isTerminal: false,
      }

      const settled = api.waitUntilSettled({ runId: 45, timeoutMs: 100 })
      await vi.advanceTimersByTimeAsync(100)

      await expect(settled).resolves.toBe('timed_out')
      expect(getAgentRun).toHaveBeenCalledWith(45)
      wrapper.unmount()
    } finally {
      vi.useRealTimers()
    }
  })

  it('hydrateFromActive then cancel uses same run id (post-refresh cancel path)', async () => {
    getActiveAgentRun.mockResolvedValue({
      has_active_run: true,
      run: snapshot({
        run_id: 501,
        session_id: 12,
        status: 'running',
        assistant_text: 'half',
        last_event_seq: 4,
      }),
    })
    cancelAgentRun.mockResolvedValue({
      run: snapshot({
        run_id: 501,
        session_id: 12,
        status: 'cancel_requested',
        assistant_text: 'half',
        last_event_seq: 4,
      }),
    })
    subscribeAgentRunEvents.mockImplementation(async () => {})

    const { api, wrapper } = mountComposable()
    const next = await api.hydrateFromActive(12)
    expect(next?.runId).toBe(501)
    await api.cancel({ client_request_id: 'stop-1' })
    expect(cancelAgentRun).toHaveBeenCalledWith(501, { client_request_id: 'stop-1' })
    expect(api.state.value.status).toBe('cancel_requested')
    expect(api.state.value.assistantText).toBe('half')
    wrapper.unmount()
  })
})
