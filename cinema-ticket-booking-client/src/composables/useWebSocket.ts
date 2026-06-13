import { ref, onUnmounted } from 'vue'

export function useWebSocket(url: string) {
  const messages = ref<string[]>([])
  const status = ref<'connecting' | 'open' | 'closed'>('connecting')

  if (typeof WebSocket === 'undefined') {
    return { messages, status, send: () => {} }
  }

  const socket = new WebSocket(url)

  socket.onopen = () => {
    status.value = 'open'
  }
  socket.onmessage = (event) => {
    messages.value.push(event.data)
  }
  socket.onclose = () => {
    status.value = 'closed'
  }
  socket.onerror = (err) => console.error('WS error', err)

  function send(msg: string) {
    if (socket.readyState === WebSocket.OPEN) {
      socket.send(msg)
    }
  }

  onUnmounted(() => socket.close())

  return { messages, status, send }
}
