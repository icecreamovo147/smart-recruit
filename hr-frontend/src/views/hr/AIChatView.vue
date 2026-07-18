<script lang="ts">
import type { CreateAgentRunRequest } from '@/types/agentRun'

export const buildApplicationAnalysisMessage = (candidateName?: string, jobTitle?: string): string => {
  const candidate = candidateName?.trim() || '该候选人'
  const job = jobTitle?.trim() || '该岗位'
  return `请分析${candidate}投递${job}的简历与岗位匹配度，并基于真实候选人、岗位和匹配评估数据给出结论。`
}

interface ApplicationAnalysisMessageLike {
  role?: string
  content?: string
}

export const resolveApplicationAnalysisMessage = (
  returnedMessages: ApplicationAnalysisMessageLike[],
  candidateName?: string,
  jobTitle?: string,
): string => returnedMessages
  .find((message) => message.role === 'user' && message.content?.trim())
  ?.content?.trim() || buildApplicationAnalysisMessage(candidateName, jobTitle)

export const buildApplicationAnalysisRunRequest = (input: {
  sessionId: number
  message: string
  applicationId: number
  clientRequestId: string
  modelId?: number
  skillCapabilityKeys?: string[]
}): CreateAgentRunRequest => ({
  session_id: input.sessionId,
  message: input.message.trim(),
  action_type: 'analyze_application',
  application_id: input.applicationId,
  client_request_id: input.clientRequestId,
  ...(input.modelId != null ? { model_id: input.modelId } : {}),
  ...(input.skillCapabilityKeys?.length ? { skill_capability_keys: [...input.skillCapabilityKeys] } : {}),
})
</script>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import { createApplicationAnalysisSession, createSession, deleteSession, getSessionMessages, listSessions, listSkillCapabilities, previewSessionContext, updateSession } from '@/api/ai'
import { listAvailableAgentSkills } from '@/api/agentSkill'
import { updateApplicationStatus } from '@/api/application'
import { listAvailableModels } from '@/api/llm'
import {
  bindRunStateToChatUi,
  createClientRequestId,
  executeConfirmChatRun,
  executeCreateChatRun,
  toAgentSkillSelectionPayload,
  type DurableChatUiBinder,
} from '@/components/hr/ai/agentRunChatFlow'
import AgentTracePanel from '@/components/AgentTracePanel.vue'
import ConversationSidebar from '@/components/chat/ConversationSidebar.vue'
import ConversationHeader from '@/components/chat/ConversationHeader.vue'
import ChatMessageList from '@/components/chat/ChatMessageList.vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import { useHrAgentRun } from '@/composables/useHrAgentRun'
import type { AgentRunResultMetadata } from '@/types/agentRun'
import type { AgentSkillSelectionPayload, ChatMessageSkill, ChatSessionListItem, Session, CandidateOption, StreamPayload, ContextUsageInfo } from '@/types/ai'
import type { CapabilityInfo } from '@/types/agent'
import type { LlmModel } from '@/types/llm'
import type { AvailableAgentSkill } from '@/types/agentSkill'
import { sanitizeAssistantProcessText } from '@/utils/hrAssistantProcess'
import {
  contextGuardCodeFrom,
  contextGuardMessage,
  contextUsageBelongsToSession,
  resolveLiveContextUsage,
  resolveSessionContextUsage,
} from '@/utils/contextUsage'

interface MessageItem {
  role: string
  content: string
  model_name?: string
  skill?: ChatMessageSkill
  skills?: ChatMessageSkill[]
  skill_id?: string | number
  skill_name?: string
  skill_command?: string
  skillId?: string | number
  skillName?: string
  skillCommand?: string
  agent_skill_ids?: number[]
  agent_skill_names?: string[]
  agentSkillIds?: number[]
  agentSkillNames?: string[]
  pending?: boolean
  failed?: boolean
  retryDisabled?: boolean
  errorCode?: string
  waitingText?: string
  process_content?: string
  processContent?: string
  context_usage?: ContextUsageInfo
  contextUsage?: ContextUsageInfo
  candidateOptions?: CandidateOption[]
  agentSkillSelection?: AgentSkillSelectionPayload
  skillSelectionRequest?: SkillSelectionRequest
  skillSelectionConfirmed?: boolean
}

interface SkillSelectionRequest {
  message: string
  sessionId: number
  modelId: number | null
  messageId?: number
  runId?: number
}

const route = useRoute()
const router = useRouter()
const sessions = ref<Session[]>([])
const messages = ref<MessageItem[]>([])
const currentSession = ref<Session | null>(null)
const input = ref('')
const loading = ref(false)
const streaming = ref(false)
const sessionLoading = ref(false)
const menuSessionId = ref(0)
/** 移动端：会话列表抽屉是否打开 */
const sessionSidebarOpen = ref(false)
/** 桌面端：会话列表是否收起 */
const sessionSidebarCollapsed = ref(false)
/** 桌面端收起后，悬停左侧热区临时浮出列表 */
const sessionSidebarPeek = ref(false)
const CHAT_SIDEBAR_COLLAPSED_KEY = 'hr-ai-chat-sidebar-collapsed'
let sidebarPeekLeaveTimer: ReturnType<typeof setTimeout> | null = null

const readSidebarCollapsed = (): boolean => {
  try {
    return localStorage.getItem(CHAT_SIDEBAR_COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

const persistSidebarCollapsed = (collapsed: boolean): void => {
  try {
    localStorage.setItem(CHAT_SIDEBAR_COLLAPSED_KEY, collapsed ? '1' : '0')
  } catch {
    // localStorage unavailable
  }
}

sessionSidebarCollapsed.value = readSidebarCollapsed()

const isMobileChatViewport = (): boolean =>
  typeof window !== 'undefined' && window.matchMedia('(max-width: 768px)').matches

const clearSidebarPeekTimer = (): void => {
  if (sidebarPeekLeaveTimer) {
    clearTimeout(sidebarPeekLeaveTimer)
    sidebarPeekLeaveTimer = null
  }
}

const openSidebarPeek = (): void => {
  if (!sessionSidebarCollapsed.value || isMobileChatViewport()) return
  clearSidebarPeekTimer()
  // 已打开时不要重复写 true，避免无意义重渲染打断 transition
  if (!sessionSidebarPeek.value) {
    sessionSidebarPeek.value = true
  }
}

const scheduleCloseSidebarPeek = (): void => {
  if (!sessionSidebarCollapsed.value) return
  clearSidebarPeekTimer()
  // 稍长延迟：从热区移入列表时不会闪关；也避免快速抖动手势导致半动画空白态
  sidebarPeekLeaveTimer = setTimeout(() => {
    sessionSidebarPeek.value = false
    sidebarPeekLeaveTimer = null
  }, 220)
}

const resetSidebarPeek = (): void => {
  clearSidebarPeekTimer()
  sessionSidebarPeek.value = false
}
const candidateName = ref('')
const candidatePosition = ref('')
const userAborted = ref(false)
const activeRunToken = ref(0)
const statusBarExpanded = ref(true)
const modelName = ref('')
const modelList = ref<LlmModel[]>([])
const selectedModelId = ref<number | null>(null)
const dataSource = ref('招聘业务数据库')
const tracePanelVisible = ref(false)
const agentSkills = ref<AvailableAgentSkill[]>([])
const selectedAgentSkillIds = ref<number[]>([])
const skillCapabilities = ref<CapabilityInfo[]>([])
const selectedSkillKeys = ref<string[]>([])
const contextUsage = ref<ContextUsageInfo | null>(null)
const contextPreviewing = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const listRef = ref<any>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null
let streamTypewriterTimer: ReturnType<typeof setInterval> | null = null
let contextPreviewTimer: ReturnType<typeof setTimeout> | null = null
let contextPreviewController: AbortController | null = null
let contextPreviewVersion = 0
const contextUsageStoragePrefix = 'hr-ai-context-usage:'

// Durable HR Agent runtime (TASK-HARS-007). Legacy sendMessageStream remains in api/ai.ts for rollout compatibility.
const agentRun = useHrAgentRun()

type StreamTextTarget = 'content' | 'process'

interface StreamTextQueue {
  index: number
  target: StreamTextTarget
  text: string
}

const streamTextQueue: StreamTextQueue[] = []

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

const renderMarkdown = (content: string): string => {
  const raw = DOMPurify.sanitize(md.render(content || ''), {
    ALLOWED_TAGS: [
      'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
      'p', 'br', 'hr',
      'strong', 'b', 'em', 'i', 'u', 's', 'del',
      'ul', 'ol', 'li',
      'code', 'pre',
      'a',
      'table', 'thead', 'tbody', 'tr', 'th', 'td',
      'blockquote',
    ],
    ALLOWED_ATTR: ['href', 'title', 'target'],
    ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|tel):|[^a-z]|[a-z+.-]+(?:[^a-z+.\-:]|$))/i,
  })
  // Add rel="noopener noreferrer" to external links opened in new tabs.
  return raw.replace(/<a\s/g, '<a rel="noopener noreferrer" ')
}

const waitingText = (message: MessageItem): string => {
  if (message?.waitingText) return message.waitingText
  return currentSession.value?.application_id ? '分析中' : '响应中'
}

const normalizeSession = (item: ChatSessionListItem): Session => ({
  id: item.session_id || 0,
  title: item.title || '新对话',
  application_id: item.application_id || 0,
  updated_at: item.updated_at || item.created_at || '',
  latest_context_usage: item.latest_context_usage || item.latestContextUsage,
  selected_model_id: item.selected_model_id ?? item.selectedModelId,
})

const agentSkillLabel = (skill: AvailableAgentSkill) => skill.display_name || skill.name

/** Resolve user-facing Skill label: prefer catalog display_name over stored technical name. */
const resolveAgentSkillLabel = (id?: number | string, fallbackName?: string): string => {
  const numericId = typeof id === 'string' ? Number(id) : id
  if (typeof numericId === 'number' && Number.isFinite(numericId) && numericId > 0) {
    const byId = agentSkills.value.find((skill) => skill.id === numericId)
    if (byId) return agentSkillLabel(byId)
  }
  const key = (fallbackName || '').trim()
  if (key) {
    const byKey = agentSkills.value.find(
      (skill) => skill.name === key || skill.display_name === key,
    )
    if (byKey) return agentSkillLabel(byKey)
  }
  return key
}

const toMessageSkill = (id?: number | string, fallbackName?: string): ChatMessageSkill | undefined => {
  const name = resolveAgentSkillLabel(id, fallbackName)
  if (!name) return undefined
  return {
    ...(id !== undefined && id !== null && id !== '' ? { id } : {}),
    name,
    command: `/${name}`,
  }
}

const buildMessageSkills = (ids: number[]): ChatMessageSkill[] =>
  ids
    .map((id) => toMessageSkill(id))
    .filter((skill): skill is ChatMessageSkill => Boolean(skill))

const normalizeSkillMeta = (message: Partial<MessageItem>, fallback?: MessageItem): ChatMessageSkill | undefined => {
  if (message.skill?.name || message.skill?.id != null) {
    const resolved = toMessageSkill(message.skill?.id, message.skill?.name)
    if (resolved) return resolved
  }
  const name = message.skill_name || message.skillName
  const id = message.skill_id ?? message.skillId
  if (name || id != null) {
    const resolved = toMessageSkill(id, name)
    if (resolved) return resolved
  }
  return fallback?.skill
}

const normalizeSkillsMeta = (message: Partial<MessageItem>, fallback?: MessageItem): ChatMessageSkill[] | undefined => {
  if (Array.isArray(message.skills) && message.skills.length > 0) {
    const resolved = message.skills
      .map((skill) => toMessageSkill(skill?.id, skill?.name))
      .filter((skill): skill is ChatMessageSkill => Boolean(skill))
    if (resolved.length > 0) return resolved
  }
  const agentSkillNames = message.agent_skill_names || message.agentSkillNames
  const agentSkillIds = message.agent_skill_ids || message.agentSkillIds || []
  if (Array.isArray(agentSkillNames) && agentSkillNames.length > 0) {
    const resolved = agentSkillNames
      .map((name, index) => toMessageSkill(agentSkillIds[index], name))
      .filter((skill): skill is ChatMessageSkill => Boolean(skill))
    if (resolved.length > 0) return resolved
  }
  if (Array.isArray(agentSkillIds) && agentSkillIds.length > 0) {
    const resolved = agentSkillIds
      .map((id) => toMessageSkill(id))
      .filter((skill): skill is ChatMessageSkill => Boolean(skill))
    if (resolved.length > 0) return resolved
  }
  if (fallback?.skills?.length) return fallback.skills
  const skill = normalizeSkillMeta(message, fallback)
  return skill ? [skill] : undefined
}

const normalizeMessage = (message: Partial<MessageItem>, fallback?: MessageItem): MessageItem => {
  const skill = normalizeSkillMeta(message, fallback)
  const skills = normalizeSkillsMeta(message, fallback)
  const processContent = sanitizeAssistantProcessText(message.processContent || message.process_content || fallback?.processContent || '')
  const contextUsageSnapshot = message.context_usage || message.contextUsage || fallback?.context_usage || fallback?.contextUsage
  const {
    processContent: _processContent,
    process_content: _processContentSnake,
    ...messageWithoutProcessContent
  } = message
  return {
    ...(messageWithoutProcessContent as MessageItem),
    ...(skill ? { skill } : {}),
    ...(skills?.length ? { skills } : {}),
    ...(processContent ? { processContent } : {}),
    ...(contextUsageSnapshot ? { context_usage: contextUsageSnapshot } : {}),
  }
}

const normalizeMessages = (list: Partial<MessageItem>[] = [], fallbackMessages: MessageItem[] = []): MessageItem[] =>
  list.map((message, index) => normalizeMessage(message, fallbackMessages[index]))

const contextUsageStorageKey = (sessionId: number): string => `${contextUsageStoragePrefix}${sessionId}`

const rememberContextUsage = (sessionId: number, usage: ContextUsageInfo) => {
  try {
    sessionStorage.setItem(contextUsageStorageKey(sessionId), JSON.stringify(usage))
  } catch {
    // Best-effort UI state; ignore quota or privacy-mode failures.
  }
}

const restorePersistedContextUsage = (session: Session, items: MessageItem[]) => {
  const persisted = resolveSessionContextUsage(session, items)
  if (persisted) {
    contextUsage.value = persisted
    rememberContextUsage(session.id, contextUsage.value)
    return
  }
  // Server snapshots are authoritative. A session without usage must show a
  // neutral placeholder instead of reviving a stale browser cache entry.
  contextUsage.value = null
}

const forgetContextUsage = (sessionId: number) => {
  try {
    sessionStorage.removeItem(contextUsageStorageKey(sessionId))
  } catch {
    // Best-effort cleanup only.
  }
}

const scrollBottom = async () => {
  await nextTick()
  listRef.value?.scrollToBottom()
}

const writeAssistantText = (index: number, target: StreamTextTarget, text: string) => {
  const message = messages.value[index]
  if (!message || !text) return
  if (target === 'process') {
    const nextProcessContent = sanitizeAssistantProcessText(`${message.processContent || ''}${text}`)
    messages.value[index] = {
      ...message,
      processContent: nextProcessContent,
    }
  } else {
    messages.value[index] = { ...message, content: `${message.content || ''}${text}`, pending: false }
  }
  scrollBottom()
}

const drainStreamTextQueue = () => {
  const item = streamTextQueue[0]
  if (!item) {
    if (streamTypewriterTimer) {
      clearInterval(streamTypewriterTimer)
      streamTypewriterTimer = null
    }
    return
  }
  const chars = [...item.text]
  const take = Math.min(chars.length, chars.length > 24 ? 3 : 2)
  const chunk = chars.slice(0, take).join('')
  item.text = chars.slice(take).join('')
  writeAssistantText(item.index, item.target, chunk)
  if (!item.text) {
    streamTextQueue.shift()
  }
}

const ensureStreamTypewriter = () => {
  if (streamTypewriterTimer) return
  streamTypewriterTimer = setInterval(drainStreamTextQueue, 18)
}

const enqueueAssistantText = (index: number, target: StreamTextTarget, text: string) => {
  if (!text) return
  const last = streamTextQueue[streamTextQueue.length - 1]
  if (last && last.index === index && last.target === target) {
    last.text += text
  } else {
    streamTextQueue.push({ index, target, text })
  }
  ensureStreamTypewriter()
}

const flushAssistantTextQueue = (index?: number) => {
  const remaining: StreamTextQueue[] = []
  for (const item of streamTextQueue) {
    if (index != null && item.index !== index) {
      remaining.push(item)
      continue
    }
    writeAssistantText(item.index, item.target, item.text)
  }
  streamTextQueue.splice(0, streamTextQueue.length, ...remaining)
  if (streamTextQueue.length === 0 && streamTypewriterTimer) {
    clearInterval(streamTypewriterTimer)
    streamTypewriterTimer = null
  }
}

const hasAssistantTextQueued = (index: number): boolean =>
  streamTextQueue.some((item) => item.index === index && item.text.length > 0)

const waitForAssistantTextQueue = (index: number): Promise<void> => new Promise((resolve) => {
  if (!hasAssistantTextQueued(index)) {
    resolve()
    return
  }
  const timer = setInterval(() => {
    if (!hasAssistantTextQueued(index)) {
      clearInterval(timer)
      resolve()
    }
  }, 30)
})

const clearAssistantTextQueue = () => {
  streamTextQueue.splice(0, streamTextQueue.length)
  if (streamTypewriterTimer) {
    clearInterval(streamTypewriterTimer)
    streamTypewriterTimer = null
  }
}

const appendAssistantDelta = (index: number, delta: string) => {
  enqueueAssistantText(index, 'content', delta)
}

const appendAssistantProcess = (index: number, delta: string) => {
  const text = sanitizeAssistantProcessText(delta)
  if (!text) return
  enqueueAssistantText(index, 'process', text)
}

const markAssistantError = (index: number, error: Error | null, explicitCode?: string) => {
  clearAssistantTextQueue()
  const message = messages.value[index]
  const errorWithGuard = error as (Error & { contextGuardCode?: string }) | null
  const guardCode = contextGuardCodeFrom(explicitCode, errorWithGuard?.contextGuardCode, error?.message, message?.errorCode)
  const content = contextGuardMessage(guardCode) || error?.message || '响应中断，请稍后重试'
  const retryDisabled = Boolean(guardCode)
  if (message?.role === 'assistant') {
    messages.value[index] = { ...message, content, pending: false, failed: true, retryDisabled, ...(guardCode ? { errorCode: guardCode } : {}) }
  } else {
    messages.value.push({ role: 'assistant', content, failed: true, retryDisabled, ...(guardCode ? { errorCode: guardCode } : {}) })
  }
  scrollBottom()
}

const safeAgentRunErrorMessage = (
  error: Error | undefined,
  errorType: string,
  errorMessage: string,
  fallback: string,
): string => contextGuardMessage(contextGuardCodeFrom(errorType, errorMessage, error?.message))
  || error?.message
  || fallback

const beginAgentRun = (): number => {
  activeRunToken.value += 1
  userAborted.value = false
  return activeRunToken.value
}

const isActiveAgentRun = (token: number): boolean => activeRunToken.value === token

const resultMetaToStreamPayload = (
  meta: AgentRunResultMetadata | null | undefined,
  sessionId?: number | null,
): StreamPayload => ({
  action: meta?.action,
  application_id: meta?.application_id,
  action_status: meta?.action_status,
  candidate_name: meta?.candidate_name,
  job_title: meta?.job_title,
  status: meta?.status,
  candidate_options: meta?.candidate_options,
  session_id: sessionId ?? undefined,
  context_usage: (meta?.context_usage as ContextUsageInfo | undefined) || undefined,
})

const makeChatUiBinder = (
  assistantIndex: number,
  expectedSessionId: number | undefined = currentSession.value?.id,
): DurableChatUiBinder => ({
  onAssistantDelta: (delta) => appendAssistantDelta(assistantIndex, delta),
  onAssistantSnapshot: (text) => {
    flushAssistantTextQueue(assistantIndex)
    const msg = messages.value[assistantIndex]
    if (!msg) return
    messages.value[assistantIndex] = {
      ...msg,
      content: text,
      pending: !text,
    }
    scrollBottom()
  },
  onProcessDelta: (delta) => appendAssistantProcess(assistantIndex, delta),
  onProcessSnapshot: (text) => {
    for (let i = streamTextQueue.length - 1; i >= 0; i--) {
      const item = streamTextQueue[i]
      if (item.index === assistantIndex && item.target === 'process') {
        streamTextQueue.splice(i, 1)
      }
    }
    const msg = messages.value[assistantIndex]
    if (!msg) return
    messages.value[assistantIndex] = {
      ...msg,
      processContent: sanitizeAssistantProcessText(text),
    }
    scrollBottom()
  },
  onModelName: (name) => {
    modelName.value = name
    const msg = messages.value[assistantIndex]
    if (msg) {
      messages.value[assistantIndex] = { ...msg, model_name: name }
    }
  },
  onContextUsage: (usage, sessionId) => {
    handleContextUsage({
      context_usage: usage,
      session_id: sessionId || expectedSessionId,
    })
  },
  onCandidateOptions: (options) => {
    const msg = messages.value[assistantIndex]
    if (!msg) return
    messages.value[assistantIndex] = {
      ...msg,
      candidateOptions: options,
      pending: false,
    }
    scrollBottom()
  },
  onResultMetadata: (meta) => {
    if (!meta?.context_usage) return
    handleContextUsage({
      context_usage: meta.context_usage as ContextUsageInfo,
      session_id: expectedSessionId,
    })
  },
  onRunError: (errorType, errorMessage) => {
    const guardCode = contextGuardCodeFrom(errorType, errorMessage)
    if (guardCode) {
      markAssistantError(assistantIndex, new Error(contextGuardMessage(guardCode) || ''), guardCode)
    }
  },
})

const finishAgentRunUi = (token: number) => {
  if (!isActiveAgentRun(token)) return
  loading.value = false
  streaming.value = false
  userAborted.value = false
}

const applySkillSelectionFromRun = (
  assistantIndex: number,
  text: string,
  session: Session,
  runId: number | null | undefined,
) => {
  const selection = toAgentSkillSelectionPayload(agentRun.state.value.confirmation)
  if (!selection) return false
  setSkillSelectionMessage(assistantIndex, text, session, selection, runId || undefined)
  return true
}

const refreshSessions = async () => {
  const data = await listSessions({ page: 1, page_size: 50 })
  sessions.value = (data.list || []).map(normalizeSession)
  if (data.model_name) {
    modelName.value = data.model_name
  }
}

/**
 * Ensure an assistant message slot exists for a restored in-flight run and seed
 * snapshot text (FR-015). Prefer reusing the last assistant bubble when it is
 * empty/pending so history is not duplicated.
 */
const ensureRestoreAssistantSlot = (
  session: Session,
  run: { assistantText: string; processText: string; modelName: string; isTerminal: boolean },
): number => {
  const lastIndex = messages.value.length - 1
  const last = lastIndex >= 0 ? messages.value[lastIndex] : null
  if (
    last?.role === 'assistant' &&
    !last.agentSkillSelection &&
    !last.failed &&
    (
      last.pending ||
      !last.content ||
      (run.assistantText && run.assistantText.startsWith(last.content || ''))
    )
  ) {
    messages.value[lastIndex] = {
      ...last,
      content: run.assistantText || last.content || '',
      processContent: sanitizeAssistantProcessText(run.processText || last.processContent || last.process_content || ''),
      pending: !run.isTerminal && !(run.assistantText && !last.pending && last.content === run.assistantText),
      waitingText: last.waitingText || (session.application_id ? '分析中' : '响应中'),
      model_name: run.modelName || last.model_name,
    }
    return lastIndex
  }

  messages.value.push({
    role: 'assistant',
    content: run.assistantText || '',
    processContent: sanitizeAssistantProcessText(run.processText || ''),
    pending: true,
    waitingText: session.application_id ? '分析中' : '响应中',
    ...(run.modelName ? { model_name: run.modelName } : {}),
  })
  return messages.value.length - 1
}

const lastUserMessageText = (): string => {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    if (messages.value[i]?.role === 'user') {
      return messages.value[i].content || ''
    }
  }
  return ''
}

/**
 * Refresh / route remount recovery (TASK-HARS-008 / FR-015):
 * query active run, hydrate reducer snapshot, subscribe after last_event_seq,
 * and bind UI so cancel / confirm / complete still work.
 */
const restoreActiveRunForSession = async (session: Session) => {
  if (!session?.id) return

  const token = beginAgentRun()
  let stopBinder: (() => void) | null = null
  try {
    const restored = await agentRun.hydrateFromActive(session.id, { autoSubscribe: true })
    if (!isActiveAgentRun(token) || userAborted.value) return
    if (currentSession.value?.id !== session.id) return
    if (!restored?.runId || restored.isTerminal) return

    const assistantIndex = ensureRestoreAssistantSlot(session, restored)
    scrollBottom()

    if (
      restored.status === 'waiting_confirmation' ||
      Boolean(restored.confirmation?.required)
    ) {
      const seeded = applySkillSelectionFromRun(
        assistantIndex,
        lastUserMessageText(),
        session,
        restored.runId,
      )
      if (!seeded) {
        // Parked without full confirmation payload — still show non-streaming state.
        loading.value = false
        streaming.value = false
        const msg = messages.value[assistantIndex]
        if (msg) {
          messages.value[assistantIndex] = { ...msg, pending: false }
        }
      }
      return
    }

    loading.value = true
    streaming.value = true
    stopBinder = bindRunStateToChatUi(agentRun.state, makeChatUiBinder(assistantIndex))

    const settlement = await agentRun.waitUntilSettled({
      runId: restored.runId,
      shouldAbort: () => userAborted.value || !isActiveAgentRun(token),
    })

    if (!isActiveAgentRun(token) || userAborted.value || settlement === 'aborted') return

    if (settlement === 'waiting_confirmation') {
      applySkillSelectionFromRun(
        assistantIndex,
        lastUserMessageText(),
        session,
        agentRun.state.value.runId || restored.runId,
      )
      return
    }

    await waitForAssistantTextQueue(assistantIndex)
    const finalPayload = resultMetaToStreamPayload(
      agentRun.state.value.resultMetadata,
      agentRun.state.value.sessionId || session.id,
    )
    if (finalPayload.session_id && currentSession.value) {
      currentSession.value = { ...currentSession.value, id: finalPayload.session_id }
    }
    await confirmAction(finalPayload)

    // Reconcile transient bubble with persisted history after completion (AC completion).
    const sid = finalPayload.session_id || session.id
    const data = await getSessionMessages(sid, { page: 1, page_size: 100 })
    messages.value = normalizeMessages(data.list || [], messages.value)
    await refreshSessions()
    scrollBottom()
  } catch (error: unknown) {
    if (userAborted.value || !isActiveAgentRun(token)) return
    // Restore failure is recoverable — keep loaded chat history intact.
    console.warn('[AIChatView] restoreActiveRunForSession failed', error)
  } finally {
    stopBinder?.()
    if (isActiveAgentRun(token)) {
      // Keep loading false when parked on skill selection; applySkillSelection already cleared flags.
      if (
        agentRun.state.value.status === 'waiting_confirmation' ||
        Boolean(agentRun.state.value.confirmation?.required)
      ) {
        loading.value = false
        streaming.value = false
      } else {
        finishAgentRunUi(token)
      }
    }
  }
}

const selectSession = async (session: Session) => {
  if (!session) return
  // Switching sessions: dispose local subscription only (backend run continues).
  userAborted.value = true
  activeRunToken.value += 1
  agentRun.dispose()
  cancelContextPreview()
  clearAssistantTextQueue()
  resetContextUsage()
  currentSession.value = session
  const storedModelId = session.selected_model_id ?? session.selectedModelId
  selectedModelId.value = storedModelId == null
    ? (session.latest_context_usage?.model_id || session.latestContextUsage?.model_id || null)
    : (storedModelId > 0 ? storedModelId : null)
  sessionLoading.value = true
  loading.value = false
  streaming.value = false
  // 移动端选中会话后收起抽屉，避免遮挡对话区
  if (isMobileChatViewport()) {
    sessionSidebarOpen.value = false
  }
  try {
    const data = await getSessionMessages(session.id, { page: 1, page_size: 100 })
    if (currentSession.value?.id !== session.id) return
    messages.value = normalizeMessages(data.list || [])
    restorePersistedContextUsage(session, messages.value)
    if (!contextUsageMatchesSelectedModel()) {
      requestContextPreview(false)
    }
    router.replace({ path: '/hr/ai', query: { session_id: String(session.id) } })
    scrollBottom()
  } finally {
    sessionLoading.value = false
    userAborted.value = false
  }
  // After history load, re-attach any still-active durable run for this session.
  await restoreActiveRunForSession(session)
}

let typewriterTimer: ReturnType<typeof setInterval> | null = null
const typewriterContent = ref('')

const startTypewriter = (fullText: string, targetIndex: number) => {
  if (typewriterTimer) clearInterval(typewriterTimer)
  typewriterContent.value = ''
  let pos = 0
  const chars = [...fullText]
  typewriterTimer = setInterval(() => {
    if (!chars[pos]) {
      clearInterval(typewriterTimer!)
      typewriterTimer = null
      return
    }
    // 一次输出 1~3 个字符，模拟流式速度变化
    const chunk = chars.slice(pos, pos + (Math.random() > 0.6 ? 1 : 2)).join('')
    pos += chunk.length
    const msg = messages.value[targetIndex]
    if (msg) {
      messages.value[targetIndex] = { ...msg, content: (msg.content || '') + chunk, pending: false }
    }
    scrollBottom()
  }, 30)
}

const pollCurrentSession = (expectedLength: number) => {
  if (pollTimer) clearInterval(pollTimer)
  let count = 0
  let startedTypewriter = false
  pollTimer = setInterval(async () => {
    if (!currentSession.value) return
    count += 1
    const data = await getSessionMessages(currentSession.value.id, { page: 1, page_size: 100 })
    const nextMessages = normalizeMessages(data.list || [], messages.value)

    // 检测是否有新消息或最后一条 assistant 消息内容更新
    if (!startedTypewriter && nextMessages.length > 0) {
      const lastMsg = nextMessages[nextMessages.length - 1]
      const isNewMessage = nextMessages.length > messages.value.length
      const hasRealContent = lastMsg.role === 'assistant' &&
        lastMsg.content &&
        lastMsg.content !== '好的，正在分析中。'
      if (hasRealContent && (isNewMessage || lastMsg.content !== messages.value[messages.value.length - 1]?.content)) {
        startedTypewriter = true
        messages.value = nextMessages
        const idx = messages.value.length - 1
        const fullText = lastMsg.content
        messages.value[idx] = { ...messages.value[idx], content: '', pending: true }
        startTypewriter(fullText, idx)
      }
    }

    // 新消息数量达到预期 或 超时 90s
    if (nextMessages.length >= expectedLength || count >= 90) {
      clearInterval(pollTimer!)
      pollTimer = null
      loading.value = false
      streaming.value = false
      if (typewriterTimer) {
        clearInterval(typewriterTimer)
        typewriterTimer = null
        const data2 = await getSessionMessages(currentSession.value!.id, { page: 1, page_size: 100 })
        messages.value = normalizeMessages(data2.list || [], messages.value)
      }
      await refreshSessions()
    }
  }, 1000)
}

const createNewSession = async () => {
  const data = await createSession({ title: '新对话' })
  const session = normalizeSession(data.session)
  sessions.value = [session, ...sessions.value]
  messages.value = []
  resetContextUsage()
  await selectSession(session)
}

const renameSession = async (session: Session) => {
  try {
    const { value } = await ElMessageBox.prompt('请输入新的会话名称', '重命名', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputValue: session.title,
      inputValidator: (val: string) => val.trim() ? true : '名称不能为空',
    })
    await updateSession(session.id, { title: value.trim() })
    session.title = value.trim()
    if (currentSession.value?.id === session.id) {
      currentSession.value.title = value.trim()
    }
    ElMessage.success('会话名称已更新')
  } catch {
    // user cancelled
  }
}

const removeSession = async (session: Session) => {
  try {
    await ElMessageBox.confirm(`确认删除会话「${session.title}」？删除后不可恢复。`, '删除会话', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  await deleteSession(session.id)
  sessions.value = sessions.value.filter((s) => s.id !== session.id)
  forgetContextUsage(session.id)
  if (currentSession.value?.id === session.id) {
    clearAssistantTextQueue()
    resetContextUsage()
    currentSession.value = null
    messages.value = []
    router.replace({ path: '/hr/ai' })
  }
  ElMessage.success('会话已删除')
}

const createAnalysisSessionFromRoute = async () => {
  const applicationId = Number(route.query.application_id || 0)
  if (!applicationId) return false
  const nameFromQuery = route.query.candidate_name || '该求职者'
  candidateName.value = String(nameFromQuery)
  candidatePosition.value = String(route.query.job_title || '')

  // Phase 1: Explicitly create the analysis session so it appears in the sidebar immediately
  // and the URL can be replaced with session_id, preventing re-analysis on refresh.
  let data: { session: ChatSessionListItem; messages: Partial<MessageItem>[] }
  try {
    data = await createApplicationAnalysisSession({ application_id: applicationId, ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}) })
  } catch {
    ElMessage.error('创建分析会话失败，请稍后重试')
    loading.value = false
    streaming.value = false
    return true
  }

  const session = normalizeSession(data.session)
  currentSession.value = session
  messages.value = normalizeMessages(data.messages || [])
  const returnedUserText = messages.value.find((message) => message.role === 'user' && message.content?.trim())?.content?.trim()
  const userText = resolveApplicationAnalysisMessage(messages.value, candidateName.value, candidatePosition.value)
  if (!returnedUserText) {
    messages.value.unshift({ role: 'user', content: userText })
  }
  restorePersistedContextUsage(session, messages.value)
  // Replace URL: remove application_id/candidate_name, set session_id so a refresh
  // will load the session normally instead of re-triggering analysis.
  await router.replace({ path: '/hr/ai', query: { session_id: String(session.id) } })
  await refreshSessions()
  scrollBottom()

  // Show pending animation while the durable run executes.
  messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: '分析中' })
  const assistantIndex = messages.value.length - 1
  loading.value = true
  streaming.value = true
  scrollBottom()

  // Phase 2: Durable run for the analysis reply (observes events; does not own execution).
  const token = beginAgentRun()
  try {
    const createPayload = buildApplicationAnalysisRunRequest({
      sessionId: session.id,
      message: userText,
      applicationId: session.application_id || applicationId,
      clientRequestId: createClientRequestId(),
      ...(selectedModelId.value != null ? { modelId: selectedModelId.value } : {}),
      ...(selectedSkillKeys.value.length > 0 ? { skillCapabilityKeys: selectedSkillKeys.value } : {}),
    })
    const result = await executeCreateChatRun(
      agentRun,
      createPayload,
      makeChatUiBinder(assistantIndex),
      { isAborted: () => userAborted.value || !isActiveAgentRun(token) },
    )
    if (result.outcome === 'aborted' || userAborted.value) {
      return true
    }
    if (result.outcome === 'waiting_confirmation') {
      applySkillSelectionFromRun(assistantIndex, userText, session, result.state.runId)
      return true
    }
    if (result.outcome === 'failed') {
      markAssistantError(assistantIndex, result.error || new Error('AI 分析请求失败，请稍后重试'), result.state.errorType)
      return true
    }
    await waitForAssistantTextQueue(assistantIndex)
    const refreshed = await getSessionMessages(session.id, { page: 1, page_size: 100 })
    messages.value = normalizeMessages(refreshed.list || [], messages.value)
    scrollBottom()
    return true
  } catch (_streamError) {
    if (userAborted.value) {
      return true
    }
    markAssistantError(assistantIndex, new Error('AI 分析请求失败，请稍后重试'))
    return true
  } finally {
    finishAgentRunUi(token)
  }
}

const confirmAction = async (data: StreamPayload) => {
  if (!data.action || !data.application_id || !data.action_status) return
  const actionKey = data.action_status === 2 ? 'screen_passed' : 'rejected'
  const actionText = data.action_status === 2 ? '通过' : '淘汰'
  let reason: string | undefined
  if (actionKey === 'rejected') {
    try {
      const { value } = await ElMessageBox.prompt(
        `确认将「${data.candidate_name || '该候选人'}」投递「${data.job_title || '该岗位'}」的申请标记为${actionText}？请输入淘汰原因。`,
        '确认更新投递状态',
        { type: 'warning', inputPlaceholder: '淘汰原因' }
      )
      reason = value
    } catch {
      return
    }
  } else {
    try {
      await ElMessageBox.confirm(
        `确认将「${data.candidate_name || '该候选人'}」投递「${data.job_title || '该岗位'}」的申请标记为${actionText}？`,
        '确认更新投递状态',
        { type: 'success' }
      )
    } catch {
      return
    }
  }
  await updateApplicationStatus(data.application_id, actionKey, reason)
  ElMessage.success(`已标记为${actionText}`)
  messages.value.push({ role: 'assistant', content: `已将「${data.candidate_name || '该候选人'}」的投递状态更新为"${actionText}"。` })
  scrollBottom()
}

const analyzeCandidateOption = async (option: CandidateOption) => {
  if (!option?.application_id || loading.value) return
  candidateName.value = option.candidate_name || ''
  candidatePosition.value = option.job_title || ''
  // Durable runs require session_id — create analysis session first (same as route entry).
  let data: { session: ChatSessionListItem; messages: Partial<MessageItem>[] }
  try {
    data = await createApplicationAnalysisSession({
      application_id: option.application_id,
      ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}),
    })
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '创建分析会话失败')
    return
  }

  const session = normalizeSession(data.session)
  currentSession.value = session
  messages.value = normalizeMessages(data.messages || [])
  const returnedUserText = messages.value.find((message) => message.role === 'user' && message.content?.trim())?.content?.trim()
  const userMessage = resolveApplicationAnalysisMessage(messages.value, option.candidate_name, option.job_title)
  if (!returnedUserText) {
    messages.value.unshift({ role: 'user', content: userMessage })
  }
  restorePersistedContextUsage(session, messages.value)
  await router.replace({ path: '/hr/ai', query: { session_id: String(session.id) } })
  await refreshSessions()

  messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: '分析中' })
  const assistantIndex = messages.value.length - 1
  loading.value = true
  streaming.value = true
  scrollBottom()

  const token = beginAgentRun()
  try {
    const result = await executeCreateChatRun(
      agentRun,
      buildApplicationAnalysisRunRequest({
        sessionId: session.id,
        message: userMessage,
        applicationId: option.application_id,
        clientRequestId: createClientRequestId(),
        ...(selectedModelId.value != null ? { modelId: selectedModelId.value } : {}),
      }),
      makeChatUiBinder(assistantIndex),
      { isAborted: () => userAborted.value || !isActiveAgentRun(token) },
    )
    if (result.outcome === 'aborted' || userAborted.value) return
    if (result.outcome === 'waiting_confirmation') {
      applySkillSelectionFromRun(assistantIndex, userMessage, session, result.state.runId)
      return
    }
    if (result.outcome === 'failed') {
      markAssistantError(assistantIndex, result.error || new Error('AI 流式响应失败'), result.state.errorType)
      ElMessage.error(safeAgentRunErrorMessage(result.error, result.state.errorType, result.state.errorMessage, 'AI 流式响应失败'))
      return
    }
    await waitForAssistantTextQueue(assistantIndex)
    await refreshSessions()
    const sid = result.state.sessionId || session.id
    const matched = sessions.value.find((item) => item.id === sid) || session
    currentSession.value = matched
    router.replace({ path: '/hr/ai', query: { session_id: matched.id } })
  } catch (error: unknown) {
    if (userAborted.value) return
    markAssistantError(assistantIndex, error instanceof Error ? error : new Error('AI 流式响应失败'))
    ElMessage.error(error instanceof Error ? error.message : 'AI 流式响应失败')
  } finally {
    finishAgentRunUi(token)
  }
}

const stopStreaming = async () => {
  if (!streaming.value && !loading.value) return
  userAborted.value = true
  activeRunToken.value += 1
  flushAssistantTextQueue()

  // Explicit stop is a cancel command — distinct from page-leave dispose.
  try {
    if (agentRun.state.value.runId && !agentRun.state.value.isTerminal) {
      await agentRun.cancel({ client_request_id: createClientRequestId() })
    }
  } catch {
    // Best-effort cancel; still tear down subscription and local UI.
  }
  agentRun.dispose()

  const msg = messages.value[messages.value.length - 1]
  if (msg?.role === 'assistant' && !msg.agentSkillSelection) {
    messages.value[messages.value.length - 1] = {
      ...msg,
      pending: false,
      content: msg.content || '已中断回复',
    }
  }
  loading.value = false
  streaming.value = false
}

const applyUserMessageSkillsBefore = (assistantIndex: number, skillIds: number[]) => {
  for (let i = assistantIndex - 1; i >= 0; i--) {
    if (messages.value[i]?.role !== 'user') continue
    const skills = buildMessageSkills(skillIds)
    messages.value[i] = {
      ...messages.value[i],
      skill: skills[0],
      skills,
      skillSelectionConfirmed: true,
    }
    return
  }
}

const setSkillSelectionMessage = (
  assistantIndex: number,
  text: string,
  session: Session,
  selection: AgentSkillSelectionPayload,
  runId?: number,
) => {
  messages.value[assistantIndex] = {
    role: 'assistant',
    content: '',
    pending: false,
    agentSkillSelection: selection,
    skillSelectionRequest: {
      message: text,
      sessionId: session.id,
      modelId: selectedModelId.value,
      messageId: selection.user_message_id,
      runId: runId || agentRun.state.value.runId || undefined,
    },
  }
  loading.value = false
  streaming.value = false
  scrollBottom()
}

const submitConfirmedSkillSelection = async (assistantIndex: number, skillIds: number[]) => {
  const current = messages.value[assistantIndex]
  const request = current?.skillSelectionRequest
  const session = currentSession.value
  if (!request || !session) return

  applyUserMessageSkillsBefore(assistantIndex, skillIds)
  messages.value[assistantIndex] = {
    role: 'assistant',
    content: '',
    pending: true,
    waitingText: session.application_id ? '分析中' : '响应中',
  }
  loading.value = true
  streaming.value = true
  scrollBottom()

  const token = beginAgentRun()
  try {
    // Prefer resuming the same durable run; fall back to create if run id was lost.
    const hasActiveRun =
      Boolean(request.runId && agentRun.state.value.runId === request.runId) ||
      Boolean(agentRun.state.value.runId && agentRun.state.value.status === 'waiting_confirmation')

    const result = hasActiveRun
      ? await executeConfirmChatRun(
        agentRun,
        {
          client_request_id: createClientRequestId(),
          ...(skillIds.length > 0 ? { agent_skill_ids: skillIds } : {}),
          agent_skill_selection_confirmed: true,
          ...(request.messageId ? { agent_skill_selection_message_id: request.messageId } : {}),
        },
        makeChatUiBinder(assistantIndex),
        { isAborted: () => userAborted.value || !isActiveAgentRun(token) },
      )
      : await executeCreateChatRun(
        agentRun,
        {
          session_id: request.sessionId,
          message: request.message,
          client_request_id: createClientRequestId(),
          ...(request.modelId != null ? { model_id: request.modelId } : {}),
          ...(skillIds.length > 0 ? { agent_skill_ids: skillIds } : {}),
          agent_skill_selection_confirmed: true,
          ...(request.messageId ? { agent_skill_selection_message_id: request.messageId } : {}),
        },
        makeChatUiBinder(assistantIndex),
        { isAborted: () => userAborted.value || !isActiveAgentRun(token) },
      )

    scrollBottom()
    if (result.outcome === 'aborted' || userAborted.value) return
    if (result.outcome === 'waiting_confirmation') {
      applySkillSelectionFromRun(assistantIndex, request.message, session, result.state.runId)
      return
    }
    if (result.outcome === 'failed') {
      markAssistantError(assistantIndex, result.error || new Error('AI 流式响应失败'), result.state.errorType)
      ElMessage.error(safeAgentRunErrorMessage(result.error, result.state.errorType, result.state.errorMessage, 'AI 流式响应失败'))
      return
    }

    await waitForAssistantTextQueue(assistantIndex)
    const finalPayload = resultMetaToStreamPayload(result.state.resultMetadata, result.state.sessionId || request.sessionId)
    if (finalPayload.session_id && currentSession.value) {
      currentSession.value = { ...currentSession.value, id: finalPayload.session_id }
    }
    await confirmAction(finalPayload)
    if (!messages.value[assistantIndex]?.content && !messages.value[assistantIndex]?.agentSkillSelection) {
      const sid = finalPayload.session_id || request.sessionId
      const data = await getSessionMessages(sid, { page: 1, page_size: 100 })
      messages.value = normalizeMessages(data.list || [], messages.value)
    }
    await refreshSessions()
  } catch (error: unknown) {
    if (userAborted.value) return
    markAssistantError(assistantIndex, error instanceof Error ? error : new Error('AI 流式响应失败'))
    const err = error as { code?: string; message?: string }
    if (err.code === 'ECONNABORTED') {
      ElMessage.warning('AI 分析耗时较长，请稍后重新发送')
    } else {
      ElMessage.error(err.message || 'AI 流式响应失败')
    }
  } finally {
    finishAgentRunUi(token)
  }
}

const submit = async () => {
  const text = input.value.trim()
  if (!text) return
  if (!currentSession.value) {
    await createNewSession()
  }
  const agentSkillIdsForMessage = [...selectedAgentSkillIds.value]
  const messageSkills = buildMessageSkills(agentSkillIdsForMessage)
  input.value = ''
  selectedAgentSkillIds.value = []
  messages.value.push({
    role: 'user',
    content: text,
    ...(messageSkills.length > 0 ? { skill: messageSkills[0], skills: messageSkills } : {}),
  })
  const session = currentSession.value
  if (!session) return
  loading.value = true
  streaming.value = true
  scrollBottom()
  const token = beginAgentRun()
  const assistantIndex = messages.value.length
  try {
    messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: session.application_id ? '分析中' : '响应中' })
    const skillKeysForMessage = [...selectedSkillKeys.value]
    selectedSkillKeys.value = []
    const result = await executeCreateChatRun(
      agentRun,
      {
        session_id: session.id,
        message: text,
        client_request_id: createClientRequestId(),
        ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}),
        ...(agentSkillIdsForMessage.length > 0 ? { agent_skill_ids: agentSkillIdsForMessage } : {}),
        ...(skillKeysForMessage.length > 0 ? { skill_capability_keys: skillKeysForMessage } : {}),
        ...(session.application_id ? { application_id: session.application_id } : {}),
      },
      makeChatUiBinder(assistantIndex),
      { isAborted: () => userAborted.value || !isActiveAgentRun(token) },
    )
    scrollBottom()
    if (result.outcome === 'aborted' || userAborted.value) return
    if (result.outcome === 'waiting_confirmation') {
      applySkillSelectionFromRun(assistantIndex, text, session, result.state.runId)
      return
    }
    if (result.outcome === 'failed') {
      selectedAgentSkillIds.value = agentSkillIdsForMessage
      selectedSkillKeys.value = skillKeysForMessage
      markAssistantError(assistantIndex, result.error || new Error('AI 流式响应失败'), result.state.errorType)
      ElMessage.error(safeAgentRunErrorMessage(result.error, result.state.errorType, result.state.errorMessage, 'AI 流式响应失败'))
      return
    }
    await waitForAssistantTextQueue(assistantIndex)
    const finalPayload = resultMetaToStreamPayload(result.state.resultMetadata, result.state.sessionId || session.id)
    if (finalPayload.session_id && currentSession.value) {
      currentSession.value = { ...currentSession.value, id: finalPayload.session_id }
    }
    await confirmAction(finalPayload)
    if (!messages.value[assistantIndex]?.content) {
      const sid = finalPayload.session_id || session.id
      const data = await getSessionMessages(sid, { page: 1, page_size: 100 })
      messages.value = normalizeMessages(data.list || [], messages.value)
    }
    await refreshSessions()
  } catch (error: unknown) {
    if (userAborted.value) return
    markAssistantError(assistantIndex, error instanceof Error ? error : new Error('AI 流式响应失败'))
    input.value = text
    selectedAgentSkillIds.value = agentSkillIdsForMessage
    const err = error as { code?: string; message?: string }
    if (err.code === 'ECONNABORTED') {
      ElMessage.warning('AI 分析耗时较长，请稍后重新发送')
    } else {
      ElMessage.error(err.message || 'AI 流式响应失败')
    }
  } finally {
    finishAgentRunUi(token)
  }
}

const retry = async (failedIndex: number) => {
  const failedMsg = messages.value[failedIndex]
  if (!failedMsg || failedMsg.role !== 'assistant' || !failedMsg.failed || failedMsg.retryDisabled) return

  let lastUserContent = ''
  let lastUserSkillIds: number[] = []
  let lastUserSkillSelectionConfirmed = false
  for (let i = failedIndex - 1; i >= 0; i--) {
    if (messages.value[i]?.role === 'user') {
      lastUserContent = messages.value[i].content
      lastUserSkillSelectionConfirmed = Boolean(messages.value[i].skillSelectionConfirmed)
      const skillsForRetry = (messages.value[i].skills || (messages.value[i].skill ? [messages.value[i].skill] : []))
        .filter((skill): skill is ChatMessageSkill => Boolean(skill))
      lastUserSkillIds = skillsForRetry
        .map((skill) => Number(skill.id))
        .filter((id) => Number.isFinite(id))
      break
    }
  }
  if (!lastUserContent) return

  const session = currentSession.value
  if (!session) return

  messages.value.splice(failedIndex, 1)
  messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: session.application_id ? '分析中' : '响应中' })
  const assistantIndex = messages.value.length - 1

  loading.value = true
  streaming.value = true
  scrollBottom()

  const token = beginAgentRun()
  try {
    const result = await executeCreateChatRun(
      agentRun,
      {
        session_id: session.id,
        message: lastUserContent,
        client_request_id: createClientRequestId(),
        ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}),
        ...(lastUserSkillIds.length > 0 ? { agent_skill_ids: lastUserSkillIds } : {}),
        ...(lastUserSkillSelectionConfirmed ? { agent_skill_selection_confirmed: true } : {}),
        ...(session.application_id ? { application_id: session.application_id } : {}),
      },
      makeChatUiBinder(assistantIndex),
      { isAborted: () => userAborted.value || !isActiveAgentRun(token) },
    )
    scrollBottom()
    if (result.outcome === 'aborted' || userAborted.value) return
    if (result.outcome === 'waiting_confirmation') {
      applySkillSelectionFromRun(assistantIndex, lastUserContent, session, result.state.runId)
      return
    }
    if (result.outcome === 'failed') {
      markAssistantError(assistantIndex, result.error || new Error('AI 流式响应失败'), result.state.errorType)
      ElMessage.error(safeAgentRunErrorMessage(result.error, result.state.errorType, result.state.errorMessage, 'AI 流式响应失败'))
      return
    }
    await waitForAssistantTextQueue(assistantIndex)
    const finalPayload = resultMetaToStreamPayload(result.state.resultMetadata, result.state.sessionId || session.id)
    if (finalPayload.session_id && currentSession.value) {
      currentSession.value = { ...currentSession.value, id: finalPayload.session_id }
    }
    await confirmAction(finalPayload)
    if (!messages.value[assistantIndex]?.content) {
      const sid = finalPayload.session_id || session.id
      const data = await getSessionMessages(sid, { page: 1, page_size: 100 })
      messages.value = normalizeMessages(data.list || [], messages.value)
    }
    await refreshSessions()
  } catch (error: unknown) {
    if (userAborted.value) return
    markAssistantError(assistantIndex, error instanceof Error ? error : new Error('AI 流式响应失败'))
    const err = error as { code?: string; message?: string }
    if (err.code === 'ECONNABORTED') {
      ElMessage.warning('AI 分析耗时较长，请稍后重新发送')
    } else {
      ElMessage.error(err.message || 'AI 流式响应失败')
    }
  } finally {
    finishAgentRunUi(token)
  }
}

onMounted(async () => {
  document.addEventListener('click', closeMenu)
  // Load available models for the model selector.
  try {
    const modelData = await listAvailableModels(1, 200)
    modelList.value = modelData.list || []
  } catch { /* non-fatal: model selector will be empty */ }
  try {
    const agentSkillData = await listAvailableAgentSkills()
    agentSkills.value = agentSkillData.list || []
  } catch { /* non-fatal: agent skill selector will be empty */ }
  try {
    const capabilityData = await listSkillCapabilities()
    skillCapabilities.value = capabilityData.list || []
  } catch { /* non-fatal: capability slash menu will be empty; backend uses agent-bound tools by default */ }
  await refreshSessions()
  if (await createAnalysisSessionFromRoute()) return
  const querySessionId = Number(route.query.session_id || 0)
  const target = sessions.value.find((item) => item.id === querySessionId) || sessions.value[0]
  if (target) {
    await selectSession(target)
  }
})

const closeMenu = () => { menuSessionId.value = 0 }

const handleContextUsage = (payload: StreamPayload) => {
  if (payload.context_usage) {
    const sessionId = payload.session_id || currentSession.value?.id || 0
    if (!contextUsageBelongsToSession(sessionId, currentSession.value?.id)) return
    const nextUsage = resolveLiveContextUsage(contextUsage.value, payload.context_usage)
    contextUsage.value = nextUsage
    if (sessionId > 0) {
      rememberContextUsage(sessionId, nextUsage)
    }
  }
}

const resetContextUsage = () => {
  contextUsage.value = null
}

const cancelContextPreview = () => {
  contextPreviewVersion += 1
  if (contextPreviewTimer) {
    clearTimeout(contextPreviewTimer)
    contextPreviewTimer = null
  }
  contextPreviewController?.abort()
  contextPreviewController = null
  contextPreviewing.value = false
}

const selectedEffectiveModel = (): LlmModel | undefined => selectedModelId.value == null
  ? modelList.value.find((model) => model.is_default) || modelList.value.find((model) => model.is_enabled)
  : modelList.value.find((model) => model.id === selectedModelId.value)

const contextUsageMatchesSelectedModel = (): boolean => {
  const model = selectedEffectiveModel()
  return Boolean(contextUsage.value && (!model || contextUsage.value.model_id === model.id))
}

const requestContextPreview = (notifyOnError = true) => {
  const session = currentSession.value
  if (!session || streaming.value) return
  cancelContextPreview()
  const version = contextPreviewVersion
  const sessionId = session.id
  const requestedModelId = selectedModelId.value ?? 0
  const skillCapabilityKeys = [...selectedSkillKeys.value]
  const agentSkillIds = [...selectedAgentSkillIds.value]
  contextPreviewing.value = true
  contextPreviewTimer = setTimeout(async () => {
    contextPreviewTimer = null
    const controller = new AbortController()
    contextPreviewController = controller
    try {
      const result = await previewSessionContext(sessionId, {
        model_id: requestedModelId,
        ...(skillCapabilityKeys.length > 0 ? { skill_capability_keys: skillCapabilityKeys } : {}),
        ...(agentSkillIds.length > 0 ? { agent_skill_ids: agentSkillIds } : {}),
      }, controller.signal)
      if (version !== contextPreviewVersion || currentSession.value?.id !== sessionId) return
      contextUsage.value = result.context_usage
      rememberContextUsage(sessionId, result.context_usage)
      const selected = result.selected_model_id
      currentSession.value = {
        ...currentSession.value,
        selected_model_id: selected,
        latest_context_usage: result.context_usage,
      }
      const sessionIndex = sessions.value.findIndex((item) => item.id === sessionId)
      if (sessionIndex >= 0) {
        sessions.value[sessionIndex] = {
          ...sessions.value[sessionIndex],
          selected_model_id: selected,
          latest_context_usage: result.context_usage,
        }
      }
    } catch (error: unknown) {
      if (version !== contextPreviewVersion || controller.signal.aborted) return
      contextUsage.value = null
      if (notifyOnError) {
        ElMessage.warning(error instanceof Error ? error.message : '模型上下文重新计算失败')
      }
    } finally {
      if (version === contextPreviewVersion) {
        contextPreviewController = null
        contextPreviewing.value = false
      }
    }
  }, 150)
}

const handleSelectedModelUpdate = (value: number | null) => {
  selectedModelId.value = value
  requestContextPreview()
}

const handleSelectedSkillKeysUpdate = (value: string[]) => {
  selectedSkillKeys.value = value
  requestContextPreview(false)
}

const handleSelectedAgentSkillIdsUpdate = (value: number[]) => {
  selectedAgentSkillIds.value = value
  requestContextPreview(false)
}

// Sync status bar model name with user selection.
watch(selectedModelId, (id) => {
  if (id != null) {
    const m = modelList.value.find((x) => x.id === id)
    if (m) {
      modelName.value = m.display_name || m.model_name
    }
  }
})
const toggleSessionSidebar = () => {
  if (isMobileChatViewport()) {
    sessionSidebarOpen.value = !sessionSidebarOpen.value
    return
  }
  sessionSidebarCollapsed.value = !sessionSidebarCollapsed.value
  persistSidebarCollapsed(sessionSidebarCollapsed.value)
  // 永久展开/收起时清掉临时浮层状态，避免状态错乱
  resetSidebarPeek()
}
const closeSessionSidebar = () => { sessionSidebarOpen.value = false }

const mobileContextTitle = computed(() => {
  if (!currentSession.value) return ''
  if (currentSession.value.application_id && candidateName.value) {
    return candidateName.value
  }
  return currentSession.value.title
})

const mobileContextSub = computed(() => {
  if (!currentSession.value) return ''
  if (currentSession.value.application_id) {
    return candidatePosition.value || '候选人分析会话'
  }
  return '招聘数据问答会话'
})

const runningModeLabel = computed(() => {
  if (currentSession.value?.application_id) return '/候选人岗位匹配复核'
  return '/数据问答'
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (typewriterTimer) clearInterval(typewriterTimer)
  clearSidebarPeekTimer()
  cancelContextPreview()
  document.removeEventListener('click', closeMenu)
})
</script>

<template>
  <section class="chat-page">
    <div
      v-if="sessionSidebarOpen"
      class="mobile-sidebar-backdrop"
      @click="closeSessionSidebar"
    ></div>
    <div
      class="chat-layout"
      :class="{
        'chat-layout--sidebar-collapsed': sessionSidebarCollapsed,
        'chat-layout--sidebar-peek': sessionSidebarCollapsed && sessionSidebarPeek,
      }"
    >
      <!-- 收起后左侧热区：悬停临时浮出会话列表 -->
      <div
        v-if="sessionSidebarCollapsed"
        class="chat-sidebar-rail"
        aria-hidden="true"
        @mouseenter="openSidebarPeek"
        @mouseleave="scheduleCloseSidebarPeek"
      />

      <div
        class="chat-sidebar-host"
        @mouseenter="openSidebarPeek"
        @mouseleave="scheduleCloseSidebarPeek"
      >
        <ConversationSidebar
          :sessions="sessions"
          :current-session="currentSession"
          :menu-session-id="menuSessionId"
          :session-sidebar-open="sessionSidebarOpen"
          @select-session="selectSession"
          @create-session="createNewSession"
          @rename-session="renameSession"
          @remove-session="removeSession"
          @menu-toggle="(id: number) => menuSessionId = id"
          @close-sidebar="closeSessionSidebar"
        />
      </div>

      <div class="chat-main">
        <ConversationHeader
          v-if="currentSession"
          :current-session="currentSession"
          :mobile-context-title="mobileContextTitle"
          :mobile-context-sub="mobileContextSub"
          :sidebar-collapsed="sessionSidebarCollapsed"
          @toggle-sidebar="toggleSessionSidebar"
          @show-trace="tracePanelVisible = true"
        />

        <div class="chat-content">
          <ChatMessageList
            ref="listRef"
            :messages="messages"
            :loading="loading"
            :streaming="streaming"
            :session-loading="sessionLoading"
            :has-session="!!currentSession"
            :running-mode-label="runningModeLabel"
            :render-markdown="renderMarkdown"
            :waiting-text="waitingText"
            @retry="retry"
            @confirm-skill-selection="submitConfirmedSkillSelection"
          />

          <ChatComposer
            :input="input"
            :loading="loading"
            :streaming="streaming"
            :model-list="modelList"
            :selected-model-id="selectedModelId"
            :context-usage="contextUsage"
            :context-previewing="contextPreviewing"
            :data-source="dataSource"
            :current-session="currentSession"
            :skill-capabilities="skillCapabilities"
            :selected-skill-keys="selectedSkillKeys"
            :agent-skills="agentSkills"
            :selected-agent-skill-ids="selectedAgentSkillIds"
            @update:input="(val: string) => input = val"
            @update:selected-model-id="handleSelectedModelUpdate"
            @update:selected-skill-keys="handleSelectedSkillKeysUpdate"
            @update:selected-agent-skill-ids="handleSelectedAgentSkillIdsUpdate"
            @submit="submit"
            @stop="stopStreaming"
          />
        </div>
      </div>
    </div>

    <AgentTracePanel
      v-model:visible="tracePanelVisible"
      :session-id="currentSession?.id ?? null"
    />
  </section>
</template>
