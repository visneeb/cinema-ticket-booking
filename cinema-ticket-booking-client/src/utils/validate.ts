/** Returns true if the string is a valid 24-character MongoDB ObjectID hex. */
export function isValidObjectId(id: string): boolean {
  return /^[0-9a-f]{24}$/i.test(id)
}

/** Formats an ISO-8601 date string for display in the showtime list. */
export function formatShowtimeDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
