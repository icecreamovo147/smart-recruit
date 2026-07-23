import request from './request'
import type { Profile, ProfileFillDraft } from '@/types/domain'

export const getProfile = (): Promise<Profile> =>
  request.get('/api/v1/candidate/profile')

export const updateProfile = (data: Profile): Promise<Profile> =>
  request.put('/api/v1/candidate/profile', data)

export const fillProfileFromResume = (options?: {
  force_refresh?: boolean
  overwrite_existing?: boolean
}): Promise<ProfileFillDraft> =>
  request.post('/api/v1/candidate/profile/fill-from-resume', {
    force_refresh: Boolean(options?.force_refresh),
    overwrite_existing: Boolean(options?.overwrite_existing),
  })

export const applyProfileFill = (data: { draft: Profile; overwrite_existing?: boolean }): Promise<Profile> =>
  request.post('/api/v1/candidate/profile/apply-fill', {
    overwrite_existing: data.overwrite_existing,
    draft: {
      ...data.draft,
      // Fill drafts expose skills as string[]; update/apply-fill accept a CSV string.
      skills: Array.isArray(data.draft.skills)
        ? data.draft.skills.join(',')
        : (data.draft.skills ?? ''),
    },
  })
