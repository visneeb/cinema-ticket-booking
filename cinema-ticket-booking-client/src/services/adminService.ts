import api from './api/api'
import type { BookingFilters, BookingListResponse, MovieItem, UserItem } from '@/types/admin'

export async function listBookings(filters: BookingFilters): Promise<BookingListResponse> {
  const { data } = await api.get('/api/admin/bookings', { params: filters })
  return data
}

export async function listMovies(): Promise<MovieItem[]> {
  const { data } = await api.get('/api/admin/movies')
  return data.items ?? []
}

export async function listUsers(): Promise<UserItem[]> {
  const { data } = await api.get('/api/admin/users')
  return data.items ?? []
}
