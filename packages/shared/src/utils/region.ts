const REGION_SEPARATOR = '/'

const MUNICIPALITIES = new Set(['北京市', '天津市', '上海市', '重庆市'])

export function formatRegion(parts: string[] | null | undefined): string {
  if (!parts?.length) return ''
  return parts.map((part) => part.trim()).filter(Boolean).join(REGION_SEPARATOR)
}

export function parseRegion(value: string | null | undefined): string[] {
  if (!value?.trim()) return []
  return value.split(REGION_SEPARATOR).map((part) => part.trim()).filter(Boolean)
}

export function isMunicipality(label: string): boolean {
  return MUNICIPALITIES.has(label.trim())
}

/** Whether a stored region value is complete for display / validation. */
export function isRegionComplete(value: string | null | undefined): boolean {
  const parts = parseRegion(value)
  if (parts.length === 0) return false
  if (isMunicipality(parts[0])) {
    const district = parts.length >= 3 ? parts[2] : parts[1]
    return Boolean(district)
  }
  return parts.length >= 3
}

/** Normalize legacy municipality values like 北京市/市辖区/朝阳区 → 北京市/朝阳区. */
export function normalizeRegionValue(value: string | null | undefined): string {
  const parts = parseRegion(value)
  if (parts.length < 2 || !isMunicipality(parts[0])) {
    return formatRegion(parts)
  }
  if (parts.length >= 3 && parts[1] === '市辖区') {
    return formatRegion([parts[0], parts[2]])
  }
  return formatRegion(parts)
}

export function regionPartsFromValue(value: string | null | undefined): {
  province: string
  city: string
  district: string
  isMunicipality: boolean
} {
  const parts = parseRegion(normalizeRegionValue(value))
  const province = parts[0] ?? ''
  const municipality = isMunicipality(province)
  if (municipality) {
    return {
      province,
      city: '',
      district: parts.length >= 3 ? (parts[2] ?? '') : (parts[1] ?? ''),
      isMunicipality: true,
    }
  }
  return {
    province,
    city: parts[1] ?? '',
    district: parts[2] ?? '',
    isMunicipality: false,
  }
}

export function regionValueFromParts(
  province: string,
  city: string,
  district: string,
): string {
  if (isMunicipality(province)) {
    return formatRegion([province, district])
  }
  return formatRegion([province, city, district])
}

export function isRegionSelectionComplete(
  province: string,
  city: string,
  district: string,
): boolean {
  if (isMunicipality(province)) {
    return Boolean(province && district)
  }
  return Boolean(province && city && district)
}

/**
 * Best-effort conversion of free-text locations into slash-separated 省/市[/区].
 * Returns a cascader-friendly value even when district is missing.
 */
export function normalizeRegionForProfile(raw: string | null | undefined): string {
  const value = (raw ?? '').trim().replace(/\s+/g, '').replace(/／/g, '/')
  if (!value) return ''
  if (value.includes('/')) {
    return normalizeRegionValue(value)
  }
  const municipalities: Array<[string, string]> = [
    ['北京市', '北京'],
    ['上海市', '上海'],
    ['天津市', '天津'],
    ['重庆市', '重庆'],
  ]
  for (const [canonical, alias] of municipalities) {
    if (value.startsWith(canonical) || value.startsWith(alias)) {
      const rest = value.replace(canonical, '').replace(alias, '').replace(/^市/, '')
      const districtMatch = rest.match(/([^省市区县]{1,12}(?:区|县|旗))$/)
      if (districtMatch?.[1]) {
        return formatRegion([canonical, districtMatch[1]])
      }
      return canonical
    }
  }
  const match = value.match(/^(?:([^省市区县]{2,10}省))?(?:([^省市区县]{1,12}市))?(?:([^省市区县]{1,12}(?:区|县|旗)))?$/)
  if (!match) return ''
  const province = match[1] || ''
  const city = match[2] || ''
  const district = match[3] || ''
  if (province && city && district) {
    return formatRegion([province, city, district])
  }
  if (province && city) {
    return formatRegion([province, city])
  }
  if (province) {
    return province
  }
  return ''
}
