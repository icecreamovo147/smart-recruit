<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Bell } from '@element-plus/icons-vue'
import request from '@/api/request'

const router = useRouter()
const unreadCount = ref(0)
let pollTimer: ReturnType<typeof setInterval> | null = null

const fetchUnreadCount = async () => {
  try {
    const data = await request.get<{ count: number }>('/api/v1/hr/notifications/unread-count')
    unreadCount.value = data.count || 0
  } catch {
    // Silently fail
  }
}

onMounted(() => {
  fetchUnreadCount()
  pollTimer = setInterval(fetchUnreadCount, 30000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

const handleClick = () => {
  router.push('/notifications')
}
</script>

<template>
  <button class="header-action-btn notification-bell" @click="handleClick" aria-label="通知">
    <el-badge :value="unreadCount" :hidden="unreadCount === 0" :max="99">
      <el-icon :size="18"><Bell /></el-icon>
    </el-badge>
  </button>
</template>

<style scoped>
.notification-bell {
  position: relative;
}
.notification-bell :deep(.el-badge__content) {
  font-size: 11px;
}
</style>
