<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { ArrowDown, MoreFilled, Plus, Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listDepartments, createDepartment, updateDepartment, updateDepartmentStatus, deleteDepartment,
  getDepartmentLocationConfig, updateDepartmentLocationConfig,
  listDepartmentsLocationMap,
} from '@/api/admin'
import { listLocations } from '@/api/admin'
import type { DepartmentLocationConfig, DepartmentNode, LocationOption } from '@/types/domain'

const loading = ref(false)
const tree = ref<DepartmentNode[]>([])
const locationNames = ref<Map<number, string>>(new Map())
const deptLocationMap = ref<Map<number, number[]>>(new Map())

const dialogVisible = ref(false)
const editing = ref<DepartmentNode | null>(null)
const form = reactive({
  parent_id: 0,
  name: '',
  sort_order: 1,
})

const normTree = (nodes: DepartmentNode[]): DepartmentNode[] =>
  nodes.map(n => ({ ...n, id: toNum(n.id), parent_id: toNum(n.parent_id), depth: toNum(n.depth), children: normTree(n.children || []) }))

// ── Left tree state ────────────────────────────────────────────
const selectedDeptId = ref<number>(0) // 0 = "all departments"
const treeFilterText = ref('')
const deptTreeRef = ref()

const findDeptById = (nodes: DepartmentNode[], id: number): DepartmentNode | null => {
  for (const node of nodes) {
    if (node.id === id) return node
    const found = findDeptById(node.children || [], id)
    if (found) return found
  }
  return null
}

const load = async () => {
  loading.value = true
  try {
    const [deptRes, locMapRes, locRes] = await Promise.all([
      listDepartments(),
      listDepartmentsLocationMap(),
      listLocations(),
    ])
    tree.value = normTree(deptRes.list || [])
    // Build location name lookup
    const m = new Map<number, string>()
    const locs = (locRes.list || []).map(l => ({ ...l, id: toNum(l.id) }))
    for (const l of locs) {
      m.set(l.id, l.name)
    }
    locationNames.value = m
    allLocations.value = locs
    // Build department → location IDs lookup
    const dm = new Map<number, number[]>()
    for (const item of locMapRes.items || []) {
      dm.set(toNum(item.department_id), (item.location_ids || []).map(toNum))
    }
    deptLocationMap.value = dm
    // Preserve selected department after refresh
    const currentId = selectedDeptId.value
    if (currentId !== 0 && !findDeptById(tree.value, currentId)) {
      selectedDeptId.value = 0
    }
    await nextTick()
    if (selectedDeptId.value !== 0) {
      deptTreeRef.value?.setCurrentKey?.(selectedDeptId.value)
    }
  } catch {
    ElMessage.error('加载部门数据失败')
  } finally {
    loading.value = false
  }
}

const toNum = (v: unknown): number => (v != null ? Number(v) : 0)

const deptLocationLabel = (deptId: number): string => {
  const ids = deptLocationMap.value.get(deptId)
  if (!ids || ids.length === 0) return '-'
  return ids.map(id => locationNames.value.get(id) || `#${id}`).join('、')
}

const parentName = ref('')

// ── Computed for left-right layout ────────────────────────────
const selectedDept = computed(() => {
  if (selectedDeptId.value === 0) return null
  return findDeptById(tree.value, selectedDeptId.value)
})

const visibleDepartments = computed(() => {
  const depts = !selectedDept.value ? tree.value : (selectedDept.value.children || [])
  return depts.map(({ children, ...rest }) => rest)
})

const currentDeptName = computed(() => selectedDept.value?.name || '全部部门')

const flattenDepartments = computed(() => {
  const result: DepartmentNode[] = []
  const walk = (nodes: DepartmentNode[]) => {
    for (const node of nodes) {
      result.push(node)
      walk(node.children || [])
    }
  }
  walk(tree.value)
  return result
})

// ── Tree methods ──────────────────────────────────────────────
const selectAllDepartments = () => {
  selectedDeptId.value = 0
  deptTreeRef.value?.setCurrentKey(null)
}

const handleDeptNodeClick = (node: DepartmentNode) => {
  selectedDeptId.value = node.id
}

const filterDeptNode = (value: string, data: DepartmentNode) => {
  if (!value) return true
  return data.name.includes(value.trim())
}

watch(treeFilterText, value => {
  deptTreeRef.value?.filter(value)
})

const openCreateUnderSelected = () => {
  if (!selectedDept.value) return
  openCreate(selectedDept.value)
}

const openCreate = (parent?: DepartmentNode) => {
  editing.value = null
  form.parent_id = toNum(parent?.id)
  parentName.value = parent?.name || ''
  form.name = ''
  // Default to max sort_order among siblings + 1 (= append to end)
  // Parent may come from visibleDepartments (children stripped), so look up full node
  const fullParent = parent ? (findDeptById(tree.value, parent.id) ?? parent) : undefined
  const siblings = fullParent ? (fullParent.children || []) : tree.value
  const maxSort = siblings.reduce((max, s) => Math.max(max, toNum(s.sort_order)), 0)
  form.sort_order = maxSort + 1
  dialogVisible.value = true
}

const openEdit = (row: DepartmentNode) => {
  editing.value = row
  form.parent_id = toNum(row.parent_id)
  form.name = row.name
  form.sort_order = toNum(row.sort_order)
  dialogVisible.value = true
}

const save = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请输入部门名称')
    return
  }
  try {
    if (editing.value) {
      await updateDepartment(editing.value.id, {
        parent_id: form.parent_id,
        name: form.name,
        sort_order: form.sort_order,
      })
      ElMessage.success('部门已更新')
    } else {
      await createDepartment({
        parent_id: form.parent_id,
        name: form.name,
        sort_order: form.sort_order,
      })
      ElMessage.success('部门已创建')
    }
    dialogVisible.value = false
    load()
  } catch {
    // error already shown by interceptor
  }
}

const handleRowCommand = (cmd: string, row: DepartmentNode) => {
  switch (cmd) {
    case 'loc-config':
      openLocConfig(row)
      break
    case 'toggle-status':
      toggleStatus(row)
      break
    case 'delete':
      remove(row)
      break
  }
}

const toggleStatus = async (row: DepartmentNode) => {
  const newStatus = row.is_active === 1 ? 0 : 1
  await updateDepartmentStatus(row.id, newStatus)
  ElMessage.success(newStatus === 1 ? '部门已启用' : '部门已停用')
  load()
}

const remove = async (row: DepartmentNode) => {
  try {
    await ElMessageBox.confirm(`确认删除部门「${row.name}」？`, '删除部门', { type: 'warning' })
  } catch {
    return
  }
  await deleteDepartment(row.id)
  ElMessage.success('部门已删除')
  load()
}

// ── Location config dialog ──────────────────────────────────────

const locDialogVisible = ref(false)
const locDept = ref<DepartmentNode | null>(null)
const locConfig = ref<DepartmentLocationConfig | null>(null)
const allLocations = ref<LocationOption[]>([])
const locForm = reactive({
  inherit_locations: 1,
  location_ids: [] as number[],
})
const locSaving = ref(false)

const isRootDept = () => locDept.value?.parent_id === 0

const selectableLocations = () => {
  if (!locConfig.value?.available_location_ids) return allLocations.value
  const set = new Set(locConfig.value.available_location_ids.map(toNum))
  return allLocations.value.filter(l => set.has(l.id))
}

const previewLocationIds = (): number[] => {
  if (!locConfig.value) return []
  // Custom mode → show the user's current selection in real-time
  if (locForm.inherit_locations === 0) return locForm.location_ids
  // Inherit mode → show effective locations from the backend
  return (locConfig.value.effective_location_ids || []).map(toNum)
}

const openLocConfig = async (row: DepartmentNode) => {
  locDept.value = row
  try {
    const cfg = await getDepartmentLocationConfig(row.id)
    locConfig.value = cfg
    // Root departments cannot inherit — force custom config
    locForm.inherit_locations = row.parent_id === 0 ? 0 : cfg.inherit_locations
    locForm.location_ids = (cfg.direct_location_ids || []).map(toNum)
  } catch {
    ElMessage.error('加载地点配置失败')
    return
  }
  locDialogVisible.value = true
}

const saveLocConfig = async () => {
  if (!locDept.value) return
  locSaving.value = true
  try {
    await updateDepartmentLocationConfig(locDept.value.id, {
      inherit_locations: locForm.inherit_locations,
      location_ids: locForm.inherit_locations === 1 ? [] : locForm.location_ids,
    })
    ElMessage.success('地点配置已保存')
    locDialogVisible.value = false
    load()
  } catch {
    // error already shown by interceptor
  } finally {
    locSaving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="console-page console-page--fill taxonomy-page">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">BASIC DATA</p>
          <h1 class="console-title">部门管理</h1>
          <p class="console-description">维护企业组织树，并配置部门可用地点。岗位、候选人台账和员工数据权限都会复用这里的组织结构。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" @click="load">刷新</el-button>
          <el-button type="primary" :icon="Plus" @click="openCreate()">新增根部门</el-button>
        </div>
      </div>
      <div class="workspace-surface__divider"></div>
      <div class="workspace-surface__body department-layout">
      <aside class="department-tree-panel">
        
        <el-input
          v-model="treeFilterText"
          placeholder="搜索部门"
          clearable
          class="department-tree-search"
        />
        <button
          class="department-tree-root"
          :class="{ 'is-active': selectedDeptId === 0 }"
          type="button"
          @click="selectAllDepartments"
        >
          全部部门
        </button>
        <el-tree
          ref="deptTreeRef"
          class="department-tree"
          :data="tree"
          node-key="id"
          default-expand-all
          highlight-current
          indent="20"
          :props="{ label: 'name', children: 'children' }"
          :filter-node-method="filterDeptNode"
          @node-click="handleDeptNodeClick"
        />
      </aside>

      <section class="department-table-panel">
        <div class="department-table-head">
          <div>
            <h2 class="department-table-head__title">{{ currentDeptName }}</h2>
            <p class="department-table-head__meta">{{ visibleDepartments.length }} 个直接子部门</p>
          </div>
          <div class="department-table-actions">
            <el-button v-if="selectedDept && selectedDept.depth < 2" type="primary" size="default" @click="openCreateUnderSelected">
              新增子部门
            </el-button>
          </div>
        </div>

        <div class="department-table-wrapper">
          <el-table v-loading="loading" :data="visibleDepartments" row-key="id" class="console-table" stripe>
            <el-table-column label="部门信息" min-width="220">
              <template #default="{ row }">
                <div class="console-entity">
                  <div class="console-entity__name">{{ row.name }}</div>
                  <div class="console-entity__meta">层级 {{ row.depth ?? 0 }} / 排序 {{ row.sort_order ?? 0 }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="部门地点" min-width="240">
              <template #default="{ row }">
                <template v-if="deptLocationMap.get(row.id)?.length">
                  <el-tag
                    v-for="id in deptLocationMap.get(row.id)"
                    :key="id"
                    size="small"
                    style="margin: 1px 4px 1px 0"
                  >
                    {{ locationNames.get(id) || `#${id}` }}
                  </el-tag>
                </template>
                <span v-else style="color: var(--text-faint)">-</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.is_active === 1 ? 'success' : 'info'" size="small">
                  {{ row.is_active === 1 ? '启用' : '停用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right" align="center">
              <template #default="{ row }">
                <el-button v-if="row.depth < 2" size="small" @click="openCreate(row)">添加子部门</el-button>
                <el-button size="small" @click="openEdit(row)">编辑</el-button>
                <el-dropdown trigger="click" popper-class="dropdown-menu-center" @command="(cmd: string) => handleRowCommand(cmd, row)">
                  <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="loc-config">地点配置</el-dropdown-item>
                      <el-dropdown-item command="toggle-status">{{ row.is_active === 1 ? '停用' : '启用' }}</el-dropdown-item>
                      <el-dropdown-item command="delete" divided style="color: var(--el-color-danger)">删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </section>
    </div>
    </div>

    <el-drawer v-model="dialogVisible" :title="editing ? '编辑部门' : '新增部门'" size="480px" :close-on-click-modal="true">
      <el-form label-width="80px">
        <el-form-item v-if="!editing && form.parent_id > 0" label="父部门">
          <el-input disabled :model-value="parentName" />
        </el-form-item>
        <el-form-item label="部门名称">
          <el-input v-model="form.name" placeholder="请输入部门名称" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="1" :max="999" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-drawer>

    <!-- Location config dialog -->
    <el-drawer v-model="locDialogVisible" :title="`地点配置 - ${locDept?.name || ''}`" size="540px" :close-on-click-modal="true">
      <el-form v-if="locConfig" label-width="120px">
        <el-form-item label="继承上级地点">
          <el-switch
            v-model="locForm.inherit_locations"
            :active-value="1"
            :inactive-value="0"
            active-text="继承"
            inactive-text="自定义"
            :disabled="isRootDept()"
          />
          <span v-if="isRootDept()" class="text-muted" style="font-size: 12px; margin-left: 8px;">
            根部门没有上级，只能使用自定义地点配置
          </span>
        </el-form-item>
        <el-form-item label="自定义地点">
          <el-select
            v-model="locForm.location_ids"
            multiple
            filterable
            placeholder="选择地点"
            :disabled="locForm.inherit_locations === 1"
            style="width: 100%"
          >
            <el-option v-for="item in selectableLocations()" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="生效地点预览">
          <el-tag
            v-for="id in previewLocationIds()"
            :key="id"
            style="margin: 2px 4px"
            size="small"
          >
            {{ allLocations.find(l => l.id === id)?.name || `#${id}` }}
          </el-tag>
          <span v-if="!previewLocationIds().length" class="text-muted">暂无</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="locDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="locSaving" @click="saveLocConfig">保存</el-button>
      </template>
    </el-drawer>
  </section>
</template>

<style>
.dropdown-menu-center .el-dropdown-menu__item {
  text-align: center;
}
</style>

<style scoped>
.workspace-surface__body.department-layout {
  padding: 0;
}

.department-tree-search {
  padding: 0 2px;
}

.department-tree-search :deep(.el-input__wrapper) {
  border-radius: 6px;
}

.department-tree-panel {
  background: transparent;
}

.department-tree :deep(.el-tree-node__content) {
  border-radius: 6px;
  padding: 0 6px;
  margin: 0 2px;
  transition: background var(--motion-fast) var(--motion-ease);
}

.department-tree :deep(.el-tree-node__content:hover) {
  background: var(--surface-muted);
}

.department-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: var(--brand-soft);
  color: var(--brand-strong);
  font-weight: 600;
}

.department-tree :deep(.el-tree-node__expand-icon) {
  font-size: 14px;
  color: var(--text-faint);
}

.department-tree :deep(.el-tree-node__label) {
  font-size: 13px;
}

.department-table-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.department-table-wrapper :deep(.el-table) {
  height: 100%;
}

.department-table-wrapper :deep(.el-table__body-wrapper) {
  overflow-y: auto;
}

.department-table-actions {
  flex-shrink: 0;
}

:deep(.console-table .el-table__header th) {
  background: var(--surface-muted);
}

@media (max-width: 768px) {
  .department-tree-panel {
    padding: 14px 14px 14px 16px;
  }
  .department-table-panel {
    padding: 14px 16px;
  }
}
</style>
