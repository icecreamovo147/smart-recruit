<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Close, MagicStick, Plus, Promotion } from '@element-plus/icons-vue'
import { listSessions } from '@/api/ai'
import type { CandidateSession } from '@/types/ai'

const router = useRouter()
const route = useRoute()

const open = ref(false)
const loading = ref(false)
const latestSession = ref<CandidateSession | null>(null)

const currentSourceQuery = computed<Record<string, string>>(() => {
  if (route.name === undefined && route.path.startsWith('/jobs/')) {
    const jobId = Number(route.params.jobId)
    if (jobId > 0) return { source: 'job', jobId: String(jobId) }
  }
  return {} as Record<string, string>
})

const openMini = async () => {
  open.value = !open.value
  if (!open.value || latestSession.value) return
  loading.value = true
  try {
    const data = await listSessions({ page: 1, page_size: 1 })
    latestSession.value = data.list?.[0] || null
  } finally {
    loading.value = false
  }
}

const goAssistant = (query: Record<string, string | number> = {}) => {
  open.value = false
  router.push({ path: '/ai-assistant', query })
}

const continueLatest = () => {
  if (latestSession.value?.session_id) {
    goAssistant({ sessionId: latestSession.value.session_id })
    return
  }
  goAssistant({ intent: 'new' })
}

const startFromCurrentPage = () => {
  goAssistant(Object.keys(currentSourceQuery.value).length > 0 ? currentSourceQuery.value : { intent: 'new' })
}
</script>

<template>
  <div class="candidate-ai-mini" :class="{ 'candidate-ai-mini--open': open }">
    <Transition name="ai-mini-pop">
      <section v-if="open" class="ai-mini-card" aria-label="AI求职助手入口">
        <header class="ai-mini-card__head">
          <div>
            <strong>AI求职助手</strong>
            <span>让助手继续帮你推进求职任务</span>
          </div>
          <button class="ai-mini-icon-btn" aria-label="关闭 AI 助手入口" @click="open = false">
            <el-icon><Close /></el-icon>
          </button>
        </header>

        <button class="ai-mini-action" :disabled="loading" @click="continueLatest">
          <el-icon><Promotion /></el-icon>
          <span>{{ latestSession ? '继续上次对话' : '开始新对话' }}</span>
        </button>
        <button class="ai-mini-action" @click="startFromCurrentPage">
          <el-icon><MagicStick /></el-icon>
          <span>{{ Object.keys(currentSourceQuery).length > 0 ? '基于当前岗位提问' : '新建求职咨询' }}</span>
        </button>
        <button class="ai-mini-action ai-mini-action--primary" @click="goAssistant()">
          <el-icon><Plus /></el-icon>
          <span>打开完整助手</span>
        </button>
      </section>
    </Transition>

    <button
      class="ai-mini-avatar"
      :class="{ 'ai-mini-avatar--active': open }"
      :aria-label="open ? '关闭 AI 助手入口' : '打开 AI 助手入口'"
      @click="openMini"
    >
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
      <span class="ai-mini-avatar__pulse"></span>
    </button>
  </div>
</template>

<style scoped>
.candidate-ai-mini {
  position: fixed;
  right: max(24px, env(safe-area-inset-right));
  bottom: max(24px, env(safe-area-inset-bottom));
  z-index: 1200;
}

.ai-mini-avatar {
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

.ai-mini-avatar:hover {
  transform: scale(1.08);
  box-shadow: 0 8px 32px rgba(37, 99, 235, 0.5);
}

.ai-mini-avatar--active {
  transform: scale(0.92);
  box-shadow: 0 4px 16px rgba(37, 99, 235, 0.3);
}

.ai-mini-avatar__pulse {
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  border: 2px solid rgba(37, 99, 235, 0.3);
  animation: ai-mini-pulse 2.5s ease-in-out infinite;
  pointer-events: none;
}

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

.ai-robot__eye {
  width: 5px;
  height: 5px;
  background: #1d4ed8;
  border-radius: 50%;
  animation: ai-blink 3s ease-in-out infinite;
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

.ai-mini-card {
  position: absolute;
  right: 0;
  bottom: 72px;
  width: min(320px, calc(100vw - 32px));
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: 0 18px 42px rgba(15, 23, 42, 0.18);
  padding: 14px;
}

.ai-mini-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.ai-mini-card__head strong {
  display: block;
  color: var(--text-primary);
  font-size: 16px;
}

.ai-mini-card__head span {
  display: block;
  color: var(--text-muted);
  font-size: 12px;
  margin-top: 4px;
}

.ai-mini-icon-btn {
  width: 30px;
  height: 30px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  display: grid;
  place-items: center;
  cursor: pointer;
}

.ai-mini-action {
  width: 100%;
  min-height: 42px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 12px;
  margin-top: 8px;
  cursor: pointer;
  font-weight: 600;
}

.ai-mini-action:hover {
  border-color: rgba(37, 99, 235, 0.35);
  color: var(--brand-strong);
}

.ai-mini-action--primary {
  background: var(--brand);
  border-color: var(--brand);
  color: #fff;
}

.ai-mini-action--primary:hover {
  color: #fff;
  filter: brightness(0.98);
}

.ai-mini-pop-enter-active,
.ai-mini-pop-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.ai-mini-pop-enter-from,
.ai-mini-pop-leave-to {
  opacity: 0;
  transform: translateY(8px) scale(0.98);
}

@keyframes ai-mini-pulse {
  0%, 100% { transform: scale(1); opacity: 0.4; }
  50% { transform: scale(1.12); opacity: 0; }
}

@keyframes ai-blink {
  0%, 95%, 100% { transform: scaleY(1); }
  97% { transform: scaleY(0.1); }
}

@keyframes ai-core-glow {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.8); }
}

@media (max-width: 768px) {
  .candidate-ai-mini {
    right: max(16px, env(safe-area-inset-right));
    bottom: max(16px, env(safe-area-inset-bottom));
  }
}
</style>
