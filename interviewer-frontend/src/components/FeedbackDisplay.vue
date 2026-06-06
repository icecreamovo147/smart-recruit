<script setup lang="ts">
import type { InterviewFeedback } from '@/types/domain'
import { RECOMMENDATION_LABEL } from '@/types/domain'

defineProps<{
  feedback: InterviewFeedback
  dimensionScores: Array<{ label: string; score: number }>
}>()

const formatDateTime = (iso: string): string => {
  if (!iso) return '-'
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}
</script>

<template>
  <div class="feedback-display">
    <div class="feedback-header">
      <el-tag type="success" size="small" effect="light">已提交</el-tag>
      <span class="feedback-time">{{ formatDateTime(feedback.submitted_at) }}</span>
    </div>

    <!-- Recommendation -->
    <div class="feedback-field">
      <label class="field-label">推荐结论</label>
      <span class="field-value recommendation">
        {{ RECOMMENDATION_LABEL[feedback.recommendation] || feedback.recommendation }}
      </span>
    </div>

    <!-- Overall score -->
    <div class="feedback-field">
      <label class="field-label">综合评分</label>
      <span class="field-value score">{{ feedback.score }} / 10</span>
    </div>

    <!-- Dimension scores -->
    <div v-if="dimensionScores.length > 0" class="feedback-field">
      <label class="field-label">维度评分</label>
      <div class="dimensions-list">
        <div v-for="dim in dimensionScores" :key="dim.label" class="dimension-row">
          <span class="dim-label">{{ dim.label }}</span>
          <div class="dim-bar-wrap">
            <div class="dim-bar" :style="{ width: `${dim.score * 10}%` }" />
          </div>
          <span class="dim-score">{{ dim.score }}</span>
        </div>
      </div>
    </div>

    <!-- Comments -->
    <div v-if="feedback.comments" class="feedback-field">
      <label class="field-label">评语</label>
      <p class="field-value comments">{{ feedback.comments }}</p>
    </div>
  </div>
</template>

<style scoped>
.feedback-display {
  padding: 4px 0;
}

.feedback-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.feedback-time {
  font-size: 13px;
  color: var(--text-muted);
}

.feedback-field {
  margin-bottom: 16px;
}

.field-label {
  display: block;
  font-size: 13px;
  color: var(--text-muted);
  margin-bottom: 6px;
}

.field-value.recommendation {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.field-value.score {
  font-size: 20px;
  font-weight: 600;
  color: var(--brand-primary);
}

.field-value.comments {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.7;
  white-space: pre-wrap;
}

.dimensions-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dimension-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.dim-label {
  font-size: 13px;
  color: var(--text-secondary);
  width: 72px;
  flex-shrink: 0;
}

.dim-bar-wrap {
  flex: 1;
  height: 8px;
  background: var(--border-subtle);
  border-radius: 4px;
  overflow: hidden;
}

.dim-bar {
  height: 100%;
  background: var(--brand-primary);
  border-radius: 4px;
  transition: width 300ms ease;
}

.dim-score {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  width: 24px;
  text-align: right;
}
</style>
