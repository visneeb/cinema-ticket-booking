import { describe, it, expect } from 'vitest'
import { movieTitle, userEmail, fmtDate } from '@/utils/formatters'

const movies = [{ id: 'abc', title: 'Inception' }]
const users = [{ uid: 'uid1', email: 'alice@example.com' }]

describe('movieTitle', () => {
  it('returns the title when movie is found', () => {
    expect(movieTitle(movies, 'abc')).toBe('Inception')
  })
  it('falls back to the id when not found', () => {
    expect(movieTitle(movies, 'unknown')).toBe('unknown')
  })
})

describe('userEmail', () => {
  it('returns email when user is found', () => {
    expect(userEmail(users, 'uid1')).toBe('alice@example.com')
  })
  it('falls back to uid when not found', () => {
    expect(userEmail(users, 'uid-nobody')).toBe('uid-nobody')
  })
})

describe('fmtDate', () => {
  it('returns dash for empty string', () => {
    expect(fmtDate('')).toBe('—')
  })
  it('returns a non-empty string for a valid ISO date', () => {
    expect(fmtDate('2026-06-15T00:00:00Z').length).toBeGreaterThan(0)
  })
})
