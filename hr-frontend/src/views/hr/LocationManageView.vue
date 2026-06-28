<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import { listLocations, createLocation, updateLocation, updateLocationStatus, deleteLocation } from '@/api/admin'
import type { LocationOption } from '@/types/domain'

const toNum = (v: unknown): number => (v != null ? Number(v) : 0)

const loading = ref(false)
const list = ref<LocationOption[]>([])
const keyword = ref('')
const statusFilter = ref('')

const dialogVisible = ref(false)
const editing = ref<LocationOption | null>(null)
const form = reactive({
  name: '',
  code: '',
  sort_order: 1,
})

const load = async () => {
  loading.value = true
  try {
    const res = await listLocations()
    list.value = (res.list || []).map(l => ({ ...l, id: toNum(l.id) }))
  } catch {
    ElMessage.error('加载地点数据失败')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  editing.value = null
  form.name = ''
  form.code = ''
  form.sort_order = 1
  dialogVisible.value = true
}

const openEdit = (row: LocationOption) => {
  editing.value = row
  form.name = row.name
  form.code = row.code || ''
  form.sort_order = toNum(row.sort_order)
  dialogVisible.value = true
}

const saveLoc = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请输入地点名称')
    return
  }
  try {
    if (editing.value) {
      await updateLocation(editing.value.id, {
        name: form.name,
        code: form.code,
        sort_order: form.sort_order,
      })
      ElMessage.success('地点已更新')
    } else {
      await createLocation({
        name: form.name,
        code: form.code,
        sort_order: form.sort_order,
      })
      ElMessage.success('地点已创建')
    }
    dialogVisible.value = false
    load()
  } catch {
    // error already shown
  }
}

const toggleStatus = async (row: LocationOption) => {
  const newStatus = row.is_active === 1 ? 0 : 1
  await updateLocationStatus(row.id, newStatus)
  ElMessage.success(newStatus === 1 ? '地点已启用' : '地点已停用')
  load()
}

const remove = async (row: LocationOption) => {
  try {
    await ElMessageBox.confirm(`确认删除地点「${row.name}」？`, '删除地点', { type: 'warning' })
  } catch {
    return
  }
  await deleteLocation(row.id)
  ElMessage.success('地点已删除')
  load()
}

const filteredList = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return list.value.filter((item) => {
    const matchesKeyword = !q
      || item.name.toLowerCase().includes(q)
      || (item.code || '').toLowerCase().includes(q)
    const matchesStatus = !statusFilter.value
      || (statusFilter.value === 'active' ? item.is_active === 1 : item.is_active !== 1)
    return matchesKeyword && matchesStatus
  })
})

const stats = computed(() => [
  { label: '地点总数', value: list.value.length, hint: '可用于岗位发布和数据范围' },
  { label: '已启用', value: list.value.filter((item) => item.is_active === 1).length, hint: '当前可选地点' },
  { label: '已停用', value: list.value.filter((item) => item.is_active !== 1).length, hint: '历史保留或暂不可用' },
])

onMounted(load)
</script>

<template>
  <section class="console-page console-page--fill taxonomy-page">
    <div class="console-header">
      <div class="console-header__copy">
        <p class="console-eyebrow">BASIC DATA</p>
        <h1 class="console-title">地点管理</h1>
        <p class="console-description">维护招聘业务中可使用的城市、园区或办公地点，供岗位发布、部门地点配置和权限数据范围复用。</p>
      </div>
      <div class="console-header__actions">
        <el-button :icon="Refresh" @click="load">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate()">新增地点</el-button>
      </div>
    </div>

    <section class="console-stats">
      <div v-for="item in stats" :key="item.label" class="console-stat">
        <div class="console-stat__label">{{ item.label }}</div>
        <div class="console-stat__value">{{ item.value }}</div>
        <div class="console-stat__hint">{{ item.hint }}</div>
      </div>
    </section>

    <div class="console-card console-card--fill">
      <div class="console-card__head">
        <div>
          <h2 class="console-card__title">地点列表</h2>
          <p class="console-card__desc">共 {{ filteredList.length }} 个匹配地点</p>
        </div>
      </div>
      <div class="console-toolbar">
        <div class="console-toolbar__filters">
          <el-input v-model="keyword" :prefix-icon="Search" clearable placeholder="搜索地点名称 / 编码" style="width: 240px" />
          <el-select v-model="statusFilter" clearable placeholder="全部状态" style="width: 140px">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="inactive" />
          </el-select>
        </div>
      </div>
      <div class="console-table-wrap">
      <el-table v-loading="loading" :data="filteredList" class="console-table" stripe>
        <el-table-column label="地点信息" min-width="220">
          <template #default="{ row }">
            <div class="console-entity">
              <div class="console-entity__name">{{ row.name }}</div>
              <div class="console-entity__meta">编码：<span class="console-code">{{ row.code || '-' }}</span></div>
            </div>
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
            <el-button text type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button text :type="row.is_active === 1 ? 'warning' : 'success'" size="small" @click="toggleStatus(row)">
              {{ row.is_active === 1 ? '停用' : '启用' }}
            </el-button>
            <el-button text type="danger" size="small" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </div>

    <el-drawer v-model="dialogVisible" :title="editing ? '编辑地点' : '新增地点'" size="480px">
      <el-form label-width="80px">
        <el-form-item label="地点名称">
          <el-input v-model="form.name" placeholder="请输入地点名称" />
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model="form.code" placeholder="可选，如 beijing" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="1" :max="999" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveLoc">保存</el-button>
      </template>
    </el-drawer>
  </section>
</template>
