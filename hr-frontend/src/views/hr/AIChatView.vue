<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import { createApplicationAnalysisSession, createSession, deleteSession, getSessionMessages, listSessions, sendMessageStream, updateSession } from '@/api/ai'
import { updateApplicationStatus } from '@/api/application'
import type { ChatMessage, ChatSessionListItem, Session, CandidateOption, StreamPayload } from '@/types/ai'
import { BusinessError } from '@/types/api'

interface MessageItem {
  role: string
  content: string
  pending?: boolean
  failed?: boolean
  waitingText?: string
  candidateOptions?: CandidateOption[]
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
const sessionSidebarOpen = ref(false)
const candidateName = ref('')
const candidatePosition = ref('')
const activeController = ref<AbortController | null>(null)
const userAborted = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const listRef = ref<any>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

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

const parseCandidateOptions = (value: unknown): CandidateOption[] => {
  if (!value) return []
  if (Array.isArray(value)) return value as CandidateOption[]
  try {
    return JSON.parse(value as string) as CandidateOption[]
  } catch {
    return []
  }
}

const normalizeSession = (item: ChatSessionListItem): Session => ({
  id: item.session_id || 0,
  title: item.title || '新对话',
  application_id: item.application_id || 0,
  updated_at: item.updated_at || item.created_at || '',
})

const scrollBottom = async () => {
  await nextTick()
  if (listRef.value) listRef.value.scrollTop = listRef.value.scrollHeight
}

const appendAssistantDelta = (index: number, delta: string) => {
  const message = messages.value[index]
  if (!message) return
  messages.value[index] = { ...message, content: `${message.content || ''}${delta}`, pending: false }
  scrollBottom()
}

const markAssistantError = (index: number, error: Error | null) => {
  const message = messages.value[index]
  const content = error?.message || '响应中断，请稍后重试'
  if (message?.role === 'assistant') {
    messages.value[index] = { ...message, content, pending: false, failed: true }
  } else {
    messages.value.push({ role: 'assistant', content, failed: true })
  }
  scrollBottom()
}

const refreshSessions = async () => {
  const data = await listSessions({ page: 1, page_size: 50 })
  sessions.value = (data.list || []).map(normalizeSession)
}

const selectSession = async (session: Session) => {
  if (!session) return
  currentSession.value = session
  sessionLoading.value = true
  try {
    const data = await getSessionMessages(session.id, { page: 1, page_size: 100 })
    messages.value = (data.list || []) as MessageItem[]
    router.replace({ path: '/hr/ai', query: { session_id: String(session.id) } })
    scrollBottom()
  } finally {
    sessionLoading.value = false
  }
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
    const nextMessages = (data.list || []) as MessageItem[]

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
        messages.value = (data2.list || []) as MessageItem[]
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
  if (currentSession.value?.id === session.id) {
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
  let data: { session: ChatSessionListItem; messages: { role: string; content: string; created_at: string }[] }
  try {
    data = await createApplicationAnalysisSession({ application_id: applicationId })
  } catch {
    ElMessage.error('创建分析会话失败，请稍后重试')
    loading.value = false
    streaming.value = false
    return true
  }

  const session = normalizeSession(data.session)
  currentSession.value = session
  messages.value = (data.messages || []) as MessageItem[]
  // Replace URL: remove application_id/candidate_name, set session_id so a refresh
  // will load the session normally instead of re-triggering analysis.
  router.replace({ path: '/hr/ai', query: { session_id: String(session.id) } })
  await refreshSessions()
  scrollBottom()

  // Show pending animation while the stream runs.
  messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: '分析中' })
  const assistantIndex = messages.value.length - 1
  loading.value = true
  streaming.value = true
  scrollBottom()

  // Phase 2: Stream the analysis reply into the already-created session.
  const controller = new AbortController()
  activeController.value = controller
  userAborted.value = false
  try {
    await sendMessageStream(
      { message: messages.value[0].content, session_id: session.id },
      {
        onDelta: (delta) => {
          const msg = messages.value[assistantIndex]
          if (msg) {
            messages.value[assistantIndex] = { ...msg, content: (msg.content || '') + delta, pending: false }
          }
          scrollBottom()
        },
        onStatus: (_eventType, eventMessage) => {
          const msg = messages.value[assistantIndex]
          if (msg) {
            messages.value[assistantIndex] = { ...msg, waitingText: eventMessage }
          }
        },
        onDone: (_payload) => { /* session already created; done payload is informational */ },
      },
      { signal: controller.signal, silentAbort: true },
    )
    // User aborted → just bail; the session + user message are already persisted.
    if (userAborted.value) {
      return true
    }
    // Normal completion: re-fetch messages to get the full persisted history.
    const refreshed = await getSessionMessages(session.id, { page: 1, page_size: 100 })
    messages.value = (refreshed.list || []) as MessageItem[]
    scrollBottom()
    return true
  } catch (_streamError) {
    if (userAborted.value) {
      return true
    }
    markAssistantError(assistantIndex, new Error('AI 分析请求失败，请稍后重试'))
    return true
  } finally {
    if (activeController.value === controller) {
      loading.value = false
      streaming.value = false
      activeController.value = null
      userAborted.value = false
    }
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
  messages.value.push({ role: 'assistant', content: `已将「${data.candidate_name || '该候选人'}」的投递状态更新为“${actionText}”。` })
  scrollBottom()
}

const analyzeCandidateOption = async (option: CandidateOption) => {
  if (!option?.application_id || loading.value) return
  candidateName.value = option.candidate_name || ''
  candidatePosition.value = option.job_title || ''
  const userMessage = `请帮我分析${option.candidate_name || '该候选人'}投递${option.job_title || '该岗位'}的简历。`
  messages.value.push({ role: 'user', content: userMessage })
  messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: '分析中' })
  const assistantIndex = messages.value.length - 1
  loading.value = true
  streaming.value = true
  scrollBottom()
  userAborted.value = false
  const controller = new AbortController()
  activeController.value = controller
  try {
    let finalPayload: StreamPayload | null = null
    let streamFailed = false
    await sendMessageStream(
      { message: userMessage, application_id: option.application_id },
      {
        onDelta: (delta) => appendAssistantDelta(assistantIndex, delta),
        onStatus: (_eventType, eventMessage) => {
          const msg = messages.value[assistantIndex]
          if (msg) {
            messages.value[assistantIndex] = { ...msg, waitingText: eventMessage }
          }
        },
        onDone: (payload) => {
          finalPayload = payload
        },
        onError: (_errorType, errorMessage) => {
          streamFailed = true
          markAssistantError(assistantIndex, new Error(errorMessage))
        },
      },
      { signal: controller.signal, silentAbort: true },
    )
    if (userAborted.value) return
    if (streamFailed) return
    await refreshSessions()
    const session = sessions.value.find((item) => item.id === finalPayload?.session_id)
    if (session) {
      currentSession.value = session
      router.replace({ path: '/hr/ai', query: { session_id: session.id } })
    }
  } catch (error: unknown) {
    if (userAborted.value) return
    markAssistantError(assistantIndex, error instanceof Error ? error : new Error('AI 流式响应失败'))
    ElMessage.error(error instanceof Error ? error.message : 'AI 流式响应失败')
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
    messages.value[messages.value.length - 1] = {
      ...msg,
      pending: false,
      content: msg.content || '已中断回复',
    }
  }
  loading.value = false
  streaming.value = false
}

const submit = async () => {
  const text = input.value.trim()
  if (!text) return
  if (!currentSession.value) {
    await createNewSession()
  }
  input.value = ''
  messages.value.push({ role: 'user', content: text })
  const session = currentSession.value
  if (!session) return
  loading.value = true
  streaming.value = true
  scrollBottom()
  const controller = new AbortController()
  activeController.value = controller
  userAborted.value = false
  const assistantIndex = messages.value.length
  try {
    messages.value.push({ role: 'assistant', content: '', pending: true, waitingText: session.application_id ? '分析中' : '响应中' })
    let finalPayload: StreamPayload | null = null
    let streamFailed = false
    await sendMessageStream(
      { message: text, session_id: session.id },
      {
        onDelta: (delta) => {
          appendAssistantDelta(assistantIndex, delta)
        },
        onStatus: (_eventType, eventMessage) => {
          const msg = messages.value[assistantIndex]
          if (msg) {
            messages.value[assistantIndex] = { ...msg, waitingText: eventMessage }
          }
        },
        onDone: (payload) => {
          finalPayload = payload
          const options = parseCandidateOptions(payload.candidate_options)
          if (options.length > 0) {
            const msg = messages.value[assistantIndex]
            if (msg) messages.value[assistantIndex] = { ...msg, candidateOptions: options, pending: false }
          }
        },
        onError: (_errorType, errorMessage) => {
          streamFailed = true
          markAssistantError(assistantIndex, new Error(errorMessage))
        },
      },
      { signal: controller.signal, silentAbort: true },
    )
    scrollBottom()
    if (streamFailed) return
    if (finalPayload) {
      if ((finalPayload as StreamPayload).session_id && currentSession.value) {
        currentSession.value = { ...currentSession.value, id: (finalPayload as StreamPayload).session_id! }
      }
      if (!userAborted.value) {
        await confirmAction(finalPayload)
      }
      if (!messages.value[assistantIndex]?.content) {
        const sid = (finalPayload as StreamPayload).session_id || session.id
        const data = await getSessionMessages(sid, { page: 1, page_size: 100 })
        messages.value = (data.list || messages.value) as MessageItem[]
      }
    }
    if (!userAborted.value) {
      await refreshSessions()
    }
  } catch (error: unknown) {
    if (userAborted.value) return
    markAssistantError(assistantIndex, error instanceof Error ? error : new Error('AI 流式响应失败'))
    input.value = text
    const err = error as { code?: string; message?: string }
    if (err.code === 'ECONNABORTED') {
      ElMessage.warning('AI 分析耗时较长，请稍后重新发送')
    } else {
      ElMessage.error(err.message || 'AI 流式响应失败')
    }
  } finally {
    // Only clear state if we're still the active request (not a stale finally).
    if (activeController.value === controller) {
      loading.value = false
      streaming.value = false
      activeController.value = null
      userAborted.value = false
    }
  }
}

onMounted(async () => {
  document.addEventListener('click', closeMenu)
  await refreshSessions()
  if (await createAnalysisSessionFromRoute()) return
  const querySessionId = Number(route.query.session_id || 0)
  const target = sessions.value.find((item) => item.id === querySessionId) || sessions.value[0]
  if (target) {
    await selectSession(target)
  }
})

const closeMenu = () => { menuSessionId.value = 0 }
const toggleSessionSidebar = () => { sessionSidebarOpen.value = !sessionSidebarOpen.value }
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

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (typewriterTimer) clearInterval(typewriterTimer)
  document.removeEventListener('click', closeMenu)
})
</script>

<template>
  <section class="chat-page">
    <div v-if="sessionSidebarOpen" class="mobile-sidebar-backdrop" @click="closeSessionSidebar"></div>
    <aside class="chat-sidebar" :class="{ 'chat-sidebar--mobile-open': sessionSidebarOpen }">
      <div class="chat-sidebar__head">
        <h2>AI 会话</h2>
        <el-button size="small" type="primary" @click="createNewSession">新建对话</el-button>
      </div>
      <div class="session-list">
        <div v-for="session in sessions" :key="session.id" class="session-item" :class="{ 'session-item--active': currentSession?.id === session.id, 'session-item--menu-open': menuSessionId === session.id }" @click="selectSession(session)">
          <div class="session-item__info">
            <span class="session-title">{{ session.title }}</span>
            <span class="session-meta">{{ session.application_id ? '简历分析' : '数据问答' }}</span>
          </div>
          <div class="session-item__actions">
            <button class="session-item__more" @click.stop="menuSessionId = menuSessionId === session.id ? 0 : session.id">⋮</button>
            <div v-if="menuSessionId === session.id" class="session-item__menu" @click.stop>
              <button @click.stop="menuSessionId = 0; renameSession(session)">重命名</button>
              <button @click.stop="menuSessionId = 0; removeSession(session)">删除会话</button>
            </div>
          </div>
        </div>
        <el-empty v-if="sessions.length === 0" description="暂无会话" />
      </div>
    </aside>

    <div class="chat-main">
      <div v-if="currentSession" class="ai-context">
        <!-- Top row: ☰ mobile toggle + tag -->
        <div class="ai-context__top">
          <button class="session-sidebar-toggle mobile-only" @click="toggleSessionSidebar">☰ 会话</button>
          <div class="ai-context__title desktop-only">{{ currentSession.title }}</div>
          <el-tag :type="currentSession.application_id ? 'success' : 'info'">{{ currentSession.application_id ? '简历分析' : '普通对话' }}</el-tag>
        </div>
        <!-- Desktop meta -->
        <div class="ai-context__meta desktop-only">{{ currentSession.application_id ? '候选人分析会话' : '招聘数据问答会话' }}</div>
        <!-- Mobile expanded info -->
        <div v-if="currentSession.application_id" class="ai-context__mobile mobile-only">
          <div class="ai-context__candidate-name">{{ mobileContextTitle }}</div>
          <div class="ai-context__candidate-pos">{{ mobileContextSub }}</div>
        </div>
      </div>

      <div class="chat-layout">
        <div ref="listRef" class="chat-list" v-loading="sessionLoading">
          <el-empty v-if="messages.length === 0 && !loading" description="暂无对话" />
          <div v-for="(message, index) in messages" :key="index" class="bubble" :class="[message.role, { 'bubble--failed': message.failed }]">
            <div v-if="message.role === 'assistant'">
              <div v-if="message.pending" class="typing-indicator">
                <span>{{ waitingText(message) }}</span>
                <span class="typing-indicator__dots">
                  <i></i>
                  <i></i>
                  <i></i>
                </span>
              </div>
              <div v-else-if="message.content" class="md-content" v-html="renderMarkdown(message.content)"></div>
            </div>
            <template v-else>{{ message.content }}</template>
            <div v-if="message.role === 'assistant' && message.candidateOptions?.length" class="candidate-options">
              <button v-for="option in message.candidateOptions" :key="option.application_id" class="candidate-option" @click="analyzeCandidateOption(option)">
                <span class="candidate-option__name">{{ option.candidate_name }}</span>
                <span>{{ option.job_title }}</span>
                <span>{{ option.masked_phone }}</span>
                <span>第 {{ option.round_no }} 轮</span>
                <strong>分析</strong>
              </button>
            </div>
          </div>
          <div v-if="loading && !streaming" class="bubble assistant">分析中...</div>
        </div>
        <div class="chat-input">
          <el-input v-model="input" :disabled="streaming" :placeholder="currentSession?.application_id ? '例如：他的项目经历和岗位要求匹配吗？也可以说“通过这个候选人”' : '例如：今天后端岗位投递了多少人？'" @keyup.enter="streaming ? undefined : submit()" />
          <el-button v-if="streaming" type="danger" plain @click="stopStreaming">中断</el-button>
          <el-button v-else type="primary" :loading="loading" :disabled="!input.trim()" @click="submit">发送</el-button>
        </div>
      </div>
    </div>
  </section>
</template>
