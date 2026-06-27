<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import { createApplicationAnalysisSession, createSession, deleteSession, getSessionMessages, listSessions, sendMessageStream, updateSession } from '@/api/ai'
import { updateApplicationStatus } from '@/api/application'
import { listModels } from '@/api/llm'
import AgentTracePanel from '@/components/AgentTracePanel.vue'
import ConversationSidebar from '@/components/chat/ConversationSidebar.vue'
import ConversationHeader from '@/components/chat/ConversationHeader.vue'
import ChatMessageList from '@/components/chat/ChatMessageList.vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import type { ChatMessage, ChatSessionListItem, Session, CandidateOption, StreamPayload } from '@/types/ai'
import type { LlmModel } from '@/types/llm'
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
const statusBarExpanded = ref(true)
const modelName = ref('')
const modelList = ref<LlmModel[]>([])
const selectedModelId = ref<number | null>(null)
const dataSource = ref('招聘业务数据库')
const tracePanelVisible = ref(false)
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
  listRef.value?.scrollToBottom()
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
  if (data.model_name) {
    modelName.value = data.model_name
  }
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
  await router.replace({ path: '/hr/ai', query: { session_id: String(session.id) } })
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
      { message: messages.value[0].content, session_id: session.id, ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}) },
      {
        onDelta: (delta) => {
          const msg = messages.value[assistantIndex]
          if (msg) {
            messages.value[assistantIndex] = { ...msg, content: (msg.content || '') + delta, pending: false }
          }
          scrollBottom()
        },
        onStatus: (_eventType, eventMessage) => {
          if (_eventType === 'model_info') { modelName.value = eventMessage; return }
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
  messages.value.push({ role: 'assistant', content: `已将「${data.candidate_name || '该候选人'}」的投递状态更新为"${actionText}"。` })
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
      { message: userMessage, application_id: option.application_id, ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}) },
      {
        onDelta: (delta) => appendAssistantDelta(assistantIndex, delta),
        onStatus: (_eventType, eventMessage) => {
          if (_eventType === 'model_info') { modelName.value = eventMessage; return }
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
      { message: text, session_id: session.id, ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}) },
      {
        onDelta: (delta) => {
          appendAssistantDelta(assistantIndex, delta)
        },
        onStatus: (_eventType, eventMessage) => {
          if (_eventType === 'model_info') { modelName.value = eventMessage; return }
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

const retry = async (failedIndex: number) => {
  const failedMsg = messages.value[failedIndex]
  if (!failedMsg || failedMsg.role !== 'assistant' || !failedMsg.failed) return

  let lastUserContent = ''
  for (let i = failedIndex - 1; i >= 0; i--) {
    if (messages.value[i]?.role === 'user') {
      lastUserContent = messages.value[i].content
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

  const controller = new AbortController()
  activeController.value = controller
  userAborted.value = false

  try {
    let finalPayload: StreamPayload | null = null
    let streamFailed = false
    await sendMessageStream(
      { message: lastUserContent, session_id: session.id, ...(selectedModelId.value != null ? { model_id: selectedModelId.value } : {}) },
      {
        onDelta: (delta) => {
          appendAssistantDelta(assistantIndex, delta)
        },
        onStatus: (_eventType, eventMessage) => {
          if (_eventType === 'model_info') { modelName.value = eventMessage; return }
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
    const err = error as { code?: string; message?: string }
    if (err.code === 'ECONNABORTED') {
      ElMessage.warning('AI 分析耗时较长，请稍后重新发送')
    } else {
      ElMessage.error(err.message || 'AI 流式响应失败')
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

onMounted(async () => {
  document.addEventListener('click', closeMenu)
  // Load available models for the model selector.
  try {
    const modelData = await listModels(1, 200)
    modelList.value = (modelData.list || []).filter((m) => m.is_enabled)
  } catch { /* non-fatal: model selector will be empty */ }
  await refreshSessions()
  if (await createAnalysisSessionFromRoute()) return
  const querySessionId = Number(route.query.session_id || 0)
  const target = sessions.value.find((item) => item.id === querySessionId) || sessions.value[0]
  if (target) {
    await selectSession(target)
  }
})

const closeMenu = () => { menuSessionId.value = 0 }

// Sync status bar model name with user selection.
watch(selectedModelId, (id) => {
  if (id != null) {
    const m = modelList.value.find((x) => x.id === id)
    if (m) modelName.value = m.display_name || m.model_name
  }
})
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
    <div
      v-if="sessionSidebarOpen"
      class="mobile-sidebar-backdrop"
      @click="closeSessionSidebar"
    ></div>
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

    <div class="chat-main">
      <ConversationHeader
        v-if="currentSession"
        :current-session="currentSession"
        :mobile-context-title="mobileContextTitle"
        :mobile-context-sub="mobileContextSub"
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
          :render-markdown="renderMarkdown"
          :waiting-text="waitingText"
          @retry="retry"
          @analyze-candidate="analyzeCandidateOption"
        />

        <ChatComposer
          :input="input"
          :loading="loading"
          :streaming="streaming"
          :model-list="modelList"
          :selected-model-id="selectedModelId"
          :data-source="dataSource"
          :current-session="currentSession"
          @update:input="(val: string) => input = val"
          @update:selected-model-id="(val: number | null) => selectedModelId = val"
          @submit="submit"
          @stop="stopStreaming"
        />
      </div>
    </div>

    <AgentTracePanel
      v-model:visible="tracePanelVisible"
      :session-id="currentSession?.id ?? null"
    />
  </section>
</template>
