<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ChatDotRound, Close, Delete, EditPen, Moon, Plus, Promotion, Sunny, User } from '@element-plus/icons-vue'
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import {
  createSession,
  deleteSession,
  getSessionMessages,
  listSessions,
  sendMessageStream,
  updateSession,
} from '@/api/ai'
import { applyJob } from '@/api/application'
import type { CandidateSession, RecommendedJob, StreamPayload } from '@/types/ai'

interface MessageItem {
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  failed?: boolean
  waitingText?: string
  actionPayload?: CandidateAIActionPayload | null
  suggestedQuestions?: string[]
}

interface CandidateAIActionPayload {
  action: string
  jobs?: RecommendedJob[]
  job_id?: number
}

const router = useRouter()

const panelOpen = ref(false)
const input = ref('')
const messages = ref<MessageItem[]>([])
const loading = ref(false)
const streaming = ref(false)
const activeController = ref<AbortController | null>(null)
const userAborted = ref(false)
const sessions = ref<CandidateSession[]>([])
const currentSession = ref<CandidateSession | null>(null)
const sessionListOpen = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const listRef = ref<any>(null)

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

const ZWS = '​' // zero-width space

const normalizeMarkdown = (content: string): string => {
  if (!content) return ''
  return content
    // Strip whitespace between ** and enclosed text.
    // AI models often output "** text **" which CommonMark doesn't parse as bold.
    .replace(/\*\* +/g, '**')
    .replace(/ +\*\*/g, '**')
    // Insert ZWS between emphasis delimiters and adjacent quotation marks.
    // In CJK text there's no whitespace around punctuation, so ** followed by
    // a quote (e.g. 是**"文字"**) makes the delimiter non-flanking per CommonMark.
    // A zero-width space restores flanking without adding visible gaps.
    .replace(/(\*\*|__|\*|_)([""""「」『』])/g, `$1${ZWS}$2`)
    .replace(/([""""「」『』])(\*\*|__|\*|_)/g, `$1${ZWS}$2`)
}

const renderMarkdown = (content: string): string => {
  const raw = DOMPurify.sanitize(md.render(normalizeMarkdown(content)), {
    ALLOWED_TAGS: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'p', 'br', 'hr', 'strong', 'b', 'em', 'i', 'u', 's', 'ul', 'ol', 'li', 'code', 'pre', 'a', 'blockquote'],
    ALLOWED_ATTR: ['href', 'target'],
    ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|tel):|[^a-z]|[a-z+.-]+(?:[^a-z+.\-:]|$))/i,
  })
  // Add rel="noopener noreferrer" to external links opened in new tabs.
  return raw.replace(/<a\s/g, '<a rel="noopener noreferrer" ')
}

const waitingText = (msg: MessageItem): string => msg.waitingText || '思考中'

const quickActions = [
  { label: '我的应聘进度怎么样？', icon: Promotion },
  { label: '根据我的简历推荐岗位', icon: Sunny },
  { label: '帮我优化简历', icon: Moon },
]

const scrollBottom = async () => {
  await nextTick()
  if (listRef.value) listRef.value.scrollTop = listRef.value.scrollHeight
}

const togglePanel = () => {
  panelOpen.value = !panelOpen.value
  if (panelOpen.value) {
    refreshSessions().then(() => {
      if (!currentSession.value && sessions.value.length > 0) {
        selectSession(sessions.value[0])
      }
    })
  }
}

const formatSessionTitle = (title: string, createdAt: string): string => {
  if (title && title !== '新对话') return title
  if (createdAt) {
    const d = new Date(createdAt)
    if (!isNaN(d.getTime())) {
      const pad = (n: number) => String(n).padStart(2, '0')
      return `对话 ${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
    }
  }
  return '新对话'
}

const refreshSessions = async () => {
  try {
    const data = await listSessions({ page: 1, page_size: 50 })
    sessions.value = (data.list || []).map((item) => ({
      session_id: item.session_id,
      title: formatSessionTitle(item.title || '', item.created_at || ''),
      created_at: item.created_at,
      updated_at: item.updated_at,
    }))
  } catch {
    // silently ignore
  }
}

const newSession = async () => {
  try {
    const data = await createSession({ title: '新对话' })
    const session: CandidateSession = {
      session_id: data.session.session_id,
      title: data.session.title || '新对话',
      created_at: data.session.created_at,
      updated_at: data.session.updated_at,
    }
    sessions.value.unshift(session)
    currentSession.value = session
    messages.value = []
  } catch {
    ElMessage.error('创建会话失败')
  }
}

const selectSession = async (session: CandidateSession) => {
  currentSession.value = session
  sessionListOpen.value = false
  try {
    const data = await getSessionMessages(session.session_id, { page: 1, page_size: 100 })
    messages.value = (data.list || []) as MessageItem[]
  } catch {
    messages.value = []
  }
  scrollBottom()
}

const removeSession = async (session: CandidateSession) => {
  try {
    await ElMessageBox.confirm(`确认删除会话「${session.title}」？`, '删除会话', { type: 'warning' })
  } catch {
    return
  }
  await deleteSession(session.session_id)
  sessions.value = sessions.value.filter((s) => s.session_id !== session.session_id)
  if (currentSession.value?.session_id === session.session_id) {
    currentSession.value = null
    messages.value = []
  }
  ElMessage.success('会话已删除')
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
  const renamed = { ...session, title }
  sessions.value = sessions.value.map((item) =>
    item.session_id === session.session_id ? { ...item, title } : item,
  )
  if (currentSession.value?.session_id === session.session_id) {
    currentSession.value = { ...currentSession.value, title: renamed.title }
  }
  ElMessage.success('会话已重命名')
}

const parseActionPayload = (raw: string): CandidateAIActionPayload | null => {
  if (!raw) return null
  try {
    return JSON.parse(raw) as CandidateAIActionPayload
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

const buildFallbackSuggestedQuestions = (userMessage: string, assistantReply: string): string[] => {
  const text = `${userMessage}\n${assistantReply}`.toLowerCase()
  if (text.includes('上传简历') || text.includes('没有上传') || text.includes('解析')) {
    return ['简历支持哪些格式？', '上传简历后能做什么？', '如何提高简历解析效果？']
  }
  if (text.includes('投递') || text.includes('应聘') || text.includes('待查看') || text.includes('已查看') || text.includes('人才库') || text.includes('通过')) {
    return ['这个状态代表什么？', '我还适合投哪些岗位？', '需要更新简历吗？']
  }
  if (text.includes('推荐') || text.includes('岗位') || text.includes('匹配')) {
    return ['为什么推荐这些岗位？', '哪些岗位我已投递？', '帮我比较前两个岗位']
  }
  if (text.includes('优化') || text.includes('简历') || text.includes('经历') || text.includes('技能')) {
    return ['我的简历最大短板是什么？', '哪些经历需要量化？', '适合投递什么岗位？']
  }
  return ['我目前的应聘进度？', '根据简历推荐岗位', '帮我优化简历建议']
}

const isLatestAssistantMessage = (index: number, msg: MessageItem): boolean =>
  msg.role === 'assistant' && index === messages.value.length - 1

const askSuggestedQuestion = (question: string) => {
  if (loading.value) return
  send(question)
}

const send = async (text?: string) => {
  const message = (text || input.value).trim()
  if (!message || loading.value) return
  input.value = ''

  if (!currentSession.value) {
    await newSession()
  }
  if (!currentSession.value) return

  const session = currentSession.value
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
    const result = { payload: null as StreamPayload | null }
    await sendMessageStream(
      { message, session_id: session.session_id },
      {
        onDelta: (delta) => {
          const msg = messages.value[assistantIndex]
          if (msg) {
            messages.value[assistantIndex] = { ...msg, content: (msg.content || '') + delta, pending: false }
          }
          scrollBottom()
        },
        onDone: (payload: StreamPayload) => {
          result.payload = payload
        },
      },
      { signal: controller.signal, silentAbort: true },
    )

    if (userAborted.value) return

    const finalPayload = result.payload
    if (finalPayload) {
      const msg = messages.value[assistantIndex]
      if (msg && finalPayload.action_payload) {
        const actionPayload = parseActionPayload(finalPayload.action_payload)
        if (actionPayload) {
          messages.value[assistantIndex] = { ...msg, actionPayload }
        }
      }
      const latestMsg = messages.value[assistantIndex]
      if (latestMsg) {
        const suggestedQuestions = normalizeSuggestedQuestions(finalPayload.suggested_questions ?? finalPayload.suggestedQuestions)
        messages.value[assistantIndex] = {
          ...latestMsg,
          suggestedQuestions: suggestedQuestions.length > 0
            ? suggestedQuestions
            : buildFallbackSuggestedQuestions(message, latestMsg.content),
        }
      }
      if (finalPayload.session_id && currentSession.value) {
        currentSession.value = { ...currentSession.value, session_id: finalPayload.session_id }
      }
    }
    await refreshSessions()
  } catch {
    if (userAborted.value) return
    const msg = messages.value[assistantIndex]
    if (msg) messages.value[assistantIndex] = { ...msg, failed: true, pending: false }
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
      suggestedQuestions: undefined,
    }
  }
  loading.value = false
  streaming.value = false
}

const retry = () => {
  const lastUser = [...messages.value].reverse().find((m) => m.role === 'user')
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
  try {
    await applyJob({ job_id: job.job_id })
    ElMessage.success('投递成功')
    job.has_applied = true
  } catch {
    // error handled by interceptor
  }
}

const clearInput = () => {
  input.value = ''
}

watch(panelOpen, (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'
    scrollBottom()
  } else {
    document.body.style.overflow = ''
  }
})

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})
</script>

<template>
  <div class="candidate-ai" :class="{ 'candidate-ai--open': panelOpen }">
    <!-- Floating avatar button -->
    <button class="ai-avatar" :class="{ 'ai-avatar--active': panelOpen }" @click="togglePanel"
      :aria-label="panelOpen ? '关闭 AI 助手' : '打开 AI 助手'">
      <div class="ai-avatar__robot">
        <div class="ai-robot__head">
          <div class="ai-robot__eye ai-robot__eye--left"></div>
          <div class="ai-robot__eye ai-robot__eye--right"></div>
          <div class="ai-robot__mouth"></div>
        </div>
        <div class="ai-robot__body">
          <div class="ai-robot__core"></div>
        </div>
      </div>
      <span class="ai-avatar__pulse"></span>
    </button>

    <!-- Chat panel -->
    <Transition name="panel-slide">
      <div v-if="panelOpen" class="ai-panel" :class="{ 'ai-panel--sidebar-open': sessionListOpen }">
        <!-- Session sidebar drawer -->
        <div class="ai-panel__sidebar" :class="{ 'ai-panel__sidebar--open': sessionListOpen }">
          <div class="ai-panel__sidebar-head">
            <span>历史会话</span>
            <button class="ai-panel__sidebar-close" @click="sessionListOpen = false" title="关闭">
              <el-icon :size="18"><Close /></el-icon>
            </button>
          </div>
          <div class="ai-panel__sidebar-list">
            <div
              v-for="session in sessions"
              :key="session.session_id"
              class="ai-session-item"
              :class="{ 'ai-session-item--active': currentSession?.session_id === session.session_id }"
              @click="selectSession(session)"
            >
              <span class="ai-session-item__title">{{ session.title }}</span>
              <div class="ai-session-item__actions">
                <button class="ai-session-item__act" title="重命名" @click.stop="renameSession(session)">
                  <el-icon :size="13"><EditPen /></el-icon>
                </button>
                <button class="ai-session-item__act ai-session-item__act--danger" title="删除" @click.stop="removeSession(session)">
                  <el-icon :size="13"><Delete /></el-icon>
                </button>
              </div>
            </div>
            <el-empty v-if="sessions.length === 0" description="暂无会话" :image-size="48" />
          </div>
        </div>

        <!-- Main chat area -->
        <div class="ai-panel__main" @click="sessionListOpen = false">
        <!-- Panel header -->
        <div class="ai-panel__head" @click.stop>
          <div class="ai-panel__title-group">
            <button class="ai-panel__sessions-btn" :class="{ 'ai-panel__sessions-btn--active': sessionListOpen }" @click="sessionListOpen = !sessionListOpen" title="历史会话">
              <el-icon :size="20"><ChatDotRound /></el-icon>
            </button>
            <span class="ai-panel__title">{{ currentSession?.title || 'AI 助手' }}</span>
          </div>
          <div class="ai-panel__head-actions">
            <el-button type="primary" :icon="Plus" @click="newSession">新会话</el-button>
            <el-button size="small" text @click="panelOpen = false" title="关闭">
              <el-icon :size="18"><Close /></el-icon>
            </el-button>
          </div>
        </div>

        <!-- Messages -->
        <div ref="listRef" class="ai-panel__messages">
          <!-- Quick actions -->
          <div v-if="messages.length === 0 && !loading" class="ai-quick">
            <p class="ai-quick__hint">我是你的 AI 求职助手，可以帮你：</p>
            <button
              v-for="action in quickActions"
              :key="action.label"
              class="ai-quick__btn"
              @click="send(action.label)"
            >
              <el-icon :size="16"><component :is="action.icon" /></el-icon>
              <span>{{ action.label }}</span>
            </button>
          </div>

          <!-- Message bubbles -->
          <div
            v-for="(msg, index) in messages"
            :key="index"
            class="ai-bubble"
            :class="[msg.role, { 'ai-bubble--failed': msg.failed }]"
          >
            <div class="ai-bubble__avatar">
              <el-icon v-if="msg.role === 'user'" :size="16"><User /></el-icon>
              <div v-else class="ai-robot__head ai-robot__head--mini">
                <div class="ai-robot__eye ai-robot__eye--left"></div>
                <div class="ai-robot__eye ai-robot__eye--right"></div>
              </div>
            </div>
            <div class="ai-bubble__body">
              <div v-if="msg.pending" class="ai-typing">
                <span>{{ waitingText(msg) }}</span>
                <span class="ai-typing__dots"><i></i><i></i><i></i></span>
              </div>
              <div v-else class="ai-bubble__content" v-html="renderMarkdown(msg.content)"></div>

              <!-- Action payload: recommend_jobs -->
              <div v-if="msg.actionPayload?.action === 'recommend_jobs' && msg.actionPayload.jobs?.length" class="ai-recommend">
                <div v-for="job in msg.actionPayload.jobs" :key="job.job_id" class="ai-recommend__card">
                  <div class="ai-recommend__info">
                    <div class="ai-recommend__title">{{ job.title }}</div>
                    <div class="ai-recommend__meta">
                      <span>{{ job.department || '部门待定' }}</span>
                      <span>{{ job.location || '地点待定' }}</span>
                      <span>{{ job.salary_range ? job.salary_range + ' 元/月' : '薪资面议' }}</span>
                    </div>
                    <div v-if="job.match_score" class="ai-recommend__score">匹配度 {{ job.match_score }}%</div>
                    <div v-if="job.reasons?.length" class="ai-recommend__reasons">
                      <span v-for="reason in job.reasons" :key="reason">{{ reason }}</span>
                    </div>
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

              <div
                v-if="isLatestAssistantMessage(index, msg) && msg.suggestedQuestions?.length"
                class="ai-suggested"
                aria-label="你可以继续问"
              >
                <button
                  v-for="question in msg.suggestedQuestions"
                  :key="question"
                  class="ai-suggested__btn"
                  :disabled="loading"
                  @click="askSuggestedQuestion(question)"
                >
                  {{ question }}
                </button>
              </div>

              <div v-if="msg.failed" class="ai-bubble__error">
                发送失败
                <el-button size="small" text type="primary" @click="retry">重试</el-button>
              </div>
            </div>
          </div>

          <div v-if="loading && !streaming" class="ai-bubble assistant">
            <div class="ai-bubble__avatar">
              <div class="ai-robot__head ai-robot__head--mini">
                <div class="ai-robot__eye ai-robot__eye--left"></div>
                <div class="ai-robot__eye ai-robot__eye--right"></div>
              </div>
            </div>
            <div class="ai-bubble__body">
              <div class="ai-typing"><span>思考中</span><span class="ai-typing__dots"><i></i><i></i><i></i></span></div>
            </div>
          </div>
        </div>

        <!-- Input -->
        <div class="ai-panel__input">
          <el-input
            v-model="input"
            placeholder="输入你的问题..."
            :disabled="streaming"
            @keyup.enter="streaming ? undefined : send()"
            clearable
            @clear="clearInput"
          />
          <el-button v-if="streaming" type="danger" plain @click="stopStreaming">
            中断
          </el-button>
          <el-button v-else type="primary" :loading="loading" :disabled="!input.trim()" @click="send()">
            发送
          </el-button>
        </div>
        </div><!-- /ai-panel__main -->
      </div>
    </Transition>
  </div>
</template>

<style scoped>
/* ---- Floating avatar ---- */

.candidate-ai {
  position: fixed;
  right: max(24px, env(safe-area-inset-right));
  bottom: max(24px, env(safe-area-inset-bottom));
  z-index: 1200;
}

.ai-avatar {
  width: 56px;
  height: 56px;
  border: none;
  border-radius: 50%;
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
  box-shadow: 0 6px 24px rgba(37, 99, 235, 0.4);
  cursor: pointer;
  display: grid;
  place-items: center;
  position: relative;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.ai-avatar:hover {
  transform: scale(1.08);
  box-shadow: 0 8px 32px rgba(37, 99, 235, 0.5);
}

.ai-avatar--active {
  transform: scale(0.92);
  box-shadow: 0 4px 16px rgba(37, 99, 235, 0.3);
}

.ai-avatar__pulse {
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  border: 2px solid rgba(37, 99, 235, 0.3);
  animation: ai-pulse 2.5s ease-in-out infinite;
  pointer-events: none;
}

@keyframes ai-pulse {
  0%, 100% { transform: scale(1); opacity: 0.4; }
  50%      { transform: scale(1.12); opacity: 0; }
}

/* ---- CSS Robot ---- */

.ai-avatar__robot {
  display: grid;
  place-items: center;
  gap: 2px;
}

.ai-robot__head {
  width: 20px;
  height: 18px;
  background: #fff;
  border-radius: 5px 5px 3px 3px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr auto;
  place-items: center;
  padding: 2px 3px;
  position: relative;
}

.ai-robot__head--mini {
  width: 16px;
  height: 14px;
  border-radius: 4px 4px 2px 2px;
  padding: 1px 2px;
}

.ai-robot__eye {
  width: 5px;
  height: 5px;
  background: #1d4ed8;
  border-radius: 50%;
  animation: ai-blink 3s ease-in-out infinite;
}

.ai-robot__head--mini .ai-robot__eye {
  width: 4px;
  height: 4px;
}

@keyframes ai-blink {
  0%, 95%, 100% { transform: scaleY(1); }
  97%            { transform: scaleY(0.1); }
}

.ai-robot__mouth {
  grid-column: 1 / -1;
  width: 8px;
  height: 2px;
  background: #93c5fd;
  border-radius: 0 0 2px 2px;
  margin-top: 2px;
}

.ai-robot__body {
  width: 22px;
  height: 12px;
  background: #fff;
  border-radius: 2px 2px 4px 4px;
  display: grid;
  place-items: center;
}

.ai-robot__core {
  width: 6px;
  height: 6px;
  background: #3b82f6;
  border-radius: 50%;
  animation: ai-core-glow 1.8s ease-in-out infinite;
}

@keyframes ai-core-glow {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%      { opacity: 0.5; transform: scale(0.8); }
}

/* ---- Chat Panel ---- */

.ai-panel {
  position: absolute;
  right: 0;
  bottom: 72px;
  width: 420px;
  max-width: calc(100vw - 48px);
  height: 560px;
  max-height: calc(100vh - 120px);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 16px;
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Panel slide transition */
.panel-slide-enter-active,
.panel-slide-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.panel-slide-enter-from,
.panel-slide-leave-to {
  opacity: 0;
  transform: translateY(12px) scale(0.97);
}

/* Panel header */
.ai-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.ai-panel__title-group {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.ai-panel__sessions-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: grid;
  place-items: center;
}

.ai-panel__sessions-btn:hover,
.ai-panel__sessions-btn--active {
  background: var(--surface-muted);
  color: var(--brand);
}

.ai-panel__title {
  font-weight: 700;
  font-size: 15px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-panel__head-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* Session sidebar drawer */
.ai-panel__sidebar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 260px;
  background: var(--surface);
  border-right: 1px solid var(--border);
  border-radius: 16px 0 0 16px;
  display: flex;
  flex-direction: column;
  transform: translateX(-100%);
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 2;
}

.ai-panel__sidebar--open {
  transform: translateX(0);
}

.ai-panel__sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  font-weight: 600;
  font-size: 14px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.ai-panel__sidebar-close {
  border: none;
  background: transparent;
  color: var(--text-faint);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: grid;
  place-items: center;
  transition: background 0.15s, color 0.15s;
}

.ai-panel__sidebar-close:hover {
  background: var(--surface-muted);
  color: var(--text-primary);
}

.ai-panel__sidebar-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.ai-panel__main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.ai-session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 13px;
  transition: background-color 0.15s;
}

.ai-session-item:hover,
.ai-session-item--active {
  background: var(--surface-muted);
  color: var(--text-primary);
}

.ai-session-item__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
}

.ai-session-item__actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
}

.ai-session-item:hover .ai-session-item__actions,
.ai-session-item--active .ai-session-item__actions {
  opacity: 1;
}

.ai-session-item__act {
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-faint);
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: background 0.15s, color 0.15s;
}

.ai-session-item__act:hover {
  background: var(--border);
  color: var(--text-secondary);
}

.ai-session-item__act--danger:hover {
  background: #fef2f2;
  color: #ef4444;
}

/* Messages */
.ai-panel__messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
}

/* Quick actions */
.ai-quick {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: center;
  padding: 20px 0;
}

.ai-quick__hint {
  margin: 0 0 4px;
  color: var(--text-muted);
  font-size: 14px;
}

.ai-quick__btn {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
  transition: border-color 0.15s, background-color 0.15s, transform 0.15s;
}

.ai-quick__btn:hover {
  border-color: var(--brand);
  background: var(--brand-soft);
  transform: translateY(-1px);
}

/* Message bubbles */
.ai-bubble {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.ai-bubble.user {
  flex-direction: row-reverse;
}

.ai-bubble__avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  background: var(--surface-muted);
  color: var(--text-secondary);
}

.ai-bubble.user .ai-bubble__avatar {
  background: var(--brand);
  color: #fff;
}

.ai-bubble.assistant .ai-bubble__avatar {
  background: var(--brand-soft);
  color: var(--brand);
}

.ai-bubble__body {
  max-width: calc(100% - 52px);
  min-width: 0;
}

.ai-bubble__content {
  background: var(--surface-muted);
  border-radius: 12px;
  padding: 10px 14px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-primary);
  word-break: break-word;
}

.ai-bubble.user .ai-bubble__content {
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
  color: #fff;
}

.ai-bubble.assistant .ai-bubble__content {
  border-bottom-left-radius: 4px;
}

.ai-bubble.user .ai-bubble__content {
  border-bottom-right-radius: 4px;
}

.ai-bubble__content :deep(p) { margin: 0 0 8px; }
.ai-bubble__content :deep(p:last-child) { margin-bottom: 0; }
.ai-bubble__content :deep(ul),
.ai-bubble__content :deep(ol) { margin: 6px 0 8px 20px; padding: 0; }
.ai-bubble__content :deep(li) { margin: 2px 0; }
.ai-bubble__content :deep(strong),
.ai-bubble__content :deep(b) { font-weight: 600; }
.ai-bubble__content :deep(code) { padding: 1px 5px; background: var(--border); border-radius: 4px; font-size: 13px; }
.ai-bubble__content :deep(pre) { margin: 8px 0; padding: 10px; background: #0f172a; color: #e2e8f0; border-radius: 8px; overflow: auto; font-size: 13px; }
.ai-bubble__content :deep(pre code) { padding: 0; background: transparent; color: inherit; }
.ai-bubble__content :deep(blockquote) { margin: 8px 0; padding: 6px 12px; border-left: 3px solid var(--brand); background: var(--brand-soft); }
.ai-bubble__content :deep(a) { color: var(--brand); text-decoration: underline; }
.ai-bubble__content :deep(hr) { border: none; border-top: 1px solid var(--border); margin: 8px 0; }

.ai-bubble--failed .ai-bubble__content {
  color: #991b1b;
  background: #fef2f2;
  border: 1px solid #fecaca;
}

.ai-bubble__error {
  margin-top: 6px;
  font-size: 12px;
  color: #ef4444;
}

/* Suggested follow-up questions */
.ai-suggested {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.ai-suggested__btn {
  max-width: 100%;
  border: 1px solid rgba(37, 99, 235, 0.28);
  border-radius: 999px;
  background: var(--surface);
  color: var(--brand);
  cursor: pointer;
  font-size: 12px;
  line-height: 1.3;
  padding: 6px 10px;
  text-align: left;
  transition:
    background-color var(--motion-fast) var(--motion-ease),
    border-color var(--motion-fast) var(--motion-ease),
    color var(--motion-fast) var(--motion-ease),
    transform var(--motion-fast) var(--motion-ease);
}

.ai-suggested__btn:hover:not(:disabled) {
  background: var(--brand-soft);
  border-color: rgba(37, 99, 235, 0.55);
  transform: translateY(-1px);
}

.ai-suggested__btn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
  transform: none;
}

/* Typing indicator */
.ai-typing {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--surface-muted);
  border-radius: 12px;
  color: var(--text-muted);
  font-size: 14px;
}

.ai-typing__dots {
  display: flex;
  gap: 3px;
  align-items: center;
}

.ai-typing__dots i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-faint);
  animation: typing-bounce 1.2s infinite ease-in-out;
}

.ai-typing__dots i:nth-child(2) { animation-delay: 0.2s; }
.ai-typing__dots i:nth-child(3) { animation-delay: 0.4s; }

@keyframes typing-bounce {
  0%, 60%, 100% { transform: translateY(0) scale(1); opacity: 0.4; }
  30%           { transform: translateY(-4px) scale(1.2); opacity: 1; }
}

/* Recommend cards */
.ai-recommend {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-recommend__card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px;
  background: var(--surface);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.ai-recommend__card:hover {
  border-color: var(--brand);
  box-shadow: var(--shadow-hover);
}

.ai-recommend__info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ai-recommend__title {
  font-weight: 700;
  font-size: 14px;
  color: var(--text-primary);
}

.ai-recommend__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 12px;
  font-size: 12px;
  color: var(--text-muted);
}

.ai-recommend__score {
  font-size: 12px;
  color: var(--brand);
  font-weight: 600;
}

.ai-recommend__reasons {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  margin-top: 4px;
}

.ai-recommend__reasons span {
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--surface-muted);
  padding: 2px 8px;
  border-radius: 4px;
}

.ai-recommend__actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex-shrink: 0;
}

/* Input */
.ai-panel__input {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

.ai-panel__input .el-input {
  flex: 1;
}

/* Mobile */
@media (max-width: 560px) {
  .ai-panel {
    width: calc(100vw - 16px);
    right: 8px;
    bottom: env(safe-area-inset-bottom, 12px);
    height: min(78dvh, 680px);
    max-height: calc(100dvh - 72px);
    border-radius: 16px 16px 8px 8px;
  }

  .ai-avatar {
    width: 48px;
    height: 48px;
  }

  .ai-panel__head {
    padding: 10px 12px;
  }

  .ai-panel__head-actions .el-button {
    min-height: var(--touch-size);
  }

  .ai-panel__sidebar {
    width: min(320px, 86vw);
    border-radius: 16px 0 0 16px;
  }

  .ai-panel__input {
    padding: 10px 12px;
    flex-wrap: wrap;
    gap: 6px;
  }

  .ai-panel__input .el-input {
    min-width: 0;
  }

  .ai-panel__messages {
    padding: 12px;
  }

  .ai-recommend__card {
    flex-direction: column;
  }

  .ai-recommend__actions {
    flex-direction: row;
    width: 100%;
  }

  /* Markdown overflow */
  .ai-bubble__content :deep(pre),
  .ai-bubble__content :deep(table) {
    display: block;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }
}

/* Reduced motion */
@media (prefers-reduced-motion: reduce) {
  .ai-avatar__pulse,
  .ai-robot__eye,
  .ai-robot__core,
  .ai-typing__dots i {
    animation: none;
  }
}
</style>
