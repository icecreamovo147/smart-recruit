<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { debugSemanticRetrieval } from '@/api/agentSkill'
import type {
  SemanticMemoryDebugItem,
  SemanticRetrievalDebugResult,
  SemanticSkillDebugItem,
} from '@/types/agentSkill'

const loading = ref(false)
const result = ref<SemanticRetrievalDebugResult | null>(null)

const form = reactive({
  query: '',
  agent_type: 'hr_recruiting_agent',
  job_id: undefined as number | undefined,
  application_id: undefined as number | undefined,
  limit: 5,
})

const skillRows = computed<SemanticSkillDebugItem[]>(() => result.value?.skills || [])
const memoryRows = computed<SemanticMemoryDebugItem[]>(() => result.value?.memories || [])

const formatScore = (value?: number) => (typeof value === 'number' ? value.toFixed(2) : '-')

const runDebug = async () => {
  const query = form.query.trim()
  if (!query) {
    ElMessage.warning('请输入查询内容')
    return
  }
  loading.value = true
  try {
    result.value = await debugSemanticRetrieval({
      query,
      agent_type: form.agent_type,
      job_id: form.job_id,
      application_id: form.application_id,
      limit: form.limit,
    })
  } catch (error) {
    console.error(error)
    ElMessage.error('语义召回调试失败')
  } finally {
    loading.value = false
  }
}

const reset = () => {
  form.query = ''
  form.agent_type = 'hr_recruiting_agent'
  form.job_id = undefined
  form.application_id = undefined
  form.limit = 5
  result.value = null
}
</script>

<template>
  <div class="semantic-debug-view">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="page-kicker">Semantic Retrieval</p>
          <h2 class="page-title">语义召回调试</h2>
          <p class="page-desc">检查指定查询下 Agent Skill 与 AI Memory 的召回排序、分数和降级状态。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" @click="reset">重置</el-button>
          <el-button type="primary" :icon="Search" :loading="loading" @click="runDebug">查询</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input
            v-model="form.query"
            class="query-input"
            type="textarea"
            :rows="2"
            maxlength="500"
            show-word-limit
            placeholder="输入招聘场景查询"
          />
          <el-input v-model="form.agent_type" class="filter-input" placeholder="Agent 类型" />
          <el-input-number v-model="form.job_id" class="number-input" :min="0" :controls="false" placeholder="Job ID" />
          <el-input-number
            v-model="form.application_id"
            class="number-input"
            :min="0"
            :controls="false"
            placeholder="Application ID"
          />
          <el-input-number v-model="form.limit" class="limit-input" :min="1" :max="20" :controls="false" />
        </div>
      </div>

      <div v-if="result" class="debug-status">
        <el-tag :type="result.embedding_available ? 'success' : 'warning'" effect="plain">
          {{ result.embedding_available ? 'Embedding 可用' : 'Fallback' }}
        </el-tag>
        <span v-if="result.fallback_reason" class="debug-status__reason">{{ result.fallback_reason }}</span>
      </div>

      <div class="result-grid">
        <section class="result-panel">
          <div class="result-panel__header">
            <h3>Agent Skills</h3>
            <span>{{ skillRows.length }} 条</span>
          </div>
          <el-table v-loading="loading" :data="skillRows" stripe :empty-text="result ? '暂无 Skill 召回' : '输入查询后查看结果'">
            <el-table-column label="Skill" min-width="220">
              <template #default="{ row }: { row: SemanticSkillDebugItem }">
                <div class="entity-cell">
                  <div class="entity-title">{{ row.display_name || row.name }}</div>
                  <div class="entity-sub">{{ row.name }}</div>
                  <div v-if="row.semantic_tags?.length" class="tag-row">
                    <el-tag v-for="tag in row.semantic_tags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="分数" width="96">
              <template #default="{ row }: { row: SemanticSkillDebugItem }">{{ formatScore(row.score) }}</template>
            </el-table-column>
            <el-table-column label="原因" min-width="180" prop="reason" show-overflow-tooltip />
            <el-table-column label="分类" width="140" prop="category" show-overflow-tooltip />
          </el-table>
        </section>

        <section class="result-panel">
          <div class="result-panel__header">
            <h3>AI Memories</h3>
            <span>{{ memoryRows.length }} 条</span>
          </div>
          <el-table v-loading="loading" :data="memoryRows" stripe :empty-text="result ? '暂无 Memory 召回' : '输入查询后查看结果'">
            <el-table-column label="Memory" min-width="260">
              <template #default="{ row }: { row: SemanticMemoryDebugItem }">
                <div class="entity-cell">
                  <div class="entity-title">{{ row.content }}</div>
                  <div class="entity-sub">{{ row.scope_type }} / {{ row.scope_id }} · {{ row.memory_type }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="分数" width="96">
              <template #default="{ row }: { row: SemanticMemoryDebugItem }">{{ formatScore(row.score) }}</template>
            </el-table-column>
            <el-table-column label="权重" width="120">
              <template #default="{ row }: { row: SemanticMemoryDebugItem }">
                {{ formatScore(row.importance) }} / {{ formatScore(row.confidence) }}
              </template>
            </el-table-column>
            <el-table-column label="原因" min-width="190" prop="reason" show-overflow-tooltip />
          </el-table>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.semantic-debug-view {
  height: 100%;
}

.workspace-surface {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-height: 100%;
  padding: 22px 24px;
}

.workspace-surface__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--admin-console-header-bg);
}

.workspace-surface__header-actions,
.workspace-surface__filters,
.debug-status,
.tag-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.workspace-surface__divider {
  height: 1px;
  background: var(--el-border-color-lighter);
}

.workspace-surface__toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.workspace-surface__filters {
  flex: 1;
  flex-wrap: wrap;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.25;
}

.page-kicker {
  margin: 0 0 6px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-color-primary);
  text-transform: uppercase;
  letter-spacing: 0;
}

.page-desc {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.query-input {
  width: min(520px, 100%);
}

.filter-input {
  width: 210px;
}

.number-input {
  width: 140px;
}

.limit-input {
  width: 90px;
}

.debug-status {
  min-height: 32px;
}

.debug-status__reason,
.entity-sub {
  color: var(--el-text-color-secondary);
}

.result-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}

.result-panel {
  min-width: 0;
}

.result-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.result-panel__header h3 {
  margin: 0;
  font-size: 16px;
}

.result-panel__header span {
  color: var(--el-text-color-secondary);
}

.entity-cell {
  min-width: 0;
}

.entity-title {
  font-weight: 600;
  line-height: 1.4;
  word-break: break-word;
}

.entity-sub {
  margin-top: 2px;
  font-size: 12px;
}

.tag-row {
  flex-wrap: wrap;
  margin-top: 6px;
}

@media (max-width: 1100px) {
  .result-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .workspace-surface {
    padding: 16px;
  }

  .workspace-surface__header {
    flex-direction: column;
  }

  .workspace-surface__header-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
