import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
  subscribeAuthState,
  withAuthStateLock,
} from './tokenStorage'

afterEach(() => {
  localStorage.clear()
})

describe('token storage', () => {
  it('publishes a token pair through one authoritative storage value', () => {
    localStorage.setItem('accessToken', 'legacy-access')
    localStorage.setItem('refreshToken', 'legacy-refresh')

    setTokens('next-access', 'next-refresh')

    expect(getAccessToken()).toBe('next-access')
    expect(getRefreshToken()).toBe('next-refresh')
    expect(localStorage.getItem('accessToken')).toBeNull()
    expect(localStorage.getItem('refreshToken')).toBeNull()
    expect(JSON.parse(localStorage.getItem('authTokenPairV1') ?? '{}')).toEqual({
      accessToken: 'next-access',
      refreshToken: 'next-refresh',
    })

    clearTokens()
    expect(getAccessToken()).toBeNull()
    expect(getRefreshToken()).toBeNull()
  })

  it('serializes auth-state writers when Web Locks are unavailable', async () => {
    const events: string[] = []
    let releaseFirst!: () => void
    const gate = new Promise<void>((resolve) => {
      releaseFirst = resolve
    })

    const first = withAuthStateLock(async () => {
      events.push('first:start')
      await gate
      events.push('first:end')
    })
    const second = withAuthStateLock(() => {
      events.push('second')
    })

    await vi.waitFor(() => expect(events).toEqual(['first:start']))
    releaseFirst()
    await Promise.all([first, second])
    expect(events).toEqual(['first:start', 'first:end', 'second'])
  })

  it('notifies the current tab after publishing and clearing auth state', () => {
    const observed: Array<string | null> = []
    const unsubscribe = subscribeAuthState((accessToken) => observed.push(accessToken))

    setTokens('next-access', 'next-refresh')
    clearTokens()
    unsubscribe()

    expect(observed).toEqual(['next-access', null])
  })
})
