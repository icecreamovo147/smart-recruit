<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listMyInterviews } from '@/api/interview'
import type { InterviewSchedule } from '@/types/domain'

const loading = ref(false)
const errorMessage = ref('')
const interviews = ref<InterviewSchedule[]>([])

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const statusLabel = (status: string): string => {
  const labels: Record<string, string> = {
    pending: '待安排',
    scheduled: '已安排',
    completed: '已完成',
    cancelled: '已取消',
  }
  return labels[status] || status
}

const statusType = (status: string): string => {
  const types: Record<string, string> = {
    pending: 'info',
    scheduled: 'warning',
    completed: 'success',
    cancelled: 'danger',
  }
  return types[status] || 'info'
}

const modeLabel = (mode: string): string => {
  const labels: Record<string, string> = {
    video: '视频面试',
    phone: '电话面试',
    onsite: '现场面试',
  }
  return labels[mode] || mode || '视频面试'
}

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await listMyInterviews()
    interviews.value = data.list || []
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '面试信息加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="my-interviews-view">
    <h2>我的面试</h2>
    <p class="text-secondary">查看您的面试安排和信息</p>

    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon closable class="mb-4" />

    <el-card v-loading="loading" class="mt-4">
      <el-empty v-if="!loading && interviews.length === 0" description="暂无面试安排" />

      <div v-else>
        <el-timeline v-if="interviews.length > 0">
          <el-timeline-item
            v-for="item in interviews"
            :key="item.interview_id"
            :timestamp="formatDateTime(item.scheduled_at)"
            placement="top"
            :color="item.status === 'completed' ? '#67c23a' : item.status === 'cancelled' ? '#f56c6c' : '#409eff'"
          >
            <el-card shadow="hover">
              <div class="interview-card">
                <div class="card-header">
                  <h3>{{ item.job_title }} - {{ item.title || modeLabel(item.mode) }}</h3>
                  <el-tag :type="statusType(item.status)" size="small">{{ statusLabel(item.status) }}</el-tag>
                </div>
                <el-descriptions :column="2" size="small" border class="mt-2">
                  <el-descriptions-item label="面试官">{{ item.interviewer_name }}</el-descriptions-item>
                  <el-descriptions-item label="轮次">第{{ item.round_no || 1 }}轮</el-descriptions-item>
                  <el-descriptions-item label="模式">{{ modeLabel(item.mode) }}</el-descriptions-item>
                  <el-descriptions-item label="时长">{{ item.duration_minutes ? item.duration_minutes + ' 分钟' : '-' }}</el-descriptions-item>
                  <el-descriptions-item v-if="item.meeting_url" label="会议链接" :span="2">
                    <el-link :href="item.meeting_url" target="_blank" type="primary">{{ item.meeting_url }}</el-link>
                  </el-descriptions-item>
                  <el-descriptions-item v-if="item.location" label="面试地点" :span="2">{{ item.location }}</el-descriptions-item>
                  <el-descriptions-item v-if="item.candidate_note" label="注意事项" :span="2">{{ item.candidate_note }}</el-descriptions-item>
                </el-descriptions>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.my-interviews-view {
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
}
.mb-4 {
  margin-bottom: 16px;
}
.mt-2 {
  margin-top: 8px;
}
.mt-4 {
  margin-top: 16px;
}
.text-secondary {
  color: #909399;
  font-size: 14px;
}
.interview-card {
  padding: 4px 0;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-header h3 {
  margin: 0;
  font-size: 16px;
}
</style>
