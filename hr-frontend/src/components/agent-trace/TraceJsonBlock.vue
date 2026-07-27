<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { formatJsonContent, previewJsonContent } from './agentTraceViewModel'

const props = withDefaults(defineProps<{
  content: string
  label?: string
  copyLabel?: string
  collapsedLength?: number
}>(), {
  label: 'JSON',
  copyLabel: '复制',
  collapsedLength: 240,
})

const expanded = ref(false)

const display = computed(() => {
  const meta = formatJsonContent(props.content || '')
  if (meta.previewLength !== props.collapsedLength) {
    return {
      ...meta,
      previewLength: props.collapsedLength,
    }
  }
  return meta
})

const visibleText = computed(() => previewJsonContent(display.value, expanded.value))
const canToggle = computed(() => display.value.formatted.length > props.collapsedLength)

const copy = async () => {
  const text = display.value.formatted || display.value.raw || ''
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const area = document.createElement('textarea')
      area.value = text
      area.style.position = 'fixed'
      area.style.left = '-9999px'
      document.body.appendChild(area)
      area.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(area)
      if (!ok) throw new Error(t('frontend.copy_failed'))
    }
    ElMessage.success(t('common.success'))
  } catch {
    ElMessage.warning(t('common.invalid_request'))
  }
}
</script>

<template>
  <div class="trace-json-block" data-testid="trace-json-block">
    <div class="trace-json-block__header">
      <span class="trace-json-block__label">{{ label }}</span>
      <div class="trace-json-block__actions">
        <el-tag v-if="display.isEmpty" size="small" type="info" effect="plain">空</el-tag>
        <el-tag v-else-if="display.isValidJson" size="small" type="success" effect="plain">
          {{ display.nestedExpanded ? 'JSON · 已展开嵌套' : 'JSON' }}
        </el-tag>
        <el-tag v-else size="small" type="warning" effect="plain">文本</el-tag>
        <el-button size="small" text data-testid="trace-json-copy" @click="copy">
          {{ copyLabel }}
        </el-button>
        <el-button
          v-if="canToggle"
          size="small"
          text
          data-testid="trace-json-toggle"
          @click="expanded = !expanded"
        >
          {{ expanded ? '收起' : '展开全部' }}
        </el-button>
      </div>
    </div>
    <div class="trace-json-block__body" :class="{ 'trace-json-block__body--collapsed': canToggle && !expanded }">
      <pre data-testid="trace-json-content">{{ visibleText }}</pre>
    </div>
  </div>
</template>

<style scoped>
.trace-json-block {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-blank);
  overflow: hidden;
}

.trace-json-block__header {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  align-items: center;
  padding: 6px 8px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: var(--el-fill-color-extra-light);
}

.trace-json-block__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.trace-json-block__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.trace-json-block__body {
  padding: 8px;
  overflow-x: auto;
}

.trace-json-block__body pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-text-color-regular);
}

.trace-json-block__body--collapsed pre {
  max-height: 8.5em;
  overflow: hidden;
}
</style>
