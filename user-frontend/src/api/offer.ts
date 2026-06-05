import request from './request'
import type { Offer } from '@/types/domain'

export const listMyOffers = (params: { page?: number; page_size?: number; cursor?: string }): Promise<{ list: Offer[]; total: number; next_cursor?: string; has_more?: boolean }> =>
  request.get('/api/v1/candidate/offers', { params })

export const getOffer = (offerId: number): Promise<{ offer: Offer }> =>
  request.get(`/api/v1/candidate/offers/${offerId}`)

export const acceptOffer = (offerId: number): Promise<void> =>
  request.post(`/api/v1/candidate/offers/${offerId}/accept`)

export const rejectOffer = (offerId: number, reason?: string): Promise<void> =>
  request.post(`/api/v1/candidate/offers/${offerId}/reject`, { reason })
