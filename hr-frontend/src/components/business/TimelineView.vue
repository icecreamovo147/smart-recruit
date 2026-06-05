<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listTimelineEvents } from '@/api/collaboration'
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
}

const loading = ref(false)
const events = ref<TimelineDisplayEvent[]>([])

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const eventTypeIcon = (type: string): string => {
  switch (type) {
    case 'status_transition': return 'el-icon-refresh'
    case 'interview': return 'el-icon-microphone'
    case 'offer': return 'el-icon-document-copy'
    case 'note': return 'el-icon-edit'
    default: return 'el-icon-info'
  }
}

const loadTimeline = async () => {
  loading.value = true
  const allEvents: TimelineDisplayEvent[] = []

  try {
    // Load aggregated timeline events from the backend
    const resp = await listTimelineEvents(props.candidateUserId)
    const timelineEvents: TimelineEventInfo[] = resp.events || []
    for (const e of timelineEvents) {
      allEvents.push({
        id: e.id,
        type: e.event_type,
        title: e.title,
        description: e.description,
        timestamp: e.timestamp,
        actorName: e.actor_name,
      })
    }
  } catch {
    // Timeline unavailable
  }

  // Sort events by timestamp descending (most recent first)
  allEvents.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())

  events.value = allEvents
}

onMounted(loadTimeline)
</script>

<template>
  <div class="timeline-view">
    <el-timeline>
      <el-timeline-item
        v-for="event in events"
        :key="event.id"
        :timestamp="formatDateTime(event.timestamp)"
        placement="top"
      >
        <div class="timeline-event">
          <div class="timeline-event-title">
            <span class="event-type-icon" :class="'event-type-' + event.type">
              <el-icon><component :is="eventTypeIcon(event.type)" /></el-icon>
            </span>
            {{ event.title }}
          </div>
          <div v-if="event.description" class="timeline-event-desc">{{ event.description }}</div>
        </div>
      </el-timeline-item>
    </el-timeline>
    <div v-if="events.length === 0 && !loading" class="no-events">暂无动态</div>
    <el-skeleton v-if="loading" :rows="3" animated />
  </div>
</template>

<style scoped>
.timeline-view {
  padding: 10px 0;
}

.timeline-event {
  padding: 0;
}

.timeline-event-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
}

.timeline-event-title .event-type-icon {
  display: inline-block;
  vertical-align: middle;
  margin-right: 4px;
}

.event-type-note {
  color: #409eff;
}

.event-type-status_transition {
  color: #67c23a;
}

.event-type-interview {
  color: #e6a23c;
}

.event-type-offer {
  color: #f56c6c;
}

.timeline-event-desc {
  font-size: 13px;
  color: #666;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

.no-events {
  text-align: center;
  color: #999;
  padding: 40px 0;
}
</style>
