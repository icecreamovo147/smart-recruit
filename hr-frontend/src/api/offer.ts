import request from './request'
import type { Offer, OfferEvent } from '@/types/domain'

export const createOffer = (data: {
  application_id: number
  title: string
  salary_range?: string
  level?: string
  work_location?: string
  start_date?: string
  expires_at?: string
  terms_json?: string
}): Promise<{ offer_id: number }> =>
  request.post('/api/v1/hr/offers', data)

export const updateOffer = (offerId: number, data: {
  title?: string
  salary_range?: string
  level?: string
  work_location?: string
  start_date?: string
  expires_at?: string
  terms_json?: string
}): Promise<void> =>
  request.put(`/api/v1/hr/offers/${offerId}`, data)

export const getOffer = (offerId: number): Promise<{ offer: Offer }> =>
  request.get(`/api/v1/hr/offers/${offerId}`)

export const listOffersByApplication = (applicationId: number): Promise<{ list: Offer[] }> =>
  request.get(`/api/v1/hr/applications/${applicationId}/offers`)

export const sendOffer = (offerId: number): Promise<void> =>
  request.post(`/api/v1/hr/offers/${offerId}/send`)

export const withdrawOffer = (offerId: number, reason?: string): Promise<void> =>
  request.post(`/api/v1/hr/offers/${offerId}/withdraw`, { reason })

export const listOfferEvents = (offerId: number): Promise<{ list: OfferEvent[] }> =>
  request.get(`/api/v1/hr/offers/${offerId}/events`)
