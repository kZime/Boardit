// src/api/axios.ts
import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setAccessToken,
  setRefreshToken,
} from '../auth/tokenStorage'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/',
})

const isAuthPath = (url?: string) =>
  !!url && /^\/?api\/(v1\/)?auth\//.test(url)

const authRequestInterceptor = (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
  const t = getAccessToken()
  if (t) {
    Object.assign(config.headers, { Authorization: `Bearer ${t}` })
  }
  return config
}

// Apply to custom instance
api.interceptors.request.use(authRequestInterceptor)

// Also apply to global axios instance (used by orval-generated code)
axios.interceptors.request.use(authRequestInterceptor)

// ===== 401 refresh: skip auth path; share one refresh promise across callers =====
let refreshPromise: Promise<string> | null = null

interface RetryableConfig {
  url?: string
  headers?: Record<string, string>
  _retry?: boolean
}

function refreshAccessToken(refreshToken: string): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = axios
      .post(
        '/api/auth/refresh',
        { refresh_token: refreshToken },
        { baseURL: api.defaults.baseURL },
      )
      .then(({ data }) => {
        if (!data?.access_token) throw new Error('bad refresh response')
        setAccessToken(data.access_token)
        if (data.refresh_token) {
          setRefreshToken(data.refresh_token)
        }
        return data.access_token as string
      })
      .catch((error) => {
        clearTokens()
        throw error
      })
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
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
    clearTokens()
    return Promise.reject(err)
  }

  let newToken: string
  try {
    newToken = await refreshAccessToken(rt)
  } catch (error) {
    return Promise.reject(error)
  }

  if (!err.config) return Promise.reject(err)
  Object.assign(err.config.headers, { Authorization: `Bearer ${newToken}` })
  ;(err.config as InternalAxiosRequestConfig & { _retry?: boolean })._retry = true
  return api(err.config)
}

export function resetAuthRefreshStateForTests() {
  refreshPromise = null
}

api.interceptors.response.use((res) => res, authResponseInterceptor)

// Also apply to global axios instance (used by orval-generated code)
axios.interceptors.response.use((res) => res, authResponseInterceptor)

export default api
