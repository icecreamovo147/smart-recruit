<script setup lang="ts">
import { ref } from 'vue'
import type { CandidateOption } from '@/types/ai'

interface MessageItem {
  role: string
  content: string
  pending?: boolean
  failed?: boolean
  waitingText?: string
  candidateOptions?: CandidateOption[]
}

defineProps<{
  messages: MessageItem[]
  loading: boolean
  streaming: boolean
  sessionLoading: boolean
  hasSession: boolean
  renderMarkdown: (content: string) => string
  waitingText: (message: MessageItem) => string
}>()

const emit = defineEmits<{
  (e: 'retry', index: number): void
  (e: 'analyze-candidate', option: CandidateOption): void
}>()

defineExpose({ scrollToBottom })

const listRef = ref<HTMLElement | null>(null)

function scrollToBottom() {
  if (listRef.value) {
    listRef.value.scrollTop = listRef.value.scrollHeight
  }
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
        <!-- Typing indicator -->
        <div v-if="message.role === 'assistant' && message.pending" class="typing-indicator">
          <span >{{ waitingText(message) }}</span>
          <span class="typing-indicator__dots">
            <i></i>
            <i></i>
            <i></i>
          </span>
        </div>

        <!-- Assistant markdown content -->
        <div
          v-else-if="message.role === 'assistant' && message.content"
          class="md-content"
          v-html="renderMarkdown(message.content)"
        ></div>

        <!-- User plain text -->
        <template v-else-if="message.role === 'user'">{{ message.content }}</template>

        <!-- Retry button -->
        <div v-if="message.role === 'assistant' && message.failed" class="bubble__retry">
          <el-button type="warning" size="small" @click="emit('retry', index)">重新发送</el-button>
        </div>

        <!-- Candidate options -->
        <div
          v-if="message.role === 'assistant' && message.candidateOptions?.length"
          class="candidate-options"
        >
          <button
            v-for="option in message.candidateOptions"
            :key="option.application_id"
            class="candidate-option"
            @click="emit('analyze-candidate', option)"
          >
            <span class="candidate-option__name">{{ option.candidate_name }}</span>
            <span>{{ option.job_title }}</span>
            <span>{{ option.masked_phone }}</span>
            <span>第 {{ option.round_no }} 轮</span>
            <strong>分析</strong>
          </button>
        </div>
      </div>
    </div>

    <!-- Loading placeholder -->
    <div v-if="loading && !streaming" class="message-wrapper assistant">
      <div class="bubble bubble--assistant">
        <div class="typing-indicator">
          <span >分析中</span>
          <span class="typing-indicator__dots">
            <i></i><i></i><i></i>
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

.candidate-options {
  display: grid;
  gap: 8px;
  margin-top: 12px;
}

.candidate-option {
  display: grid;
  grid-template-columns: minmax(72px, 1fr) minmax(120px, 1.4fr) minmax(96px, 1fr) auto auto;
  gap: 10px;
  align-items: center;
  width: 100%;
  padding: 9px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 13px;
  text-align: left;
  transition: transform 0.12s ease, border-color 0.12s ease, background-color 0.12s ease;
}

.candidate-option:hover {
  transform: translateY(-1px);
  border-color: rgba(37, 99, 235, 0.55);
  background: var(--brand-soft);
}

.candidate-option__name {
  font-weight: 700;
  color: var(--text-primary);
}

.candidate-option strong {
  color: var(--brand);
}

@media (max-width: 768px) {
  .candidate-option {
    grid-template-columns: 1fr;
    gap: 4px;
    font-size: 12px;
  }

  .chat-welcome {
    padding: 40px 16px 32px;
  }
}
</style>
