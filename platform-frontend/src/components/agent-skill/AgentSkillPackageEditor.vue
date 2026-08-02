<script setup lang="ts">
import { computed, watch } from 'vue'
import { ArrowDown, ArrowUp, Delete, Plus } from '@element-plus/icons-vue'
import {
  activationPolicyForRisk,
  createNextEmptySection,
  type AgentSkillPackageEditorModel,
} from './packageEditor'

const model = defineModel<AgentSkillPackageEditorModel>({ required: true })
defineProps<{ skillNameReadonly?: boolean }>()

const agentTypeOptions = [
  { value: 'hr_recruiting_agent', label: 'HR 招聘助手' },
  { value: 'candidate_assistant', label: '候选人 AI 助手' },
  { value: 'resume_profile_extractor', label: '简历画像抽取' },
  { value: 'job_requirement_extractor', label: '岗位要求抽取' },
  { value: 'candidate_match_evaluator', label: '人岗匹配评估' },
]

const activationPolicy = computed(() => activationPolicyForRisk(model.value.risk))
const outputModeOptions = computed(() => {
  if (model.value.compositionRole === 'supporting') return [{ value: 'none', label: '无输出契约' }]
  const options = [
    { value: 'none', label: '无输出契约' },
    { value: 'advisory', label: 'Advisory（提示约束）' },
  ]
  if (['resume_profile_extractor', 'job_requirement_extractor', 'candidate_match_evaluator'].includes(model.value.agentType)) {
    options.push({ value: 'strict', label: 'Strict（结构化强校验）' })
  }
  return options
})

watch(() => model.value.compositionRole, (role) => {
  if (role === 'supporting') {
    model.value.outputMode = 'none'
    model.value.outputSchemaId = ''
    model.value.outputSchemaJson = ''
  }
})

watch(() => model.value.agentType, (agentType) => {
  if (model.value.outputMode === 'strict'
    && !['resume_profile_extractor', 'job_requirement_extractor', 'candidate_match_evaluator'].includes(agentType)) {
    model.value.outputMode = 'advisory'
    model.value.outputSchemaId = ''
    model.value.outputSchemaJson = ''
  }
})

const addSection = () => {
  if (model.value.sections.length >= 20) return
  model.value.sections.push(createNextEmptySection(model.value.sections))
}

const removeSection = (index: number) => {
  model.value.sections.splice(index, 1)
}

const moveSection = (index: number, offset: -1 | 1) => {
  const target = index + offset
  if (target < 0 || target >= model.value.sections.length) return
  const [section] = model.value.sections.splice(index, 1)
  model.value.sections.splice(target, 0, section)
}
</script>

<template>
  <div class="package-editor">
    <section class="editor-section">
      <div class="editor-section__head">
        <div>
          <h3>不可变版本 Manifest</h3>
          <p>以下字段随版本保存。注册表显示名称与描述在编辑器顶部独立维护。</p>
        </div>
      </div>

      <div class="field-grid field-grid--four">
        <label class="field">
          <span>唯一标识</span>
          <el-input
            v-model="model.skillName"
            data-testid="skill-name"
            placeholder="candidate_job_match"
            :disabled="skillNameReadonly"
          />
        </label>
        <label class="field">
          <span>显示名称</span>
          <el-input v-model="model.displayName" placeholder="候选人与岗位匹配" />
        </label>
        <label class="field">
          <span>适用 Agent</span>
          <el-select v-model="model.agentType" filterable>
            <el-option v-for="item in agentTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </label>
        <label class="field">
          <span>治理分类</span>
          <el-input v-model="model.category" placeholder="candidate_match" />
        </label>
      </div>

      <label class="field">
        <span>Skill 描述</span>
        <el-input v-model="model.description" type="textarea" :rows="2" placeholder="说明 Skill 的业务目标和适用边界" />
      </label>

      <div class="field-grid field-grid--four">
        <label class="field">
          <span>业务场景</span>
          <el-input v-model="model.scenario" placeholder="candidate_job_match" />
        </label>
        <label class="field">
          <span>优先级</span>
          <el-input-number v-model="model.priority" :min="-1000" :max="1000" controls-position="right" />
        </label>
        <label class="field">
          <span>风险等级</span>
          <el-select v-model="model.risk">
            <el-option label="低风险" value="low" />
            <el-option label="中风险" value="medium" />
            <el-option label="高风险" value="high" />
            <el-option label="关键风险" value="critical" />
          </el-select>
        </label>
        <label class="field">
          <span>激活策略（服务端派生）</span>
          <el-input :model-value="activationPolicy" data-testid="activation-policy" readonly />
        </label>
      </div>

      <div class="field-grid">
        <label class="field">
          <span>依赖能力</span>
          <el-select v-model="model.requiredCapabilities" multiple filterable allow-create default-first-option>
            <el-option v-for="item in model.requiredCapabilities" :key="item" :label="item" :value="item" />
          </el-select>
        </label>
        <label class="field">
          <span>触发关键词</span>
          <el-select v-model="model.triggerKeywords" multiple filterable allow-create default-first-option>
            <el-option v-for="item in model.triggerKeywords" :key="item" :label="item" :value="item" />
          </el-select>
        </label>
        <label class="field">
          <span>语义标签</span>
          <el-select v-model="model.semanticTags" multiple filterable allow-create default-first-option>
            <el-option v-for="item in model.semanticTags" :key="item" :label="item" :value="item" />
          </el-select>
        </label>
        <label class="field">
          <span>评估标准</span>
          <el-select v-model="model.evaluationCriteria" multiple filterable allow-create default-first-option>
            <el-option v-for="item in model.evaluationCriteria" :key="item" :label="item" :value="item" />
          </el-select>
        </label>
      </div>
    </section>

    <section class="editor-section">
      <div class="editor-section__head">
        <div>
          <h3>组合与输出契约</h3>
          <p>每次运行最多加载一个 Primary 和一个 Supporting；Supporting 不产生最终输出。</p>
        </div>
      </div>
      <div class="field-grid field-grid--three">
        <label class="field">
          <span>组合角色</span>
          <el-radio-group v-model="model.compositionRole" data-testid="composition-role">
            <el-radio-button value="primary">Primary</el-radio-button>
            <el-radio-button value="supporting">Supporting</el-radio-button>
          </el-radio-group>
        </label>
        <label class="field">
          <span>输出模式</span>
          <el-select v-model="model.outputMode" data-testid="output-mode">
            <el-option v-for="item in outputModeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </label>
        <label class="field">
          <span>Schema ID</span>
          <el-input v-model="model.outputSchemaId" :disabled="model.outputMode === 'none'" placeholder="registered.schema.id" />
        </label>
      </div>
      <label v-if="model.outputMode !== 'none'" class="field">
        <span>JSON Schema</span>
        <el-input v-model="model.outputSchemaJson" type="textarea" :rows="5" placeholder='{"type":"object"}' />
      </label>
    </section>

    <section class="editor-section">
      <div class="editor-section__head">
        <div>
          <h3>Core 指令</h3>
          <p>Core 每次激活时完整加载，服务端限制为 800 tokens；客户端不计算或提交 Token。</p>
        </div>
      </div>
      <el-input
        v-model="model.coreMarkdown"
        data-testid="core-markdown"
        type="textarea"
        :rows="12"
        placeholder="使用 Markdown 编写稳定、精简、每次都需要的核心指令。"
      />
    </section>

    <section class="editor-section">
      <div class="editor-section__head">
        <div>
          <h3>Reference Sections</h3>
          <p>Reference Section 按召回信号动态加载，顺序决定同分和预算不足时的稳定优先级。</p>
        </div>
        <el-button data-testid="add-section" :icon="Plus" :disabled="model.sections.length >= 20" @click="addSection">
          添加 Section
        </el-button>
      </div>

      <el-empty v-if="model.sections.length === 0" description="暂无 Reference Section，可只使用 Core。" />
      <div v-else class="section-list">
        <article v-for="(section, index) in model.sections" :key="`${section.sectionKey}-${index}`" class="reference-card">
          <div class="reference-card__head">
            <div>
              <strong>Section {{ index + 1 }}</strong>
              <span>ordinal: {{ index }}</span>
            </div>
            <div class="reference-card__actions">
              <el-button circle :icon="ArrowUp" :disabled="index === 0" @click="moveSection(index, -1)" />
              <el-button circle :icon="ArrowDown" :disabled="index === model.sections.length - 1" @click="moveSection(index, 1)" />
              <el-button
                circle
                type="danger"
                plain
                :icon="Delete"
                :data-testid="`delete-section-${index}`"
                @click="removeSection(index)"
              />
            </div>
          </div>
          <div class="field-grid field-grid--three">
            <label class="field">
              <span>Section Key</span>
              <el-input v-model="section.sectionKey" :data-testid="`section-key-${index}`" placeholder="screening_evidence" />
            </label>
            <label class="field">
              <span>标题</span>
              <el-input v-model="section.title" placeholder="筛选证据规则" />
            </label>
            <label class="field">
              <span>优先级</span>
              <el-input-number v-model="section.priority" :min="-1000" :max="1000" controls-position="right" />
            </label>
          </div>
          <label class="field">
            <span>描述</span>
            <el-input v-model="section.description" placeholder="说明该 Section 何时有用" />
          </label>
          <div class="field-grid field-grid--three">
            <label class="field">
              <span>触发词</span>
              <el-select v-model="section.triggerTerms" multiple filterable allow-create default-first-option>
                <el-option v-for="item in section.triggerTerms" :key="item" :label="item" :value="item" />
              </el-select>
            </label>
            <label class="field">
              <span>语义标签</span>
              <el-select v-model="section.semanticTags" multiple filterable allow-create default-first-option>
                <el-option v-for="item in section.semanticTags" :key="item" :label="item" :value="item" />
              </el-select>
            </label>
            <label class="field">
              <span>Planner Intents</span>
              <el-select v-model="section.plannerIntents" multiple filterable allow-create default-first-option>
                <el-option v-for="item in section.plannerIntents" :key="item" :label="item" :value="item" />
              </el-select>
            </label>
          </div>
          <label class="field">
            <span>Section Markdown</span>
            <el-input v-model="section.contentMarkdown" type="textarea" :rows="7" placeholder="填写仅在匹配时加载的参考规则或领域知识。" />
          </label>
        </article>
      </div>
    </section>
  </div>
</template>

<style scoped>
.package-editor {
  display: grid;
  gap: 16px;
}

.editor-section {
  display: grid;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.editor-section__head,
.reference-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.editor-section__head h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 15px;
}

.editor-section__head p {
  margin: 5px 0 0;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.field-grid--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.field-grid--four {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.field {
  display: grid;
  min-width: 0;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.field :deep(.el-select),
.field :deep(.el-input-number) {
  width: 100%;
}

.section-list {
  display: grid;
  gap: 14px;
}

.reference-card {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: var(--surface-muted);
}

.reference-card__head > div:first-child {
  display: flex;
  align-items: baseline;
  gap: 10px;
}

.reference-card__head span {
  color: var(--text-faint);
  font-size: 12px;
}

.reference-card__actions {
  display: flex;
  gap: 6px;
}

@media (max-width: 920px) {
  .field-grid,
  .field-grid--three,
  .field-grid--four {
    grid-template-columns: 1fr;
  }
}
</style>
