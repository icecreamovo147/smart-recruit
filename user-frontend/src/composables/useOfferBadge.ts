import { ref } from 'vue'
import { listMyOffers } from '@/api/offer'

/**
 * Shared reactive state for the pending-offer badge shown in the top nav.
 * Imported by App.vue (reads + refreshes on mount) and by JobProgressView.vue
 * (refreshes after accept/reject).
 */
const pendingOfferCount = ref(0)

export function useOfferBadge() {
  const refreshPendingOfferCount = async () => {
    try {
      const data = await listMyOffers({ page_size: 50 })
      pendingOfferCount.value = (data.list || []).filter((o) => o.status === 'sent').length
    } catch {
      // Silently fail – badge stays at previous value or 0
    }
  }

  return { pendingOfferCount, refreshPendingOfferCount }
}
