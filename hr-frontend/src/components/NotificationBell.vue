<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { Bell } from '@element-plus/icons-vue'
import { ElNotification } from 'element-plus'
import {
  listNotifications,
  getNotificationSummary,
  openNotificationStream,
  markNotificationRead,
  markAllNotificationsRead,
} from '@/api/notification'
import type { NotificationItem, NotificationStreamEvent } from '@/types/notification'

const router = useRouter()
const unreadCount = ref(0)
const list = ref<NotificationItem[]>([])
const listLoaded = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const popoverRef = ref<any>(null)
let pollTimer: ReturnType<typeof setTimeout> | null = null
let audioCtx: AudioContext | null = null
const seenNotificationIds = new Set<number>()
const MAX_SEEN_IDS = 500
const pruneSeenIds = () => {
  if (seenNotificationIds.size > MAX_SEEN_IDS) {
    const ids = [...seenNotificationIds]
    seenNotificationIds.clear()
    ids.slice(-200).forEach((id) => seenNotificationIds.add(id))
  }
}
let notificationBaselineReady = false
const notificationStaggerMs = 350
const notificationTimers = new Set<ReturnType<typeof setTimeout>>()
let polling = false
let latestNotificationId = 0
let streamAbort: AbortController | null = null
let streamReconnectTimer: ReturnType<typeof setTimeout> | null = null
let streamConnected = false

const scheduleNextPoll = () => {
  const baseMs = streamConnected ? 60000 : document.visibilityState === 'hidden' ? 60000 : 15000
  const jitter = baseMs + Math.random() * 5000
  pollTimer = setTimeout(async () => {
    await pollNotifications()
    scheduleNextPoll()
  }, jitter)
}

const fetchList = async () => {
  try {
    const data = await listNotifications({ page: 1, page_size: 20 })
    list.value = data.list || []
    listLoaded.value = true
  } catch {
    if (!listLoaded.value) list.value = []
  }
}

const pollNotifications = async () => {
  if (polling) return
  polling = true
  try {
    const summary = await getNotificationSummary()
    const nextLatestId = summary.latest_notification_id || 0
    const wasBaselineReady = notificationBaselineReady
    const shouldFetchRecent = nextLatestId > 0 && (!notificationBaselineReady || nextLatestId !== latestNotificationId)
    unreadCount.value = summary.unread || 0
    latestNotificationId = nextLatestId
    notificationBaselineReady = true
    if (shouldFetchRecent) {
      await fetchRecentNotifications(wasBaselineReady)
    }
  } catch {
    // silently ignore
  } finally {
    polling = false
  }
}

const fetchRecentNotifications = async (notifyNew: boolean) => {
  const listData = await listNotifications({ page: 1, page_size: 5 })
  const nextList = listData.list || []
  list.value = nextList.length > 0 ? nextList : list.value
  listLoaded.value = true
  const freshUnread = notifyNew
    ? nextList
        .filter((item) => !item.is_read && !seenNotificationIds.has(item.notification_id))
        .sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
    : []
  // 避免与 SSE 路径竞态导致重复弹窗：SSE 在线时弹窗由 SSE 负责
  if (!streamConnected && freshUnread.length > 0) {
    notifyFreshUnread(freshUnread)
  }
  // 在弹窗逻辑之后才标记已见，防止 SSE 推送晚于 poll 拉取时遗漏
  freshUnread.forEach((item) => seenNotificationIds.add(item.notification_id))
  nextList.forEach((item) => seenNotificationIds.add(item.notification_id))
  pruneSeenIds()
}

const handlePopoverShow = () => {
  fetchList()
}

const handleClick = async (item: NotificationItem) => {
  try {
    await markNotificationRead(item.notification_id)
    item.is_read = true
    if (unreadCount.value > 0) unreadCount.value--
  } catch {
    // silently ignore
  }
  popoverRef.value?.hide()
  if (item.link) {
    router.push(item.link)
  }
}

const openNotification = async (item: NotificationItem) => {
  await handleClick(item)
}

const showDesktopNotification = (item: NotificationItem) => {
  ElNotification({
    title: item.title || '新的通知',
    message: item.content,
    position: 'top-right',
    duration: 6000,
    type: 'info',
    onClick: () => {
      openNotification(item)
    },
  })
}

const notifyFreshUnread = (items: NotificationItem[]) => {
  items.forEach((item, index) => {
    const timer = window.setTimeout(() => {
      notificationTimers.delete(timer)
      void playSound()
      showDesktopNotification(item)
    }, index * notificationStaggerMs)
    notificationTimers.add(timer)
  })
}

const connectNotificationStream = () => {
  streamAbort?.abort()
  const controller = new AbortController()
  streamAbort = controller
  readNotificationStream(controller.signal)
    .catch(() => {})
    .finally(() => {
      streamConnected = false
      if (!controller.signal.aborted && streamAbort === controller) {
        streamReconnectTimer = window.setTimeout(connectNotificationStream, 5000)
      }
    })
}

const readNotificationStream = async (signal: AbortSignal) => {
  const response = await openNotificationStream(signal)
  if (!response.ok || !response.body) throw new Error('notification stream unavailable')
  streamConnected = true
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (!signal.aborted) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const blocks = buffer.split(/\r?\n\r?\n/)
    buffer = blocks.pop() || ''
    blocks.forEach(handleStreamBlock)
  }
}

const handleStreamBlock = (block: string) => {
  const data = block
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trim())
    .join('\n')
  if (!data) return
  try {
    handleNotificationEvent(JSON.parse(data) as NotificationStreamEvent)
  } catch {
    // silently ignore malformed stream payloads
  }
}

const handleNotificationEvent = (event: NotificationStreamEvent) => {
  if (event.type !== 'notification_created') return
  unreadCount.value = event.unread ?? unreadCount.value + 1
  latestNotificationId = Math.max(latestNotificationId, event.notification_id)
  if (!seenNotificationIds.has(event.notification_id)) {
    seenNotificationIds.add(event.notification_id)
    pruneSeenIds()
    // Build a temporary item for immediate desktop notification.
    // The correct notification_type will be supplied via REST fetch below.
    const item: NotificationItem = {
      notification_id: event.notification_id,
      type: event.notification_type || '',
      title: event.title,
      content: event.content,
      link: event.link,
      is_read: false,
      created_at: event.created_at,
    }
    list.value = [item, ...list.value.filter((n) => n.notification_id !== item.notification_id)].slice(0, 20)
    listLoaded.value = true
    notifyFreshUnread([item])
    // Replace with REST versions that carry fully correct notification types.
    fetchRecentNotifications(true).catch(() => {})
  }
}

const handleMarkAllRead = async () => {
  try {
    await markAllNotificationsRead()
    list.value.forEach((n) => (n.is_read = true))
    unreadCount.value = 0
  } catch {
    // silently ignore
  }
}

const formatTime = (ts: string) => {
  if (!ts) return ''
  const d = new Date(ts)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const unlockSound = () => {
  try {
    if (!audioCtx) audioCtx = new AudioContext()
    if (audioCtx.state === 'suspended') {
      audioCtx.resume().catch(() => {})
    }
  } catch {
    // browser may block auto-play silently
  }
}

const playSound = async () => {
  try {
    if (!audioCtx) audioCtx = new AudioContext()
    if (audioCtx.state === 'suspended') {
      await audioCtx.resume()
    }
    const osc = audioCtx.createOscillator()
    const gain = audioCtx.createGain()
    osc.connect(gain)
    gain.connect(audioCtx.destination)
    osc.type = 'sine'
    osc.frequency.setValueAtTime(880, audioCtx.currentTime)
    osc.frequency.setValueAtTime(1100, audioCtx.currentTime + 0.08)
    gain.gain.setValueAtTime(0.18, audioCtx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, audioCtx.currentTime + 0.24)
    osc.start(audioCtx.currentTime)
    osc.stop(audioCtx.currentTime + 0.24)
  } catch {
    // browser may block auto-play silently
  }
}

onMounted(() => {
  pollNotifications()
  scheduleNextPoll()
  connectNotificationStream()
  document.addEventListener('pointerdown', unlockSound, { once: true })
  document.addEventListener('keydown', unlockSound, { once: true })
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  if (pollTimer) clearTimeout(pollTimer)
  if (streamReconnectTimer) clearTimeout(streamReconnectTimer)
  streamAbort?.abort()
  notificationTimers.forEach((timer) => clearTimeout(timer))
  notificationTimers.clear()
  if (audioCtx) {
    audioCtx.close().catch(() => {})
    audioCtx = null
  }
  document.removeEventListener('pointerdown', unlockSound)
  document.removeEventListener('keydown', unlockSound)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

const onVisibilityChange = () => {
  if (document.visibilityState === 'visible') {
    pollNotifications()
    if (pollTimer) clearTimeout(pollTimer)
    scheduleNextPoll()
  }
}
</script>

<template>
  <el-popover
    ref="popoverRef"
    placement="bottom-end"
    :width="380"
    trigger="click"
    :persistent="true"
    @show="handlePopoverShow"
  >
    <template #reference>
      <button class="notif-bell">
        <el-badge :value="unreadCount" :hidden="unreadCount === 0" :max="99">
          <el-icon :size="20"><Bell /></el-icon>
        </el-badge>
      </button>
    </template>

    <div class="notif-panel">
      <div class="notif-panel__head">
        <span>通知</span>
        <el-button v-if="unreadCount > 0" size="small" text type="primary" @click="handleMarkAllRead">
          全部已读
        </el-button>
      </div>

      <el-scrollbar v-if="list.length > 0" max-height="360px">
        <div
          v-for="item in list"
          :key="item.notification_id"
          class="notif-item"
          :class="{ 'notif-item--unread': !item.is_read }"
          @click="handleClick(item)"
        >
          <div class="notif-item__title">
            <span v-if="!item.is_read" class="notif-dot"></span>
            {{ item.title }}
          </div>
          <div class="notif-item__content">{{ item.content }}</div>
          <div class="notif-item__time">{{ formatTime(item.created_at) }}</div>
        </div>
      </el-scrollbar>

      <div v-else-if="!listLoaded" class="notif-placeholder"></div>
      <el-empty v-else description="暂无通知" :image-size="60" />
    </div>
  </el-popover>
</template>

<style scoped>
.notif-bell {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 4px 8px;
  display: flex;
  align-items: center;
  border-radius: 8px;
  transition: background 0.15s;
}
.notif-bell:hover {
  background: var(--surface-muted);
  color: var(--text-primary);
}

.notif-panel {
  min-height: 220px;
}

.notif-panel__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 0 8px;
  font-weight: 600;
  font-size: 14px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 4px;
}

.notif-item {
  padding: 10px 8px;
  border-radius: 8px;
  cursor: pointer;
  transition:
    background var(--motion-fast, 120ms) var(--motion-ease, ease-out),
    transform var(--motion-fast, 120ms) var(--motion-ease, ease-out);
  border-bottom: 1px solid var(--surface-muted);
}
.notif-item:last-child {
  border-bottom: none;
}
.notif-item:hover {
  background: var(--surface-muted);
  transform: translateX(2px);
}
.notif-item--unread {
  background: var(--brand-soft);
}
.notif-item__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 6px;
}
.notif-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--brand);
  flex-shrink: 0;
}
.notif-item__content {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
  line-height: 1.4;
}
.notif-item__time {
  font-size: 11px;
  color: var(--text-faint);
  margin-top: 4px;
}

.notif-placeholder {
  min-height: 160px;
}
</style>
