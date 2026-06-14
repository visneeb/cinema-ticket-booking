/** Server statuses: AVAILABLE | LOCKED | BOOKED. SELECTED is client-only visual state. */
export type SeatStatus = 'AVAILABLE' | 'LOCKED' | 'BOOKED' | 'SELECTED'

export interface Seat {
  id: string // ObjectID hex
  label: string
  status: SeatStatus
  showtime_id: string // ObjectID hex
}

export interface SeatEvent {
  seat_id: string // ObjectID hex
  showtime_id: string // ObjectID hex
  status: SeatStatus
}

export interface SeatLock {
  seat_id: string // ObjectID hex
  seconds_left: number
}

export interface Showtime {
  id: string // ObjectID hex
  movie_id: string // ObjectID hex
  starts_at: string // ISO-8601 date string
  title: string
  description: string
}
