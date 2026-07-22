<script setup lang="ts">
import { searchSchools } from '@shared/utils/schoolSearch'

const model = defineModel<string>({ default: '' })

const querySchools = (query: string, cb: (results: Array<{ value: string }>) => void) => {
  cb(searchSchools(query).map((value) => ({ value })))
}
</script>

<template>
  <el-autocomplete
    v-model="model"
    :fetch-suggestions="querySchools"
    placeholder="搜索或输入学校名称"
    clearable
    style="width: 100%"
    :trigger-on-focus="true"
  />
</template>
