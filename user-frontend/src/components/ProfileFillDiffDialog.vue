<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ProfileFillFieldDiff } from '@/types/domain'

const props = defineProps<{
  modelValue: boolean
  loading?: boolean
  reparseLoading?: boolean
  refreshed?: boolean
  refreshReason?: string
  warnings?: string[]
  diffs?: ProfileFillFieldDiff[]
  overwriteExisting: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'update:overwriteExisting': [value: boolean]
  confirm: []
  reparse: []
}>()

const localOverwrite = ref(props.overwriteExisting)

watch(
  () => props.overwriteExisting,
  (value) => {
    localOverwrite.value = value
  },
)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const refreshText = computed(() => {
  if (props.refreshed) {
    switch (props.refreshReason) {
      case 'forced':
        return '已重新解析简历'
      case 'heuristic':
        return '已重新解析（原画像为启发式结果）'
      case 'input_changed':
        return '已重新解析（简历文本已更新）'
      case 'missing':
        return '已解析简历并生成画像'
      default:
        return '已重新解析简历'
    }
  }
  return '复用已有解析结果'
})

const displayDiffs = computed(() => {
  const source = props.diffs || []
  return source.map((item) => {
    if (item.action === 'unsupported') return item
    const before = (item.before || '').trim()
    const after = (item.after || '').trim()
    if (!after) return { ...item, action: 'skip' }
    if (!before) return { ...item, action: 'fill' }
    if (localOverwrite.value && before !== after) return { ...item, action: 'overwrite' }
    return { ...item, action: 'skip' }
  })
})

const grouped = computed(() => {
  const fill = displayDiffs.value.filter((item) => item.action === 'fill')
  const overwrite = displayDiffs.value.filter((item) => item.action === 'overwrite')
  const skip = displayDiffs.value.filter((item) => item.action === 'skip')
  const unsupported = displayDiffs.value.filter((item) => item.action === 'unsupported')
  return { fill, overwrite, skip, unsupported }
})

const actionLabel = (action: string) => {
  switch (action) {
    case 'fill':
      return '将填充'
    case 'overwrite':
      return '将覆盖'
    case 'skip':
      return '将跳过'
    case 'unsupported':
      return '不支持'
    default:
      return action
  }
}

const actionType = (action: string): 'success' | 'warning' | 'info' | 'danger' => {
  switch (action) {
    case 'fill':
      return 'success'
    case 'overwrite':
      return 'warning'
    case 'unsupported':
      return 'danger'
    default:
      return 'info'
  }
}

const onOverwriteChange = (value: boolean | string | number) => {
  const next = Boolean(value)
  localOverwrite.value = next
  emit('update:overwriteExisting', next)
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="从简历填充预览"
    width="640px"
    destroy-on-close
    :close-on-click-modal="false"
  >
    <div v-loading="loading || reparseLoading" class="fill-diff">
      <el-alert
        :title="refreshText"
        :type="refreshed ? 'success' : 'info'"
        :closable="false"
        show-icon
        class="fill-diff__alert"
      />
      <el-alert
        v-for="(warning, index) in warnings || []"
        :key="`warn-${index}`"
        :title="warning"
        type="warning"
        :closable="false"
        show-icon
        class="fill-diff__alert"
      />

      <div class="fill-diff__toolbar">
        <el-checkbox :model-value="localOverwrite" @change="onOverwriteChange">
          覆盖已有内容
        </el-checkbox>
      </div>

      <template v-for="section in [
        { key: 'fill', title: '将填充', items: grouped.fill },
        { key: 'overwrite', title: '将覆盖', items: grouped.overwrite },
        { key: 'skip', title: '将跳过', items: grouped.skip },
        { key: 'unsupported', title: '不支持写入', items: grouped.unsupported },
      ]" :key="section.key">
        <div v-if="section.items.length" class="fill-diff__section">
          <h3>{{ section.title }}（{{ section.items.length }}）</h3>
          <div v-for="item in section.items" :key="`${section.key}-${item.field}`" class="fill-diff__row">
            <div class="fill-diff__row-head">
              <strong>{{ item.label || item.field }}</strong>
              <el-tag size="small" :type="actionType(item.action)">{{ actionLabel(item.action) }}</el-tag>
            </div>
            <div v-if="item.action !== 'unsupported'" class="fill-diff__compare">
              <div>
                <span class="fill-diff__muted">当前</span>
                <p>{{ item.before || '（空）' }}</p>
              </div>
              <div>
                <span class="fill-diff__muted">简历</span>
                <p>{{ item.after || '（空）' }}</p>
              </div>
            </div>
            <p v-else class="fill-diff__note">{{ item.after }}</p>
          </div>
        </div>
      </template>

      <el-empty
        v-if="!displayDiffs.length"
        description="暂无可预览的字段变更"
        :image-size="72"
      />
    </div>

    <template #footer>
      <el-button type="primary" plain :disabled="loading || reparseLoading" @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="reparseLoading" :disabled="loading" @click="emit('reparse')">
        重新解析并刷新
      </el-button>
      <el-button type="primary" :loading="loading" :disabled="reparseLoading" @click="emit('confirm')">
        确认填充
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.fill-diff {
  min-height: 160px;
}

.fill-diff__alert {
  margin-bottom: 10px;
}

.fill-diff__toolbar {
  margin: 8px 0 16px;
}

.fill-diff__section {
  margin-bottom: 16px;
}

.fill-diff__section h3 {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
}

.fill-diff__row {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 8px;
}

.fill-diff__row-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.fill-diff__compare {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.fill-diff__compare p,
.fill-diff__note {
  margin: 4px 0 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.5;
}

.fill-diff__muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

@media (max-width: 640px) {
  .fill-diff__compare {
    grid-template-columns: 1fr;
  }
}
</style>
