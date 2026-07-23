<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { regionData } from 'element-china-area-data'
import {
  isMunicipality,
  regionPartsFromValue,
  regionValueFromParts,
} from '@shared/utils/region'

type RegionNode = {
  value: string
  label: string
  children?: RegionNode[]
}

const model = defineModel<string>({ default: '' })
const emit = defineEmits<{ selecting: [] }>()

const province = ref('')
const city = ref('')
const district = ref('')
const syncing = ref(false)

const provinces = computed(() => (regionData as RegionNode[]).map((item) => item.label))

const isMunicipalityProvince = computed(() => isMunicipality(province.value))

const findProvinceNode = (label: string): RegionNode | undefined =>
  (regionData as RegionNode[]).find((item) => item.label === label)

const cities = computed(() => {
  if (!province.value || isMunicipalityProvince.value) return []
  return (findProvinceNode(province.value)?.children ?? []).map((item) => item.label)
})

const districts = computed(() => {
  if (!province.value) return []
  const provinceNode = findProvinceNode(province.value)
  if (!provinceNode) return []

  if (isMunicipalityProvince.value) {
    const municipalityCity = provinceNode.children?.find((item) => item.label === '市辖区') ?? provinceNode.children?.[0]
    return (municipalityCity?.children ?? []).map((item) => item.label)
  }

  if (!city.value) return []
  const cityNode = provinceNode.children?.find((item) => item.label === city.value)
  return (cityNode?.children ?? []).map((item) => item.label)
})

const districtDisabled = computed(() => {
  if (!province.value) return true
  if (isMunicipalityProvince.value) return false
  return !city.value
})

const normalize = (value: string | null | undefined): string => (value ?? '').trim()

const syncFromModel = (value: string) => {
  syncing.value = true
  const parsed = regionPartsFromValue(value)
  province.value = parsed.province
  city.value = parsed.city
  district.value = parsed.district
  syncing.value = false
}

const commitModel = () => {
  // Persist partial selections (e.g. province+city after clearing district)
  // so the model watch does not treat '' as an external reset and wipe upstream levels.
  const allEmpty = !province.value && !city.value && !district.value
  const next = allEmpty
    ? ''
    : regionValueFromParts(province.value, city.value, district.value)

  if (next !== model.value) {
    model.value = next
  }
}

const onProvinceChange = (value: string | null | undefined) => {
  if (syncing.value) return
  emit('selecting')
  province.value = normalize(value)
  city.value = ''
  district.value = ''
  commitModel()
}

const onCityChange = (value: string | null | undefined) => {
  if (syncing.value) return
  emit('selecting')
  city.value = normalize(value)
  district.value = ''
  commitModel()
}

const onDistrictChange = (value: string | null | undefined) => {
  if (syncing.value) return
  emit('selecting')
  district.value = normalize(value)
  commitModel()
}

watch(
  () => model.value,
  (value) => {
    const current = regionValueFromParts(province.value, city.value, district.value)
    if (value !== current) {
      syncFromModel(value)
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="region-selects" :class="{ 'region-selects--municipality': isMunicipalityProvince }">
    <el-select
      :model-value="province"
      placeholder="省"
      filterable
      clearable
      :validate-event="false"
      class="region-selects__item"
      @update:model-value="onProvinceChange"
    >
      <el-option v-for="item in provinces" :key="`province-${item}`" :label="item" :value="item" />
    </el-select>
    <el-select
      v-if="!isMunicipalityProvince"
      :model-value="city"
      placeholder="市"
      filterable
      clearable
      :disabled="!province"
      :validate-event="false"
      class="region-selects__item"
      @update:model-value="onCityChange"
    >
      <el-option v-for="item in cities" :key="`city-${item}`" :label="item" :value="item" />
    </el-select>
    <el-select
      :model-value="district"
      placeholder="区"
      filterable
      clearable
      :disabled="districtDisabled"
      :validate-event="false"
      class="region-selects__item"
      @update:model-value="onDistrictChange"
    >
      <el-option v-for="item in districts" :key="`district-${item}`" :label="item" :value="item" />
    </el-select>
  </div>
</template>

<style scoped>
.region-selects {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  width: 100%;
}

.region-selects--municipality {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.region-selects__item {
  width: 100%;
  min-width: 0;
}

@media (max-width: 640px) {
  .region-selects,
  .region-selects--municipality {
    grid-template-columns: 1fr;
  }
}
</style>
