import { ref, reactive, computed } from 'vue'
import { listBookings, listMovies, listUsers } from '@/services/adminService'
import type { BookingItem, BookingFilters, MovieItem, UserItem } from '@/types/admin'

export function useAdminBookings() {
  const filters = reactive<BookingFilters>({
    movie_id: '',
    user_id: '',
    date_from: '',
    date_to: '',
    status: '',
    page: 1,
    page_size: 20,
  })

  const bookings = ref<BookingItem[]>([])
  const total = ref(0)
  const movies = ref<MovieItem[]>([])
  const users = ref<UserItem[]>([])
  const loading = ref(false)
  const error = ref('')

  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filters.page_size)))

  async function fetchData() {
    loading.value = true
    error.value = ''
    try {
      const [bookingRes, moviesRes, usersRes] = await Promise.all([
        listBookings(filters),
        listMovies(),
        listUsers(),
      ])
      bookings.value = bookingRes.items ?? []
      total.value = bookingRes.total
      movies.value = moviesRes
      users.value = usersRes
    } catch (e) {
      error.value = 'Failed to load admin data.'
      console.error(e)
    } finally {
      loading.value = false
    }
  }

  function applyFilters() {
    filters.page = 1
    fetchData()
  }

  function resetFilters() {
    Object.assign(filters, {
      movie_id: '',
      user_id: '',
      date_from: '',
      date_to: '',
      status: '',
      page: 1,
    })
    fetchData()
  }

  function prevPage() {
    if (filters.page > 1) {
      filters.page--
      fetchData()
    }
  }

  function nextPage() {
    if (filters.page < totalPages.value) {
      filters.page++
      fetchData()
    }
  }

  return {
    filters,
    bookings,
    total,
    movies,
    users,
    loading,
    error,
    totalPages,
    fetchData,
    applyFilters,
    resetFilters,
    prevPage,
    nextPage,
  }
}
