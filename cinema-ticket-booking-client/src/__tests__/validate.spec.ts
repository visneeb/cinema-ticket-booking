import { describe, it, expect } from 'vitest'
import { isValidObjectId } from '@/utils/validate'

describe('isValidObjectId', () => {
  it('returns true for a valid 24-char hex', () => {
    expect(isValidObjectId('507f1f77bcf86cd799439011')).toBe(true)
  })
  it('returns false for a short string', () => {
    expect(isValidObjectId('abc123')).toBe(false)
  })
  it('returns false for empty string', () => {
    expect(isValidObjectId('')).toBe(false)
  })
  it('returns false for 24 chars that are not hex', () => {
    expect(isValidObjectId('gggggggggggggggggggggggg')).toBe(false)
  })
})
