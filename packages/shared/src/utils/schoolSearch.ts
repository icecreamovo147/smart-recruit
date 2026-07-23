import { SCHOOL_NAMES } from '../data/schools'

export function searchSchools(query: string, limit = 20): string[] {
  const normalized = query.trim().toLowerCase()
  if (!normalized) {
    return SCHOOL_NAMES.slice(0, limit)
  }
  return SCHOOL_NAMES.filter((name) => name.toLowerCase().includes(normalized)).slice(0, limit)
}
