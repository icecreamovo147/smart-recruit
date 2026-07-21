<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ChatDotRound,
  Delete,
  EditPen,
  Expand,
  Fold,
  Memo,
  Plus,
  Position,
  Promotion,
  Search,
  Star,
  Suitcase,
  User,
  WarningFilled,
} from '@element-plus/icons-vue'
import { formatShanghaiDateTime } from '@shared/utils/format'
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import {
  createSession,
  deleteSession,
  getSessionMessages,
  listSessions,
  sendMessageStream,
  updateSession,
  type CandidateAISessionSource,
} from '@/api/ai'
import { listAvailableModels } from '@/api/llm'
import { applyJob } from '@/api/application'
import { getJobDetail } from '@/api/job'
import { getBillingAccount } from '@/api/billing'
import type { CandidateSession, RecommendedJob, StreamPayload } from '@/types/ai'
import type { LlmModel } from '@/types/llm'

interface MessageItem {
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  failed?: boolean
  failureCode?: string
  retryable?: boolean
  waitingText?: string
  actionPayload?: CandidateAIActionPayload | null
  suggestedQuestions?: string[]
  model_name?: string
}

interface CandidateAIActionPayload {
  action: string
  jobs?: RecommendedJob[]
  job_id?: number
}

interface CandidateAIProcessContent {
  suggested_questions?: unknown
  suggestedQuestions?: unknown
  delivery_status?: string
  error_code?: string
  retryable?: boolean
}

const route = useRoute()
const router = useRouter()
const SELECTED_MODEL_STORAGE_KEY = 'candidate-ai-selected-model-id'

const sessions = ref<CandidateSession[]>([])
const totalSessions = ref(0)
const currentSession = ref<CandidateSession | null>(null)
const messages = ref<MessageItem[]>([])
const input = ref('')
const keyword = ref('')
const activeType = ref('')
const loading = ref(false)
const sessionsLoading = ref(false)
const messagesLoading = ref(false)
const streaming = ref(false)
const sessionListCollapsed = ref(false)
const activeController = ref<AbortController | null>(null)
const userAborted = ref(false)
const modelList = ref<LlmModel[]>([])
const selectedModelId = ref<number | null>(null)
const sourceContext = ref<CandidateAISessionSource>({})
const sourceHint = ref('')
const lastFallbackSignature = ref('')
const availableCredits = ref<number | null>(null)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const messageListRef = ref<any>(null)

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

const sessionTypes = [
  { label: '全部', value: '' },
  { label: '简历优化', value: 'resume' },
  { label: '职位匹配', value: 'job_match' },
  { label: '面试准备', value: 'interview' },
  { label: 'Offer 咨询', value: 'offer' },
  { label: '求职进展', value: 'progress' },
]

const starterActions = [
  { label: '帮我优化简历', icon: Memo, type: 'resume' },
  { label: '根据我的简历推荐岗位', icon: Suitcase, type: 'job_match' },
  { label: '帮我准备一场面试', icon: Star, type: 'interview' },
  { label: '我的应聘进展怎么样？', icon: Promotion, type: 'progress' },
]

const renderMarkdown = (content: string): string => {
  const raw = DOMPurify.sanitize(md.render(content || ''), {
    ALLOWED_TAGS: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'p', 'br', 'hr', 'strong', 'b', 'em', 'i', 'u', 's', 'del', 'ul', 'ol', 'li', 'code', 'pre', 'a', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'blockquote'],
    ALLOWED_ATTR: ['href', 'title', 'target'],
    ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|tel):|[^a-z]|[a-z+.-]+(?:[^a-z+.\-:]|$))/i,
  })
  return raw.replace(/<a\s/g, '<a rel="noopener noreferrer" ')
}

const formatSessionTitle = (title: string, createdAt?: string): string => {
  if (title && title !== '新对话' && title !== 'New chat') return title
  if (!createdAt) return '新对话'
  const formatted = formatShanghaiDateTime(createdAt, '', false)
  if (!formatted) return '新对话'
  return `对话 ${formatted.slice(5, 10)} ${formatted.slice(11)}`
}

const normalizedSessions = computed(() =>
  sessions.value.map((session) => ({
    ...session,
    title: formatSessionTitle(session.title, session.created_at),
  })),
)

const currentSourceLabel = computed(() => {
  if (currentSession.value?.source_title) return currentSession.value.source_title
  if (sourceHint.value) return sourceHint.value
  return ''
})

const latestSuggestedQuestions = computed(() => {
  const latest = messages.value[messages.value.length - 1]
  if (latest?.role !== 'assistant' || !latest.suggestedQuestions?.length) return []
  return latest.suggestedQuestions
})

const quotaExhausted = computed(() => availableCredits.value !== null && availableCredits.value <= 0)

const failureTitle = (msg: MessageItem): string => {
  if (msg.failureCode === 'insufficient_credits') return 'AI 额度已用完'
  if (msg.failureCode === '42921' || msg.failureCode === 'risk_blocked') return '请求暂时受限'
  if (msg.failureCode === '42901') return '今日使用次数已达上限'
  return '消息未发送成功'
}

const isQuotaFailure = (msg: MessageItem): boolean => msg.failureCode === 'insufficient_credits'

const refreshBillingAccess = async () => {
  try {
    const billing = await getBillingAccount()
    const value = Number(billing.available_credits)
    availableCredits.value = Number.isFinite(value) ? Math.max(0, value) : null
  } catch {
    availableCredits.value = null
  }
}

const transientFailureKey = (sessionId: number) => `candidate-ai-transient-failure:${sessionId}`

const saveTransientFailure = (sessionId: number, message: MessageItem) => {
  if (!sessionId || isQuotaFailure(message)) return
  sessionStorage.setItem(transientFailureKey(sessionId), JSON.stringify({
    role: 'assistant',
    content: message.content,
    failed: true,
    failureCode: message.failureCode,
    retryable: message.retryable,
  }))
}

const loadTransientFailure = (sessionId: number): MessageItem | null => {
  const raw = sessionStorage.getItem(transientFailureKey(sessionId))
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as MessageItem
    return parsed.role === 'assistant' && parsed.failed ? parsed : null
  } catch {
    sessionStorage.removeItem(transientFailureKey(sessionId))
    return null
  }
}

const clearTransientFailure = (sessionId: number) => {
  if (sessionId) sessionStorage.removeItem(transientFailureKey(sessionId))
}

const modelLabel = (model: LlmModel): string => model.display_name || model.model_name

const scrollBottom = async () => {
  await nextTick()
  const sb = messageListRef.value as { setScrollTop?: (top: number) => void; wrapRef?: HTMLElement } | null
  const wrap = sb?.wrapRef
  const top = wrap?.scrollHeight ?? 999999
  if (typeof sb?.setScrollTop === 'function') sb.setScrollTop(top)
  else if (wrap) wrap.scrollTop = top
}

const parseActionPayload = (raw: string): CandidateAIActionPayload | null => {
  if (!raw) return null
  try {
    return JSON.parse(raw) as CandidateAIActionPayload
  } catch {
    return null
  }
}

const parseProcessContent = (raw?: string): CandidateAIProcessContent | null => {
  if (!raw) return null
  try {
    return JSON.parse(raw) as CandidateAIProcessContent
  } catch {
    return null
  }
}

const normalizeSuggestedQuestions = (value: unknown): string[] => {
  let source = value
  if (typeof value === 'string') {
    try {
      source = JSON.parse(value)
    } catch {
      source = value.split(/[，,；;、\n]/)
    }
  }
  if (!Array.isArray(source)) return []
  const result: string[] = []
  for (const item of source) {
    const question = String(item || '').trim()
    if (!question || result.includes(question)) continue
    result.push(question)
    if (result.length === 3) break
  }
  return result
}

const fallbackSuggestedQuestions = (message: string): string[] => {
  if (message.includes('简历')) return ['我的简历最大短板是什么？', '哪些经历需要量化？', '适合投递什么岗位？']
  if (message.includes('岗位') || message.includes('匹配')) return ['为什么推荐这些岗位？', '哪些岗位我已投递？', '帮我比较前两个岗位']
  if (message.includes('面试')) return ['面试前需要准备什么？', '帮我模拟一个面试问题', '这个岗位会问哪些重点？']
  return ['我有哪些投递进展？', '有哪些岗位适合我？', '我接下来应该准备什么？']
}

const modelNameFromPayload = (payload: StreamPayload): string | undefined =>
  payload.context_usage?.model_name || payload.model_name || undefined

const resolveSelectedModelDisplayName = (): string | undefined => {
  if (selectedModelId.value != null) {
    const selected = modelList.value.find((model) => model.id === selectedModelId.value)
    if (selected) return selected.display_name || selected.model_name
  }
  const defaultModel = modelList.value.find((model) => model.is_default)
  return defaultModel?.display_name || defaultModel?.model_name
}

const loadModels = async () => {
  try {
    const data = await listAvailableModels(1, 200)
    modelList.value = data.list || []
    const stored = sessionStorage.getItem(SELECTED_MODEL_STORAGE_KEY)
    if (stored && modelList.value.some((model) => model.id === Number(stored))) {
      selectedModelId.value = Number(stored)
    }
  } catch {
    modelList.value = []
  }
}

const loadSessions = async (extra: Partial<CandidateAISessionSource> = {}) => {
  sessionsLoading.value = true
  try {
    const data = await listSessions({
      page: 1,
      page_size: 50,
      keyword: keyword.value || undefined,
      session_type: activeType.value || undefined,
      ...extra,
    })
    sessions.value = data.list || []
    totalSessions.value = data.total || 0
    return sessions.value
  } finally {
    sessionsLoading.value = false
  }
}

const syncCurrentSessionFromList = async () => {
  if (sessions.value.length === 0) {
    currentSession.value = null
    messages.value = []
    return
  }
  if (currentSession.value && sessions.value.some((item) => item.session_id === currentSession.value?.session_id)) return
  const latest = sessions.value[0]
  await loadMessages(latest)
  router.replace({ path: '/ai-assistant', query: { sessionId: latest.session_id } })
}

const loadMessages = async (session: CandidateSession) => {
  currentSession.value = session
  messagesLoading.value = true
  try {
    const data = await getSessionMessages(session.session_id, { page: 1, page_size: 100 })
    messages.value = (data.list || []).map((item) => {
      const process = parseProcessContent(item.process_content)
      return {
        role: item.role === 'assistant' ? 'assistant' : 'user',
        content: item.content,
        model_name: item.model_name || undefined,
        failed: process?.delivery_status === 'failed',
        failureCode: process?.error_code,
        retryable: process?.retryable,
        suggestedQuestions: normalizeSuggestedQuestions(
          item.suggested_questions
          ?? item.suggestedQuestions
          ?? process?.suggested_questions
          ?? process?.suggestedQuestions,
        ),
      } satisfies MessageItem
    })
    const transient = loadTransientFailure(session.session_id)
    if (transient) {
      const persisted = messages.value.some((item) => item.failed && item.failureCode === transient.failureCode && item.content === transient.content)
      if (persisted) clearTransientFailure(session.session_id)
      else messages.value.push(transient)
    }
  } catch {
    messages.value = []
  } finally {
    messagesLoading.value = false
    scrollBottom()
  }
}

const selectSession = (session: CandidateSession) => {
  router.replace({ path: '/ai-assistant', query: { sessionId: session.session_id } })
  void loadMessages(session)
}

const createNewSession = async (type = 'general') => {
  const sessionSource = { ...sourceContext.value, session_type: type }
  const data = await createSession({ title: '新对话', ...sessionSource })
  sessions.value.unshift(data.session)
  await loadMessages(data.session)
  router.replace({ path: '/ai-assistant', query: { sessionId: data.session.session_id } })
}

const renameSession = async (session: CandidateSession) => {
  let title = ''
  try {
    const result = await ElMessageBox.prompt('请输入新的会话名称', '重命名会话', {
      confirmButtonText: '确认重命名',
      cancelButtonText: '取消',
      inputValue: session.title,
      inputPattern: /\S+/,
      inputErrorMessage: '会话名称不能为空',
    })
    title = String(result.value || '').trim()
  } catch {
    return
  }
  if (!title || title === session.title) return
  await updateSession(session.session_id, { title })
  sessions.value = sessions.value.map((item) => item.session_id === session.session_id ? { ...item, title } : item)
  if (currentSession.value?.session_id === session.session_id) currentSession.value = { ...currentSession.value, title }
  ElMessage.success('会话已重命名')
}

const removeSession = async (session: CandidateSession) => {
  try {
    await ElMessageBox.confirm(`确认删除会话「${session.title}」？`, '删除会话', { type: 'warning' })
  } catch {
    return
  }
  await deleteSession(session.session_id)
  sessions.value = sessions.value.filter((item) => item.session_id !== session.session_id)
  if (currentSession.value?.session_id === session.session_id) {
    currentSession.value = null
    messages.value = []
    router.replace('/ai-assistant')
  }
  ElMessage.success('会话已删除')
}

const ensureSessionBeforeSend = async (message: string, type = 'general') => {
  if (currentSession.value) return currentSession.value
  const source = { ...sourceContext.value, session_type: sourceContext.value.session_type || type }
  const data = await createSession({ title: message.slice(0, 32), ...source })
  currentSession.value = data.session
  sessions.value.unshift(data.session)
  router.replace({ path: '/ai-assistant', query: { sessionId: data.session.session_id } })
  return data.session
}

const send = async (text?: string, type = 'general') => {
  const message = (text || input.value).trim()
  if (!message || loading.value || quotaExhausted.value) return
  input.value = ''
  const session = await ensureSessionBeforeSend(message, type)
  if (!session) return

  messages.value.push({ role: 'user', content: message })
  messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: '思考中' })
  const assistantIndex = messages.value.length - 1
  loading.value = true
  streaming.value = true
  userAborted.value = false
  const controller = new AbortController()
  activeController.value = controller
  scrollBottom()

  try {
    let streamFailed = false
    const result = { payload: null as StreamPayload | null }
    await sendMessageStream(
      {
        message,
        session_id: session.session_id,
        ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}),
        ...sourceContext.value,
      },
      {
        onDelta: (delta) => {
          const msg = messages.value[assistantIndex]
          if (msg) messages.value[assistantIndex] = { ...msg, content: (msg.content || '') + delta, pending: false }
          scrollBottom()
        },
        onStatus: (_eventType, eventMessage, payload) => {
          const msg = messages.value[assistantIndex]
          if (!msg) return
          const modelName = modelNameFromPayload(payload)
          messages.value[assistantIndex] = { ...msg, waitingText: eventMessage, ...(modelName ? { model_name: modelName } : {}) }
          const usage = payload.context_usage
          if (usage?.model_fallback_reason && usage.effective_model_id) {
            const signature = `${usage.capability_version_id || 0}:${usage.requested_model_id || 0}:${usage.effective_model_id}`
            if (signature !== lastFallbackSignature.value) {
              lastFallbackSignature.value = signature
              ElMessage.warning(`所选模型当前不可用，已按平台能力版本切换为 ${modelName || '默认模型'}`)
            }
          }
        },
        onDone: (payload) => {
          result.payload = payload
        },
        onError: (errorType, errorMessage) => {
          streamFailed = true
          const failureCode = errorType === '40201' ? 'insufficient_credits' : errorType
          const msg = messages.value[assistantIndex]
          if (msg) {
            const failedMessage = { ...msg, content: errorMessage, pending: false, failed: true, failureCode, retryable: failureCode !== 'insufficient_credits' }
            messages.value[assistantIndex] = failedMessage
            saveTransientFailure(session.session_id, failedMessage)
          }
          if (failureCode === 'insufficient_credits') availableCredits.value = 0
        },
      },
      { signal: controller.signal, silentAbort: true },
    )

    if (userAborted.value || streamFailed) return
    clearTransientFailure(session.session_id)
    const finalPayload = result.payload
    const latest = messages.value[assistantIndex]
    if (finalPayload && latest) {
      const actionPayload = finalPayload.action_payload ? parseActionPayload(finalPayload.action_payload) : null
      const suggestedQuestions = normalizeSuggestedQuestions(finalPayload.suggested_questions ?? finalPayload.suggestedQuestions)
      messages.value[assistantIndex] = {
        ...latest,
        actionPayload,
        model_name: modelNameFromPayload(finalPayload) || latest.model_name || resolveSelectedModelDisplayName(),
        suggestedQuestions: suggestedQuestions.length > 0 ? suggestedQuestions : fallbackSuggestedQuestions(message),
      }
    }
    await loadSessions()
  } catch {
    if (!userAborted.value) {
      const msg = messages.value[assistantIndex]
      if (msg) messages.value[assistantIndex] = { ...msg, failed: true, pending: false }
    }
  } finally {
    if (activeController.value === controller) {
      loading.value = false
      streaming.value = false
      activeController.value = null
      userAborted.value = false
    }
  }
}

const stopStreaming = () => {
  const controller = activeController.value
  if (!controller || !streaming.value) return
  userAborted.value = true
  controller.abort()
  const msg = messages.value[messages.value.length - 1]
  if (msg?.role === 'assistant') {
    messages.value[messages.value.length - 1] = { ...msg, pending: false, content: msg.content || '已中断回复' }
  }
  loading.value = false
  streaming.value = false
}

const retry = () => {
  const lastUser = [...messages.value].reverse().find((msg) => msg.role === 'user')
  if (lastUser) send(lastUser.content)
}

const handleApply = async (job: RecommendedJob) => {
  try {
    await ElMessageBox.confirm(`确认投递「${job.title}」？`, '投递确认', {
      type: 'info',
      confirmButtonText: '确认投递',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  await applyJob({ job_id: job.job_id })
  ElMessage.success('投递成功')
  job.has_applied = true
}

const applyRouteContext = async () => {
  const sessionId = Number(route.query.sessionId || 0)
  const source = String(route.query.source || '')
  const jobId = Number(route.query.jobId || route.query.sourceId || 0)
  if (source === 'job' && jobId > 0) {
    sourceContext.value = { session_type: 'job_match', source_type: 'job', source_id: jobId }
    try {
      const job = await getJobDetail(jobId)
      sourceContext.value.source_title = job.title
      sourceHint.value = job.title
    } catch {
      sourceHint.value = `岗位 ${jobId}`
    }
    await loadSessions({ source_type: 'job', source_id: jobId })
    if (sessions.value.length > 0) await loadMessages(sessions.value[0])
    input.value = '帮我分析这个岗位和我的简历匹配度'
    return
  }
  await loadSessions()
  if (sessionId > 0) {
    const session = sessions.value.find((item) => item.session_id === sessionId)
    if (session) {
      await loadMessages(session)
    } else {
      ElMessage.warning('会话不存在或已被删除，已打开最近会话')
      if (sessions.value.length > 0) {
        const latest = sessions.value[0]
        await loadMessages(latest)
        router.replace({ path: '/ai-assistant', query: { sessionId: latest.session_id } })
      }
    }
    return
  }
  if (sessions.value.length > 0) {
    const latest = sessions.value[0]
    await loadMessages(latest)
    router.replace({ path: '/ai-assistant', query: { sessionId: latest.session_id } })
  }
}

watch(keyword, async () => {
  await loadSessions()
  await syncCurrentSessionFromList()
})

watch(activeType, async () => {
  await loadSessions()
  await syncCurrentSessionFromList()
})

watch(selectedModelId, (id) => {
  if (id != null) sessionStorage.setItem(SELECTED_MODEL_STORAGE_KEY, String(id))
  else sessionStorage.removeItem(SELECTED_MODEL_STORAGE_KEY)
})

onMounted(async () => {
  await Promise.all([loadModels(), refreshBillingAccess()])
  await applyRouteContext()
})
</script>

<template>
  <section class="ai-workspace" :class="{ 'ai-workspace--sidebar-collapsed': sessionListCollapsed }">
    <aside class="ai-sidebar">
      <div class="ai-sidebar__head">
        <div>
          <h1>AI求职助手</h1>
          <span>{{ totalSessions }} 个会话</span>
        </div>
        <el-button type="primary" :icon="Plus" @click="createNewSession()">新建</el-button>
      </div>

      <el-input v-model="keyword" :prefix-icon="Search" placeholder="搜索会话" clearable />

      <el-scrollbar class="ai-type-tabs">
        <button
          v-for="item in sessionTypes"
          :key="item.value"
          class="ai-type-tab"
          :class="{ 'ai-type-tab--active': activeType === item.value }"
          @click="activeType = item.value"
        >
          {{ item.label }}
        </button>
      </el-scrollbar>

      <el-scrollbar v-loading="sessionsLoading" class="ai-session-list">
        <div class="ai-session-list__inner">
          <button
            v-for="session in normalizedSessions"
            :key="session.session_id"
            class="ai-session"
            :class="{ 'ai-session--active': currentSession?.session_id === session.session_id }"
            @click="selectSession(session)"
          >
            <span class="ai-session__title">{{ session.title }}</span>
            <span class="ai-session__preview">{{ session.last_message_preview || session.summary || session.source_title || '暂无消息' }}</span>
            <span class="ai-session__meta">{{ session.source_title || session.session_type || 'general' }}</span>
            <span class="ai-session__actions">
              <el-icon @click.stop="renameSession(session)"><EditPen /></el-icon>
              <el-icon @click.stop="removeSession(session)"><Delete /></el-icon>
            </span>
          </button>
          <el-empty v-if="!sessionsLoading && sessions.length === 0" description="暂无会话" :image-size="72" />
        </div>
      </el-scrollbar>
    </aside>

    <main class="ai-chat">
      <header class="ai-chat__head">
        <el-tooltip :content="sessionListCollapsed ? '展开会话列表' : '收起会话列表'" placement="bottom">
          <el-button
            class="ai-chat__sidebar-toggle"
            circle
            :icon="sessionListCollapsed ? Expand : Fold"
            :aria-label="sessionListCollapsed ? '展开会话列表' : '收起会话列表'"
            @click="sessionListCollapsed = !sessionListCollapsed"
          />
        </el-tooltip>
        <div>
          <h2>{{ currentSession?.title || '开始一次求职咨询' }}</h2>
          <el-tag v-if="currentSourceLabel" effect="plain" type="primary">{{ currentSourceLabel }}</el-tag>
        </div>
      </header>

      <div class="ai-chat__content">
        <el-scrollbar ref="messageListRef" v-loading="messagesLoading" class="ai-messages">
        <div v-if="messages.length === 0 && !messagesLoading" class="ai-welcome">
          <div class="ai-welcome__mark">
            <el-icon><ChatDotRound /></el-icon>
          </div>
          <h2>把求职问题交给 AI 助手</h2>
          <p>可以围绕简历、岗位匹配、面试准备、Offer 和投递进展继续聊。</p>
          <div class="ai-starters">
            <button
              v-for="action in starterActions"
              :key="action.label"
              class="ai-starter"
              @click="send(action.label, action.type)"
            >
              <el-icon><component :is="action.icon" /></el-icon>
              <span>{{ action.label }}</span>
            </button>
          </div>
        </div>

        <div
          v-for="(msg, index) in messages"
          :key="index"
          class="ai-message"
          :class="[msg.role, { 'ai-message--failed': msg.failed }]"
        >
          <div class="ai-message__avatar">
            <el-icon v-if="msg.role === 'user'"><User /></el-icon>
            <el-icon v-else><Position /></el-icon>
          </div>
          <div class="ai-message__body">
            <div v-if="msg.failed" class="ai-failure-card" :class="{ 'ai-failure-card--quota': isQuotaFailure(msg) }" role="alert">
              <div class="ai-failure-card__icon"><el-icon><WarningFilled /></el-icon></div>
              <div class="ai-failure-card__main">
                <strong>{{ failureTitle(msg) }}</strong>
                <p>{{ msg.content }}</p>
                <div class="ai-failure-card__actions">
                  <el-button v-if="isQuotaFailure(msg)" size="small" type="primary" @click="router.push('/billing')">查看 AI 套餐</el-button>
                  <el-button v-else-if="msg.retryable !== false" size="small" plain @click="retry">重新发送</el-button>
                </div>
              </div>
            </div>
            <div v-else-if="msg.pending" class="ai-typing" role="status" aria-live="polite">
              <span class="ai-typing__text">{{ msg.waitingText || '思考中' }}</span>
              <span class="ai-typing__dots" aria-hidden="true">
                <span />
                <span />
                <span />
              </span>
            </div>
            <div v-else class="ai-message__content" v-html="renderMarkdown(msg.content)"></div>
            <el-tag
              v-if="msg.role === 'assistant' && msg.model_name && !msg.pending"
              class="ai-message__model"
              size="small"
              effect="plain"
            >
              {{ msg.model_name }}
            </el-tag>

            <div v-if="msg.actionPayload?.action === 'recommend_jobs' && msg.actionPayload.jobs?.length" class="ai-recommend">
              <div v-for="job in msg.actionPayload.jobs" :key="job.job_id" class="ai-recommend__card">
                <div>
                  <strong>{{ job.title }}</strong>
                  <p>{{ job.department || '部门待定' }} · {{ job.location || '地点待定' }} · {{ job.salary_range ? job.salary_range + ' 元/月' : '薪资面议' }}</p>
                  <span v-if="job.match_score">匹配度 {{ job.match_score }}%</span>
                </div>
                <div class="ai-recommend__actions">
                  <el-tag v-if="job.has_applied" type="success" size="small">已投递</el-tag>
                  <template v-else>
                    <el-button size="small" @click="router.push(`/jobs/${job.job_id}`)">查看</el-button>
                    <el-button size="small" type="primary" @click="handleApply(job)">投递</el-button>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>
        </el-scrollbar>

        <div v-if="latestSuggestedQuestions.length" class="ai-suggested ai-suggested--composer">
          <button v-for="question in latestSuggestedQuestions" :key="question" :disabled="loading || quotaExhausted" @click="send(question)">
            {{ question }}
          </button>
        </div>

        <footer class="ai-composer">
          <div v-if="quotaExhausted" class="ai-quota-notice" role="status">
            <div><el-icon><WarningFilled /></el-icon><span><strong>AI 额度已用完</strong>购买套餐或加量包后即可继续对话。</span></div>
            <el-button size="small" type="primary" @click="router.push('/billing')">查看套餐</el-button>
          </div>
          <div class="ai-composer__card">
            <div class="ai-composer__input-area">
              <el-input
                v-model="input"
                type="textarea"
                :autosize="{ minRows: 2, maxRows: 6 }"
                resize="none"
                placeholder="输入你的求职问题..."
                :disabled="streaming || quotaExhausted"
                class="ai-composer__text-input"
                @keydown.enter.exact.prevent="streaming ? undefined : send()"
              />
            </div>
            <div class="ai-composer__toolbar">
              <div class="ai-composer__toolbar-left">
                <el-select
                  v-if="modelList.length > 0"
                  v-model="selectedModelId"
                  size="small"
                  placeholder="选择模型"
                  class="ai-composer__model-select"
                  clearable
                  :disabled="quotaExhausted"
                >
                  <el-option
                    v-for="model in modelList"
                    :key="model.id"
                    :label="modelLabel(model)"
                    :value="model.id"
                  >
                    <span class="ai-composer__model-option">
                      <span class="ai-composer__model-name">{{ modelLabel(model) }}</span>
                      <span v-if="model.is_default" class="ai-composer__model-default">默认</span>
                    </span>
                  </el-option>
                </el-select>
              </div>
              <el-button
                v-if="streaming"
                type="danger"
                plain
                class="ai-composer__send-btn"
                @click="stopStreaming"
              >
                中断
              </el-button>
              <el-button
                v-else
                type="primary"
                :icon="Position"
                :loading="loading"
                :disabled="!input.trim() || quotaExhausted"
                class="ai-composer__send-btn"
                @click="send()"
              >
                发送
              </el-button>
            </div>
          </div>
        </footer>
      </div>
    </main>
  </section>
</template>

<style scoped>
.ai-workspace {
  box-sizing: border-box;
  width: 100%;
  height: min(760px, calc(100dvh - 96px));
  min-height: min(620px, calc(100dvh - 96px));
  display: grid;
  grid-template-columns: 312px minmax(0, 1fr);
  gap: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  box-shadow: var(--shadow-card);
  transition: grid-template-columns 420ms var(--motion-ease);
}

.ai-workspace--sidebar-collapsed {
  grid-template-columns: 0 minmax(0, 1fr);
}

.ai-sidebar {
  box-sizing: border-box;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  border-right: 1px solid var(--border);
  background: var(--surface-muted);
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
  opacity: 1;
  transition:
    padding 420ms var(--motion-ease),
    opacity 300ms var(--motion-ease),
    border-color 420ms var(--motion-ease);
}

.ai-workspace--sidebar-collapsed .ai-sidebar {
  padding-right: 0;
  padding-left: 0;
  opacity: 0;
  pointer-events: none;
  border-right-color: transparent;
}

.ai-sidebar__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.ai-sidebar__head h1,
.ai-chat__head h2,
.ai-welcome h2 {
  margin: 0;
  font-size: 18px;
  letter-spacing: 0;
}

.ai-sidebar__head span {
  color: var(--text-muted);
  font-size: 12px;
}

.ai-type-tabs {
  flex: 0 0 auto;
  height: auto;
  max-height: none;
}

.ai-type-tabs :deep(.el-scrollbar__wrap) {
  height: auto;
  max-height: none;
  overflow-x: auto;
  overflow-y: hidden;
}

.ai-type-tabs :deep(.el-scrollbar__view) {
  display: flex;
  gap: 8px;
}

.ai-type-tabs :deep(.el-scrollbar__bar.is-vertical) {
  display: none;
}

.ai-type-tab {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text-secondary);
  border-radius: 8px;
  padding: 7px 10px;
  white-space: nowrap;
  cursor: pointer;
}

.ai-type-tab--active {
  color: var(--brand-strong);
  border-color: rgba(37, 99, 235, 0.35);
  background: rgba(37, 99, 235, 0.08);
}

.ai-session-list {
  flex: 1;
  min-height: 0;
}

.ai-session-list :deep(.el-scrollbar__wrap) {
  height: 100%;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.ai-session-list__inner {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 2px;
}

.ai-session {
  width: 100%;
  position: relative;
  display: grid;
  gap: 5px;
  text-align: left;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--text-primary);
  padding: 12px 58px 12px 12px;
  cursor: pointer;
}

.ai-session:hover,
.ai-session--active {
  background: var(--surface);
  border-color: var(--border);
}

.ai-session--active {
  border-color: color-mix(in srgb, var(--brand) 34%, var(--border));
  background: color-mix(in srgb, var(--brand-soft) 45%, var(--surface));
}

.ai-session__title,
.ai-session__preview {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-session__title {
  font-weight: 700;
}

.ai-session__preview,
.ai-session__meta {
  color: var(--text-muted);
  font-size: 12px;
}

.ai-session__actions {
  position: absolute;
  top: 12px;
  right: 10px;
  display: flex;
  gap: 8px;
  color: var(--text-muted);
}

.ai-chat {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  background: var(--surface);
}

.ai-chat__content {
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 16px;
}

.ai-chat__head {
  min-height: 56px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 12px;
}

.ai-chat__sidebar-toggle {
  flex: 0 0 auto;
}

.ai-messages {
  box-sizing: border-box;
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  padding: 4px 8px 12px;
}

.ai-messages :deep(.el-scrollbar__wrap) {
  height: 100%;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.ai-welcome {
  min-height: 100%;
  display: grid;
  place-content: center;
  justify-items: center;
  gap: 14px;
  color: var(--text-secondary);
  text-align: center;
  font-size: 14px;
}

.ai-welcome p {
  margin: 0;
  font-size: 13px;
}

.ai-welcome__mark {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  color: var(--brand-strong);
  background: rgba(37, 99, 235, 0.1);
  font-size: 24px;
}

.ai-starters {
  display: grid;
  grid-template-columns: repeat(2, minmax(180px, 1fr));
  gap: 10px;
  margin-top: 6px;
}

.ai-starter {
  min-height: 44px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  font-size: 13px;
}

.ai-message {
  display: flex;
  gap: 10px;
  margin-bottom: 14px;
}

.ai-message.user {
  flex-direction: row-reverse;
}

.ai-message__avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  background: var(--surface-muted);
  color: var(--brand-strong);
}

.ai-message__body {
  max-width: min(760px, 78%);
  display: grid;
  gap: 8px;
  justify-items: start;
}

.ai-message.user .ai-message__body {
  justify-items: end;
}

.ai-message__content,
.ai-typing {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px 12px;
  background: var(--surface-muted);
  font-size: 14px;
  line-height: 1.6;
}

.ai-failure-card {
  width: min(520px, 100%);
  box-sizing: border-box;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 15px;
  border: 1px solid color-mix(in srgb, var(--el-color-warning) 34%, var(--border));
  border-radius: 12px;
  background: color-mix(in srgb, var(--el-color-warning-light-9) 78%, var(--surface));
  box-shadow: 0 8px 22px rgba(120, 72, 12, 0.06);
}

.ai-failure-card--quota {
  border-color: color-mix(in srgb, var(--brand) 28%, var(--border));
  background: linear-gradient(135deg, color-mix(in srgb, var(--brand-soft) 68%, var(--surface)), var(--surface));
  box-shadow: 0 8px 22px rgba(37, 99, 235, 0.08);
}

.ai-failure-card__icon {
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  border-radius: 9px;
  background: color-mix(in srgb, var(--el-color-warning) 16%, transparent);
  color: var(--el-color-warning-dark-2);
  font-size: 17px;
}

.ai-failure-card--quota .ai-failure-card__icon {
  background: color-mix(in srgb, var(--brand) 13%, transparent);
  color: var(--brand-strong);
}

.ai-failure-card__main {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.ai-failure-card__main > strong {
  color: var(--text-primary);
  font-size: 14px;
  line-height: 1.4;
}

.ai-failure-card__main p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.55;
}

.ai-failure-card__actions {
  display: flex;
  gap: 8px;
  margin-top: 3px;
}

.ai-typing {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
}

.ai-typing__text {
  white-space: nowrap;
}

.ai-typing__dots {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 12px;
}

.ai-typing__dots span {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--brand-strong);
  opacity: 0.35;
  animation: ai-typing-dot 1.2s infinite ease-in-out;
}

.ai-typing__dots span:nth-child(2) {
  animation-delay: 160ms;
}

.ai-typing__dots span:nth-child(3) {
  animation-delay: 320ms;
}

@keyframes ai-typing-dot {
  0%,
  80%,
  100% {
    transform: translateY(0);
    opacity: 0.35;
  }

  40% {
    transform: translateY(-4px);
    opacity: 1;
  }
}

.ai-message.user .ai-message__content {
  background: var(--brand);
  border-color: var(--brand);
  color: #ffffff;
  box-shadow: 0 8px 18px rgba(37, 99, 235, 0.16);
}

.ai-message__model {
  justify-self: start;
  width: auto;
  max-width: max-content;
  color: var(--brand-strong);
  border-color: color-mix(in srgb, var(--brand) 28%, var(--border));
  background: color-mix(in srgb, var(--brand-soft) 42%, var(--surface));
}

.ai-message__content :deep(p) {
  margin: 0 0 6px;
  font-size: inherit;
}

.ai-message__content :deep(h1),
.ai-message__content :deep(h2),
.ai-message__content :deep(h3),
.ai-message__content :deep(h4) {
  margin: 0 0 6px;
  font-size: 15px;
  line-height: 1.45;
}

.ai-message__content :deep(h5),
.ai-message__content :deep(h6) {
  margin: 0 0 4px;
  font-size: 14px;
  line-height: 1.45;
}

.ai-message__content :deep(ul),
.ai-message__content :deep(ol) {
  margin: 0 0 6px;
  padding-left: 1.25em;
}

.ai-message__content :deep(li) {
  margin-bottom: 2px;
}

.ai-message__content :deep(code) {
  font-size: 0.92em;
}

.ai-message__content :deep(pre) {
  margin: 0 0 6px;
  padding: 8px 10px;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
}

.ai-message__content :deep(p:last-child) {
  margin-bottom: 0;
}

.ai-recommend {
  display: grid;
  gap: 10px;
}

.ai-recommend__card {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.ai-recommend__card p {
  margin: 4px 0;
  color: var(--text-muted);
  font-size: 12px;
}

.ai-recommend__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ai-suggested {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.ai-suggested--composer {
  flex-shrink: 0;
  padding: 4px 8px 10px;
}

.ai-suggested button {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--brand-strong);
  padding: 6px 10px;
  cursor: pointer;
  font-size: 13px;
  line-height: 1.4;
}

.ai-composer {
  flex-shrink: 0;
}

.ai-quota-notice {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 10px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--brand) 24%, var(--border));
  border-radius: 12px;
  background: color-mix(in srgb, var(--brand-soft) 58%, var(--surface));
}

.ai-quota-notice > div {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 13px;
}

.ai-quota-notice .el-icon,
.ai-quota-notice strong {
  color: var(--brand-strong);
}

.ai-composer__card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 12px 14px;
  min-height: 120px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.05);
  transition:
    border-color var(--motion-normal) var(--motion-ease),
    box-shadow var(--motion-normal) var(--motion-ease);
}

.ai-composer__card:hover,
.ai-composer__card:focus-within {
  border-color: var(--el-color-primary-light-7);
  box-shadow: 0 10px 28px rgba(37, 99, 235, 0.08);
}

.ai-composer__input-area {
  min-height: 54px;
}

.ai-composer__text-input {
  width: 100%;
}

.ai-composer__text-input :deep(.el-textarea__inner) {
  box-shadow: none;
  background: transparent;
  padding: 0;
  font-size: 14px;
  line-height: 1.65;
  color: var(--text-primary);
  border: 0;
  min-height: 48px !important;
}

.ai-composer__text-input :deep(.el-textarea__inner::placeholder) {
  color: var(--text-faint);
}

.ai-composer__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
}

.ai-composer__toolbar-left {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.ai-composer__model-select {
  width: 150px;
  flex-shrink: 0;
}

.ai-composer__model-select :deep(.el-select__wrapper) {
  min-height: 32px;
  border-radius: 999px;
  background: var(--surface-muted);
  box-shadow: 0 0 0 1px var(--border) inset;
}

.ai-composer__model-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}

.ai-composer__model-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-composer__model-default {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--brand-strong);
}

.ai-composer__send-btn {
  flex-shrink: 0;
  min-width: 68px;
  height: 34px;
  border-radius: 10px;
  padding: 0 12px;
}

@media (max-width: 768px) {
  .ai-workspace {
    height: auto;
    min-height: 0;
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(560px, calc(100dvh - 116px));
  }

  .ai-workspace--sidebar-collapsed {
    grid-template-columns: 1fr;
    grid-template-rows: 0 minmax(560px, calc(100dvh - 116px));
  }

  .ai-sidebar {
    max-height: 280px;
    border-right: 0;
    border-bottom: 1px solid var(--border);
    transition:
      max-height 420ms var(--motion-ease),
      padding 420ms var(--motion-ease),
      opacity 300ms var(--motion-ease),
      border-color 420ms var(--motion-ease);
  }

  .ai-workspace--sidebar-collapsed .ai-sidebar {
    max-height: 0;
    padding-top: 0;
    padding-bottom: 0;
    border-bottom-color: transparent;
  }

  .ai-chat__head {
    align-items: flex-start;
  }

  .ai-chat__content {
    padding: 12px;
  }

  .ai-composer__toolbar {
    align-items: flex-end;
    gap: 8px;
  }

  .ai-composer__toolbar-left {
    flex-wrap: wrap;
    gap: 8px;
  }

  .ai-composer__model-select {
    width: 140px;
  }

  .ai-composer__card {
    padding: 10px 12px 12px;
  }

  .ai-message__body {
    max-width: 86%;
  }

  .ai-starters {
    grid-template-columns: 1fr;
  }
}
</style>
