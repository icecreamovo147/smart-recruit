<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { formatShanghaiDateTime } from '@shared/utils/format'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowDown,
  Connection,
  Delete,
  Edit,
  Grid,
  Plus,
  Search,
  Setting,
} from '@element-plus/icons-vue'
import {
  listEmbeddingProviders,
  createEmbeddingProvider,
  updateEmbeddingProvider,
  deleteEmbeddingProvider,
  testEmbeddingModel,
  listEmbeddingModels,
  createEmbeddingModel,
  updateEmbeddingModel,
  setDefaultEmbeddingModel,
} from '@/api/embedding'
import { useAuthStore } from '@/stores/auth'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { PagePanel } from '@/components/admin-console'
import type {
  EmbeddingProvider,
  EmbeddingModel,
  CreateEmbeddingProviderPayload,
  UpdateEmbeddingProviderPayload,
  CreateEmbeddingModelPayload,
  UpdateEmbeddingModelPayload,
} from '@shared/types/embedding'

const canManage = computed(() => useAuthStore().can(PLATFORM_PERMISSIONS.AI_CONFIG_MANAGE))

const props = defineProps<{
  section?: 'providers' | 'models'
}>()

// ====== Provider State ======

const providerList = ref<EmbeddingProvider[]>([])
const providerTotal = ref(0)
const providerPage = ref(1)
const providerPageSize = ref(20)
const providerLoading = ref(false)
const providerError = ref('')
const providerSearch = ref('')
const providerTypeFilter = ref('')
const providerStatusFilter = ref('')

const loadProviders = async () => {
  providerLoading.value = true
  providerError.value = ''
  try {
    const data = await listEmbeddingProviders(providerPage.value, providerPageSize.value)
    providerList.value = data.list || []
    providerTotal.value = data.total || 0
  } catch (e: unknown) {
    providerError.value = (e as { message?: string }).message || '加载 Provider 列表失败'
  } finally {
    providerLoading.value = false
  }
}

// ---- Provider Drawer ----

const providerDrawerVisible = ref(false)
const providerDrawerTitle = ref('')
const providerSaving = ref(false)
const isEditingProvider = ref(false)
const editingProviderId = ref(0)
const providerForm = reactive({
  name: '',
  provider_type: 'bailian',
  endpoint: '',
  api_key: '',
  extra_headers_json: '',
  is_enabled: true,
})

const resetProviderForm = () => {
  providerForm.name = ''
  providerForm.provider_type = 'bailian'
  providerForm.endpoint = ''
  providerForm.api_key = ''
  providerForm.extra_headers_json = ''
  providerForm.is_enabled = true
}

const openCreateProvider = () => {
  isEditingProvider.value = false
  editingProviderId.value = 0
  providerDrawerTitle.value = '新增 Embedding Provider'
  resetProviderForm()
  providerDrawerVisible.value = true
}

const openEditProvider = (row: EmbeddingProvider) => {
  isEditingProvider.value = true
  editingProviderId.value = row.id
  providerDrawerTitle.value = '编辑 Embedding Provider'
  providerForm.name = row.name
  providerForm.provider_type = row.provider_type
  providerForm.endpoint = row.endpoint
  providerForm.api_key = ''
  providerForm.extra_headers_json = row.extra_headers_json || ''
  providerForm.is_enabled = row.is_enabled
  providerDrawerVisible.value = true
}

const saveProvider = async () => {
  if (!providerForm.name) { ElMessage.warning('请输入 Provider 名称'); return }
  if (!providerForm.endpoint) { ElMessage.warning('请输入 Endpoint'); return }
  if (!isEditingProvider.value && !providerForm.api_key) { ElMessage.warning('请输入 API Key'); return }
  providerSaving.value = true
  try {
    if (isEditingProvider.value) {
      const payload: UpdateEmbeddingProviderPayload = {
        name: providerForm.name,
        provider_type: providerForm.provider_type,
        endpoint: providerForm.endpoint,
        is_enabled: providerForm.is_enabled,
        is_enabled_set: true,
      }
      if (providerForm.api_key) payload.api_key = providerForm.api_key
      if (providerForm.extra_headers_json !== undefined) {
        payload.extra_headers_json = providerForm.extra_headers_json
        payload.extra_headers_set = true
      }
      await updateEmbeddingProvider(editingProviderId.value, payload)
      ElMessage.success('Provider 已更新')
    } else {
      await createEmbeddingProvider({
        name: providerForm.name,
        provider_type: providerForm.provider_type,
        endpoint: providerForm.endpoint,
        api_key: providerForm.api_key,
        extra_headers_json: providerForm.extra_headers_json || undefined,
      })
      ElMessage.success('Provider 已创建')
    }
    providerDrawerVisible.value = false
    await loadProviders()
  } finally {
    providerSaving.value = false
  }
}

const handleDeleteProvider = async (row: EmbeddingProvider) => {
  try {
    await ElMessageBox.confirm(`确认删除 Provider「${row.name}」？`, '删除确认', { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' })
  } catch { return }
  try {
    await deleteEmbeddingProvider(row.id)
    ElMessage.success('Provider 已删除')
    await loadProviders()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ====== Model State ======

const modelList = ref<EmbeddingModel[]>([])
const modelTotal = ref(0)
const modelPage = ref(1)
const modelPageSize = ref(20)
const modelLoading = ref(false)
const modelError = ref('')
const modelProviderFilter = ref(0)
const modelSearch = ref('')
const modelStatusFilter = ref('')
const modelDefaultFilter = ref('')

const loadModels = async () => {
  modelLoading.value = true
  modelError.value = ''
  try {
    const data = await listEmbeddingModels(modelPage.value, modelPageSize.value, modelProviderFilter.value || undefined)
    modelList.value = data.list || []
    modelTotal.value = data.total || 0
  } catch (e: unknown) {
    modelError.value = (e as { message?: string }).message || '加载 Model 列表失败'
  } finally {
    modelLoading.value = false
  }
}

// ---- Model Drawer ----

const modelDrawerVisible = ref(false)
const modelDrawerTitle = ref('')
const modelSaving = ref(false)
const isEditingModel = ref(false)
const editingModelId = ref(0)
const modelForm = reactive({
  provider_id: 0,
  model_name: '',
  display_name: '',
  embedding_dim: 1024,
  input_token_limit: 0,
  batch_size: 1,
  timeout_seconds: 30,
  max_retries: 2,
  is_default: false,
  is_enabled: true,
})

const resetModelForm = () => {
  modelForm.provider_id = providerList.value.length > 0 ? providerList.value[0].id : 0
  modelForm.model_name = ''
  modelForm.display_name = ''
  modelForm.embedding_dim = 1024
  modelForm.input_token_limit = 0
  modelForm.batch_size = 1
  modelForm.timeout_seconds = 30
  modelForm.max_retries = 2
  modelForm.is_default = false
  modelForm.is_enabled = true
}

const openCreateModel = () => {
  isEditingModel.value = false
  editingModelId.value = 0
  modelDrawerTitle.value = '新增 Embedding Model'
  resetModelForm()
  modelDrawerVisible.value = true
}

const openEditModel = (row: EmbeddingModel) => {
  isEditingModel.value = true
  editingModelId.value = row.id
  modelDrawerTitle.value = '编辑 Embedding Model'
  modelForm.provider_id = row.provider_id
  modelForm.model_name = row.model_name
  modelForm.display_name = row.display_name
  modelForm.embedding_dim = row.embedding_dim
  modelForm.input_token_limit = row.input_token_limit
  modelForm.batch_size = row.batch_size
  modelForm.timeout_seconds = row.timeout_seconds
  modelForm.max_retries = row.max_retries
  modelForm.is_default = row.is_default
  modelForm.is_enabled = row.is_enabled
  modelDrawerVisible.value = true
}

const saveModel = async () => {
  if (!modelForm.model_name) { ElMessage.warning('请输入 Model 名称'); return }
  if (!modelForm.provider_id) { ElMessage.warning('请选择 Provider'); return }
  modelSaving.value = true
  try {
    if (isEditingModel.value) {
      await updateEmbeddingModel(editingModelId.value, {
        model_name: modelForm.model_name,
        display_name: modelForm.display_name,
        embedding_dim: modelForm.embedding_dim,
        embedding_dim_set: true,
        input_token_limit: modelForm.input_token_limit,
        input_token_limit_set: true,
        batch_size: modelForm.batch_size,
        batch_size_set: true,
        timeout_seconds: modelForm.timeout_seconds,
        timeout_seconds_set: true,
        max_retries: modelForm.max_retries,
        max_retries_set: true,
        is_enabled: modelForm.is_enabled,
        is_enabled_set: true,
        is_default: modelForm.is_default,
        is_default_set: true,
      })
      ElMessage.success('Model 已更新')
    } else {
      await createEmbeddingModel({
        provider_id: modelForm.provider_id,
        model_name: modelForm.model_name,
        display_name: modelForm.display_name || undefined,
        embedding_dim: modelForm.embedding_dim,
        input_token_limit: modelForm.input_token_limit,
        batch_size: modelForm.batch_size,
        timeout_seconds: modelForm.timeout_seconds,
        max_retries: modelForm.max_retries,
        is_default: modelForm.is_default,
      })
      ElMessage.success('Model 已创建')
    }
    modelDrawerVisible.value = false
    await loadModels()
  } finally {
    modelSaving.value = false
  }
}

const handleSetDefault = async (row: EmbeddingModel) => {
  try {
    await setDefaultEmbeddingModel(row.id)
    ElMessage.success(`已设置「${row.model_name}」为默认模型`)
    await loadModels()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '设置默认模型失败')
  }
}

const handleDeleteModel = async (row: EmbeddingModel) => {
  try {
    await ElMessageBox.confirm(`确认删除 Model「${row.model_name}」？`, '删除确认', { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' })
  } catch { return }
  try {
    await deleteEmbeddingProvider(row.id)
    ElMessage.success('Model 已删除')
    await loadModels()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ---- Test Model ----

/** Concurrent tests: track every model currently in-flight (not a single scalar id). */
const testingModelIds = ref<Set<number>>(new Set())

const isTestingModel = (id: number): boolean => testingModelIds.value.has(id)

const markModelTesting = (id: number, testing: boolean) => {
  const next = new Set(testingModelIds.value)
  if (testing) next.add(id)
  else next.delete(id)
  testingModelIds.value = next
}

const handleTestModel = async (row: EmbeddingModel) => {
  if (isTestingModel(row.id)) return
  markModelTesting(row.id, true)
  try {
    const result = await testEmbeddingModel({ provider_id: row.provider_id, model_id: row.id })
    if (result.success) {
      ElMessage.success(`「${row.display_name || row.model_name}」测试成功：维度 ${result.dimension}，耗时 ${result.latency_ms}ms`)
    } else {
      ElMessage.error(`「${row.display_name || row.model_name}」测试失败：${result.detail || '未知错误'}`)
    }
    await loadModels()
  } catch (e: unknown) {
    ElMessage.error(`「${row.display_name || row.model_name}」${(e as { message?: string }).message || '测试连接失败'}`)
  } finally {
    markModelTesting(row.id, false)
  }
}

// ====== Tab Switch ======

const activeTab = ref<'providers' | 'models'>(props.section || 'providers')
const isSingleSection = computed(() => Boolean(props.section))
const currentSection = computed<'providers' | 'models'>(() => props.section || activeTab.value)

const onTabChange = (tab: string | number) => {
  if (tab === 'providers') {
    loadProviders()
  } else if (tab === 'models') {
    if (providerList.value.length === 0) {
      loadProviders().then(() => loadModels())
    } else {
      loadModels()
    }
  }
}

// ====== Computed Views ======

const providerTypeOptions = computed(() => {
  const set = new Set(providerList.value.map((item) => item.provider_type).filter(Boolean))
  return Array.from(set)
})

const filteredProviders = computed(() => {
  const keyword = providerSearch.value.trim().toLowerCase()
  return providerList.value.filter((item) => {
    const matchesKeyword = !keyword
      || item.name.toLowerCase().includes(keyword)
      || item.endpoint.toLowerCase().includes(keyword)
    const matchesType = !providerTypeFilter.value || item.provider_type === providerTypeFilter.value
    const matchesStatus = !providerStatusFilter.value
      || (providerStatusFilter.value === 'enabled' ? item.is_enabled : !item.is_enabled)
    return matchesKeyword && matchesType && matchesStatus
  })
})

const filteredModels = computed(() => {
  const keyword = modelSearch.value.trim().toLowerCase()
  return modelList.value.filter((item) => {
    const matchesKeyword = !keyword
      || item.model_name.toLowerCase().includes(keyword)
      || (item.display_name || '').toLowerCase().includes(keyword)
      || (item.provider_name || '').toLowerCase().includes(keyword)
    const matchesStatus = !modelStatusFilter.value
      || (modelStatusFilter.value === 'enabled' ? item.is_enabled : !item.is_enabled)
    const matchesDefault = !modelDefaultFilter.value
      || (modelDefaultFilter.value === 'default' ? item.is_default : !item.is_default)
    return matchesKeyword && matchesStatus && matchesDefault
  })
})

// ====== Helpers ======

const providerTypeLabel = (t: string): string => {
  const map: Record<string, string> = {
    bailian: '百炼 Bailian',
  }
  return map[t] || t || '未知'
}

const formatTime = (s?: string): string => {
  return formatShanghaiDateTime(s)
}

const maskApiKey = (key: string): string => {
  if (!key || key.length < 8) return key || '-'
  return key.slice(0, 4) + '****' + key.slice(-4)
}

const testStatusTag = (status: string): 'success' | 'danger' | 'info' | 'warning' => {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  return 'info'
}

const testStatusLabel = (status: string): string => {
  if (status === 'success') return '通过'
  if (status === 'failed') return '失败'
  return '未测试'
}

const formatNumber = (value: number): string => Number(value || 0).toLocaleString()

const resetProviderFilters = () => {
  providerSearch.value = ''
  providerTypeFilter.value = ''
  providerStatusFilter.value = ''
}

const resetModelFilters = () => {
  modelSearch.value = ''
  modelProviderFilter.value = 0
  modelStatusFilter.value = ''
  modelDefaultFilter.value = ''
  modelPage.value = 1
  loadModels()
}

// ====== Init ======

onMounted(() => {
  onTabChange(currentSection.value)
})
</script>

<template>
  <div class="llm-config-view">
    <PagePanel>
      <div class="workspace-surface">
      <el-tabs v-if="!isSingleSection" v-model="activeTab" class="console-tabs" @tab-change="onTabChange">
        <el-tab-pane label="Provider" name="providers" />
        <el-tab-pane label="Model" name="models" />
      </el-tabs>

      <div v-if="!isSingleSection" class="workspace-surface__divider"></div>

      <template v-if="currentSection === 'providers'">
        <div class="workspace-surface__toolbar">
          <div class="workspace-surface__filters">
            <el-input v-model="providerSearch" :prefix-icon="Search" clearable placeholder="搜索名称 / Endpoint" style="width: 200px" />
            <el-select v-model="providerTypeFilter" clearable placeholder="类型" style="width: 140px">
              <el-option v-for="type in providerTypeOptions" :key="type" :value="type" :label="providerTypeLabel(type)" />
            </el-select>
            <el-select v-model="providerStatusFilter" clearable placeholder="状态" style="width: 110px">
              <el-option value="enabled" label="启用" />
              <el-option value="disabled" label="禁用" />
            </el-select>
          </div>
          <div class="workspace-surface__actions">
            <el-button @click="resetProviderFilters">重置</el-button>
            <el-button v-if="canManage" type="primary" :icon="Plus" @click="openCreateProvider">新增 Provider</el-button>
          </div>
        </div>

        <div class="workspace-surface__body">
          <el-table
            height="100%"
            v-loading="providerLoading"
            :data="filteredProviders"
            class="console-table"
            style="width: 100%"
            :empty-text="providerError || '暂无 Provider'"
          >
            <el-table-column label="Provider 信息" min-width="220">
              <template #default="{ row }: { row: EmbeddingProvider }">
                <div class="entity-cell">
                  <div class="entity-icon"><Setting /></div>
                  <div>
                    <strong>{{ row.name }}</strong>
                    <span>ID {{ row.id }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="Endpoint" min-width="260">
              <template #default="{ row }: { row: EmbeddingProvider }">
                <el-tooltip :content="row.endpoint" placement="top">
                  <span class="url-text">{{ row.endpoint }}</span>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="API Key" width="170">
              <template #default="{ row }: { row: EmbeddingProvider }">
                <span class="secret-text">{{ maskApiKey(row.api_key_masked) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="140">
              <template #default="{ row }: { row: EmbeddingProvider }">
                <el-tag type="info" effect="plain">{{ providerTypeLabel(row.provider_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="启用状态" width="110">
              <template #default="{ row }: { row: EmbeddingProvider }">
                <el-tag :type="row.is_enabled ? 'success' : 'info'" effect="light">
                  {{ row.is_enabled ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }: { row: EmbeddingProvider }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }: { row: EmbeddingProvider }">
                <div class="row-actions">
                  <el-button v-if="canManage" size="small" :icon="Edit" @click="openEditProvider(row)">编辑</el-button>
                  <el-dropdown trigger="click">
                    <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item :icon="Delete" @click="handleDeleteProvider(row)">删除</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="workspace-surface__pagination">
          <el-pagination
            v-model:current-page="providerPage"
            v-model:page-size="providerPageSize"
            :total="providerTotal"
            :page-sizes="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            @current-change="loadProviders"
            @size-change="(s: number) => { providerPageSize = s; providerPage = 1; loadProviders() }"
          />
        </div>
      </template>

      <template v-else>
        <div class="workspace-surface__toolbar">
          <div class="workspace-surface__filters">
            <el-input v-model="modelSearch" :prefix-icon="Search" clearable placeholder="搜索模型 / 显示名称 / Provider" style="width: 230px" />
            <el-select
              v-model="modelProviderFilter"
              clearable
              placeholder="全部 Provider"
              style="width: 165px"
              @change="() => { modelPage = 1; loadModels() }"
            >
              <el-option :value="0" label="全部 Provider" />
              <el-option v-for="p in providerList" :key="p.id" :value="p.id" :label="p.name" />
            </el-select>
            <el-select v-model="modelStatusFilter" clearable placeholder="状态" style="width: 110px">
              <el-option value="enabled" label="启用" />
              <el-option value="disabled" label="禁用" />
            </el-select>
            <el-select v-model="modelDefaultFilter" clearable placeholder="默认" style="width: 130px">
              <el-option value="default" label="默认 Model" />
              <el-option value="custom" label="非默认" />
            </el-select>
          </div>
          <div class="workspace-surface__actions">
            <el-button @click="resetModelFilters">重置</el-button>
            <el-button v-if="canManage" type="primary" :icon="Plus" @click="openCreateModel">新增 Model</el-button>
          </div>
        </div>

        <div class="workspace-surface__body">
          <el-table
            height="100%"
            v-loading="modelLoading"
            :data="filteredModels"
            class="console-table"
            style="width: 100%"
            :empty-text="modelError || '暂无 Model'"
          >
            <el-table-column label="模型信息" min-width="230">
              <template #default="{ row }: { row: EmbeddingModel }">
                <div class="entity-cell">
                  <div class="entity-icon model"><Grid /></div>
                  <div>
                    <strong>{{ row.display_name || row.model_name }}</strong>
                    <span>{{ row.model_name }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="Provider" min-width="150">
              <template #default="{ row }: { row: EmbeddingModel }">
                {{ row.provider_name || `Provider ${row.provider_id}` }}
              </template>
            </el-table-column>
            <el-table-column label="参数摘要" min-width="180">
              <template #default="{ row }: { row: EmbeddingModel }">
                <div class="metric-stack">
                  <span>维度 {{ formatNumber(row.embedding_dim) }}</span>
                  <span>Batch {{ row.batch_size }}</span>
                  <span>{{ row.timeout_seconds }}s 超时 / {{ row.max_retries }} 重试</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="默认 / 状态" width="140">
              <template #default="{ row }: { row: EmbeddingModel }">
                <div class="status-stack">
                  <el-tag v-if="row.is_default" type="warning" effect="light">默认</el-tag>
                  <el-tag v-else type="info" effect="plain">非默认</el-tag>
                  <el-tag :type="row.is_enabled ? 'success' : 'info'" effect="light">
                    {{ row.is_enabled ? '启用' : '禁用' }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="测试状态" width="120">
              <template #default="{ row }: { row: EmbeddingModel }">
                <el-tooltip :content="row.last_test_error || testStatusLabel(row.last_test_status)" placement="top">
                  <el-tag :type="testStatusTag(row.last_test_status)" effect="light">
                    {{ testStatusLabel(row.last_test_status) }}
                  </el-tag>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }: { row: EmbeddingModel }">
                <div class="row-actions">
                  <el-button size="small" :icon="Connection" :loading="isTestingModel(row.id)" @click="handleTestModel(row)">测试</el-button>
                  <el-button v-if="canManage" size="small" :icon="Edit" @click="openEditModel(row)">编辑</el-button>
                  <el-dropdown trigger="click">
                    <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item v-if="!row.is_default" :icon="Setting" @click="handleSetDefault(row)">设为默认</el-dropdown-item>
                        <el-dropdown-item :icon="Delete" @click="handleDeleteModel(row)">删除</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="workspace-surface__pagination">
          <el-pagination
            v-model:current-page="modelPage"
            v-model:page-size="modelPageSize"
            :total="modelTotal"
            :page-sizes="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            @current-change="loadModels"
            @size-change="(s: number) => { modelPageSize = s; modelPage = 1; loadModels() }"
          />
        </div>
      </template>
    </div>
    </PagePanel>

    <el-drawer
      v-model="providerDrawerVisible"
      :title="providerDrawerTitle"
      size="520px"
      :close-on-click-modal="true"
      class="config-drawer"
    >
      <el-form :model="providerForm" label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="providerForm.name" placeholder="例如：阿里云百炼" />
        </el-form-item>
        <el-form-item label="Provider 类型" required>
          <el-select v-model="providerForm.provider_type" style="width: 100%">
            <el-option value="bailian" label="百炼 Bailian" />
          </el-select>
        </el-form-item>
        <el-form-item label="Endpoint" required>
          <el-input v-model="providerForm.endpoint" placeholder="https://dashscope.aliyuncs.com/..." />
        </el-form-item>
        <el-form-item :label="isEditingProvider ? 'API Key（留空不修改）' : 'API Key'" :required="!isEditingProvider">
          <el-input
            v-model="providerForm.api_key"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="输入 API Key（编辑时留空表示不修改）"
          />
        </el-form-item>
        <el-form-item label="Extra Headers">
          <el-input
            v-model="providerForm.extra_headers_json"
            type="textarea"
            :rows="4"
            placeholder='可选，JSON 格式，如 {"X-DashScope-Plugin": "..."}'
          />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="providerForm.is_enabled" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="drawer-footer">
          <el-button @click="providerDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="providerSaving" @click="saveProvider">保存</el-button>
        </div>
      </template>
    </el-drawer>

    <el-drawer
      v-model="modelDrawerVisible"
      :title="modelDrawerTitle"
      size="560px"
      :close-on-click-modal="true"
      class="config-drawer"
    >
      <el-form :model="modelForm" label-position="top">
        <el-form-item label="所属 Provider" required>
          <el-select v-model="modelForm.provider_id" style="width: 100%" placeholder="选择 Provider">
            <el-option v-for="p in providerList" :key="p.id" :value="p.id" :label="p.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型名称" required>
          <el-input v-model="modelForm.model_name" placeholder="例如：text-embedding-v4" />
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="modelForm.display_name" placeholder="例如：百炼文本向量 v4" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="向量维度">
            <el-input-number v-model="modelForm.embedding_dim" :min="1" :max="8192" controls-position="right" />
          </el-form-item>
          <el-form-item label="Input Token 上限">
            <el-input-number v-model="modelForm.input_token_limit" :min="0" :max="128000" controls-position="right" />
          </el-form-item>
          <el-form-item label="Batch Size">
            <el-input-number v-model="modelForm.batch_size" :min="1" :max="100" controls-position="right" />
          </el-form-item>
          <el-form-item label="超时（秒）">
            <el-input-number v-model="modelForm.timeout_seconds" :min="1" :max="300" controls-position="right" />
          </el-form-item>
          <el-form-item label="最大重试次数">
            <el-input-number v-model="modelForm.max_retries" :min="0" :max="10" controls-position="right" />
          </el-form-item>
        </div>
        <div class="switch-row">
          <el-form-item label="设为默认">
            <el-switch v-model="modelForm.is_default" active-text="默认" inactive-text="普通" />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="modelForm.is_enabled" active-text="启用" inactive-text="禁用" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <div class="drawer-footer">
          <el-button @click="modelDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="modelSaving" @click="saveModel">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.llm-config-view {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding-bottom: 24px;
  color: var(--el-text-color-primary);
}

.llm-config-view > .page-panel {
  flex: 1;
  min-height: 0;
}

.console-table {
  border: 0;
  border-radius: 0;
}

.console-table :deep(.el-table__header th) {
  background: var(--table-header-solid);
  color: var(--text-secondary);
  font-weight: 700;
}

.entity-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 12px;
}

.entity-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--el-color-primary);
  border-radius: 7px;
  background: var(--el-color-primary-light-9);
  flex: 0 0 auto;
  font-size: 14px;
}

.entity-icon.model {
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.14);
}

.entity-cell strong,
.entity-cell span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entity-cell strong {
  max-width: 220px;
  font-weight: 700;
}

.entity-cell span {
  max-width: 240px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.url-text {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
  text-overflow: ellipsis;
  vertical-align: middle;
  white-space: nowrap;
}

.secret-text {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--el-text-color-secondary);
}

.row-actions,
.status-stack {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.metric-stack {
  display: flex;
  flex-direction: column;
  gap: 3px;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}

.form-grid :deep(.el-input-number) {
  width: 100%;
}

.switch-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

@media (max-width: 720px) {
  .form-grid,
  .switch-row {
    grid-template-columns: 1fr;
  }
}
</style>
