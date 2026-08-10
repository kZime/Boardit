import { describe, expect, it } from 'vitest'

import { authErrorMessage, validateLogin, validateRegistration } from './form'

describe('auth form validation', () => {
  it('normalizes valid login credentials', () => {
    expect(validateLogin({ email: ' user@example.com ', password: ' password ' })).toEqual({
      ok: true,
      value: { email: 'user@example.com', password: 'password' },
    })
  })

  it('applies one registration policy to all forms', () => {
    expect(validateRegistration({ username: 'a!', email: 'bad', password: 'short' })).toEqual({
      ok: false,
      error: 'Username must be 3-32 letters, numbers, underscores, or hyphens',
    })
    expect(validateRegistration({ username: 'writer', email: 'writer@example.com', password: 'short' })).toEqual({
      ok: false,
      error: 'Password must be at least 8 characters',
    })
  })

  it('extracts API errors without unsafe casts at call sites', () => {
    expect(authErrorMessage({ response: { data: { error: 'email exists' } } }, 'FAILED')).toBe('email exists')
    expect(authErrorMessage(new Error('offline'), 'FAILED')).toBe('FAILED')
  })
})
