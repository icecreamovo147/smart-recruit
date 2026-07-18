<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import type { Session } from '@/types/ai'

const props = defineProps<{
  sessions: Session[]
  currentSession: Session | null
  menuSessionId: number
  sessionSidebarOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'select-session', session: Session): void
  (e: 'create-session'): void
  (e: 'rename-session', session: Session): void
  (e: 'remove-session', session: Session): void
  (e: 'batch-remove-sessions', sessionIds: number[]): void
  (e: 'menu-toggle', id: number): void
  (e: 'close-sidebar'): void
}>()

const selectMode = ref(false)
const selectedIds = ref<Set<number>>(new Set())

const toggleSelectMode = () => {
  selectMode.value = !selectMode.value
  if (!selectMode.value) {
    selectedIds.value = new Set()
  }
}

const toggleSession = (id: number) => {
  const next = new Set(selectedIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selectedIds.value = next
}

const allSessionsSelected = computed(() => {
  if (props.sessions.length === 0) return false
  return props.sessions.every((s) => selectedIds.value.has(s.id))
})

const selectAllSessions = () => {
  if (allSessionsSelected.value) {
    selectedIds.value = new Set()
  } else {
    selectedIds.value = new Set(props.sessions.map((s) => s.id))
  }
}

const cancelSelect = () => {
  selectMode.value = false
  selectedIds.value = new Set()
}

const batchRemove = async () => {
  const ids = [...selectedIds.value]
  if (ids.length === 0) return
  try {
    await ElMessageBox.confirm(
      `确认删除选中的 ${ids.length} 个会话？删除后不可恢复。`,
      '批量删除会话',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  emit('batch-remove-sessions', ids)
  selectMode.value = false
  selectedIds.value = new Set()
}

type TimeGroup = 'today' | 'yesterday' | 'older'

const GROUP_LABELS: Record<TimeGroup, string> = {
  today: '今天',
  yesterday: '昨天',
  older: '更早',
}

const getTimeGroup = (dateStr?: string): TimeGroup => {
  if (!dateStr) return 'older'
  const now = new Date()
  const then = new Date(dateStr)
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const yesterdayStart = todayStart - 86400000
  const thenTime = then.getTime()
  if (thenTime >= todayStart) return 'today'
  if (thenTime >= yesterdayStart) return 'yesterday'
  return 'older'
}

const groupedSessions = computed(() => {
  const groups: { key: TimeGroup; label: string; items: Session[] }[] = [
    { key: 'today', label: GROUP_LABELS.today, items: [] },
    { key: 'yesterday', label: GROUP_LABELS.yesterday, items: [] },
    { key: 'older', label: GROUP_LABELS.older, items: [] },
  ]
  for (const s of props.sessions) {
    const g = getTimeGroup(s.updated_at)
    const group = groups.find((x) => x.key === g)!
    group.items.push(s)
  }
  return groups.filter((g) => g.items.length > 0)
})

const formatSessionTime = (dateStr?: string): string => {
  if (!dateStr) return ''
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diff = Math.floor((now - then) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`
  return new Date(dateStr).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
</script>

<template>
  <aside class="chat-sidebar" :class="{ 'chat-sidebar--mobile-open': sessionSidebarOpen }">
    <div class="chat-sidebar__head">
      <h2 class="chat-sidebar__title">AI 会话</h2>
      <div class="chat-sidebar__head-actions">
        <el-button
          v-if="!selectMode"
          size="small"
          text
          class="chat-sidebar__manage-btn"
          @click="toggleSelectMode"
        >
          管理
        </el-button>
        <el-button size="small" type="primary" class="chat-sidebar__new-btn" @click="emit('create-session')">
          新建对话
        </el-button>
      </div>
    </div>
    <div class="session-list">
      <template v-if="sessions.length === 0">
        <el-empty description="暂无会话" :image-size="64" />
      </template>
      <template v-for="group in groupedSessions" :key="group.key">
        <div class="session-group__header">{{ group.label }}</div>
        <div
          v-for="session in group.items"
          :key="session.id"
          class="session-item"
          :class="{
            'session-item--active': !selectMode && currentSession?.id === session.id,
            'session-item--menu-open': menuSessionId === session.id,
            'session-item--selecting': selectMode,
            'session-item--checked': selectMode && selectedIds.has(session.id),
          }"
          @click="selectMode ? toggleSession(session.id) : emit('select-session', session)"
        >
          <div v-if="selectMode" class="session-item__checkbox" @click.stop="toggleSession(session.id)">
            <span class="session-item__checkmark" :class="{ 'session-item__checkmark--checked': selectedIds.has(session.id) }">
              <svg v-if="selectedIds.has(session.id)" width="12" height="12" viewBox="0 0 12 12" fill="none">
                <path d="M2.5 6l2.5 2.5 4.5-5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </span>
          </div>
          <div class="session-item__content">
            <div class="session-item__header">
              <span class="session-item__title">{{ session.title }}</span>
              <span class="session-item__time">{{ formatSessionTime(session.updated_at) }}</span>
            </div>
            <div class="session-item__meta">
              <span class="session-item__type-badge" :class="{ 'session-item__type-badge--analysis': session.application_id }">
                {{ session.application_id ? '简历分析' : '数据问答' }}
              </span>
            </div>
          </div>
          <div v-if="!selectMode" class="session-item__actions">
            <button
              class="session-item__more"
              @click.stop="emit('menu-toggle', menuSessionId === session.id ? 0 : session.id)"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                <circle cx="8" cy="3" r="1.5"/>
                <circle cx="8" cy="8" r="1.5"/>
                <circle cx="8" cy="13" r="1.5"/>
              </svg>
            </button>
            <div v-if="menuSessionId === session.id" class="session-item__menu" @click.stop>
              <button @click.stop="emit('menu-toggle', 0); emit('rename-session', session)">重命名</button>
              <button @click.stop="emit('menu-toggle', 0); emit('remove-session', session)">删除会话</button>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- Multi-select action bar -->
    <transition name="slide-up">
      <div v-if="selectMode" class="chat-sidebar__select-bar">
        <div class="chat-sidebar__select-bar-inner">
          <label class="chat-sidebar__select-all" @click="selectAllSessions">
            <span class="session-item__checkmark" :class="{ 'session-item__checkmark--checked': allSessionsSelected }">
              <svg v-if="allSessionsSelected" width="12" height="12" viewBox="0 0 12 12" fill="none">
                <path d="M2.5 6l2.5 2.5 4.5-5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </span>
            <span>全选</span>
          </label>
          <span class="chat-sidebar__select-count">已选 {{ selectedIds.size }} 项</span>
          <div class="chat-sidebar__select-actions">
            <el-button size="small" @click="cancelSelect">取消</el-button>
            <el-button size="small" type="danger" :disabled="selectedIds.size === 0" @click="batchRemove">
              删除
            </el-button>
          </div>
        </div>
      </div>
    </transition>
  </aside>
</template>

<style scoped>
.chat-sidebar__title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: 0.2px;
}

.session-group__header {
  padding: 10px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.chat-sidebar__new-btn {
  font-size: 12px;
  height: 30px;
  padding: 0 14px;
}

.session-item__content {
  flex: 1;
  min-width: 0;
}

.session-item__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.session-item__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  font-size: 13px;
  color: var(--text-primary);
}

.session-item__time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-faint);
  white-space: nowrap;
}

.session-item__meta {
  margin-top: 4px;
  display: flex;
  align-items: center;
}

.session-item__type-badge {
  display: inline-block;
  padding: 1px 8px;
  font-size: 11px;
  font-weight: 500;
  border-radius: 4px;
  background: var(--surface-muted);
  color: var(--text-muted);
  letter-spacing: 0.3px;
}

.session-item__type-badge--analysis {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
}

:root[data-theme='dark'] .session-item__type-badge--analysis {
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
}

.session-item__more svg {
  display: block;
}

/* ---- Multi-select mode ---- */
.chat-sidebar__head-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.chat-sidebar__manage-btn {
  font-size: 12px;
  color: var(--text-muted);
}

.chat-sidebar__manage-btn:hover {
  color: var(--brand);
}

.session-item--selecting {
  cursor: pointer;
}

.session-item--selecting.session-item--checked {
  background: var(--brand-soft);
}

.session-item__checkbox {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  padding: 0 4px 0 8px;
}

.session-item__checkmark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: 1.5px solid var(--border);
  border-radius: 4px;
  background: var(--surface);
  color: #fff;
  transition: all 0.15s ease;
  box-sizing: border-box;
}

.session-item__checkmark--checked {
  background: var(--brand);
  border-color: var(--brand);
}

.chat-sidebar__select-bar {
  flex-shrink: 0;
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.chat-sidebar__select-bar-inner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
}

.chat-sidebar__select-all {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}

.chat-sidebar__select-all:hover {
  color: var(--brand);
}

.chat-sidebar__select-count {
  font-size: 12px;
  color: var(--text-muted);
  white-space: nowrap;
}

.chat-sidebar__select-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

/* slide-up transition */
.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
