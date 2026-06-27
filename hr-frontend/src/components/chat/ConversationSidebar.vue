<script setup lang="ts">
import { computed } from 'vue'
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
  (e: 'menu-toggle', id: number): void
  (e: 'close-sidebar'): void
}>()

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
      <el-button size="small" type="primary" class="chat-sidebar__new-btn" @click="emit('create-session')">
        新建对话
      </el-button>
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
            'session-item--active': currentSession?.id === session.id,
            'session-item--menu-open': menuSessionId === session.id,
          }"
          @click="emit('select-session', session)"
        >
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
          <div class="session-item__actions">
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
</style>
