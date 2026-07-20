<script setup lang="ts">
import { ArrowDown, MoreFilled } from '@element-plus/icons-vue'

export interface RowActionItem {
  label: string
  command: string
  divided?: boolean
  danger?: boolean
  disabled?: boolean
}

defineProps<{
  actions: RowActionItem[]
}>()

const emit = defineEmits<{
  command: [cmd: string]
}>()
</script>

<template>
  <el-dropdown trigger="click" @command="(cmd: string) => emit('command', cmd)">
    <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="(action, idx) in actions"
          :key="idx"
          :command="action.command"
          :divided="action.divided"
          :disabled="action.disabled"
          :class="{ 'row-action--danger': action.danger }"
        >
          {{ action.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<style scoped>
:deep(.row-action--danger) {
  color: var(--el-color-danger) !important;
}
</style>
