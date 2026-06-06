<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listTimelineEvents } from '@/api/collaboration'
import { getHRStatusLabel } from '@/types/domain'
import type { TimelineEventInfo } from '@/types/domain'

const props = defineProps<{
  candidateUserId: number
}>()

interface TimelineDisplayEvent {
  id: string
  type: string
  title: string
  description: string
  timestamp: string
  actorName: string
  statusFrom?: string
  statusTo?: string
}

const loading = ref(false)
const errorMessage = ref('')
const events = ref<TimelineDisplayEvent[]>([])

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const statusKeyPattern = /\b(applied|viewed|screening|screen_passed|interview_pending|interviewing|interview_passed|offer_pending|offer_sent|offer_accepted|offer_rejected|hired|rejected|withdrawn)\b/g

const formatStatusText = (value: string): string => {
  return String(value || '').replace(statusKeyPattern, (key) => getHRStatusLabel(key, key))
}

const parseStatusTransition = (title: string): { title: string; from?: string; to?: string } => {
  const match = String(title || '').match(/^(.*?)\s*变更状态:\s*([a-z_]+)\s*→\s*([a-z_]+)$/)
  if (!match) return { title: formatStatusText(title) }
  const actor = match[1].trim()
  const from = getHRStatusLabel(match[2], match[2])
  const to = getHRStatusLabel(match[3], match[3])
  return {
    title: actor ? `${actor} 更新了投递状态` : '投递状态已更新',
    from,
    to,
  }
}

const eventTypeMeta = (type: string): { label: string; tone: string } => {
  switch (type) {
    case 'status_transition': return { label: '状态', tone: 'success' }
    case 'interview': return { label: '面试', tone: 'warning' }
    case 'offer': return { label: 'Offer', tone: 'danger' }
    case 'note': return { label: '备注', tone: 'primary' }
    default: return { label: '动态', tone: 'info' }
  }
}

const loadTimeline = async () => {
  loading.value = true
  errorMessage.value = ''
  const allEvents: TimelineDisplayEvent[] = []

  try {
    // Load aggregated timeline events from the backend
    const resp = await listTimelineEvents(props.candidateUserId)
    const timelineEvents: TimelineEventInfo[] = resp.events || []
    for (const e of timelineEvents) {
      const transition = e.event_type === 'status_transition' ? parseStatusTransition(e.title) : undefined
      allEvents.push({
        id: e.id,
        type: e.event_type,
        title: transition?.title || formatStatusText(e.title),
        description: formatStatusText(e.description),
        timestamp: e.timestamp,
        actorName: e.actor_name,
        statusFrom: transition?.from,
        statusTo: transition?.to,
      })
    }
  } catch {
    errorMessage.value = '动态时间线加载失败'
  } finally {
    loading.value = false
  }

  // Sort events by timestamp descending (most recent first)
  allEvents.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())

  events.value = allEvents
}

onMounted(loadTimeline)
</script>

<template>
  <div class="timeline-view">
    <el-timeline v-if="events.length > 0" class="activity-timeline">
      <el-timeline-item
        v-for="event in events"
        :key="event.id"
        :timestamp="formatDateTime(event.timestamp)"
        placement="top"
        :type="eventTypeMeta(event.type).tone"
      >
        <div class="timeline-event" :class="'timeline-event--' + event.type">
          <div class="timeline-event-head">
            <el-tag size="small" effect="light" :type="eventTypeMeta(event.type).tone">
              {{ eventTypeMeta(event.type).label }}
            </el-tag>
            <span v-if="event.actorName" class="timeline-actor">{{ event.actorName }}</span>
          </div>
          <div class="timeline-event-title">{{ event.title }}</div>
          <div v-if="event.statusFrom || event.statusTo" class="status-flow">
            <span class="status-chip">{{ event.statusFrom || '-' }}</span>
            <span class="status-arrow">→</span>
            <span class="status-chip status-chip--next">{{ event.statusTo || '-' }}</span>
          </div>
          <div v-if="event.description" class="timeline-event-desc">{{ event.description }}</div>
        </div>
      </el-timeline-item>
    </el-timeline>
    <el-alert v-if="errorMessage && !loading" class="timeline-error" type="warning" :title="errorMessage" show-icon :closable="false" />
    <div v-if="events.length === 0 && !loading" class="no-events">暂无动态</div>
    <el-skeleton v-if="loading" :rows="3" animated />
  </div>
</template>

<style scoped>
.timeline-view {
  padding: 8px 2px 2px;
}

.timeline-error {
  margin-bottom: 12px;
}

.activity-timeline {
  --el-timeline-node-size-normal: 12px;
  padding-left: 2px;
}

.activity-timeline :deep(.el-timeline-item) {
  padding-bottom: 18px;
}

.activity-timeline :deep(.el-timeline-item__tail) {
  border-left-color: #e2e8f0;
}

.activity-timeline :deep(.el-timeline-item__timestamp) {
  margin-bottom: 8px;
  color: #64748b;
  font-size: 13px;
  font-weight: 500;
}

.timeline-event {
  max-width: 860px;
  padding: 12px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.timeline-event--status_transition {
  border-left: 3px solid #67c23a;
}

.timeline-event--interview {
  border-left: 3px solid #e6a23c;
}

.timeline-event--offer {
  border-left: 3px solid #f56c6c;
}

.timeline-event--note {
  border-left: 3px solid #409eff;
}

.timeline-event-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.timeline-actor {
  color: #64748b;
  font-size: 12px;
}

.timeline-event-title {
  font-weight: 600;
  font-size: 15px;
  color: #1f2937;
  line-height: 1.5;
}

.status-flow {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 2px 10px;
  border-radius: 999px;
  background: #f1f5f9;
  color: #475569;
  font-size: 13px;
  font-weight: 600;
}

.status-chip--next {
  background: #ecfdf5;
  color: #047857;
}

.status-arrow {
  color: #94a3b8;
  font-weight: 700;
}

.timeline-event-desc {
  margin-top: 10px;
  font-size: 13px;
  color: #64748b;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.no-events {
  text-align: center;
  color: #94a3b8;
  padding: 40px 0 34px;
}
</style>
