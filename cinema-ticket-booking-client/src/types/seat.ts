/** Server-persisted statuses. 'SELECTED' is client-only optimistic state. */
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
  seat_id: string
  seconds_left: number
}
