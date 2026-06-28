<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowUp, CircleCheck, Document, Plus, Refresh, Search, Switch, WarningFilled } from '@element-plus/icons-vue'
import { createAgentSkill, listAgentSkills, previewAgentSkill, updateAgentSkillStatus } from '@/api/agentSkill'
import { DataTableCard, EmptyGuide, FilterToolbar, PageHeader, StatCard, StatusTag } from '@/components/admin-console'
import type {
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

const form = reactive({
  name: '',
  display_name: '',
  description: '',
  category: 'recruiting',
  version: '1.0.0',
  is_enabled: true,
})

const nodes = ref<AgentSkillNode[]>([
  createNode('trigger'),
  createNode('context'),
  createNode('instruction'),
  createNode('output'),
  createNode('constraint'),
])

function createNode(type: AgentSkillNodeType): AgentSkillNode {
  const config = NODE_TYPES.find((item) => item.type === type)!
  return {
    id: `${type}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    type,
    title: config.label,
    content: '',
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

const stats = computed(() => {
  const enabled = list.value.filter((item) => item.is_enabled).length
  return [
    { label: 'Agent Skill 总数', value: total.value || list.value.length },
    { label: '已启用', value: enabled },
    { label: '已停用', value: Math.max((total.value || list.value.length) - enabled, 0) },
    { label: '当前节点数', value: nodes.value.length },
  ]
})

const addNode = (type: AgentSkillNodeType) => {
  nodes.value.push(createNode(type))
}

const removeNode = (id: string) => {
  nodes.value = nodes.value.filter((node) => node.id !== id)
}

const moveNode = (index: number, direction: -1 | 1) => {
  const nextIndex = index + direction
  if (nextIndex < 0 || nextIndex >= nodes.value.length) return
  const next = [...nodes.value]
  const [item] = next.splice(index, 1)
  next.splice(nextIndex, 0, item)
  nodes.value = next
}

const resetBuilder = () => {
  form.name = ''
  form.display_name = ''
  form.description = ''
  form.category = 'recruiting'
  form.version = '1.0.0'
  form.is_enabled = true
  nodes.value = [
    createNode('trigger'),
    createNode('context'),
    createNode('instruction'),
    createNode('output'),
    createNode('constraint'),
  ]
  previewMarkdown.value = ''
  validation.value = { valid: false, errors: [], warnings: [] }
}

const buildLocalValidation = (): AgentSkillValidation => {
  const errors: string[] = []
  const warnings: string[] = []
  if (!form.name.trim()) errors.push('请填写唯一标识')
  if (!/^[a-z][a-z0-9_-]{1,127}$/.test(form.name.trim())) errors.push('唯一标识只能使用小写字母开头，包含小写字母、数字、下划线或短横线，长度 2-128')
  if (!form.display_name.trim()) errors.push('请填写显示名称')
  if (!form.description.trim()) errors.push('请填写技能描述，描述会写入 SKILL.md frontmatter')
  if (!nodes.value.some((node) => node.type === 'trigger' && node.content.trim())) errors.push('至少需要一个有内容的触发场景节点')
  if (!nodes.value.some((node) => node.type === 'instruction' && node.content.trim())) errors.push('至少需要一个有内容的执行指令节点')
  if (!nodes.value.some((node) => node.type === 'output' && node.content.trim())) warnings.push('建议补充输出格式节点，方便助手稳定回答')
  if (!nodes.value.some((node) => node.type === 'constraint' && node.content.trim())) warnings.push('建议补充约束边界节点，降低越权或编造风险')
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
  nodes.value.forEach((node, index) => {
    lines.push('', `## ${index + 1}. ${node.title || NODE_TYPE_LABEL[node.type]}`, '', `> node_type: ${node.type}`, '')
    lines.push(node.content.trim() || `（待补充${NODE_TYPE_LABEL[node.type]}）`)
  })
  return lines.join('\n')
})

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
  nodes: nodes.value.map((node, index) => ({
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

watch([() => form.name, () => form.display_name, () => form.description, () => form.category, () => form.version, nodes], () => {
  validation.value = buildLocalValidation()
  previewMarkdown.value = localMarkdown.value
}, { deep: true, immediate: true })

onMounted(loadList)
</script>

<template>
  <section class="agent-skill-page">
    <PageHeader
      title="Agent Skill 管理"
      description="用流程节点编排生成数据库版 SKILL.md，供 AI 助手手动选择使用。"
    >
      <template #primary>
        <el-button type="primary" :icon="CircleCheck" :loading="saving" @click="saveSkill">创建 Agent Skill</el-button>
      </template>
      <template #secondary>
        <el-button :icon="Refresh" @click="loadList">刷新列表</el-button>
      </template>
    </PageHeader>

    <div class="agent-skill-stats">
      <StatCard
        v-for="item in stats"
        :key="item.label"
        :title="item.label"
        :value="String(item.value)"
      />
    </div>

    <section class="builder-shell">
      <aside class="node-palette">
        <div class="section-title">节点类型</div>
        <button
          v-for="item in NODE_TYPES"
          :key="item.type"
          class="node-type"
          type="button"
          @click="addNode(item.type)"
        >
          <span class="node-type__label">{{ item.label }}</span>
          <span class="node-type__desc">{{ item.description }}</span>
        </button>
      </aside>

      <main class="node-canvas">
        <div class="builder-title-row">
          <div>
            <h2>流程化创建</h2>
            <p>按节点顺序生成 SKILL.md，本页面只编排提示规则，不执行工作流。</p>
          </div>
          <el-button plain @click="resetBuilder">清空重置</el-button>
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

        <div class="node-list">
          <article v-for="(node, index) in nodes" :key="node.id" class="node-card">
            <div class="node-card__head">
              <div class="node-card__index">{{ index + 1 }}</div>
              <el-select v-model="node.type" class="node-card__type">
                <el-option v-for="item in NODE_TYPES" :key="item.type" :label="item.label" :value="item.type" />
              </el-select>
              <el-input v-model="node.title" class="node-card__title" placeholder="节点标题" />
              <div class="node-card__actions">
                <el-button :icon="ArrowUp" circle :disabled="index === 0" @click="moveNode(index, -1)" />
                <el-button :icon="ArrowDown" circle :disabled="index === nodes.length - 1" @click="moveNode(index, 1)" />
                <el-button text type="danger" @click="removeNode(node.id)">删除</el-button>
              </div>
            </div>
            <el-input
              v-model="node.content"
              type="textarea"
              :rows="4"
              :placeholder="NODE_TYPES.find((item) => item.type === node.type)?.placeholder"
            />
          </article>
        </div>
      </main>

      <aside class="preview-panel">
        <div class="preview-panel__head">
          <div>
            <div class="section-title">SKILL.md 只读预览</div>
            <p>由左侧节点实时生成，不提供原生 Markdown 编辑。</p>
          </div>
          <el-button :icon="Document" :loading="previewLoading" @click="refreshPreview">后端预览</el-button>
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
      </aside>
    </section>

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
      <EmptyGuide v-if="!loading && list.length === 0" title="暂无 Agent Skill" description="从上方节点编排器创建第一个数据库版 SKILL.md。" />
    </DataTableCard>
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

.agent-skill-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.builder-shell {
  display: grid;
  grid-template-columns: 220px minmax(0, 1.15fr) minmax(360px, 0.85fr);
  gap: 14px;
  align-items: start;
}

.node-palette,
.node-canvas,
.preview-panel {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: var(--admin-console-card-shadow);
}

.node-palette,
.preview-panel {
  padding: 14px;
}

.node-canvas {
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

.node-type {
  width: 100%;
  margin-top: 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
  background: var(--surface-muted);
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}

.node-type:hover {
  border-color: var(--brand);
}

.node-type__label,
.skill-name {
  display: block;
  font-weight: 700;
}

.node-type__desc,
.skill-key,
.preview-panel p,
.builder-title-row p {
  margin: 4px 0 0;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.builder-title-row,
.preview-panel__head,
.node-card__head {
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

.node-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.node-card {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  background: var(--surface);
}

.node-card__head {
  margin-bottom: 10px;
}

.node-card__index {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--brand);
  color: #fff;
  font-weight: 700;
  flex-shrink: 0;
}

.node-card__type {
  width: 124px;
}

.node-card__title {
  flex: 1;
}

.node-card__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.preview-panel {
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
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
  min-height: 360px;
  max-height: 680px;
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

@media (max-width: 1280px) {
  .builder-shell {
    grid-template-columns: 200px minmax(0, 1fr);
  }

  .preview-panel {
    grid-column: 1 / -1;
    position: static;
  }
}

@media (max-width: 860px) {
  .agent-skill-stats,
  .builder-shell,
  .meta-grid {
    grid-template-columns: 1fr;
  }

  .node-card__head {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .node-card__type,
  .node-card__title {
    width: 100%;
  }
}
</style>
