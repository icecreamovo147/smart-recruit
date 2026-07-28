<script setup lang="ts">
import { ref } from 'vue'
import type { AgentSkillSelectionPayload, CandidateOption, ChatMessageSkill } from '@/types/ai'
import type { AgentSkillRuntimeEvidence } from '@shared/types/agentRun'

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
  agent_skill_runtime_evidence?: AgentSkillRuntimeEvidence[]
}

const props = defineProps<{
  messages: MessageItem[]
  loading: boolean
  streaming: boolean
  sessionLoading: boolean
  hasSession: boolean
  renderMarkdown: (content: string) => string
  waitingText: (message: MessageItem) => string
  interactionDisabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'retry', index: number): void
  (e: 'confirm-skill-selection', index: number, skillIds: number[]): void
  (e: 'reject-skill-selection', index: number): void
  (e: 'cancel-pending-run', index: number): void
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

const skillBadgeText = (skill: ChatMessageSkill): string => {
  const label = skill.command || (skill.name ? `/${skill.name}` : '')
  return skill.version ? `${label} v${skill.version}` : label
}

const messageSkillBadges = (message: MessageItem): string[] =>
  messageSkills(message)
    .map(skillBadgeText)
    .filter(Boolean)

const selectedAgentSkillVersionIds = ref<Record<number, number[]>>({})

const skillSelectionLabel = (candidate: AgentSkillSelectionPayload['candidates'][number]): string =>
  candidate.display_name || candidate.name || `Skill #${candidate.skill_id}`

const skillSelectionReason = (candidate: AgentSkillSelectionPayload['candidates'][number]): string => {
  if (candidate.reason) return candidate.reason
  if (candidate.category || candidate.scenario) return [candidate.category, candidate.scenario].filter(Boolean).join(' · ')
  return '系统推荐候选 Skill'
}

const isMCPConfirmation = (selection: AgentSkillSelectionPayload): boolean =>
  selection.confirmation_kind === 'mcp_tool'

const confirmationTitle = (selection: AgentSkillSelectionPayload): string => {
  if (isMCPConfirmation(selection)) return '确认执行 MCP 工具'
  return selection.candidates.length ? '确认启用 Agent Skill' : 'Agent Skill 确认信息异常'
}

const confirmationDescription = (selection: AgentSkillSelectionPayload): string => {
  if (isMCPConfirmation(selection)) {
    return selection.reason && selection.reason !== 'confirmation_required'
      ? selection.reason
      : '该工具需要你的明确授权。拒绝后本次运行将安全取消。'
  }
  return selection.candidates.length
    ? '以下确认仅授权列出的精确 Skill 版本，不会授权 MCP 工具。'
    : '未能读取精确 Skill 版本，请取消本次运行后重试。'
}

const defaultSelectionIds = (selection: AgentSkillSelectionPayload): number[] =>
  selection.recommended_agent_skill_version_ids?.length
    ? selection.recommended_agent_skill_version_ids
    : selection.candidates
      .filter((candidate) => candidate.recommended)
      .map((candidate) => candidate.version_id)

const selectionIds = (index: number, selection: AgentSkillSelectionPayload): number[] =>
  selectedAgentSkillVersionIds.value[index] ?? defaultSelectionIds(selection)

const toggleSkillSelection = (index: number, selection: AgentSkillSelectionPayload, id: number) => {
  if (props.interactionDisabled) return
  const current = selectionIds(index, selection)
  const next = current.includes(id) ? current.filter((item) => item !== id) : [...current, id]
  selectedAgentSkillVersionIds.value = { ...selectedAgentSkillVersionIds.value, [index]: next }
}

const confirmSkillSelection = (index: number, selection: AgentSkillSelectionPayload) => {
  if (props.interactionDisabled) return
  emit('confirm-skill-selection', index, selectionIds(index, selection))
}

const rejectSkillSelection = (index: number) => {
  if (props.interactionDisabled) return
  emit('reject-skill-selection', index)
}

const cancelPendingRun = (index: number) => {
  if (props.interactionDisabled) return
  emit('cancel-pending-run', index)
}

const shortHash = (hash: string | undefined): string => {
  const normalized = String(hash || '').trim()
  return normalized ? normalized.slice(0, 10) : '—'
}

const candidateMeta = (
  candidate: AgentSkillSelectionPayload['candidates'][number],
): string => [
  `v${candidate.version || '—'}`,
  shortHash(candidate.compiled_hash),
  candidate.composition_role,
  candidate.risk,
  candidate.activation_policy,
  `${Math.max(0, Number(candidate.core_estimated_tokens) || 0)} tokens`,
].join(' · ')

const runtimeEvidence = (message: MessageItem): AgentSkillRuntimeEvidence[] =>
  Array.isArray(message.agent_skill_runtime_evidence)
    ? message.agent_skill_runtime_evidence.filter(
        (item): item is AgentSkillRuntimeEvidence =>
          Boolean(item && Number.isFinite(Number(item.version_id)) && Number(item.version_id) > 0),
      )
    : []

const evidenceSections = (evidence: AgentSkillRuntimeEvidence) =>
  Array.isArray(evidence?.sections)
    ? evidence.sections.filter((section) => Boolean(section?.section_key))
    : []

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
          :disabled="interactionDisabled"
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
          'bubble--pending': message.role === 'assistant' && message.pending && !message.content,
          'bubble--failed': message.failed,
        }"
      >
        <template v-if="message.role === 'assistant'">
          <div v-if="message.agentSkillSelection" class="skill-confirmation">
            <div class="skill-confirmation__header">
              <div>
                <div class="skill-confirmation__title">
                  {{ confirmationTitle(message.agentSkillSelection) }}
                </div>
                <div class="skill-confirmation__desc">
                  {{ confirmationDescription(message.agentSkillSelection) }}
                </div>
              </div>
              <el-tag size="small" effect="plain">待确认</el-tag>
            </div>
            <div class="skill-confirmation__list">
              <button
                v-for="candidate in message.agentSkillSelection.candidates"
                :key="candidate.version_id"
                class="skill-confirmation__option"
                :class="{ 'skill-confirmation__option--selected': selectionIds(index, message.agentSkillSelection).includes(candidate.version_id) }"
                type="button"
                :disabled="interactionDisabled"
                @click="toggleSkillSelection(index, message.agentSkillSelection, candidate.version_id)"
              >
                <span class="skill-confirmation__check">
                  {{ selectionIds(index, message.agentSkillSelection).includes(candidate.version_id) ? '✓' : '' }}
                </span>
                <span class="skill-confirmation__body">
                  <span class="skill-confirmation__name">
                    {{ skillSelectionLabel(candidate) }}
                    <em v-if="candidate.recommended">推荐</em>
                  </span>
                  <span class="skill-confirmation__meta">{{ candidateMeta(candidate) }}</span>
                  <span class="skill-confirmation__reason">{{ skillSelectionReason(candidate) }}</span>
                </span>
              </button>
            </div>
            <div class="skill-confirmation__actions">
              <el-button
                v-if="isMCPConfirmation(message.agentSkillSelection)"
                size="small"
                :disabled="interactionDisabled"
                @click="rejectSkillSelection(index)"
              >
                拒绝并取消
              </el-button>
              <el-button
                v-else
                size="small"
                :disabled="interactionDisabled"
                @click="rejectSkillSelection(index)"
              >
                拒绝 Skill
              </el-button>
              <el-button
                v-if="!isMCPConfirmation(message.agentSkillSelection)"
                size="small"
                :disabled="interactionDisabled"
                @click="cancelPendingRun(index)"
              >
                取消运行
              </el-button>
              <el-button
                type="primary"
                size="small"
                :disabled="interactionDisabled || (!isMCPConfirmation(message.agentSkillSelection) && selectionIds(index, message.agentSkillSelection).length === 0)"
                @click="confirmSkillSelection(index, message.agentSkillSelection)"
              >
                {{ message.agentSkillSelection.confirmation_kind === 'mcp_tool' ? '确认执行' : '确认启用' }}
              </el-button>
            </div>
          </div>

          <div
            v-if="runtimeEvidence(message).length > 0"
            class="agent-skill-evidence"
            aria-label="Agent Skill 运行证据"
          >
            <div class="agent-skill-evidence__title">已应用 Agent Skill</div>
            <div
              v-for="evidence in runtimeEvidence(message)"
              :key="evidence.version_id"
              class="agent-skill-evidence__item"
            >
              <div class="agent-skill-evidence__header">
                <strong>{{ evidence.display_name || evidence.skill_name }}</strong>
                <span>v{{ evidence.version }} · {{ shortHash(evidence.compiled_hash) }}</span>
              </div>
              <div class="agent-skill-evidence__meta">
                {{ evidence.composition_role }} · {{ evidence.risk }} ·
                {{ evidence.activation_policy }} · {{ evidence.loaded_tokens }} tokens
              </div>
              <div class="agent-skill-evidence__reason">{{ evidence.decision_reason }}</div>
              <div v-if="evidenceSections(evidence).length > 0" class="agent-skill-evidence__sections">
                <span
                  v-for="section in evidenceSections(evidence)"
                  :key="section.section_id"
                  :class="{ 'agent-skill-evidence__section--dropped': !section.included }"
                >
                  {{ section.section_key }} · {{ section.estimated_tokens }} tokens · {{ section.decision_reason }}
                </span>
              </div>
            </div>
          </div>

          <div v-if="message.processContent && !message.pending" class="assistant-process">
            <div class="assistant-process__label">执行过程</div>
            <div class="assistant-process__content md-content" v-html="renderMarkdown(message.processContent)"></div>
          </div>

          <!-- Typing indicator -->
          <div
            v-if="message.pending && !message.content"
            class="assistant-typing"
            role="status"
            aria-live="polite"
          >
            <span class="assistant-typing__text">{{ waitingText(message) || '思考中' }}</span>
            <span class="assistant-typing__dots" aria-hidden="true">
              <span></span>
              <span></span>
              <span></span>
            </span>
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
          <el-button type="warning" size="small" :disabled="interactionDisabled" @click="interactionDisabled ? undefined : emit('retry', index)">重新发送</el-button>
        </div>

      </div>
    </div>

    <!-- Loading placeholder -->
    <div v-if="loading && !streaming" class="message-wrapper assistant">
      <div class="bubble bubble--assistant bubble--pending">
        <div class="assistant-typing" role="status" aria-live="polite">
          <span class="assistant-typing__text">思考中</span>
          <span class="assistant-typing__dots" aria-hidden="true">
            <span></span>
            <span></span>
            <span></span>
          </span>
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

.skill-confirmation__meta {
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.skill-confirmation__actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

.agent-skill-evidence {
  width: min(560px, 100%);
  display: grid;
  gap: 8px;
  margin-bottom: 10px;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
}

.agent-skill-evidence__title {
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 700;
}

.agent-skill-evidence__item {
  display: grid;
  gap: 3px;
  padding-top: 7px;
  border-top: 1px solid var(--border);
  color: var(--text-secondary);
  font-size: 11px;
}

.agent-skill-evidence__header {
  display: flex;
  justify-content: space-between;
  gap: 10px;
}

.agent-skill-evidence__header span,
.agent-skill-evidence__meta {
  color: var(--text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.agent-skill-evidence__reason {
  overflow-wrap: anywhere;
}

.agent-skill-evidence__sections {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.agent-skill-evidence__sections span {
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--brand-soft);
}

.agent-skill-evidence__section--dropped {
  opacity: 0.62;
  text-decoration: line-through;
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

.bubble--pending {
  padding: 10px 12px;
  border-radius: 8px;
}

.assistant-typing {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
}

.assistant-typing__text {
  white-space: nowrap;
}

.assistant-typing__dots {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 12px;
}

.assistant-typing__dots span {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--brand-strong);
  opacity: 0.35;
  animation: assistant-typing-dot 1.2s infinite ease-in-out;
}

.assistant-typing__dots span:nth-child(2) {
  animation-delay: 160ms;
}

.assistant-typing__dots span:nth-child(3) {
  animation-delay: 320ms;
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

@keyframes assistant-typing-dot {
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

@media (max-width: 768px) {
  .chat-welcome {
    padding: 40px 16px 32px;
  }
}
</style>
