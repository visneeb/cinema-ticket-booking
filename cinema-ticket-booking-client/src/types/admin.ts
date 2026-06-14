export interface BookingItem {
  id: string
  showtime_id: string
  user_id: string
  seat_id: string
  status: 'BOOKED' | 'LOCKED'
  created_at: string
}

export interface MovieItem {
  id: string
  title: string
}

export interface UserItem {
  uid: string
  email: string
}

export interface BookingFilters {
  movie_id: string
  user_id: string
  date_from: string
  date_to: string
  status: string
  page: number
  page_size: number
}

export interface BookingListResponse {
  items: BookingItem[]
  total: number
}
