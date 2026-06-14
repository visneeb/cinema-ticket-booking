import { ref } from 'vue'
import { getShowtimes } from '@/services/api/showtimeService'
import type { Showtime } from '@/types/seat'

export function useShowtimes() {
  const showtimes = ref<Showtime[]>([])
  const loading = ref(false)
  const error = ref('')

  async function fetchShowtimes() {
    loading.value = true
    error.value = ''
    try {
      showtimes.value = await getShowtimes()
    } catch {
      error.value = 'Failed to load showtimes. Please refresh the page.'
    } finally {
      loading.value = false
    }
  }

  return { showtimes, loading, error, fetchShowtimes }
}
