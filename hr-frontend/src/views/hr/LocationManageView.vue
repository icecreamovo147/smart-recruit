<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listLocations, createLocation, updateLocation, updateLocationStatus, deleteLocation } from '@/api/admin'
import type { LocationOption } from '@/types/domain'

const toNum = (v: unknown): number => (v != null ? Number(v) : 0)

const loading = ref(false)
const list = ref<LocationOption[]>([])

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

onMounted(load)
</script>

<template>
  <section class="taxonomy-page">
    <div class="page-header">
      <h1 class="page-title">地点管理</h1>
      <el-button type="primary" @click="openCreate()">新增地点</el-button>
    </div>
    <div class="content-surface">
      <el-table v-loading="loading" :data="list" border>
        <el-table-column prop="name" label="地点名称" min-width="160" />
        <el-table-column prop="code" label="编码" width="120" />
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

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑地点' : '新增地点'" width="480px">
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
    </el-dialog>
  </section>
</template>
