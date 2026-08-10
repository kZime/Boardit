import axios, { type AxiosError, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  authResponseInterceptor,
  resetAuthRefreshStateForTests,
} from './axios'

function unauthorizedError(adapter?: AxiosRequestConfig['adapter']): AxiosError {
  return {
    config: {
      adapter,
      headers: {},
      method: 'get',
      url: '/api/v1/notes',
    },
    response: { status: 401 },
  } as AxiosError
}

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
  resetAuthRefreshStateForTests()
})

describe('401 refresh coordination', () => {
  it('rejects every waiting request when the shared refresh fails', async () => {
    localStorage.setItem('refreshToken', 'expired-refresh')
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
    expect(localStorage.getItem('refreshToken')).toBeNull()
  })

  it('retries all waiting requests with one refreshed access token', async () => {
    localStorage.setItem('refreshToken', 'valid-refresh')
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
    resolveRefresh({ data: { access_token: 'new-access', refresh_token: 'new-refresh' } })

    await expect(first).resolves.toMatchObject({ status: 200 })
    await expect(second).resolves.toMatchObject({ status: 200 })
    expect(adapter).toHaveBeenCalledTimes(2)
    expect(localStorage.getItem('accessToken')).toBe('new-access')
    expect(localStorage.getItem('refreshToken')).toBe('new-refresh')
  })
})
