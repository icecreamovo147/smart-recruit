<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ChatDotRound, Plus, Refresh, Search, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { createCapabilityFromTemplate, getCapability, listCapabilities } from '@/api/capability'
import { useAuthStore } from '@/stores/auth'
import type { CapabilityInfo, CapabilityPoint } from '@/types/capability'

type TagType = 'success' | 'info' | 'warning' | 'danger' | 'primary'

const auth = useAuthStore()
const list = ref<CapabilityInfo[]>([])
const total = ref(0)
const loading = ref(false)
const error = ref('')
const keyword = ref('')
const statusFilter = ref('')
const detailVisible = ref(false)
const detailLoading = ref(false)
const selectedCapability = ref<CapabilityInfo | null>(null)
const createDialogVisible = ref(false)
const creating = ref(false)
const createForm = reactive({
  template_key: 'candidate_match',
  display_name: '',
  description: '',
  scenarios_text: '',
  instruction: '',
})

const canCreateCapability = computed(() => auth.isRecruitingAdmin || auth.isSystemAdmin || auth.isLegacyAdmin)

const templateOptions = [
  { value: 'candidate_match', label: '候选人匹配评分' },
  { value: 'resume_summary', label: '简历总结' },
  { value: 'interview_questions', label: '面试题生成' },
  { value: 'offer_copy', label: 'Offer 文案辅助' },
  { value: 'custom', label: '自定义招聘能力' },
]

const loadList = async () => {
  loading.value = true
  error.value = ''
  try {
    const data = await listCapabilities()
    list.value = data.list || []
    total.value = data.total ?? list.value.length
  } catch (e: unknown) {
    error.value = (e as { message?: string }).message || '加载能力列表失败'
  } finally {
    loading.value = false
  }
}

const normalizeStatus = (status?: string): 'available' | 'unavailable' | 'preparing' => {
  const value = (status || '').toLowerCase()
  if (!value || ['enabled', 'active', 'online', 'published', 'available'].includes(value)) return 'available'
  if (['disabled', 'inactive', 'offline', 'unavailable'].includes(value)) return 'unavailable'
  return 'preparing'
}

const statusLabel = (status?: string): string => {
  const normalized = normalizeStatus(status)
  if (normalized === 'available') return '可用'
  if (normalized === 'unavailable') return '不可用'
  return '准备中'
}

const statusType = (status?: string): TagType => {
  const normalized = normalizeStatus(status)
  if (normalized === 'available') return 'success'
  if (normalized === 'unavailable') return 'info'
  return 'warning'
}

const scenariosOf = (item: CapabilityInfo | null): string[] => {
  if (!item?.scenarios) return []
  if (Array.isArray(item.scenarios)) return item.scenarios.filter(Boolean)
  return item.scenarios
    .split(/[,\n，、]/)
    .map((scenario) => scenario.trim())
    .filter(Boolean)
}

const pointsOf = (item: CapabilityInfo | null): CapabilityPoint[] => item?.tools || []

const capabilityTitle = (item: CapabilityInfo | null): string =>
  item?.display_name || item?.name || '未命名能力'

const categoryLabel = (category?: string): string => {
  const labels: Record<string, string> = {
    candidate_screening: '候选人分析',
    interview: '面试辅助',
    offer: 'Offer 辅助',
    job: '岗位辅助',
    recruiting: '招聘协作',
  }
  return labels[category || ''] || category || ''
}

const pointTitle = (item: CapabilityPoint): string =>
  item.display_name || item.name || '能力点'

const formatTime = (value?: string): string => {
  if (!value) return '-'
  const time = new Date(value)
  if (Number.isNaN(time.getTime())) return value
  return time.toLocaleString('zh-CN')
}

const filteredCapabilities = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return list.value.filter((item) => {
    const scenarios = scenariosOf(item).join(' ').toLowerCase()
    const points = pointsOf(item).map((point) => `${pointTitle(point)} ${point.description || ''}`).join(' ').toLowerCase()
    const matchesKeyword = !q
      || capabilityTitle(item).toLowerCase().includes(q)
      || (item.description || '').toLowerCase().includes(q)
      || (item.category || '').toLowerCase().includes(q)
      || scenarios.includes(q)
      || points.includes(q)
    const matchesStatus = !statusFilter.value || normalizeStatus(item.status) === statusFilter.value
    return matchesKeyword && matchesStatus
  })
})

const stats = computed(() => {
  const available = list.value.filter((item) => normalizeStatus(item.status) === 'available').length
  const points = list.value.reduce((sum, item) => sum + (item.tools_count ?? pointsOf(item).length), 0)
  const categories = new Set(list.value.map((item) => item.category).filter(Boolean)).size
  return [
    { label: '可用能力', value: available },
    { label: '能力点', value: points },
    { label: '业务分类', value: categories },
  ]
})

const resetFilters = () => {
  keyword.value = ''
  statusFilter.value = ''
}

const openCreateDialog = () => {
  createForm.template_key = 'candidate_match'
  createForm.display_name = ''
  createForm.description = ''
  createForm.scenarios_text = ''
  createForm.instruction = ''
  createDialogVisible.value = true
}

const scenariosFromText = (value: string): string[] =>
  value
    .split(/[,\n，、]/)
    .map((item) => item.trim())
    .filter(Boolean)

const submitCreateCapability = async () => {
  if (!createForm.display_name.trim()) {
    ElMessage.warning('请输入能力名称')
    return
  }
  if (!createForm.description.trim()) {
    ElMessage.warning('请输入能力说明')
    return
  }
  creating.value = true
  try {
    await createCapabilityFromTemplate({
      template_key: createForm.template_key,
      display_name: createForm.display_name.trim(),
      description: createForm.description.trim(),
      scenarios: scenariosFromText(createForm.scenarios_text),
      instruction: createForm.instruction.trim() || undefined,
    })
    ElMessage.success('能力已添加')
    createDialogVisible.value = false
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '添加能力失败')
  } finally {
    creating.value = false
  }
}

const openDetail = async (row: CapabilityInfo) => {
  detailVisible.value = true
  selectedCapability.value = row
  detailLoading.value = true
  try {
    const data = await getCapability(row.id)
    selectedCapability.value = data.capability || row
  } catch {
    selectedCapability.value = row
  } finally {
    detailLoading.value = false
  }
}

const promptUseInAssistant = () => {
  ElMessage.info('请在 AI 数据助手中使用该招聘能力')
}

onMounted(() => {
  loadList()
})
</script>

<template>
  <div class="capability-center-view">
    <section class="page-header">
      <div>
        <p class="page-kicker">AI 招聘能力中心</p>
        <h2 class="page-title">招聘能力</h2>
        <p class="page-desc">查看当前可用于招聘流程的 AI 能力、适用场景和业务能力点。</p>
      </div>
      <div class="page-actions">
        <el-button v-if="canCreateCapability" type="primary" :icon="Plus" @click="openCreateDialog">
          添加能力
        </el-button>
        <el-button :icon="Refresh" @click="loadList">刷新</el-button>
      </div>
    </section>

    <section class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">能力总数</div>
        <div class="stat-value">{{ total || list.length }}</div>
      </div>
      <div v-for="item in stats" :key="item.label" class="stat-card">
        <div class="stat-label">{{ item.label }}</div>
        <div class="stat-value">{{ item.value }}</div>
      </div>
    </section>

    <el-card class="workbench-card" shadow="never">
      <div class="filter-toolbar">
        <el-input
          v-model="keyword"
          class="filter-search"
          :prefix-icon="Search"
          clearable
          placeholder="搜索能力名称 / 适用场景 / 能力点"
        />
        <el-select v-model="statusFilter" class="filter-select" clearable placeholder="全部状态">
          <el-option value="available" label="可用" />
          <el-option value="preparing" label="准备中" />
          <el-option value="unavailable" label="不可用" />
        </el-select>
        <div class="filter-actions">
          <el-button @click="resetFilters">重置</el-button>
        </div>
      </div>

      <el-table
        v-loading="loading"
        :data="filteredCapabilities"
        stripe
        style="width: 100%"
        :empty-text="error || '暂无匹配能力'"
        @row-click="openDetail"
      >
        <el-table-column label="能力名称" min-width="260">
          <template #default="{ row }: { row: CapabilityInfo }">
            <div class="capability-cell">
              <div class="capability-title">{{ capabilityTitle(row) }}</div>
              <div v-if="row.category" class="capability-category">{{ categoryLabel(row.category) }}</div>
              <div v-if="row.description" class="capability-desc">{{ row.description }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="适用场景" min-width="220">
          <template #default="{ row }: { row: CapabilityInfo }">
            <div v-if="scenariosOf(row).length" class="tag-list">
              <el-tag v-for="scenario in scenariosOf(row)" :key="scenario" size="small" effect="plain">
                {{ scenario }}
              </el-tag>
            </div>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="可用状态" width="110">
          <template #default="{ row }: { row: CapabilityInfo }">
            <el-tag :type="statusType(row.status)" effect="light">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="能力点" width="100">
          <template #default="{ row }: { row: CapabilityInfo }">
            {{ row.tools_count ?? pointsOf(row).length }}
          </template>
        </el-table-column>
        <el-table-column label="最近更新" width="170">
          <template #default="{ row }: { row: CapabilityInfo }">
            {{ formatTime(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }: { row: CapabilityInfo }">
            <el-button size="small" :icon="View" @click.stop="openDetail(row)">详情</el-button>
            <el-button size="small" type="primary" plain :icon="ChatDotRound" @click.stop="promptUseInAssistant">
              试用
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer
      v-model="detailVisible"
      class="capability-drawer"
      direction="rtl"
      size="520px"
      :title="capabilityTitle(selectedCapability)"
    >
      <div v-loading="detailLoading" class="drawer-body">
        <div class="drawer-meta">
          <el-tag :type="statusType(selectedCapability?.status)" effect="light">
            {{ statusLabel(selectedCapability?.status) }}
          </el-tag>
          <span v-if="selectedCapability?.category" class="drawer-category">{{ categoryLabel(selectedCapability.category) }}</span>
        </div>

        <p class="drawer-desc">{{ selectedCapability?.description || '暂无能力说明' }}</p>

        <section class="drawer-section">
          <h3>适用场景</h3>
          <div v-if="scenariosOf(selectedCapability).length" class="tag-list">
            <el-tag v-for="scenario in scenariosOf(selectedCapability)" :key="scenario" effect="plain">
              {{ scenario }}
            </el-tag>
          </div>
          <el-empty v-else description="暂未配置适用场景" :image-size="80" />
        </section>

        <section class="drawer-section">
          <h3>能力点</h3>
          <div v-if="pointsOf(selectedCapability).length" class="point-list">
            <div v-for="point in pointsOf(selectedCapability)" :key="point.id || point.name || point.display_name" class="point-item">
              <div class="point-title">{{ pointTitle(point) }}</div>
              <div class="point-desc">{{ point.description || '暂无说明' }}</div>
            </div>
          </div>
          <el-empty v-else description="暂未配置能力点" :image-size="80" />
        </section>

        <section class="drawer-section">
          <h3>最近更新</h3>
          <div class="muted">{{ formatTime(selectedCapability?.updated_at) }}</div>
        </section>

        <div class="drawer-actions">
          <el-button type="primary" :icon="ChatDotRound" @click="promptUseInAssistant">在 AI 助手中使用</el-button>
        </div>
      </div>
    </el-drawer>

    <el-dialog
      v-model="createDialogVisible"
      title="添加招聘能力"
      width="640px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="createForm" label-width="104px">
        <el-form-item label="能力模板">
          <el-select v-model="createForm.template_key" style="width: 100%">
            <el-option
              v-for="item in templateOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="能力名称" required>
          <el-input v-model="createForm.display_name" placeholder="例如：技术岗候选人匹配评分" />
        </el-form-item>
        <el-form-item label="能力说明" required>
          <el-input
            v-model="createForm.description"
            type="textarea"
            :rows="3"
            placeholder="说明这个能力会帮 HR 完成什么招聘任务"
          />
        </el-form-item>
        <el-form-item label="适用场景">
          <el-input
            v-model="createForm.scenarios_text"
            type="textarea"
            :rows="2"
            placeholder="例如：简历初筛、候选人详情、面试准备"
          />
        </el-form-item>
        <el-form-item label="输出要求">
          <el-input
            v-model="createForm.instruction"
            type="textarea"
            :rows="5"
            placeholder="可选。描述希望 AI 如何输出结果，例如包含评分、理由和风险点"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="submitCreateCapability">发布能力</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.capability-center-view {
  padding: 24px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.page-kicker {
  margin: 0 0 6px;
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 700;
}

.page-title {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 24px;
  font-weight: 700;
}

.page-desc {
  margin: 8px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.page-actions {
  display: flex;
  flex-shrink: 0;
  gap: 10px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.stat-card,
.workbench-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.stat-card {
  padding: 16px;
}

.stat-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.stat-value {
  margin-top: 8px;
  color: var(--el-text-color-primary);
  font-size: 26px;
  font-weight: 700;
}

.filter-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.filter-search {
  max-width: 360px;
}

.filter-select {
  width: 150px;
}

.filter-actions {
  margin-left: auto;
}

.capability-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
  padding: 4px 0;
}

.capability-title {
  color: var(--el-text-color-primary);
  font-weight: 700;
}

.capability-category,
.capability-desc,
.muted {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.capability-desc {
  display: -webkit-box;
  overflow: hidden;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.drawer-body {
  min-height: 320px;
}

.drawer-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}

.drawer-category {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.drawer-desc {
  margin: 0 0 22px;
  color: var(--el-text-color-primary);
  line-height: 1.7;
}

.drawer-section {
  margin-top: 22px;
}

.drawer-section h3 {
  margin: 0 0 12px;
  color: var(--el-text-color-primary);
  font-size: 15px;
}

.point-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.point-item {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px;
  background: var(--el-fill-color-extra-light);
}

.point-title {
  color: var(--el-text-color-primary);
  font-weight: 700;
}

.point-desc {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.drawer-actions {
  position: sticky;
  bottom: 0;
  margin-top: 28px;
  padding-top: 16px;
  background: var(--el-bg-color);
}

@media (max-width: 900px) {
  .capability-center-view {
    padding: 16px;
  }

  .page-header,
  .filter-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .filter-search,
  .filter-select {
    width: 100%;
    max-width: none;
  }

  .filter-actions {
    margin-left: 0;
  }
}
</style>
