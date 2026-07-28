import { t } from '@shared/i18n'
import { onUnmounted, ref, shallowRef, watch, type Ref, type ShallowRef } from 'vue'
import {
  cancelAgentRun,
  confirmAgentRun,
  createAgentRun,
  getActiveAgentRun,
  getAgentRun,
  subscribeAgentRunEvents,
} from '@/api/agentRun'
import type {
  AgentRunEvent,
  CancelAgentRunRequest,
  ConfirmAgentRunRequest,
  CreateAgentRunRequest,
} from '@shared/types/agentRun'
import {
  applyAgentRunEvents,
  createInitialHrAgentRunState,
  hydrateFromSnapshot,
  reduceAgentRunEvent,
  type HrAgentRunState,
} from '@/utils/hrAgentRunReducer'

export type HrAgentRunSettlement = 'terminal' | 'waiting_confirmation' | 'timed_out' | 'aborted'

export interface WaitUntilSettledOptions {
  /** Prefer matching this run id when provided. */
  runId?: number | null
  /** When true, resolve as aborted (e.g. user stop / leave). */
  shouldAbort?: () => boolean
  /** Poll interval for shouldAbort when state is idle (ms). */
  pollMs?: number
  /** Maximum wait before one final snapshot refresh and timed_out (default 180s). */
  timeoutMs?: number
}

export interface UseHrAgentRunOptions {
  /** When true (default), register onUnmounted dispose that aborts subscription only. */
  autoDispose?: boolean
  /**
   * When true (default), resubscribe with last_event_seq after unexpected stream drops
   * while the run is still non-terminal. Bounded exponential backoff.
   */
  autoReconnect?: boolean
  /** Max reconnect attempts after a drop (default 8). */
  maxReconnectAttempts?: number
  /** Base backoff delay in ms (default 400). */
  reconnectBaseMs?: number
  /** Cap backoff delay in ms (default 8000). */
  reconnectMaxMs?: number
}

export interface UseHrAgentRunReturn {
  state: ShallowRef<HrAgentRunState>
  isSubscribing: Ref<boolean>
  isStarting: Ref<boolean>
  subscriptionError: Ref<string>
  /** Number of reconnect attempts since last successful event / fresh subscribe. */
  reconnectAttempt: Ref<number>
  startRun: (payload: CreateAgentRunRequest, options?: { autoSubscribe?: boolean }) => Promise<HrAgentRunState>
  subscribe: (runId: number, afterSeq?: number, options?: { isReconnect?: boolean }) => Promise<void>
  hydrateFromActive: (sessionId: number, options?: { autoSubscribe?: boolean }) => Promise<HrAgentRunState | null>
  hydrateFromRunId: (runId: number, options?: { autoSubscribe?: boolean }) => Promise<HrAgentRunState>
  cancel: (payload?: CancelAgentRunRequest) => Promise<HrAgentRunState>
  confirm: (payload?: ConfirmAgentRunRequest) => Promise<HrAgentRunState>
  applyEvent: (event: AgentRunEvent) => void
  /**
   * Wait until the current (or specified) run reaches a terminal status,
   * parking at waiting_confirmation, or is aborted by the caller / dispose.
   */
  waitUntilSettled: (options?: WaitUntilSettledOptions) => Promise<HrAgentRunSettlement>
  /** Abort subscription only — does not call cancelAgentRun. Also aborts waiters. */
  dispose: () => void
  reset: () => void
}

function isWaitingConfirmation(state: HrAgentRunState): boolean {
  if (state.isTerminal) return false
  if (state.status === 'waiting_confirmation') return true
  return Boolean(state.confirmation?.required)
}

/**
 * Vue composable owning durable HR Agent run create/subscribe/hydrate/cancel/confirm.
 * Aborting or disposing a subscription never cancels the backend run unless `cancel()` is called.
 * Unexpected SSE drops while a run is active trigger bounded reconnect with last_event_seq.
 */
export function useHrAgentRun(options: UseHrAgentRunOptions = {}): UseHrAgentRunReturn {
  const autoDispose = options.autoDispose !== false
  const autoReconnect = options.autoReconnect !== false
  const maxReconnectAttempts =
    options.maxReconnectAttempts && options.maxReconnectAttempts > 0
      ? options.maxReconnectAttempts
      : 8
  const reconnectBaseMs =
    options.reconnectBaseMs && options.reconnectBaseMs > 0 ? options.reconnectBaseMs : 400
  const reconnectMaxMs =
    options.reconnectMaxMs && options.reconnectMaxMs > 0 ? options.reconnectMaxMs : 8000

  const state = shallowRef<HrAgentRunState>(createInitialHrAgentRunState())
  const isSubscribing = ref(false)
  const isStarting = ref(false)
  const subscriptionError = ref('')
  const reconnectAttempt = ref(0)

  let abortController: AbortController | null = null
  let subscribeGeneration = 0
  let hydrateGeneration = 0
  let disposed = false
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  const waiters = new Set<(outcome: HrAgentRunSettlement) => void>()

  const setState = (next: HrAgentRunState): void => {
    state.value = next
  }

  const clearReconnectTimer = (): void => {
    if (reconnectTimer != null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  const abortAllWaiters = (): void => {
    const pending = [...waiters]
    waiters.clear()
    for (const finish of pending) {
      finish('aborted')
    }
  }

  const abortSubscriptionOnly = (): void => {
    subscribeGeneration += 1
    clearReconnectTimer()
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    isSubscribing.value = false
  }

  const applyEvent = (event: AgentRunEvent): void => {
    setState(reduceAgentRunEvent(state.value, event))
  }

  const reset = (): void => {
    disposed = true
    hydrateGeneration += 1
    reconnectAttempt.value = 0
    abortSubscriptionOnly()
    abortAllWaiters()
    subscriptionError.value = ''
    setState(createInitialHrAgentRunState())
  }

  const dispose = (): void => {
    // Explicit contract: dispose/abort subscription does not cancel the run.
    disposed = true
    hydrateGeneration += 1
    reconnectAttempt.value = 0
    abortSubscriptionOnly()
    abortAllWaiters()
  }

  const scheduleReconnect = (runId: number, generation: number): void => {
    if (!autoReconnect || disposed) return
    if (generation !== subscribeGeneration) return
    if (state.value.isTerminal || state.value.runId !== runId) return
    if (reconnectAttempt.value >= maxReconnectAttempts) {
      if (!subscriptionError.value) {
        subscriptionError.value = '订阅中断，请刷新页面重试'
      }
      return
    }

    const attempt = reconnectAttempt.value
    reconnectAttempt.value = attempt + 1
    const delay = Math.min(reconnectMaxMs, reconnectBaseMs * 2 ** attempt)
    clearReconnectTimer()
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      if (disposed || generation !== subscribeGeneration) return
      if (state.value.isTerminal || state.value.runId !== runId) return
      void subscribe(runId, state.value.lastEventSeq, { isReconnect: true }).catch(() => {
        /* onDone/catch path will schedule another attempt when still active */
      })
    }, delay)
  }

  const subscribe = async (
    runId: number,
    afterSeq?: number,
    subscribeOptions: { isReconnect?: boolean } = {},
  ): Promise<void> => {
    if (!Number.isFinite(runId) || runId <= 0) {
      throw new Error(t('validation.invalid_id'))
    }

    disposed = false
    if (!subscribeOptions.isReconnect) {
      reconnectAttempt.value = 0
    }

    abortSubscriptionOnly()
    const generation = subscribeGeneration
    const controller = new AbortController()
    abortController = controller
    isSubscribing.value = true
    subscriptionError.value = ''

    const cursor =
      afterSeq !== undefined && Number.isFinite(afterSeq)
        ? afterSeq
        : state.value.runId === runId
          ? state.value.lastEventSeq
          : 0

    if (state.value.runId !== runId) {
      setState(
        createInitialHrAgentRunState({
          runId,
          lastEventSeq: cursor > 0 ? cursor : 0,
        }),
      )
    }

    try {
      await subscribeAgentRunEvents(
        runId,
        cursor,
        {
          onEvent: (event) => {
            if (generation !== subscribeGeneration) return
            // Successful delivery resets reconnect budget.
            reconnectAttempt.value = 0
            applyEvent(event)
          },
          onError: (error) => {
            if (generation !== subscribeGeneration) return
            subscriptionError.value = error.message || '订阅失败'
          },
          onDone: () => {
            if (generation !== subscribeGeneration) return
            isSubscribing.value = false
            if (abortController === controller) {
              abortController = null
            }
            // Stream ended without intentional dispose: reconnect while run is active.
            if (
              !disposed &&
              !state.value.isTerminal &&
              state.value.runId === runId &&
              generation === subscribeGeneration
            ) {
              scheduleReconnect(runId, generation)
            }
          },
        },
        { signal: controller.signal },
      )
    } catch (error) {
      if (generation !== subscribeGeneration) return
      if (error instanceof Error && error.name === 'AbortError') {
        isSubscribing.value = false
        return
      }
      subscriptionError.value = error instanceof Error ? error.message : '订阅失败'
      isSubscribing.value = false
      if (
        !disposed &&
        !state.value.isTerminal &&
        state.value.runId === runId &&
        generation === subscribeGeneration
      ) {
        scheduleReconnect(runId, generation)
      }
      // Do not throw on reconnect path so fire-and-forget callers stay quiet;
      // still throw for the first non-reconnect attempt so callers can observe hard failures.
      if (subscribeOptions.isReconnect) {
        return
      }
      throw error
    } finally {
      if (generation === subscribeGeneration && abortController === controller) {
        isSubscribing.value = false
        abortController = null
      }
    }
  }

  const startRun = async (
    payload: CreateAgentRunRequest,
    startOptions: { autoSubscribe?: boolean } = {},
  ): Promise<HrAgentRunState> => {
    disposed = false
    isStarting.value = true
    subscriptionError.value = ''
    try {
      const response = await createAgentRun(payload)
      const next = hydrateFromSnapshot(response.run)
      setState(next)
      if (startOptions.autoSubscribe !== false && next.runId) {
        // Fire-and-forget subscription; callers can also await subscribe separately.
        void subscribe(next.runId, next.lastEventSeq).catch(() => {
          /* subscriptionError already set inside subscribe */
        })
      }
      return next
    } finally {
      isStarting.value = false
    }
  }

  const hydrateFromActive = async (
    sessionId: number,
    hydrateOptions: { autoSubscribe?: boolean } = {},
  ): Promise<HrAgentRunState | null> => {
    disposed = false
    const generation = ++hydrateGeneration
    const response = await getActiveAgentRun(sessionId, { silentError: true })
    // dispose/reset/newer hydrate invalidates this response (session switch race).
    if (generation !== hydrateGeneration || disposed) {
      return null
    }
    if (!response.has_active_run || !response.run) {
      // Do not wipe an unrelated current run unless same session.
      if (state.value.sessionId === sessionId) {
        abortSubscriptionOnly()
        setState(createInitialHrAgentRunState())
      }
      return null
    }
    const next = hydrateFromSnapshot(response.run)
    setState(next)
    if (hydrateOptions.autoSubscribe !== false && next.runId && !next.isTerminal) {
      void subscribe(next.runId, next.lastEventSeq).catch(() => {
        /* subscriptionError already set inside subscribe */
      })
    }
    return next
  }

  const hydrateFromRunId = async (
    runId: number,
    hydrateOptions: { autoSubscribe?: boolean } = {},
  ): Promise<HrAgentRunState> => {
    disposed = false
    const generation = ++hydrateGeneration
    const response = await getAgentRun(runId)
    if (generation !== hydrateGeneration || disposed) {
      return state.value
    }
    const next = hydrateFromSnapshot(response.run)
    setState(next)
    if (hydrateOptions.autoSubscribe !== false && next.runId && !next.isTerminal) {
      void subscribe(next.runId, next.lastEventSeq).catch(() => {
        /* subscriptionError already set inside subscribe */
      })
    }
    return next
  }

  const cancel = async (payload: CancelAgentRunRequest = {}): Promise<HrAgentRunState> => {
    const runId = state.value.runId
    if (!runId) {
      throw new Error(t('common.invalid_request'))
    }
    // Explicit cancel command — distinct from subscription abort.
    const response = await cancelAgentRun(runId, payload)
    const next = hydrateFromSnapshot(response.run)
    // Preserve text already reduced from events if snapshot is thinner.
    setState({
      ...next,
      assistantText: next.assistantText || state.value.assistantText,
      processText: next.processText || state.value.processText,
      lastEventSeq: Math.max(next.lastEventSeq, state.value.lastEventSeq),
      resultMetadata: next.resultMetadata ?? state.value.resultMetadata,
      confirmation: next.confirmation ?? state.value.confirmation,
    })
    return state.value
  }

  const confirm = async (payload: ConfirmAgentRunRequest = {}): Promise<HrAgentRunState> => {
    const runId = state.value.runId
    if (!runId) {
      throw new Error(t('common.invalid_request'))
    }
    const response = await confirmAgentRun(runId, payload)
    const next = hydrateFromSnapshot(response.run)
    setState({
      ...next,
      assistantText: next.assistantText || state.value.assistantText,
      processText: next.processText || state.value.processText,
      lastEventSeq: Math.max(next.lastEventSeq, state.value.lastEventSeq),
      resultMetadata: next.resultMetadata ?? state.value.resultMetadata,
      // Confirm clears parked confirmation on the backend; drop local parking flag promptly.
      confirmation: next.confirmation,
    })
    if (!state.value.isTerminal) {
      void subscribe(runId, state.value.lastEventSeq).catch(() => {
        /* subscriptionError already set inside subscribe */
      })
    }
    return state.value
  }

  const waitUntilSettled = (options: WaitUntilSettledOptions = {}): Promise<HrAgentRunSettlement> => {
    const pollMs = options.pollMs && options.pollMs > 0 ? options.pollMs : 50
    const timeoutMs = options.timeoutMs && options.timeoutMs > 0 ? options.timeoutMs : 180_000

    return new Promise((resolve) => {
      let settled = false
      let pollTimer: ReturnType<typeof setInterval> | null = null
      let timeoutTimer: ReturnType<typeof setTimeout> | null = null
      // Assigned after watch registration; finish may run on immediate evaluate.
      let stopWatch: (() => void) | null = null

      const finish = (outcome: HrAgentRunSettlement) => {
        if (settled) return
        settled = true
        waiters.delete(finish)
        stopWatch?.()
        stopWatch = null
        if (pollTimer) {
          clearInterval(pollTimer)
          pollTimer = null
        }
        if (timeoutTimer) {
          clearTimeout(timeoutTimer)
          timeoutTimer = null
        }
        resolve(outcome)
      }

      waiters.add(finish)

      // Already disposed before waiter registered.
      if (disposed) {
        finish('aborted')
        return
      }

      const evaluate = (): boolean => {
        if (disposed || options.shouldAbort?.()) {
          finish('aborted')
          return true
        }
        const current = state.value
        const targetRunId = options.runId
        if (
          targetRunId != null &&
          targetRunId > 0 &&
          current.runId != null &&
          current.runId !== targetRunId
        ) {
          return false
        }
        if (current.isTerminal) {
          finish('terminal')
          return true
        }
        if (isWaitingConfirmation(current)) {
          finish('waiting_confirmation')
          return true
        }
        return false
      }

      stopWatch = watch(
        () => state.value,
        () => {
          evaluate()
        },
        { deep: true, immediate: true, flush: 'sync' },
      )

      if (!settled) {
        pollTimer = setInterval(() => {
          evaluate()
        }, pollMs)
        timeoutTimer = setTimeout(async () => {
          if (settled) return
          const targetRunId = options.runId || state.value.runId
          if (targetRunId && targetRunId > 0) {
            try {
              const response = await getAgentRun(targetRunId)
              const next = hydrateFromSnapshot(response.run)
              setState({
                ...next,
                assistantText: next.assistantText || state.value.assistantText,
                processText: next.processText || state.value.processText,
                lastEventSeq: Math.max(next.lastEventSeq, state.value.lastEventSeq),
                resultMetadata: next.resultMetadata ?? state.value.resultMetadata,
              })
              if (evaluate()) return
            } catch {
              // The timeout result remains authoritative when the final refresh is unavailable.
            }
          }
          finish('timed_out')
        }, timeoutMs)
      }
    })
  }

  if (autoDispose) {
    onUnmounted(() => {
      dispose()
    })
  }

  return {
    state,
    isSubscribing,
    isStarting,
    subscriptionError,
    reconnectAttempt,
    startRun,
    subscribe,
    hydrateFromActive,
    hydrateFromRunId,
    cancel,
    confirm,
    applyEvent,
    waitUntilSettled,
    dispose,
    reset,
  }
}

/** Test helper: apply a batch of events to composable-local state without network. */
export function reduceEventsForTest(
  initial: HrAgentRunState,
  events: AgentRunEvent[],
): HrAgentRunState {
  return applyAgentRunEvents(initial, events)
}
