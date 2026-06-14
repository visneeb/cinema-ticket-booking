import { useWebSocket } from './useWebSocket'
import { auth } from '@/firebase'

export async function useShowtimeSocket(showtimeId: string) {
  const token = await auth.currentUser?.getIdToken()
  return useWebSocket(`/ws/showtime/${showtimeId}?token=${token}`)
}
