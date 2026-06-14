import { ref, onUnmounted } from 'vue'
import type { SeatEvent } from '@/types/seat'

export type WsStatus = 'connecting' | 'open' | 'closed' | 'error'

/**
 * Opens a WebSocket to /ws/showtimes/:showtimeId and returns typed seat events.
 * Auto-reconnects every 3 s on disconnect.
 */
export function useShowtimeSocket(showtimeId: string) {
  const lastEvent = ref<SeatEvent | null>(null)
  // No showtime → stay 'open' so no reconnect banner is shown on the home page
  const wsStatus = ref<WsStatus>(showtimeId ? 'connecting' : 'open')

  let socket: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let destroyed = false

  function wsUrl() {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}/ws/showtimes/${showtimeId}`
  }

  function connect() {
    if (destroyed || !showtimeId) return
    wsStatus.value = 'connecting'
    socket = new WebSocket(wsUrl())

    socket.onopen = () => {
      wsStatus.value = 'open'
    }

    socket.onmessage = (evt) => {
      try {
        lastEvent.value = JSON.parse(evt.data) as SeatEvent
      } catch {
        console.warn('[WS] unparseable message:', evt.data)
      }
    }

    socket.onerror = () => {
      wsStatus.value = 'error'
    }

    socket.onclose = () => {
      if (destroyed) return
      wsStatus.value = 'closed'
      reconnectTimer = setTimeout(connect, 3000)
    }
  }

  connect()

  onUnmounted(() => {
    destroyed = true
    if (reconnectTimer) clearTimeout(reconnectTimer)
    socket?.close()
  })

  return { lastEvent, wsStatus }
}
