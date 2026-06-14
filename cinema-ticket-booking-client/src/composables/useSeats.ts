import { ref } from 'vue'
import type { Ref } from 'vue'
import { getSeats } from '@/services/showtimeService'
import { isValidObjectId } from '@/utils/validate'
import type { Seat } from '@/types/seat'

export function useSeats(showtimeId: Ref<string>) {
  const seats = ref<Seat[]>([])
  const loading = ref(false)
  const error = ref('')

  async function fetchSeats() {
    if (!isValidObjectId(showtimeId.value)) return
    loading.value = true
    error.value = ''
    try {
      seats.value = await getSeats(showtimeId.value)
    } catch {
      error.value = 'Failed to load seats. Please refresh the page.'
    } finally {
      loading.value = false
    }
  }

  return { seats, loading, error, fetchSeats }
}
