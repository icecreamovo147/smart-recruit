<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Check } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listNotifications, markNotificationRead, markAllNotificationsRead } from '@/api/notification'
import type { NotificationItem } from '@/types/domain'

const router = useRouter()
const loading = ref(false)
const notifications = ref<NotificationItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20

const loadNotifications = async () => {
  loading.value = true
  try {
    const data = await listNotifications({ page: page.value, page_size: pageSize })
    notifications.value = data.list || []
    total.value = data.total || 0
  } catch {
    // Silent
  } finally {
    loading.value = false
  }
}

const navigateByLink = (link: string) => {
  // Backend may prefix links with /hr (shared with hr-frontend); strip it.
  let path = link.replace(/^\/hr/, '') || '/'
  // Interview notifications: route to detail page when biz context is available
  // Fallback: if the path doesn't match any known route, go to /interviews
  const knownPrefixes = ['/interviews', '/dashboard', '/notifications', '/profile']
  if (!knownPrefixes.some((p) => path === p || path.startsWith(p + '/'))) {
    path = '/interviews'
  }
  router.push(path)
}

const handleMarkRead = async (item: NotificationItem) => {
  if (item.is_read) {
    if (item.link) navigateByLink(item.link)
    return
  }
  try {
    await markNotificationRead(item.notification_id)
    item.is_read = 1
    if (item.link) navigateByLink(item.link)
  } catch {
    // Silent
  }
}

const handleMarkAllRead = async () => {
  try {
    await markAllNotificationsRead()
    notifications.value.forEach((n) => (n.is_read = 1))
    ElMessage.success('已全部标记为已读')
  } catch {
    // Silent
  }
}

const formatTime = (iso: string): string => {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return '刚刚'
  if (diffMin < 60) return `${diffMin} 分钟前`
  const diffHour = Math.floor(diffMin / 60)
  if (diffHour < 24) return `${diffHour} 小时前`
  const diffDay = Math.floor(diffHour / 24)
  if (diffDay < 7) return `${diffDay} 天前`
  return `${d.getMonth() + 1}/${d.getDate()}`
}

const handlePageChange = (p: number) => {
  page.value = p
  loadNotifications()
}

onMounted(loadNotifications)
</script>

<template>
  <div class="notification-view">
    <div class="notification-toolbar">
      <el-button type="primary" link @click="handleMarkAllRead">
        <el-icon><Check /></el-icon>
        全部标记已读
      </el-button>
    </div>

    <div v-loading="loading" class="notification-list">
      <div
        v-for="item in notifications"
        :key="item.notification_id"
        class="notification-item"
        :class="{ unread: !item.is_read }"
        @click="handleMarkRead(item)"
      >
        <div class="notification-dot" v-if="!item.is_read" />
        <div class="notification-content">
          <h4 class="notification-title">{{ item.title }}</h4>
          <p class="notification-body">{{ item.body }}</p>
          <span class="notification-time">{{ formatTime(item.created_at) }}</span>
        </div>
      </div>

      <el-empty v-if="!loading && notifications.length === 0" description="暂无通知" />
    </div>

    <div v-if="total > pageSize" class="notification-pagination">
      <el-pagination
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<style scoped>
.notification-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.notification-list {
  display: flex;
  flex-direction: column;
  min-height: 200px;
}

.notification-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: background-color 120ms ease;
}

.notification-item:hover {
  background: var(--surface-secondary);
}

.notification-item.unread {
  background: var(--surface-active);
}

.notification-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--brand-primary);
  flex-shrink: 0;
  margin-top: 6px;
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.notification-body {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.notification-time {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
  display: inline-block;
}

.notification-pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>
