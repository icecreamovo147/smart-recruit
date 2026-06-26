<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Refresh, Connection } from '@element-plus/icons-vue'
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

// ====== Provider State ======

const providerList = ref<LlmProvider[]>([])
const providerTotal = ref(0)
const providerPage = ref(1)
const providerPageSize = ref(20)
const providerLoading = ref(false)
const providerError = ref('')

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

// ---- Provider Dialog ----

const providerDialogVisible = ref(false)
const providerDialogTitle = ref('')
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
  providerDialogTitle.value = '新增 Provider'
  resetProviderForm()
  providerDialogVisible.value = true
}

const openEditProvider = (row: LlmProvider) => {
  isEditingProvider.value = true
  editingProviderId.value = row.id
  providerDialogTitle.value = '编辑 Provider'
  providerForm.name = row.name
  providerForm.base_url = row.base_url
  providerForm.api_key = ''      // empty = do not change
  providerForm.provider_type = row.provider_type
  providerForm.extra_headers_json = row.extra_headers_json || ''
  providerForm.is_enabled = row.is_enabled
  providerDialogVisible.value = true
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
    providerDialogVisible.value = false
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
    if (result.success) {
      ElMessage.success('连接测试成功')
    } else {
      ElMessage.error(`连接测试失败：${result.detail || '未知错误'}`)
    }
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '连接测试失败')
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

// Flat list sorted by provider for grouped display
const groupedModels = computed(() => {
  const groups: Record<string, { provider_id: number; provider_name: string; models: LlmModel[] }> = {}
  for (const m of modelList.value) {
    const key = m.provider_name || `Provider ${m.provider_id}`
    if (!groups[key]) {
      groups[key] = { provider_id: m.provider_id, provider_name: key, models: [] }
    }
    groups[key].models.push(m)
  }
  return Object.values(groups).sort((a, b) => a.provider_name.localeCompare(b.provider_name, 'zh'))
})

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

// ---- Model Dialog ----

const modelDialogVisible = ref(false)
const modelDialogTitle = ref('')
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
  modelForm.max_concurrency = 1
  modelForm.timeout_seconds = 60
  modelForm.is_default = false
  modelForm.is_enabled = true
}

const openCreateModel = () => {
  isEditingModel.value = false
  editingModelId.value = 0
  modelDialogTitle.value = '新增 Model'
  resetModelForm()
  modelDialogVisible.value = true
}

const openEditModel = (row: LlmModel) => {
  isEditingModel.value = true
  editingModelId.value = row.id
  modelDialogTitle.value = '编辑 Model'
  modelForm.provider_id = row.provider_id
  modelForm.model_name = row.model_name
  modelForm.display_name = row.display_name
  modelForm.temperature = row.temperature
  modelForm.top_p = row.top_p
  modelForm.max_tokens = row.max_tokens
  modelForm.max_concurrency = row.max_concurrency
  modelForm.timeout_seconds = row.timeout_seconds
  modelForm.is_default = row.is_default
  modelForm.is_enabled = row.is_enabled
  modelDialogVisible.value = true
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
        max_concurrency: modelForm.max_concurrency,
        timeout_seconds: modelForm.timeout_seconds,
        is_default: modelForm.is_default,
      }
      await createModel(payload)
      ElMessage.success('Model 已创建')
    }
    modelDialogVisible.value = false
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

const activeTab = ref('providers')

const onTabChange = (tab: string) => {
  if (tab === 'providers') {
    loadProviders()
  } else if (tab === 'models') {
    // Ensure provider list is loaded for the dropdown; then load models
    if (providerList.value.length === 0) {
      loadProviders().then(() => loadModels())
    } else {
      loadModels()
    }
  }
}

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

// ====== Init ======

onMounted(() => {
  loadProviders()
})
</script>

<template>
  <div class="llm-config-view">
    <h2 class="page-title">模型配置</h2>

    <el-tabs v-model="activeTab" @tab-change="onTabChange">
      <!-- ── Provider 管理 ──────────────────────────────────────────── -->
      <el-tab-pane label="Provider 管理" name="providers">
        <div class="toolbar">
          <el-button type="primary" :icon="Plus" @click="openCreateProvider">新增 Provider</el-button>
        </div>

        <el-table
          v-loading="providerLoading"
          :data="providerList"
          stripe
          border
          style="width: 100%"
          :empty-text="providerError || '暂无数据'"
        >
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="base_url" label="Base URL" min-width="240" show-overflow-tooltip />
          <el-table-column label="API Key" width="180">
            <template #default="{ row }: { row: LlmProvider }">
              <span style="font-family: monospace">{{ maskApiKey(row.api_key_masked) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="类型" width="140">
            <template #default="{ row }: { row: LlmProvider }">
              {{ providerTypeLabel(row.provider_type) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }: { row: LlmProvider }">
              <el-tag :type="row.is_enabled ? 'success' : 'info'" size="small">
                {{ row.is_enabled ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="170">
            <template #default="{ row }: { row: LlmProvider }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }: { row: LlmProvider }">
              <el-button size="small" :icon="Connection" :loading="testingId === row.id" @click="handleTestConnection(row)">
                测试
              </el-button>
              <el-button size="small" :icon="Edit" @click="openEditProvider(row)">
                编辑
              </el-button>
              <el-button size="small" type="danger" :icon="Delete" @click="handleDeleteProvider(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-wrap">
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
      </el-tab-pane>

      <!-- ── Model 管理 ────────────────────────────────────────────── -->
      <el-tab-pane label="Model 管理" name="models">
        <div class="toolbar">
          <div class="toolbar-left">
            <el-button type="primary" :icon="Plus" @click="openCreateModel">新增 Model</el-button>
            <el-select
              v-model="modelProviderFilter"
              placeholder="全部 Provider"
              clearable
              style="width: 200px"
              @change="() => { modelPage = 1; loadModels() }"
            >
              <el-option :value="0" label="全部 Provider" />
              <el-option
                v-for="p in providerList"
                :key="p.id"
                :value="p.id"
                :label="p.name"
              />
            </el-select>
          </div>
          <el-button :icon="Refresh" @click="loadModels">刷新</el-button>
        </div>

        <el-table
          v-loading="modelLoading"
          :data="modelList"
          stripe
          border
          style="width: 100%"
          :empty-text="modelError || '暂无数据'"
        >
          <el-table-column prop="model_name" label="模型名称" min-width="160" />
          <el-table-column prop="display_name" label="显示名称" min-width="140" />
          <el-table-column prop="provider_name" label="Provider" width="140" />
          <el-table-column label="Temperature" width="120">
            <template #default="{ row }: { row: LlmModel }">
              {{ row.temperature.toFixed(2) }}
            </template>
          </el-table-column>
          <el-table-column label="Top P" width="100">
            <template #default="{ row }: { row: LlmModel }">
              {{ row.top_p.toFixed(2) }}
            </template>
          </el-table-column>
          <el-table-column label="Max Tokens" width="110">
            <template #default="{ row }: { row: LlmModel }">
              {{ row.max_tokens.toLocaleString() }}
            </template>
          </el-table-column>
          <el-table-column label="默认" width="80">
            <template #default="{ row }: { row: LlmModel }">
              <el-tag v-if="row.is_default" type="warning" size="small">默认</el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }: { row: LlmModel }">
              <el-tag :type="row.is_enabled ? 'success' : 'info'" size="small">
                {{ row.is_enabled ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="170">
            <template #default="{ row }: { row: LlmModel }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }: { row: LlmModel }">
              <el-button size="small" :icon="Edit" @click="openEditModel(row)">
                编辑
              </el-button>
              <el-button size="small" type="danger" :icon="Delete" @click="handleDeleteModel(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-wrap">
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
      </el-tab-pane>
    </el-tabs>

    <!-- ── Provider Dialog ────────────────────────────────────────── -->
    <el-dialog
      v-model="providerDialogVisible"
      :title="providerDialogTitle"
      width="540px"
      :close-on-click-modal="false"
    >
      <el-form :model="providerForm" label-width="110px">
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
        <el-form-item
          :label="isEditingProvider ? 'API Key（留空不修改）' : 'API Key'"
          :required="!isEditingProvider"
        >
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
            :rows="2"
            placeholder='可选，JSON 格式，如 {"X-Custom-Header": "value"}'
          />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="providerForm.is_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="providerDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="providerSaving" @click="saveProvider">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- ── Model Dialog ───────────────────────────────────────────── -->
    <el-dialog
      v-model="modelDialogVisible"
      :title="modelDialogTitle"
      width="540px"
      :close-on-click-modal="false"
    >
      <el-form :model="modelForm" label-width="120px">
        <el-form-item label="所属 Provider" required>
          <el-select v-model="modelForm.provider_id" style="width: 100%" placeholder="选择 Provider">
            <el-option
              v-for="p in providerList"
              :key="p.id"
              :value="p.id"
              :label="p.name"
            />
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
            <el-slider
              v-model="modelForm.temperature"
              :min="0"
              :max="2"
              :step="0.01"
              style="flex: 1"
            />
            <span class="slider-value">{{ modelForm.temperature.toFixed(2) }}</span>
          </div>
        </el-form-item>
        <el-form-item label="Top P">
          <div class="slider-with-value">
            <el-slider
              v-model="modelForm.top_p"
              :min="0"
              :max="1"
              :step="0.01"
              style="flex: 1"
            />
            <span class="slider-value">{{ modelForm.top_p.toFixed(2) }}</span>
          </div>
        </el-form-item>
        <el-form-item label="Max Tokens">
          <el-input-number v-model="modelForm.max_tokens" :min="1" :max="1000000" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="最大并发">
          <el-input-number v-model="modelForm.max_concurrency" :min="1" :max="100" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="超时（秒）">
          <el-input-number v-model="modelForm.timeout_seconds" :min="1" :max="600" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="modelForm.is_default" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="modelForm.is_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="modelDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="modelSaving" @click="saveModel">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.llm-config-view {
  padding-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 16px;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 8px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
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

.slider-value {
  min-width: 40px;
  text-align: right;
  font-family: monospace;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
