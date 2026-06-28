<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheck, Document, Refresh, Search, Switch, WarningFilled } from '@element-plus/icons-vue'
import { createAgentSkill, listAgentSkills, previewAgentSkill, updateAgentSkillStatus } from '@/api/agentSkill'
import { DataTableCard, EmptyGuide, FilterToolbar, PageHeader, StatusTag } from '@/components/admin-console'
import AgentSkillCanvasEditor from '@/components/agent-skill/AgentSkillCanvasEditor.vue'
import type {
  AgentSkillCanvasFlow,
  AgentSkillInfo,
  AgentSkillNode,
  AgentSkillNodeType,
  AgentSkillValidation,
  CreateAgentSkillPayload,
} from '@/types/agentSkill'

const NODE_TYPES: { type: AgentSkillNodeType; label: string; description: string; placeholder: string }[] = [
  { type: 'trigger', label: '触发场景', description: '定义何时适合使用这个 Agent Skill', placeholder: '例如：当 HR 要求比较候选人与岗位 JD 的匹配度时使用。' },
  { type: 'context', label: '上下文', description: '列出助手应读取或引用的业务信息', placeholder: '例如：候选人简历、岗位职责、任职要求、当前投递状态。' },
  { type: 'instruction', label: '执行指令', description: '说明助手应该如何推理和组织回答', placeholder: '例如：先提取硬性条件，再比较项目经历，最后给出风险提示。' },
  { type: 'condition', label: '条件分支', description: '描述不同输入情况下的处理规则', placeholder: '例如：如果简历缺少关键字段，需要明确说明无法判断。' },
  { type: 'output', label: '输出格式', description: '规定回答结构、字段和语气', placeholder: '例如：输出匹配结论、证据、疑点、下一步建议四段。' },
  { type: 'constraint', label: '约束边界', description: '声明安全、合规、不可做事项', placeholder: '例如：不得编造候选人经历，不得输出歧视性判断。' },
]

const NODE_TYPE_LABEL = NODE_TYPES.reduce<Record<AgentSkillNodeType, string>>((acc, item) => {
  acc[item.type] = item.label
  return acc
}, {} as Record<AgentSkillNodeType, string>)

const list = ref<AgentSkillInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const saving = ref(false)
const previewLoading = ref(false)
const keyword = ref('')
const statusFilter = ref('')
const previewMarkdown = ref('')
const validation = ref<AgentSkillValidation>({ valid: false, errors: [], warnings: [] })
const selectedNodeId = ref('')
const activeTab = ref<'builder' | 'list'>('builder')
const previewDrawerVisible = ref(false)

const form = reactive({
  name: '',
  display_name: '',
  description: '',
  category: 'recruiting',
  version: '1.0.0',
  is_enabled: true,
})

const flow = ref<AgentSkillCanvasFlow>(createDefaultFlow())

function createCanvasNode(type: AgentSkillNodeType, index: number) {
  const config = NODE_TYPES.find((item) => item.type === type)!
  return {
    id: `${type}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    type,
    position: {
      x: 80 + (index % 2) * 300,
      y: 80 + Math.floor(index / 2) * 150,
    },
    data: {
      title: config.label,
      content: '',
    },
  }
}

function createDefaultFlow(): AgentSkillCanvasFlow {
  const nodes = (['trigger', 'context', 'instruction', 'output', 'constraint'] as AgentSkillNodeType[])
    .map((type, index) => createCanvasNode(type, index))
  return {
    format: 'canvas.v1',
    version: '1.0.0',
    type: 'agent-skill',
    nodes,
    edges: nodes.slice(0, -1).map((node, index) => ({
      id: `edge-${node.id}-${nodes[index + 1].id}`,
      source: node.id,
      target: nodes[index + 1].id,
    })),
    viewport: { x: 0, y: 0, zoom: 1 },
  }
}

const loadList = async () => {
  loading.value = true
  try {
    const data = await listAgentSkills({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      enabled_only: statusFilter.value === 'enabled' ? true : undefined,
    })
    const rawList = data.list || []
    list.value = statusFilter.value === 'disabled'
      ? rawList.filter((item) => !item.is_enabled)
      : rawList
    total.value = data.total || list.value.length
  } finally {
    loading.value = false
  }
}

const resetBuilder = () => {
  form.name = ''
  form.display_name = ''
  form.description = ''
  form.category = 'recruiting'
  form.version = '1.0.0'
  form.is_enabled = true
  flow.value = createDefaultFlow()
  selectedNodeId.value = flow.value.nodes[0]?.id || ''
  previewMarkdown.value = ''
  validation.value = { valid: false, errors: [], warnings: [] }
}

const canvasNodes = computed<AgentSkillNode[]>(() => flow.value.nodes.map((node, index) => ({
  id: node.id,
  type: node.type,
  title: node.data.title,
  content: node.data.content,
  order: index + 1,
})))

const selectedNode = computed(() => (
  flow.value.nodes.find((node) => node.id === selectedNodeId.value)
  || null
))

const buildLocalValidation = (): AgentSkillValidation => {
  const errors: string[] = []
  const warnings: string[] = []
  if (!form.name.trim()) errors.push('请填写唯一标识')
  if (!/^[a-z][a-z0-9_-]{1,127}$/.test(form.name.trim())) errors.push('唯一标识只能使用小写字母开头，包含小写字母、数字、下划线或短横线，长度 2-128')
  if (!form.display_name.trim()) errors.push('请填写显示名称')
  if (!form.description.trim()) errors.push('请填写技能描述，描述会写入 SKILL.md frontmatter')
  if (!canvasNodes.value.some((node) => node.type === 'trigger' && node.content.trim())) errors.push('至少需要一个有内容的触发场景节点')
  if (!canvasNodes.value.some((node) => node.type === 'instruction' && node.content.trim())) errors.push('至少需要一个有内容的执行指令节点')
  if (!canvasNodes.value.some((node) => node.type === 'output' && node.content.trim())) errors.push('至少需要一个有内容的输出格式节点')
  if (!canvasNodes.value.some((node) => node.type === 'constraint' && node.content.trim())) errors.push('至少需要一个有内容的约束边界节点')
  if (!flow.value.nodes.length) errors.push('画布至少需要一个节点')
  return { valid: errors.length === 0, errors, warnings }
}

const localMarkdown = computed(() => {
  const lines = [
    `# ${form.display_name.trim() || '未命名 Agent Skill'}`,
    '',
    '## Metadata',
    `- name: ${form.name.trim() || 'agent.skill.name'}`,
    `- version: ${form.version.trim() || '1.0.0'}`,
    `- category: ${form.category.trim() || 'recruiting'}`,
  ]
  if (form.description.trim()) {
    lines.push(`- description: ${form.description.trim()}`)
  }
  canvasNodes.value.forEach((node, index) => {
    lines.push('', `## ${index + 1}. ${node.title || NODE_TYPE_LABEL[node.type]}`, '', `> node_type: ${node.type}`, '')
    lines.push(node.content.trim() || `（待补充${NODE_TYPE_LABEL[node.type]}）`)
  })
  return lines.join('\n')
})

const flowJson = computed(() => JSON.stringify({
  ...flow.value,
  format: 'canvas.v1',
  version: flow.value.version || '1.0.0',
  type: 'agent-skill',
  nodes: flow.value.nodes.map((node) => ({
    id: node.id,
    type: node.type,
    position: node.position,
    data: {
      title: node.data.title?.trim() || NODE_TYPE_LABEL[node.type],
      content: node.data.content?.trim() || '',
    },
  })),
  edges: flow.value.edges,
  viewport: flow.value.viewport || { x: 0, y: 0, zoom: 1 },
}))

const payload = (): CreateAgentSkillPayload => ({
  name: form.name.trim(),
  display_name: form.display_name.trim(),
  description: form.description.trim(),
  category: form.category.trim(),
  version: form.version.trim(),
  is_enabled: form.is_enabled,
  is_enabled_set: true,
  is_manual_invocable: true,
  is_manual_invocable_set: true,
  activate: true,
  flow_json: flowJson.value,
  nodes: canvasNodes.value.map((node, index) => ({
    id: node.id,
    type: node.type,
    title: node.title.trim() || NODE_TYPE_LABEL[node.type],
    content: node.content.trim(),
    order: index + 1,
  })),
})

const refreshPreview = async () => {
  const localValidation = buildLocalValidation()
  validation.value = localValidation
  previewMarkdown.value = localMarkdown.value
  if (!localValidation.valid) return
  previewLoading.value = true
  try {
    const data = await previewAgentSkill(payload())
    previewMarkdown.value = data.skill_md || localMarkdown.value
    validation.value = data.validation || { ...localValidation, valid: localValidation.errors.length === 0 }
  } finally {
    previewLoading.value = false
  }
}

const openPreviewDrawer = async () => {
  previewDrawerVisible.value = true
  await refreshPreview()
}

const saveSkill = async () => {
  await refreshPreview()
  if (!validation.value.valid) {
    ElMessage.warning('请先修复校验错误')
    return
  }
  saving.value = true
  try {
    await createAgentSkill(payload())
    ElMessage.success('Agent Skill 已创建')
    resetBuilder()
    await loadList()
  } finally {
    saving.value = false
  }
}

const toggleStatus = async (row: AgentSkillInfo) => {
  await updateAgentSkillStatus(row.id, { is_enabled: !row.is_enabled })
  ElMessage.success(row.is_enabled ? 'Agent Skill 已停用' : 'Agent Skill 已启用')
  await loadList()
}

const formatTime = (value?: string) => {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

watch([() => form.name, () => form.display_name, () => form.description, () => form.category, () => form.version, flow], () => {
  validation.value = buildLocalValidation()
  previewMarkdown.value = localMarkdown.value
}, { deep: true, immediate: true })

onMounted(() => {
  selectedNodeId.value = flow.value.nodes[0]?.id || ''
  loadList()
})
</script>

<template>
  <section class="agent-skill-page">
    <PageHeader
      title="Agent Skill 管理"
      description="用流程节点编排生成数据库版 SKILL.md，供 AI 助手手动选择使用。"
    >
      <template #primary>
        <el-button
          v-if="activeTab === 'builder'"
          type="primary"
          :icon="CircleCheck"
          :loading="saving"
          @click="saveSkill"
        >
          创建 Agent Skill
        </el-button>
      </template>
      <template #secondary>
        <el-button v-if="activeTab === 'builder'" plain @click="resetBuilder">清空重置</el-button>
        <el-button v-else :icon="Refresh" @click="loadList">刷新列表</el-button>
      </template>
    </PageHeader>

    <el-tabs v-model="activeTab" class="agent-skill-tabs">
      <el-tab-pane label="画布编排" name="builder" />
      <el-tab-pane label="列表管理" name="list" />
    </el-tabs>

    <section v-show="activeTab === 'builder'" class="builder-shell">
      <main class="canvas-workbench">
        <div class="builder-title-row">
          <div>
            <h2>画布式流程编排</h2>
            <p>按 canvas.v1 flow 生成 SKILL.md，本页面只编排提示规则，不执行工作流。</p>
          </div>
          <el-button :icon="Document" :loading="previewLoading" @click="openPreviewDrawer">预览 SKILL.md</el-button>
        </div>

        <div class="meta-grid">
          <el-input v-model="form.name" placeholder="唯一标识，例如 resume_matching" clearable>
            <template #prepend>标识</template>
          </el-input>
          <el-input v-model="form.display_name" placeholder="显示名称，例如 简历匹配分析" clearable>
            <template #prepend>名称</template>
          </el-input>
          <el-input v-model="form.category" placeholder="分类" clearable>
            <template #prepend>分类</template>
          </el-input>
          <el-input v-model="form.version" placeholder="版本" clearable>
            <template #prepend>版本</template>
          </el-input>
        </div>
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="2"
          placeholder="简要描述这个 Agent Skill 的招聘业务用途"
        />

        <div class="canvas-frame">
          <AgentSkillCanvasEditor
            v-model="flow"
            v-model:selected-node-id="selectedNodeId"
            class="canvas-editor"
            :show-preview="false"
          />
        </div>
      </main>

    </section>

    <section v-show="activeTab === 'list'" class="list-shell">
      <FilterToolbar>
        <el-input
          v-model="keyword"
          class="filter-input"
          placeholder="搜索名称/标识"
          clearable
          :prefix-icon="Search"
          @keyup.enter="loadList"
        />
        <el-select v-model="statusFilter" class="filter-select" placeholder="状态" clearable @change="loadList">
          <el-option label="已启用" value="enabled" />
          <el-option label="已停用" value="disabled" />
        </el-select>
        <template #actions>
          <el-button :icon="Search" type="primary" plain @click="loadList">查询</el-button>
        </template>
      </FilterToolbar>

      <DataTableCard title="Agent Skill 列表" :result-count="total">
        <el-table v-loading="loading" :data="list" row-key="id">
          <el-table-column prop="display_name" label="名称" min-width="180">
            <template #default="{ row }">
              <div class="skill-name">{{ row.display_name || row.name }}</div>
              <div class="skill-key">{{ row.name }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" min-width="240" show-overflow-tooltip />
          <el-table-column label="当前版本" width="110">
            <template #default="{ row }">
              <el-tag v-if="row.current_version_id" size="small" type="success">#{{ row.current_version_id }}</el-tag>
              <el-tag v-else size="small" type="info">未发布</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <StatusTag :status="row.is_enabled ? 'enabled' : 'disabled'">
                {{ row.is_enabled ? '已启用' : '已停用' }}
              </StatusTag>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="180">
            <template #default="{ row }">{{ formatTime(row.updated_at || row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button :icon="Switch" link type="primary" @click="toggleStatus(row)">
                {{ row.is_enabled ? '停用' : '启用' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <EmptyGuide v-if="!loading && list.length === 0" title="暂无 Agent Skill" description="从画布编排中创建第一个数据库版 SKILL.md。" />
      </DataTableCard>
    </section>

    <el-drawer
      v-model="previewDrawerVisible"
      title="SKILL.md 只读预览"
      size="min(560px, 92vw)"
      direction="rtl"
      class="agent-skill-preview-drawer"
      destroy-on-close
    >
      <div class="preview-drawer-body" v-loading="previewLoading">
        <div class="preview-panel__head">
          <div>
            <div class="section-title">生成结果</div>
            <p>由画布 flow 生成，不提供原生 Markdown 编辑。</p>
          </div>
          <el-button :icon="Document" :loading="previewLoading" @click="refreshPreview">重新预览</el-button>
        </div>

        <div v-if="selectedNode" class="selected-node-summary">
          <span>{{ NODE_TYPE_LABEL[selectedNode.type] }}</span>
          <strong>{{ selectedNode.data.title || NODE_TYPE_LABEL[selectedNode.type] }}</strong>
        </div>

        <div class="validation-box" :class="{ 'validation-box--ok': validation.valid }">
          <el-icon><CircleCheck v-if="validation.valid" /><WarningFilled v-else /></el-icon>
          <span>{{ validation.valid ? '校验通过' : '需要补充' }}</span>
        </div>
        <ul v-if="validation.errors.length || validation.warnings.length" class="validation-list">
          <li v-for="item in validation.errors" :key="`e-${item}`" class="validation-list__error">{{ item }}</li>
          <li v-for="item in validation.warnings" :key="`w-${item}`">{{ item }}</li>
        </ul>

        <pre class="markdown-preview">{{ previewMarkdown }}</pre>
      </div>
    </el-drawer>
  </section>
</template>

<style scoped>
.agent-skill-page {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
  overflow-x: hidden;
  overflow-y: auto;
  padding-bottom: 24px;
}

.agent-skill-tabs {
  flex-shrink: 0;
}

.agent-skill-tabs :deep(.el-tabs__header) {
  margin: 0;
}

.agent-skill-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.agent-skill-tabs :deep(.el-tabs__item) {
  height: 36px;
  padding: 0 18px;
  color: var(--text-muted);
  font-weight: 700;
}

.agent-skill-tabs :deep(.el-tabs__item.is-active) {
  color: var(--brand);
}

.builder-shell {
  min-height: 760px;
}

.list-shell {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.canvas-workbench {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: var(--admin-console-card-shadow);
}

.canvas-workbench {
  min-width: 0;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
}

.skill-name {
  display: block;
  font-weight: 700;
}

.skill-key,
.preview-drawer-body p,
.builder-title-row p {
  margin: 4px 0 0;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.builder-title-row,
.preview-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.builder-title-row h2 {
  margin: 0;
  font-size: 18px;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.canvas-frame {
  min-height: 560px;
  height: clamp(560px, 58vh, 760px);
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
}

.canvas-editor {
  width: 100%;
  height: 100%;
  min-height: 560px;
}

.preview-drawer-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 100%;
}

.selected-node-summary {
  display: grid;
  gap: 4px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
  background: var(--surface-muted);
}

.selected-node-summary span {
  color: var(--text-faint);
  font-size: 12px;
  font-weight: 700;
}

.selected-node-summary strong {
  overflow: hidden;
  color: var(--text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.validation-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border-radius: 8px;
  background: color-mix(in srgb, #d97706 12%, var(--surface));
  color: #b45309;
  font-weight: 700;
}

.validation-box--ok {
  background: color-mix(in srgb, #16a34a 12%, var(--surface));
  color: #15803d;
}

.validation-list {
  margin: 0;
  padding-left: 18px;
  color: var(--text-secondary);
  font-size: 13px;
}

.validation-list__error {
  color: #dc2626;
}

.markdown-preview {
  min-height: 420px;
  max-height: calc(100vh - 300px);
  margin: 0;
  padding: 14px;
  overflow: auto;
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-primary);
  white-space: pre-wrap;
  line-height: 1.55;
  font-size: 12px;
}

.filter-input {
  width: 260px;
}

.filter-select {
  width: 140px;
}

@media (max-width: 860px) {
  .builder-shell,
  .meta-grid {
    grid-template-columns: 1fr;
  }

  .builder-shell {
    min-height: 0;
  }

  .canvas-frame {
    height: 560px;
  }
}
</style>
