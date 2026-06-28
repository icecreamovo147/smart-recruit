<script setup lang="ts">
import { computed } from 'vue'
import StatusTag from './StatusTag.vue'
import type { ConsoleStatus } from '@/types/ui'

const props = withDefaults(defineProps<{
  status?: Extract<ConsoleStatus, 'healthy' | 'error' | 'untested' | 'warning'>
  label?: string
  latencyMs?: number
}>(), {
  status: 'untested',
})

const text = computed(() => {
  if (props.label) return props.label
  if (props.status === 'healthy' && props.latencyMs !== undefined) return `健康 · ${props.latencyMs}ms`
  if (props.status === 'warning') return '需关注'
  if (props.status === 'error') return '连接异常'
  return '未测试'
})
</script>

<template>
  <StatusTag :status="status" :label="text" />
</template>
