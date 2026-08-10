import axios, { type AxiosError, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  authResponseInterceptor,
  default as api,
  resetAuthRefreshStateForTests,
} from './axios'
import { getAccessToken, getRefreshToken, subscribeAuthState } from '../auth/tokenStorage'

function unauthorizedError(
  adapter?: AxiosRequestConfig['adapter'],
  accessToken = getAccessToken(),
): AxiosError {
  return {
    config: {
      adapter,
      headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
      method: 'get',
      url: '/api/v1/notes',
    },
    response: { status: 401 },
  } as AxiosError
}

function tokenFor(subject: string, marker: string): string {
  const payload = btoa(JSON.stringify({ sub: subject }))
    .replaceAll('+', '-')
    .replaceAll('/', '_')
    .replace(/=+$/, '')
  return `e30.${payload}.${marker}`
}

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
  resetAuthRefreshStateForTests()
})

describe('401 refresh coordination', () => {
  it('rejects every waiting request when the shared refresh fails', async () => {
    const expiredAccess = tokenFor('user-1', 'expired-access')
    const expiredRefresh = tokenFor('user-1', 'expired-refresh')
    localStorage.setItem('accessToken', expiredAccess)
    localStorage.setItem('refreshToken', expiredRefresh)
    let rejectRefresh!: (error: Error) => void
    const refresh = new Promise<never>((_, reject) => {
      rejectRefresh = reject
    })
    const post = vi.spyOn(axios, 'post').mockReturnValue(refresh)

    const first = authResponseInterceptor(unauthorizedError())
    const second = authResponseInterceptor(unauthorizedError())
    await vi.waitFor(() => expect(post).toHaveBeenCalledTimes(1))
    rejectRefresh(new Error('refresh failed'))

    await expect(first).rejects.toThrow('refresh failed')
    await expect(second).rejects.toThrow('refresh failed')
    expect(getRefreshToken()).toBe(expiredRefresh)
  })

  it('clears and publishes signed-out state after the current refresh token is rejected', async () => {
    const expiredAccess = tokenFor('user-1', 'expired-access')
    const expiredRefresh = tokenFor('user-1', 'expired-refresh')
    localStorage.setItem('accessToken', expiredAccess)
    localStorage.setItem('refreshToken', expiredRefresh)
    const refreshError = Object.assign(new Error('refresh rejected'), {
      isAxiosError: true,
      response: { status: 401 },
    })
    vi.spyOn(axios, 'post').mockRejectedValue(refreshError)
    const observed: Array<string | null> = []
    const unsubscribe = subscribeAuthState((accessToken) => observed.push(accessToken))

    try {
      await expect(authResponseInterceptor(unauthorizedError())).rejects.toBe(refreshError)
      expect(getAccessToken()).toBeNull()
      expect(getRefreshToken()).toBeNull()
      expect(observed).toContain(null)
    } finally {
      unsubscribe()
    }
  })

  it('preserves the token bundle after a transient refresh server error', async () => {
    const access = tokenFor('user-1', 'access')
    const refresh = tokenFor('user-1', 'refresh')
    localStorage.setItem('accessToken', access)
    localStorage.setItem('refreshToken', refresh)
    const refreshError = Object.assign(new Error('refresh unavailable'), {
      isAxiosError: true,
      response: { status: 503 },
    })
    vi.spyOn(axios, 'post').mockRejectedValue(refreshError)

    await expect(authResponseInterceptor(unauthorizedError())).rejects.toBe(refreshError)
    expect(getAccessToken()).toBe(access)
    expect(getRefreshToken()).toBe(refresh)
  })

  it('retries all waiting requests with one refreshed access token', async () => {
    const oldAccess = tokenFor('user-1', 'old-access')
    const oldRefresh = tokenFor('user-1', 'old-refresh')
    const newAccess = tokenFor('user-1', 'new-access')
    const newRefresh = tokenFor('user-1', 'new-refresh')
    localStorage.setItem('accessToken', oldAccess)
    localStorage.setItem('refreshToken', oldRefresh)
    let resolveRefresh!: (response: { data: Record<string, string> }) => void
    const refresh = new Promise<{ data: Record<string, string> }>((resolve) => {
      resolveRefresh = resolve
    })
    const post = vi.spyOn(axios, 'post').mockReturnValue(refresh)
    const adapter = vi.fn(async (config: AxiosRequestConfig): Promise<AxiosResponse> => ({
      config: config as AxiosResponse['config'],
      data: { ok: true },
      headers: {},
      status: 200,
      statusText: 'OK',
    }))

    const first = authResponseInterceptor(unauthorizedError(adapter))
    const second = authResponseInterceptor(unauthorizedError(adapter))
    await vi.waitFor(() => expect(post).toHaveBeenCalledTimes(1))
    resolveRefresh({ data: { access_token: newAccess, refresh_token: newRefresh } })

    await expect(first).resolves.toMatchObject({ status: 200 })
    await expect(second).resolves.toMatchObject({ status: 200 })
    expect(adapter).toHaveBeenCalledTimes(2)
    expect(getAccessToken()).toBe(newAccess)
    expect(getRefreshToken()).toBe(newRefresh)
  })

  it('preserves and reuses a token pair rotated by another tab', async () => {
    const oldRefresh = tokenFor('user-1', 'old-refresh')
    const winnerAccess = tokenFor('user-1', 'winner-access')
    const winnerRefresh = tokenFor('user-1', 'winner-refresh')
    localStorage.setItem('accessToken', tokenFor('user-1', 'expired-access'))
    localStorage.setItem('refreshToken', oldRefresh)
    let rejectRefresh!: (error: Error) => void
    const refresh = new Promise<never>((_, reject) => {
      rejectRefresh = reject
    })
    vi.spyOn(axios, 'post').mockReturnValue(refresh)
    const adapter = vi.fn(async (config: AxiosRequestConfig): Promise<AxiosResponse> => ({
      config: config as AxiosResponse['config'],
      data: { ok: true },
      headers: {},
      status: 200,
      statusText: 'OK',
    }))

    const request = authResponseInterceptor(unauthorizedError(adapter))
    await vi.waitFor(() => expect(axios.post).toHaveBeenCalledTimes(1))

    // Simulate another browser tab winning rotation and updating shared storage.
    localStorage.setItem('accessToken', winnerAccess)
    localStorage.setItem('refreshToken', winnerRefresh)
    rejectRefresh(new Error('concurrent refresh lost'))

    await expect(request).resolves.toMatchObject({ status: 200 })
    expect(adapter).toHaveBeenCalledOnce()
    expect(getAccessToken()).toBe(winnerAccess)
    expect(getRefreshToken()).toBe(winnerRefresh)
  })

  it('does not replay a request after another account replaces shared tokens', async () => {
    const oldRefresh = tokenFor('user-1', 'old-refresh')
    const otherAccess = tokenFor('user-2', 'other-access')
    const otherRefresh = tokenFor('user-2', 'other-refresh')
    localStorage.setItem('accessToken', tokenFor('user-1', 'expired-access'))
    localStorage.setItem('refreshToken', oldRefresh)
    let rejectRefresh!: (error: Error) => void
    const refresh = new Promise<never>((_, reject) => {
      rejectRefresh = reject
    })
    vi.spyOn(axios, 'post').mockReturnValue(refresh)
    const adapter = vi.fn()

    const request = authResponseInterceptor(unauthorizedError(adapter))
    await vi.waitFor(() => expect(axios.post).toHaveBeenCalledTimes(1))
    localStorage.setItem('accessToken', otherAccess)
    localStorage.setItem('refreshToken', otherRefresh)
    rejectRefresh(new Error('authentication state changed'))

    await expect(request).rejects.toThrow('authentication state changed')
    expect(adapter).not.toHaveBeenCalled()
    expect(getAccessToken()).toBe(otherAccess)
    expect(getRefreshToken()).toBe(otherRefresh)
  })

  it('does not refresh a request whose original access token belongs to another account', async () => {
    const originalAccess = tokenFor('user-1', 'original-access')
    const currentAccess = tokenFor('user-2', 'current-access')
    const currentRefresh = tokenFor('user-2', 'current-refresh')
    localStorage.setItem('accessToken', currentAccess)
    localStorage.setItem('refreshToken', currentRefresh)
    const post = vi.spyOn(axios, 'post')
    const adapter = vi.fn()
    const error = unauthorizedError(adapter, originalAccess)

    await expect(authResponseInterceptor(error)).rejects.toBe(error)
    expect(post).not.toHaveBeenCalled()
    expect(adapter).not.toHaveBeenCalled()
  })

  it('keeps the validated subject if shared tokens change before retry dispatch', async () => {
    const oldAccess = tokenFor('user-1', 'old-access')
    const oldRefresh = tokenFor('user-1', 'old-refresh')
    const winnerAccess = tokenFor('user-1', 'winner-access')
    const winnerRefresh = tokenFor('user-1', 'winner-refresh')
    const otherAccess = tokenFor('user-2', 'other-access')
    const otherRefresh = tokenFor('user-2', 'other-refresh')
    localStorage.setItem('accessToken', oldAccess)
    localStorage.setItem('refreshToken', oldRefresh)
    let rejectRefresh!: (error: Error) => void
    const refresh = new Promise<never>((_, reject) => {
      rejectRefresh = reject
    })
    vi.spyOn(axios, 'post').mockReturnValue(refresh)
    const adapter = vi.fn(async (config: AxiosRequestConfig): Promise<AxiosResponse> => ({
      config: config as AxiosResponse['config'],
      data: { ok: true },
      headers: {},
      status: 200,
      statusText: 'OK',
    }))
    const switchAccountBeforeAuthInjection = api.interceptors.request.use((config) => {
      localStorage.setItem('accessToken', otherAccess)
      localStorage.setItem('refreshToken', otherRefresh)
      return config
    })

    try {
      const request = authResponseInterceptor(unauthorizedError(adapter))
      await vi.waitFor(() => expect(axios.post).toHaveBeenCalledTimes(1))
      localStorage.setItem('accessToken', winnerAccess)
      localStorage.setItem('refreshToken', winnerRefresh)
      rejectRefresh(new Error('concurrent refresh lost'))

      await expect(request).resolves.toMatchObject({ status: 200 })
      expect(adapter).toHaveBeenCalledOnce()
      expect(adapter.mock.calls[0][0].headers?.Authorization).toBe(`Bearer ${winnerAccess}`)
      expect(getAccessToken()).toBe(otherAccess)
      expect(getRefreshToken()).toBe(otherRefresh)
    } finally {
      api.interceptors.request.eject(switchAccountBeforeAuthInjection)
    }
  })

  it('does not join an in-flight refresh that belongs to an older token', async () => {
    const oldRefresh = tokenFor('user-1', 'old-refresh')
    const currentRefresh = tokenFor('user-1', 'current-refresh')
    const currentAccess = tokenFor('user-1', 'current-access')
    const nextRefresh = tokenFor('user-1', 'next-refresh')
    const nextAccess = tokenFor('user-1', 'next-access')
    localStorage.setItem('accessToken', tokenFor('user-1', 'old-access'))
    localStorage.setItem('refreshToken', oldRefresh)
    let rejectOldRefresh!: (error: Error) => void
    const oldRequest = new Promise<never>((_, reject) => {
      rejectOldRefresh = reject
    })
    const post = vi.spyOn(axios, 'post').mockImplementation((_url, body) => {
      const refreshToken = (body as { refresh_token: string }).refresh_token
      if (refreshToken === oldRefresh) return oldRequest
      return Promise.resolve({ data: { access_token: nextAccess, refresh_token: nextRefresh } })
    })
    const adapter = vi.fn(async (config: AxiosRequestConfig): Promise<AxiosResponse> => ({
      config: config as AxiosResponse['config'],
      data: { ok: true },
      headers: {},
      status: 200,
      statusText: 'OK',
    }))

    const first = authResponseInterceptor(unauthorizedError(adapter))
    await vi.waitFor(() => expect(post).toHaveBeenCalledTimes(1))
    localStorage.setItem('accessToken', currentAccess)
    localStorage.setItem('refreshToken', currentRefresh)
    const second = authResponseInterceptor(unauthorizedError(adapter))
    rejectOldRefresh(new Error('old refresh lost'))
    await vi.waitFor(() => expect(post).toHaveBeenCalledTimes(2))

    await expect(second).resolves.toMatchObject({ status: 200 })
    await expect(first).resolves.toMatchObject({ status: 200 })
    expect(getAccessToken()).toBe(nextAccess)
    expect(getRefreshToken()).toBe(nextRefresh)
  })
})
