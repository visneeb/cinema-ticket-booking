import type { MovieItem, UserItem } from '@/types/admin'

export function fmtDate(iso: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

export function movieTitle(movies: MovieItem[], id: string): string {
  return movies.find((m) => m.id === id)?.title ?? id
}

export function userEmail(users: UserItem[], uid: string): string {
  return users.find((u) => u.uid === uid)?.email ?? uid
}