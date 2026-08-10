// src/api/axios.ts
import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { jwtDecode } from 'jwt-decode'
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
  withAuthStateLock,
} from '../auth/tokenStorage'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/',
})

export const AUTH_HTTP_TIMEOUT_MS = 10_000

const isAuthPath = (url?: string) =>
  !!url && /^\/?api\/(v1\/)?auth\//.test(url)

interface AuthenticatedRequestConfig extends InternalAxiosRequestConfig {
  _authTokenOverride?: string
  _retry?: boolean
}

const authRequestInterceptor = (config: AuthenticatedRequestConfig): InternalAxiosRequestConfig => {
  // A retried request must keep the token whose subject was already validated.
  // Re-reading shared localStorage here could switch the request to another
  // account if a different tab logs in between validation and dispatch.
  const t = config._authTokenOverride ?? getAccessToken()
  if (t) {
    Object.assign(config.headers, { Authorization: `Bearer ${t}` })
  }
  return config
}

// Apply to custom instance
api.interceptors.request.use(authRequestInterceptor)

// Also apply to global axios instance (used by orval-generated code)
axios.interceptors.request.use(authRequestInterceptor)

// ===== 401 refresh: skip auth path; share one promise per refresh token =====
interface RefreshedTokens {
  accessToken: string
  refreshToken: string
}

const refreshPromises = new Map<string, Promise<RefreshedTokens>>()

function haveSameSubject(...tokens: string[]): boolean {
  try {
    const subjects = tokens.map((token) => jwtDecode<{ sub?: string | number }>(token).sub)
    return subjects[0] !== undefined && subjects.every((subject) => String(subject) === String(subjects[0]))
  } catch {
    return false
  }
}

interface RetryableConfig {
  url?: string
  headers?: {
    Authorization?: string
    authorization?: string
    get?: (name: string) => unknown
  }
  _authTokenOverride?: string
  _retry?: boolean
}

function getBearerToken(headers: RetryableConfig['headers']): string | null {
  const value = typeof headers?.get === 'function'
    ? headers.get('Authorization')
    : headers?.Authorization ?? headers?.authorization
  if (typeof value !== 'string') return null
  const match = /^Bearer\s+(.+)$/i.exec(value)
  return match?.[1] ?? null
}

function refreshAccessToken(refreshToken: string): Promise<RefreshedTokens> {
  const existing = refreshPromises.get(refreshToken)
  if (existing) return existing

  const promise = withAuthStateLock(async () => {
    const currentAccessToken = getAccessToken()
    const currentRefreshToken = getRefreshToken()
    if (currentRefreshToken !== refreshToken) {
      if (
        currentAccessToken &&
        currentRefreshToken &&
        haveSameSubject(refreshToken, currentAccessToken, currentRefreshToken)
      ) {
        return { accessToken: currentAccessToken, refreshToken: currentRefreshToken }
      }
      throw new Error('authentication state changed before refresh')
    }

    let data: Record<string, unknown>
    try {
      const response = await axios.post(
        '/api/auth/refresh',
        { refresh_token: refreshToken },
        { baseURL: api.defaults.baseURL, timeout: AUTH_HTTP_TIMEOUT_MS },
      )
      data = response.data as Record<string, unknown>
    } catch (error) {
      // A deterministic refresh rejection means this tab's current session is
      // unusable. Network failures and server errors remain retryable and must
      // not sign the user out.
      if (
        axios.isAxiosError(error) &&
        error.response?.status === 401 &&
        getRefreshToken() === refreshToken
      ) {
        clearTokens()
      }
      throw error
    }
    if (!data?.access_token || !data?.refresh_token) {
      throw new Error('bad refresh response')
    }
    const refreshed = {
      accessToken: data.access_token as string,
      refreshToken: data.refresh_token as string,
    }
    if (!haveSameSubject(refreshToken, refreshed.accessToken, refreshed.refreshToken)) {
      throw new Error('refresh response changed authentication subject')
    }
    setTokens(refreshed.accessToken, refreshed.refreshToken)
    return refreshed
  }).finally(() => {
    if (refreshPromises.get(refreshToken) === promise) {
      refreshPromises.delete(refreshToken)
    }
  })
  refreshPromises.set(refreshToken, promise)
  return promise
}

export const authResponseInterceptor = async (err: AxiosError) => {
  const status = err?.response?.status
  const config: RetryableConfig = (err?.config as RetryableConfig) ?? {}

  // not 401 or already retried: pass to upper layer
  if (status !== 401 || config._retry) return Promise.reject(err)

  // auth routes: no auto refresh, pass to upper layer
  if (isAuthPath(config.url)) return Promise.reject(err)

  // non-auth request: try refresh, but must have refresh_token
  const rt = getRefreshToken()
  if (!rt) {
    return Promise.reject(err)
  }
  const originalAccessToken = getBearerToken(config.headers)
  if (!originalAccessToken || !haveSameSubject(originalAccessToken, rt)) {
    return Promise.reject(err)
  }

  let newToken: string
  try {
    const refreshed = await refreshAccessToken(rt)
    if (!haveSameSubject(originalAccessToken, rt, refreshed.accessToken, refreshed.refreshToken)) {
      return Promise.reject(new Error('refresh response changed authentication subject'))
    }
    newToken = refreshed.accessToken
  } catch (error) {
    // localStorage is shared across tabs. If another tab won the same-token
    // refresh race, reuse its replacement access token for this request.
    const replacementRefreshToken = getRefreshToken()
    const replacementAccessToken = getAccessToken()
    if (
      replacementRefreshToken &&
      replacementRefreshToken !== rt &&
      replacementAccessToken &&
      haveSameSubject(
        originalAccessToken,
        rt,
        replacementRefreshToken,
        replacementAccessToken,
      )
    ) {
      newToken = replacementAccessToken
    } else {
      return Promise.reject(error)
    }
  }

  if (!err.config) return Promise.reject(err)
  Object.assign(err.config.headers, { Authorization: `Bearer ${newToken}` })
  const retryConfig = err.config as AuthenticatedRequestConfig
  retryConfig._authTokenOverride = newToken
  retryConfig._retry = true
  return api(err.config)
}

export function resetAuthRefreshStateForTests() {
  refreshPromises.clear()
}

api.interceptors.response.use((res) => res, authResponseInterceptor)

// Also apply to global axios instance (used by orval-generated code)
axios.interceptors.response.use((res) => res, authResponseInterceptor)

export default api
