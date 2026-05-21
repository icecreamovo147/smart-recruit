import axios from 'axios'
import request from './request'
import type { ConfirmPayload, PresignPayload, PresignResponse, ResumeInfo } from '@/types/domain'

export const getResume = (): Promise<{ resume: ResumeInfo | null }> =>
  request.get('/api/v1/candidate/resume')

export const presignResume = (data: PresignPayload): Promise<PresignResponse> =>
  request.post('/api/v1/candidate/resume/presign', data)

export const confirmResume = (data: ConfirmPayload): Promise<{ resume_id: number }> =>
  request.post('/api/v1/candidate/resume/confirm', data)

export const putResumeFile = (uploadUrl: string, file: File): Promise<void> =>
  axios.put(uploadUrl, file, {
    headers: { 'Content-Type': file.type || 'application/octet-stream' },
  }) as unknown as Promise<void>
