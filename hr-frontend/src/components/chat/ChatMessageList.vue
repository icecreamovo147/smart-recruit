<script setup lang="ts">
import { ref } from 'vue'
import type { AgentSkillSelectionPayload, CandidateOption, ChatMessageSkill } from '@/types/ai'

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
  pending?: boolean
  failed?: boolean
  retryDisabled?: boolean
  errorCode?: string
  waitingText?: string
  process_content?: string
  processContent?: string
  candidateOptions?: CandidateOption[]
  agentSkillSelection?: AgentSkillSelectionPayload
}

const props = defineProps<{
  messages: MessageItem[]
  loading: boolean
  streaming: boolean
  sessionLoading: boolean
  hasSession: boolean
  runningModeLabel: string
  renderMarkdown: (content: string) => string
  waitingText: (message: MessageItem) => string
}>()

const emit = defineEmits<{
  (e: 'retry', index: number): void
  (e: 'confirm-skill-selection', index: number, skillIds: number[]): void
}>()

defineExpose({ scrollToBottom })

const listRef = ref<HTMLElement | null>(null)

function scrollToBottom() {
  if (listRef.value) {
    listRef.value.scrollTop = listRef.value.scrollHeight
  }
}

const messageSkills = (message: MessageItem): ChatMessageSkill[] => {
  if (message.role !== 'user') return []
  if (message.skills?.length) return message.skills
  if (message.skill?.name) return [message.skill]
  const name = message.skill_name || message.skillName
  if (!name) return []
  return [{
    id: message.skill_id ?? message.skillId,
    name,
    command: message.skill_command || message.skillCommand || `/${name}`,
  }]
}

const skillBadgeText = (skill: ChatMessageSkill): string =>
  skill.command || (skill.name ? `/${skill.name}` : '')

const messageSkillBadges = (message: MessageItem): string[] =>
  messageSkills(message)
    .map(skillBadgeText)
    .filter(Boolean)

const loadingBadges = (index: number, fallbackMode: string): string[] => {
  const previousUserMessage = [...props.messages.slice(0, index)]
    .reverse()
    .find((message) => message.role === 'user')
  const badges = previousUserMessage ? messageSkillBadges(previousUserMessage) : []
  return badges.length > 0 ? badges : [fallbackMode]
}

const loadingTitle = (message: MessageItem): string => {
  const text = props.waitingText(message)
  if (text.includes('分析')) return '正在分析你的问题'
  if (text.includes('查询') || text.includes('检索')) return '正在查询相关数据'
  if (text.includes('生成') || text.includes('响应')) return '正在处理你的请求'
  return text || '正在处理你的请求'
}

const loadingDescription = (message: MessageItem): string => {
  const text = props.waitingText(message)
  if (text.includes('分析')) return '正在理解问题，并结合招聘数据进行分析...'
  if (text.includes('查询') || text.includes('检索')) return '正在查询相关数据，整理可用信息...'
  if (text.includes('生成') || text.includes('响应')) return '正在组织答案内容，稍后会开始输出...'
  return `${text}，请稍候...`
}

const selectedSkillIds = ref<Record<number, number[]>>({})

const skillSelectionLabel = (candidate: AgentSkillSelectionPayload['candidates'][number]): string =>
  candidate.display_name || candidate.name || `Skill #${candidate.id}`

const skillSelectionReason = (candidate: AgentSkillSelectionPayload['candidates'][number]): string => {
  if (candidate.reason) return candidate.reason
  if (candidate.category || candidate.scenario) return [candidate.category, candidate.scenario].filter(Boolean).join(' · ')
  return '系统推荐候选 Skill'
}

const defaultSelectionIds = (selection: AgentSkillSelectionPayload): number[] =>
  selection.recommended_agent_skill_ids?.length
    ? selection.recommended_agent_skill_ids
    : selection.candidates.filter((candidate) => candidate.recommended).map((candidate) => candidate.id)

const selectionIds = (index: number, selection: AgentSkillSelectionPayload): number[] =>
  selectedSkillIds.value[index] ?? defaultSelectionIds(selection)

const toggleSkillSelection = (index: number, selection: AgentSkillSelectionPayload, id: number) => {
  const current = selectionIds(index, selection)
  const next = current.includes(id) ? current.filter((item) => item !== id) : [...current, id]
  selectedSkillIds.value = { ...selectedSkillIds.value, [index]: next }
}

const confirmSkillSelection = (index: number, selection: AgentSkillSelectionPayload) => {
  emit('confirm-skill-selection', index, selectionIds(index, selection))
}

const skipSkillSelection = (index: number) => {
  selectedSkillIds.value = { ...selectedSkillIds.value, [index]: [] }
  emit('confirm-skill-selection', index, [])
}

const quickHints = [
  '今天后端岗位投递了多少人？',
  '最近一周的面试通过率是多少？',
  '帮我分析各部门岗位分布情况',
]
</script>

<template>
  <div ref="listRef" class="chat-messages" v-loading="sessionLoading">
    <!-- Welcome State -->
    <div v-if="messages.length === 0 && !loading && hasSession" class="chat-welcome">
      <div class="chat-welcome__icon">
        <svg width="44" height="44" viewBox="0 0 44 44" fill="none">
          <rect width="44" height="44" rx="12" fill="var(--brand-soft)"/>
          <circle cx="22" cy="22" r="11" stroke="var(--brand)" stroke-width="2.2" fill="none"/>
          <path d="M22 14v10l-7 3.5" stroke="var(--brand)" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
      <h3 class="chat-welcome__title">招聘数据助手</h3>
      <p class="chat-welcome__desc">可以问我招聘数据、候选人分析、岗位统计等问题</p>
      <div class="chat-welcome__hints">
        <button
          v-for="hint in quickHints"
          :key="hint"
          class="chat-welcome__hint"
        >
          {{ hint }}
        </button>
      </div>
    </div>

    <!-- Empty (no session) -->
    <el-empty
      v-if="messages.length === 0 && !loading && !hasSession"
      description="选择或创建一个会话开始对话"
      :image-size="72"
    />

    <!-- Messages -->
    <div
      v-for="(message, index) in messages"
      :key="index"
      class="message-wrapper"
      :class="[message.role]"
    >
      <div
        class="bubble"
        :class="{
          'bubble--user': message.role === 'user',
          'bubble--assistant': message.role === 'assistant',
          'bubble--failed': message.failed,
        }"
      >
        <template v-if="message.role === 'assistant'">
          <div v-if="message.agentSkillSelection" class="skill-confirmation">
            <div class="skill-confirmation__header">
              <div>
                <div class="skill-confirmation__title">
                  {{ message.agentSkillSelection.candidates.length ? '确认本次要调用的 Skill' : 'Skill 候选加载异常' }}
                </div>
                <div class="skill-confirmation__desc">
                  {{ message.agentSkillSelection.candidates.length ? '系统匹配到多个候选，请选择后继续生成回答。' : '未能读取到可选候选，可跳过 Skill 继续生成回答。' }}
                </div>
              </div>
              <el-tag size="small" effect="plain">待确认</el-tag>
            </div>
            <div class="skill-confirmation__list">
              <button
                v-for="candidate in message.agentSkillSelection.candidates"
                :key="candidate.id"
                class="skill-confirmation__option"
                :class="{ 'skill-confirmation__option--selected': selectionIds(index, message.agentSkillSelection).includes(candidate.id) }"
                type="button"
                @click="toggleSkillSelection(index, message.agentSkillSelection, candidate.id)"
              >
                <span class="skill-confirmation__check">
                  {{ selectionIds(index, message.agentSkillSelection).includes(candidate.id) ? '✓' : '' }}
                </span>
                <span class="skill-confirmation__body">
                  <span class="skill-confirmation__name">
                    {{ skillSelectionLabel(candidate) }}
                    <em v-if="candidate.recommended">推荐</em>
                  </span>
                  <span class="skill-confirmation__reason">{{ skillSelectionReason(candidate) }}</span>
                </span>
              </button>
            </div>
            <div class="skill-confirmation__actions">
              <el-button size="small" @click="skipSkillSelection(index)">不使用 Skill</el-button>
              <el-button type="primary" size="small" @click="confirmSkillSelection(index, message.agentSkillSelection)">继续</el-button>
            </div>
          </div>

          <div v-if="message.processContent" class="assistant-process">
            <div class="assistant-process__label">执行过程</div>
            <div class="assistant-process__content md-content" v-html="renderMarkdown(message.processContent)"></div>
          </div>

          <!-- Typing indicator -->
          <div v-if="message.pending && !message.content" class="assistant-loading-card">
            <div class="assistant-loading-card__badges">
              <span
                v-for="badge in loadingBadges(index, runningModeLabel)"
                :key="badge"
                class="assistant-loading-card__badge"
              >
                {{ badge }}
              </span>
            </div>
            <div class="assistant-loading-card__header">
              <span class="assistant-loading-card__pulse"></span>
              <div>
                <div class="assistant-loading-card__title">{{ loadingTitle(message) }}</div>
                <div class="assistant-loading-card__desc">{{ loadingDescription(message) }}</div>
              </div>
            </div>
            <div class="assistant-loading-card__steps">
              <div class="assistant-loading-card__step assistant-loading-card__step--done">
                <span>✓</span>
                <p>已接收问题</p>
              </div>
              <div class="assistant-loading-card__step assistant-loading-card__step--active">
                <span></span>
                <p>{{ waitingText(message) || '正在分析问题并查询数据' }}</p>
              </div>
              <div class="assistant-loading-card__step">
                <span></span>
                <p>等待生成最终回答</p>
              </div>
            </div>
            <div class="assistant-loading-card__footer">
              <span class="assistant-loading-card__dots" aria-hidden="true">
                <i></i>
                <i></i>
                <i></i>
              </span>
              <span>可随时在下方中断本次生成</span>
            </div>
            <div class="assistant-loading-card__skeleton" aria-hidden="true">
              <i></i>
              <i></i>
              <i></i>
            </div>
          </div>

          <!-- Assistant markdown content -->
          <div
            v-if="message.content"
            class="md-content"
            v-html="renderMarkdown(message.content)"
          ></div>
        </template>

        <!-- User plain text -->
        <template v-else-if="message.role === 'user'">
          <div v-if="messageSkills(message).length" class="message-skill-badges">
            <span
              v-for="skill in messageSkills(message)"
              :key="`${skill.id || skill.name}-${skillBadgeText(skill)}`"
              class="message-skill-badge"
            >
              {{ skillBadgeText(skill) }}
            </span>
          </div>
          <div class="bubble__user-text">{{ message.content }}</div>
        </template>

        <div
          v-if="message.role === 'assistant' && message.model_name && !message.pending"
          class="bubble__meta"
        >
          <el-tag size="small" effect="plain">{{ message.model_name }}</el-tag>
        </div>

        <!-- Retry button -->
        <div v-if="message.role === 'assistant' && message.failed && !message.retryDisabled" class="bubble__retry">
          <el-button type="warning" size="small" @click="emit('retry', index)">重新发送</el-button>
        </div>

      </div>
    </div>

    <!-- Loading placeholder -->
    <div v-if="loading && !streaming" class="message-wrapper assistant">
      <div class="bubble bubble--assistant">
        <div class="assistant-loading-card">
          <div class="assistant-loading-card__badges">
            <span class="assistant-loading-card__badge">{{ runningModeLabel }}</span>
          </div>
          <div class="assistant-loading-card__header">
            <span class="assistant-loading-card__pulse"></span>
            <div>
              <div class="assistant-loading-card__title">正在处理你的请求</div>
              <div class="assistant-loading-card__desc">正在理解问题并查询相关数据...</div>
            </div>
          </div>
          <div class="assistant-loading-card__footer">
            <span class="assistant-loading-card__dots" aria-hidden="true">
              <i></i><i></i><i></i>
            </span>
            <span>可随时在下方中断本次生成</span>
          </div>
          <div class="assistant-loading-card__skeleton" aria-hidden="true">
            <i></i>
            <i></i>
            <i></i>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-wrapper {
  display: flex;
  margin-bottom: 20px;
}

.message-wrapper.user {
  justify-content: flex-end;
}

.message-wrapper.assistant {
  justify-content: flex-start;
}

.chat-welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  padding-top: min(18vh, 120px);
  padding-bottom: 40px;
  text-align: center;
}

.chat-welcome__icon {
  margin-bottom: 14px;
  opacity: 0.82;
}

.chat-welcome__title {
  margin: 0 0 4px;
  font-size: 17px;
  font-weight: 700;
  color: var(--text-primary);
}

.chat-welcome__desc {
  margin: 0 0 24px;
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.6;
}

.chat-welcome__hints {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: center;
  max-width: 680px;
}

.chat-welcome__hint {
  display: inline-flex;
  align-items: center;
  height: 38px;
  padding: 0 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  color: var(--text-secondary);
  font-size: 12.5px;
  cursor: pointer;
  transition: border-color 0.15s ease, background-color 0.15s ease;
}

.chat-welcome__hint:hover {
  border-color: var(--brand);
  background: var(--brand-soft);
  color: var(--brand);
}

.bubble__retry {
  margin-top: 10px;
}

.skill-confirmation {
  width: min(560px, 100%);
  display: grid;
  gap: 12px;
}

.skill-confirmation__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.skill-confirmation__title {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
}

.skill-confirmation__desc {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 12.5px;
  line-height: 1.5;
}

.skill-confirmation__list {
  display: grid;
  gap: 8px;
}

.skill-confirmation__option {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-secondary);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s ease, background-color 0.15s ease;
}

.skill-confirmation__option:hover,
.skill-confirmation__option--selected {
  border-color: color-mix(in srgb, var(--brand) 52%, var(--border));
  background: var(--brand-soft);
}

.skill-confirmation__check {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin-top: 1px;
  border: 1px solid color-mix(in srgb, var(--brand) 36%, var(--border));
  border-radius: 4px;
  color: var(--brand);
  font-size: 12px;
  font-weight: 800;
  line-height: 1;
}

.skill-confirmation__body {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.skill-confirmation__name {
  min-width: 0;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-confirmation__name em {
  margin-left: 6px;
  color: var(--brand);
  font-size: 11px;
  font-style: normal;
  font-weight: 700;
}

.skill-confirmation__reason {
  min-width: 0;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-confirmation__actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

.assistant-process {
  margin-bottom: 8px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-muted);
}

.assistant-process__label {
  margin-bottom: 4px;
  color: var(--text-faint);
  font-size: 11px;
  font-weight: 700;
}

.assistant-process__content {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 13px;
  line-height: 1.5;
}

.assistant-process__content :deep(p) {
  margin: 3px 0;
}

.assistant-process__content :deep(ul),
.assistant-process__content :deep(ol) {
  margin: 4px 0 4px 16px;
}

.assistant-process__content :deep(li) {
  margin: 2px 0;
}

.assistant-loading-card {
  width: min(520px, 100%);
  display: grid;
  gap: 12px;
  padding: 2px 0 0;
}

.assistant-loading-card__badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.assistant-loading-card__badge {
  min-width: 0;
  max-width: 100%;
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.08);
  color: var(--brand-strong);
  font-size: 12px;
  font-weight: 600;
  line-height: 24px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.assistant-loading-card__header {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.assistant-loading-card__pulse {
  position: relative;
  flex: 0 0 auto;
  width: 10px;
  height: 10px;
  margin-top: 8px;
  border-radius: 999px;
  background: var(--brand);
  box-shadow: 0 0 0 4px rgba(37, 99, 235, 0.1);
}

.assistant-loading-card__pulse::after {
  content: "";
  position: absolute;
  inset: -5px;
  border-radius: inherit;
  border: 1px solid rgba(37, 99, 235, 0.32);
  animation: assistant-loading-pulse 1.8s ease-out infinite;
}

.assistant-loading-card__title {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
}

.assistant-loading-card__desc {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.55;
}

.assistant-loading-card__steps {
  display: grid;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--surface) 58%, var(--surface-muted));
}

.assistant-loading-card__step {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-faint);
  font-size: 12.5px;
  line-height: 1.35;
}

.assistant-loading-card__step span {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-faint);
  font-size: 10px;
  font-weight: 700;
}

.assistant-loading-card__step p {
  margin: 0;
}

.assistant-loading-card__step--done {
  color: var(--text-secondary);
}

.assistant-loading-card__step--done span {
  border-color: rgba(34, 197, 94, 0.32);
  background: rgba(34, 197, 94, 0.12);
  color: #16a34a;
}

.assistant-loading-card__step--active {
  color: var(--text-primary);
  font-weight: 600;
}

.assistant-loading-card__step--active span {
  border-color: rgba(37, 99, 235, 0.34);
  background: rgba(37, 99, 235, 0.12);
}

.assistant-loading-card__step--active span::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: inherit;
  background: var(--brand);
  animation: typing-bounce 1.2s infinite ease-in-out;
}

.assistant-loading-card__footer {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.4;
}

.assistant-loading-card__dots {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
}

.assistant-loading-card__dots i {
  display: block;
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: var(--brand);
  animation: typing-bounce 1.2s infinite ease-in-out;
}

.assistant-loading-card__dots i:nth-child(2) {
  animation-delay: 0.2s;
}

.assistant-loading-card__dots i:nth-child(3) {
  animation-delay: 0.4s;
}

.assistant-loading-card__skeleton {
  display: grid;
  gap: 8px;
  padding-top: 2px;
}

.assistant-loading-card__skeleton i {
  display: block;
  height: 9px;
  border-radius: 999px;
  background: linear-gradient(
    90deg,
    rgba(148, 163, 184, 0.16) 0%,
    rgba(148, 163, 184, 0.34) 42%,
    rgba(148, 163, 184, 0.16) 84%
  );
  background-size: 220% 100%;
  animation: assistant-loading-shimmer 1.7s ease-in-out infinite;
}

.assistant-loading-card__skeleton i:nth-child(1) {
  width: 92%;
}

.assistant-loading-card__skeleton i:nth-child(2) {
  width: 78%;
  animation-delay: 0.12s;
}

.assistant-loading-card__skeleton i:nth-child(3) {
  width: 54%;
  animation-delay: 0.24s;
}

.bubble__user-text {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.message-skill-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 7px;
}

.message-skill-badge {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  height: 24px;
  padding: 0 8px;
  border: 1px solid rgba(183, 110, 0, 0.12);
  border-radius: 999px;
  background: #fff4e5;
  color: #b76e00;
  font-size: 12px;
  font-weight: 600;
  line-height: 24px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bubble__meta {
  display: flex;
  justify-content: flex-start;
  margin-top: 8px;
  line-height: 1;
}

.bubble__meta :deep(.el-tag) {
  max-width: 100%;
  height: 20px;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
}

:global(:root[data-theme='dark']) .message-skill-badge {
  border-color: rgba(245, 158, 11, 0.18);
  background: rgba(245, 158, 11, 0.14);
  color: #f8c471;
}

:global(:root[data-theme='dark']) .assistant-loading-card__badge {
  background: rgba(59, 130, 246, 0.16);
  color: #93c5fd;
}

:global(:root[data-theme='dark']) .assistant-loading-card__steps {
  background: rgba(15, 23, 42, 0.42);
}

@keyframes assistant-loading-pulse {
  0% {
    opacity: 0.6;
    transform: scale(0.72);
  }
  100% {
    opacity: 0;
    transform: scale(1.8);
  }
}

@keyframes assistant-loading-shimmer {
  0% {
    background-position: 120% 0;
  }
  100% {
    background-position: -120% 0;
  }
}

@media (max-width: 768px) {
  .chat-welcome {
    padding: 40px 16px 32px;
  }

  .assistant-loading-card {
    width: 100%;
  }
}
</style>
