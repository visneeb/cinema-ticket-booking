import api from './api/api'
import type { Seat, SeatLock } from '@/types/seat'
import type { Showtime } from '@/types/showtime'

export async function getShowtimes(): Promise<Showtime[]> {
  const { data } = await api.get<Showtime[]>('/api/showtimes')
  return data
}

export async function getSeats(showtimeId: string): Promise<Seat[]> {
  const { data } = await api.get<Seat[]>(`/api/showtimes/${showtimeId}/seats`)
  return data
}

export async function getMyLocks(showtimeId: string): Promise<SeatLock[]> {
  const { data } = await api.get<{ locks: SeatLock[] }>(`/api/showtimes/${showtimeId}/my-lock`)
  return data.locks
}

export async function lockSeat(showtimeId: string, seatId: string): Promise<number> {
  const { data } = await api.post<{ seconds_left: number }>(
    `/api/showtimes/${showtimeId}/seats/${seatId}/lock`,
  )
  return data.seconds_left ?? 300
}

export async function unlockSeat(showtimeId: string, seatId: string): Promise<void> {
  await api.delete(`/api/showtimes/${showtimeId}/seats/${seatId}/lock`)
}

export async function bookSeat(showtimeId: string, seatId: string): Promise<void> {
  await api.post(`/api/showtimes/${showtimeId}/seats/${seatId}/book`)
}
