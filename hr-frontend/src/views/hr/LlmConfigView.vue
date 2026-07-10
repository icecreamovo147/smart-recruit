<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowDown,
  Connection,
  Delete,
  Edit,
  Grid,
  MoreFilled,
  Plus,
  Refresh,
  Search,
  Setting,
} from '@element-plus/icons-vue'
import {
  listProviders,
  createProvider,
  updateProvider,
  deleteProvider,
  testProviderConnection,
  listModels,
  createModel,
  updateModel,
  deleteModel,
} from '@/api/llm'
import type {
  LlmProvider,
  LlmModel,
  CreateProviderPayload,
  UpdateProviderPayload,
  CreateModelPayload,
  UpdateModelPayload,
} from '@/types/llm'

const props = defineProps<{
  section?: 'providers' | 'models'
}>()

type ConnectionStatus = 'unknown' | 'success' | 'failed'

interface ProviderHealth {
  status: ConnectionStatus
  detail: string
  testedAt: string
}

// ====== Provider State ======

const providerList = ref<LlmProvider[]>([])
const providerTotal = ref(0)
const providerPage = ref(1)
const providerPageSize = ref(20)
const providerLoading = ref(false)
const providerError = ref('')

const providerSearch = ref('')
const providerTypeFilter = ref('')
const providerStatusFilter = ref('')
const providerHealthFilter = ref('')
const providerHealthMap = reactive<Record<number, ProviderHealth>>({})

const loadProviders = async () => {
  providerLoading.value = true
  providerError.value = ''
  try {
    const data = await listProviders(providerPage.value, providerPageSize.value)
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
  base_url: '',
  api_key: '',
  provider_type: 'openai',
  extra_headers_json: '',
  is_enabled: true,
})

const resetProviderForm = () => {
  providerForm.name = ''
  providerForm.base_url = ''
  providerForm.api_key = ''
  providerForm.provider_type = 'openai'
  providerForm.extra_headers_json = ''
  providerForm.is_enabled = true
}

const openCreateProvider = () => {
  isEditingProvider.value = false
  editingProviderId.value = 0
  providerDrawerTitle.value = '新增 Provider'
  resetProviderForm()
  providerDrawerVisible.value = true
}

const openEditProvider = (row: LlmProvider) => {
  isEditingProvider.value = true
  editingProviderId.value = row.id
  providerDrawerTitle.value = '编辑 Provider'
  providerForm.name = row.name
  providerForm.base_url = row.base_url
  providerForm.api_key = ''      // empty = do not change
  providerForm.provider_type = row.provider_type
  providerForm.extra_headers_json = row.extra_headers_json || ''
  providerForm.is_enabled = row.is_enabled
  providerDrawerVisible.value = true
}

const saveProvider = async () => {
  if (!providerForm.name) {
    ElMessage.warning('请输入 Provider 名称')
    return
  }
  if (!providerForm.base_url) {
    ElMessage.warning('请输入 Base URL')
    return
  }
  if (!isEditingProvider.value && !providerForm.api_key) {
    ElMessage.warning('请输入 API Key')
    return
  }
  providerSaving.value = true
  try {
    if (isEditingProvider.value) {
      const payload: UpdateProviderPayload = {
        name: providerForm.name,
        base_url: providerForm.base_url,
        provider_type: providerForm.provider_type,
        is_enabled: providerForm.is_enabled,
        is_enabled_set: true,
      }
      if (providerForm.api_key) {
        payload.api_key = providerForm.api_key
      }
      if (providerForm.extra_headers_json !== undefined) {
        payload.extra_headers_json = providerForm.extra_headers_json
        payload.extra_headers_set = true
      }
      await updateProvider(editingProviderId.value, payload)
      ElMessage.success('Provider 已更新')
    } else {
      const payload: CreateProviderPayload = {
        name: providerForm.name,
        base_url: providerForm.base_url,
        api_key: providerForm.api_key,
        provider_type: providerForm.provider_type,
      }
      if (providerForm.extra_headers_json) {
        payload.extra_headers_json = providerForm.extra_headers_json
      }
      await createProvider(payload)
      ElMessage.success('Provider 已创建')
    }
    providerDrawerVisible.value = false
    await loadProviders()
  } finally {
    providerSaving.value = false
  }
}

const handleDeleteProvider = async (row: LlmProvider) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 Provider「${row.name}」？删除后关联的 Model 也将失效。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteProvider(row.id)
    ElMessage.success('Provider 已删除')
    await loadProviders()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ---- Test Connection ----

const testingId = ref(0)

const handleTestConnection = async (row: LlmProvider) => {
  testingId.value = row.id
  try {
    const result = await testProviderConnection(row.id)
    providerHealthMap[row.id] = {
      status: result.success ? 'success' : 'failed',
      detail: result.detail || (result.success ? '连接正常' : '未知错误'),
      testedAt: new Date().toISOString(),
    }
    if (result.success) {
      ElMessage.success('连接测试成功')
    } else {
      ElMessage.error(`连接测试失败：${result.detail || '未知错误'}`)
    }
  } catch (e: unknown) {
    const detail = (e as { message?: string }).message || '连接测试失败'
    providerHealthMap[row.id] = {
      status: 'failed',
      detail,
      testedAt: new Date().toISOString(),
    }
    ElMessage.error(detail)
  } finally {
    testingId.value = 0
  }
}

// ====== Model State ======

const modelList = ref<LlmModel[]>([])
const modelTotal = ref(0)
const modelPage = ref(1)
const modelPageSize = ref(20)
const modelLoading = ref(false)
const modelError = ref('')
const modelProviderFilter = ref(0) // 0 = all
const modelSearch = ref('')
const modelStatusFilter = ref('')
const modelDefaultFilter = ref('')

const loadModels = async () => {
  modelLoading.value = true
  modelError.value = ''
  try {
    const data = await listModels(modelPage.value, modelPageSize.value, modelProviderFilter.value || undefined)
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
  temperature: 0.7,
  top_p: 1.0,
  max_tokens: 4096,
  context_window_tokens: 0,
  max_concurrency: 1,
  timeout_seconds: 60,
  is_default: false,
  is_enabled: true,
})

const resetModelForm = () => {
  modelForm.provider_id = providerList.value.length > 0 ? providerList.value[0].id : 0
  modelForm.model_name = ''
  modelForm.display_name = ''
  modelForm.temperature = 0.7
  modelForm.top_p = 1.0
  modelForm.max_tokens = 4096
  modelForm.context_window_tokens = 0
  modelForm.max_concurrency = 1
  modelForm.timeout_seconds = 60
  modelForm.is_default = false
  modelForm.is_enabled = true
}

const openCreateModel = () => {
  isEditingModel.value = false
  editingModelId.value = 0
  modelDrawerTitle.value = '新增 Model'
  resetModelForm()
  modelDrawerVisible.value = true
}

const openEditModel = (row: LlmModel) => {
  isEditingModel.value = true
  editingModelId.value = row.id
  modelDrawerTitle.value = '编辑 Model'
  modelForm.provider_id = row.provider_id
  modelForm.model_name = row.model_name
  modelForm.display_name = row.display_name
  modelForm.temperature = row.temperature
  modelForm.top_p = row.top_p
  modelForm.max_tokens = row.max_tokens
  modelForm.context_window_tokens = row.context_window_tokens || 0
  modelForm.max_concurrency = row.max_concurrency
  modelForm.timeout_seconds = row.timeout_seconds
  modelForm.is_default = row.is_default
  modelForm.is_enabled = row.is_enabled
  modelDrawerVisible.value = true
}

const saveModel = async () => {
  if (!modelForm.model_name) {
    ElMessage.warning('请输入模型名称')
    return
  }
  if (!modelForm.provider_id) {
    ElMessage.warning('请选择 Provider')
    return
  }
  modelSaving.value = true
  try {
    if (isEditingModel.value) {
      const payload: UpdateModelPayload = {
        model_name: modelForm.model_name,
        display_name: modelForm.display_name || undefined,
        temperature: modelForm.temperature,
        temperature_set: true,
        top_p: modelForm.top_p,
        top_p_set: true,
        max_tokens: modelForm.max_tokens,
        max_tokens_set: true,
        context_window_tokens: modelForm.context_window_tokens,
        context_window_tokens_set: true,
        max_concurrency: modelForm.max_concurrency,
        max_concurrency_set: true,
        timeout_seconds: modelForm.timeout_seconds,
        timeout_seconds_set: true,
        is_enabled: modelForm.is_enabled,
        is_enabled_set: true,
        is_default: modelForm.is_default,
        is_default_set: true,
      }
      await updateModel(editingModelId.value, payload)
      ElMessage.success('Model 已更新')
    } else {
      const payload: CreateModelPayload = {
        provider_id: modelForm.provider_id,
        model_name: modelForm.model_name,
        display_name: modelForm.display_name,
        temperature: modelForm.temperature,
        top_p: modelForm.top_p,
        max_tokens: modelForm.max_tokens,
        context_window_tokens: modelForm.context_window_tokens || undefined,
        max_concurrency: modelForm.max_concurrency,
        timeout_seconds: modelForm.timeout_seconds,
        is_default: modelForm.is_default,
      }
      await createModel(payload)
      ElMessage.success('Model 已创建')
    }
    modelDrawerVisible.value = false
    await loadModels()
  } finally {
    modelSaving.value = false
  }
}

const handleDeleteModel = async (row: LlmModel) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 Model「${row.model_name}」？`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteModel(row.id)
    ElMessage.success('Model 已删除')
    await loadModels()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ====== Tab Switch ======

const activeTab = ref<'providers' | 'models'>(props.section || 'providers')
const isSingleSection = computed(() => Boolean(props.section))
const currentSection = computed<'providers' | 'models'>(() => props.section || activeTab.value)
const pageTitle = computed(() => {
  if (props.section === 'providers') return 'Provider 配置'
  if (props.section === 'models') return 'Model 配置'
  return '模型配置'
})
const pageDescription = computed(() => {
  if (props.section === 'providers') return '统一管理大模型服务商、API 入口、密钥和连接健康状态。'
  if (props.section === 'models') return '统一管理模型参数、默认路由、并发和可用状态，支撑 HR AI 对话、分析与自动化能力。'
  return '统一管理大模型服务商、模型参数和默认路由，支撑 HR AI 对话、分析与自动化能力。'
})

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
    const healthStatus = providerHealthMap[item.id]?.status || 'unknown'
    const matchesKeyword = !keyword
      || item.name.toLowerCase().includes(keyword)
      || item.base_url.toLowerCase().includes(keyword)
    const matchesType = !providerTypeFilter.value || item.provider_type === providerTypeFilter.value
    const matchesStatus = !providerStatusFilter.value
      || (providerStatusFilter.value === 'enabled' ? item.is_enabled : !item.is_enabled)
    const matchesHealth = !providerHealthFilter.value || healthStatus === providerHealthFilter.value
    return matchesKeyword && matchesType && matchesStatus && matchesHealth
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
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    azure: 'Azure OpenAI',
    ollama: 'Ollama',
    google: 'Google AI',
    other: '其他',
  }
  return map[t] || t || '未知'
}

const formatTime = (s?: string): string => {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
}

const maskApiKey = (key: string): string => {
  if (!key || key.length < 8) return key || '-'
  return key.slice(0, 4) + '****' + key.slice(-4)
}

const providerHealth = (row: LlmProvider): ProviderHealth => (
  providerHealthMap[row.id] || { status: 'unknown', detail: '尚未在本次会话中测试连接', testedAt: '' }
)

const healthTagType = (status: ConnectionStatus): 'success' | 'danger' | 'info' => {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  return 'info'
}

const healthLabel = (status: ConnectionStatus): string => {
  if (status === 'success') return '连接正常'
  if (status === 'failed') return '连接失败'
  return '未测试'
}

const inferModelCapabilities = (modelName: string): string[] => {
  const name = modelName.toLowerCase()
  const tags = ['chat']
  if (name.includes('embed')) tags.push('embedding')
  if (name.includes('vision') || name.includes('4o') || name.includes('vl')) tags.push('vision')
  if (name.includes('rerank')) tags.push('rerank')
  if (name.includes('tool') || name.includes('function') || name.includes('gpt') || name.includes('claude')) tags.push('tool-call')
  return Array.from(new Set(tags))
}

const formatNumber = (value: number): string => Number(value || 0).toLocaleString()

const resetProviderFilters = () => {
  providerSearch.value = ''
  providerTypeFilter.value = ''
  providerStatusFilter.value = ''
  providerHealthFilter.value = ''
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
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="page-kicker">AI Platform Console</p>
          <h2 class="page-title">{{ pageTitle }}</h2>
          <p class="page-description">{{ pageDescription }}</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" @click="currentSection === 'providers' ? loadProviders() : loadModels()">刷新</el-button>
          <el-button v-if="currentSection === 'providers'" type="primary" :icon="Plus" @click="openCreateProvider">新增 Provider</el-button>
          <el-button v-else type="primary" :icon="Plus" @click="openCreateModel">新增 Model</el-button>
        </div>
      </div>

      <el-tabs v-if="!isSingleSection" v-model="activeTab" class="console-tabs" @tab-change="onTabChange">
        <el-tab-pane label="Provider" name="providers" />
        <el-tab-pane label="Model" name="models" />
      </el-tabs>

      <div class="workspace-surface__divider"></div>

      <template v-if="currentSection === 'providers'">
         <div class="workspace-surface__toolbar">
          <div class="workspace-surface__filters">
            <el-input v-model="providerSearch" :prefix-icon="Search" clearable placeholder="搜索名称 / Base URL" style="width: 200px" />
            <el-select v-model="providerTypeFilter" clearable placeholder="类型" style="width: 140px">
              <el-option v-for="type in providerTypeOptions" :key="type" :value="type" :label="providerTypeLabel(type)" />
            </el-select>
            <el-select v-model="providerStatusFilter" clearable placeholder="状态" style="width: 110px">
              <el-option value="enabled" label="启用" />
              <el-option value="disabled" label="禁用" />
            </el-select>
            <el-select v-model="providerHealthFilter" clearable placeholder="连接状态" style="width: 140px">
              <el-option value="success" label="连接正常" />
              <el-option value="failed" label="连接失败" />
              <el-option value="unknown" label="未测试" />
            </el-select>
          </div>
          <div class="workspace-surface__actions">
            <el-button @click="resetProviderFilters">重置</el-button>
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
              <template #default="{ row }: { row: LlmProvider }">
                <div class="entity-cell">
                  <div class="entity-icon"><Setting /></div>
                  <div>
                    <strong>{{ row.name }}</strong>
                    <span>ID {{ row.id }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="连接状态" width="140">
              <template #default="{ row }: { row: LlmProvider }">
                <el-tooltip :content="`${providerHealth(row).detail}${providerHealth(row).testedAt ? ' · ' + formatTime(providerHealth(row).testedAt) : ''}`" placement="top">
                  <el-tag :type="healthTagType(providerHealth(row).status)" effect="light">
                    {{ healthLabel(providerHealth(row).status) }}
                  </el-tag>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="Base URL" min-width="260">
              <template #default="{ row }: { row: LlmProvider }">
                <el-tooltip :content="row.base_url" placement="top">
                  <span class="url-text">{{ row.base_url }}</span>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="API Key" width="170">
              <template #default="{ row }: { row: LlmProvider }">
                <span class="secret-text">{{ maskApiKey(row.api_key_masked) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="140">
              <template #default="{ row }: { row: LlmProvider }">
                <el-tag type="info" effect="plain">{{ providerTypeLabel(row.provider_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="启用状态" width="110">
              <template #default="{ row }: { row: LlmProvider }">
                <el-tag :type="row.is_enabled ? 'success' : 'info'" effect="light">
                  {{ row.is_enabled ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }: { row: LlmProvider }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }: { row: LlmProvider }">
                <div class="row-actions">
                  <el-button size="small" :icon="Connection" :loading="testingId === row.id" @click="handleTestConnection(row)">测试</el-button>
                  <el-button size="small" :icon="Edit" @click="openEditProvider(row)">编辑</el-button>
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
              <template #default="{ row }: { row: LlmModel }">
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
              <template #default="{ row }: { row: LlmModel }">
                {{ row.provider_name || `Provider ${row.provider_id}` }}
              </template>
            </el-table-column>
            <el-table-column label="能力标签" min-width="220">
              <template #default="{ row }: { row: LlmModel }">
                <div class="capability-tags">
                  <el-tag v-for="tag in inferModelCapabilities(row.model_name)" :key="tag" size="small" effect="plain">
                    {{ tag }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="参数摘要" min-width="180">
              <template #default="{ row }: { row: LlmModel }">
                <div class="metric-stack">
                  <span>Temp {{ row.temperature.toFixed(2) }}</span>
                  <span>Top P {{ row.top_p.toFixed(2) }}</span>
                  <span>{{ formatNumber(row.max_tokens) }} tok (输出)</span>
                  <span v-if="row.context_window_tokens > 0">{{ formatNumber(row.context_window_tokens) }} tok (窗口)</span>
                  <span v-else style="color: var(--text-faint);">窗口未配置</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="并发 / 超时" width="150">
              <template #default="{ row }: { row: LlmModel }">
                <div class="metric-stack">
                  <span>{{ row.max_concurrency }} 并发</span>
                  <span>{{ row.timeout_seconds }}s 超时</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="默认 / 状态" width="140">
              <template #default="{ row }: { row: LlmModel }">
                <div class="status-stack">
                  <el-tag v-if="row.is_default" type="warning" effect="light">默认</el-tag>
                  <el-tag v-else type="info" effect="plain">非默认</el-tag>
                  <el-tag :type="row.is_enabled ? 'success' : 'info'" effect="light">
                    {{ row.is_enabled ? '启用' : '禁用' }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="130" fixed="right">
              <template #default="{ row }: { row: LlmModel }">
                <div class="row-actions">
                  <el-button size="small" :icon="Edit" @click="openEditModel(row)">编辑</el-button>
                  <el-dropdown trigger="click">
                    <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
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

    <el-drawer
      v-model="providerDrawerVisible"
      :title="providerDrawerTitle"
      size="520px"
      :close-on-click-modal="false"
      class="config-drawer"
    >
      <el-form :model="providerForm" label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="providerForm.name" placeholder="例如：OpenAI" />
        </el-form-item>
        <el-form-item label="Provider 类型" required>
          <el-select v-model="providerForm.provider_type" style="width: 100%">
            <el-option value="openai" label="OpenAI" />
            <el-option value="anthropic" label="Anthropic" />
            <el-option value="azure" label="Azure OpenAI" />
            <el-option value="ollama" label="Ollama" />
            <el-option value="google" label="Google AI" />
            <el-option value="other" label="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="Base URL" required>
          <el-input v-model="providerForm.base_url" placeholder="例如：https://api.openai.com" />
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
            placeholder='可选，JSON 格式，如 {"X-Custom-Header": "value"}'
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
      :close-on-click-modal="false"
      class="config-drawer"
    >
      <el-form :model="modelForm" label-position="top">
        <el-form-item label="所属 Provider" required>
          <el-select v-model="modelForm.provider_id" style="width: 100%" placeholder="选择 Provider">
            <el-option v-for="p in providerList" :key="p.id" :value="p.id" :label="p.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型名称" required>
          <el-input v-model="modelForm.model_name" placeholder="例如：gpt-4o" />
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="modelForm.display_name" placeholder="例如：GPT-4o（推荐）" />
        </el-form-item>
        <el-form-item label="Temperature">
          <div class="slider-with-value">
            <el-slider v-model="modelForm.temperature" :min="0" :max="2" :step="0.01" />
            <span class="slider-value">{{ modelForm.temperature.toFixed(2) }}</span>
          </div>
        </el-form-item>
        <el-form-item label="Top P">
          <div class="slider-with-value">
            <el-slider v-model="modelForm.top_p" :min="0" :max="1" :step="0.01" />
            <span class="slider-value">{{ modelForm.top_p.toFixed(2) }}</span>
          </div>
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="Max Tokens（输出上限）">
            <el-input-number v-model="modelForm.max_tokens" :min="1" :max="1000000" :step="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="上下文窗口（总）">
            <el-input-number v-model="modelForm.context_window_tokens" :min="0" :max="10000000" :step="1024" controls-position="right" />
            <div style="font-size: 11px; color: var(--text-faint); margin-top: 4px;">0 = 未知。请根据模型文档填写总上下文窗口（输入 + 输出）。</div>
          </el-form-item>
          <el-form-item label="最大并发">
            <el-input-number v-model="modelForm.max_concurrency" :min="1" :max="100" :step="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="超时（秒）">
            <el-input-number v-model="modelForm.timeout_seconds" :min="1" :max="600" :step="1" controls-position="right" />
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

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px;
  margin-bottom: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--admin-console-header-bg);
}

.page-kicker {
  margin: 0 0 6px;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  color: var(--el-color-primary);
  text-transform: uppercase;
  letter-spacing: .08em;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.25;
}

.page-description {
  max-width: 720px;
  margin: 8px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.page-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.console-tabs :deep(.el-tabs__header) {
  padding: 0 16px;
  margin-bottom: 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.console-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.console-tabs :deep(.el-tabs__item) {
  height: 46px;
  font-weight: 600;
}

.panel-card {
  padding: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.panel-head h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
}

.panel-head p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
}

.filter-bar {
  display: grid;
  grid-template-columns: minmax(220px, 1.4fr) repeat(3, minmax(140px, .7fr)) auto;
  gap: 10px;
  padding: 12px;
  margin-bottom: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-extra-light);
}

.model-filter-bar {
  grid-template-columns: minmax(240px, 1.4fr) repeat(3, minmax(150px, .7fr)) auto;
}

.console-table {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.console-table :deep(.el-table__header th) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
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
  width: 34px;
  height: 34px;
  color: var(--el-color-primary);
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
  flex: 0 0 auto;
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
.capability-tags,
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

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.slider-with-value {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.slider-with-value :deep(.el-slider) {
  flex: 1;
}

.slider-value {
  min-width: 44px;
  text-align: right;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  color: var(--el-text-color-secondary);
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

@media (max-width: 1100px) {
  .filter-bar,
  .model-filter-bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .page-header,
  .panel-head {
    flex-direction: column;
  }

  .page-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .filter-bar,
  .model-filter-bar,
  .form-grid,
  .switch-row {
    grid-template-columns: 1fr;
  }
}
</style>
