// src/api/axios.ts
import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { getAccessToken, setAccessToken, getRefreshToken } from '../contexts/AuthContext'

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

// ===== 401 refresh: skip auth path; no refresh_token; no hard redirect =====
let isRefreshing = false
let subscribers: Array<(token: string) => void> = []

function onRefreshed(token: string) {
  subscribers.forEach((cb) => cb(token))
  subscribers = []
}

interface RetryableConfig {
  url?: string
  headers?: Record<string, string>
  _retry?: boolean
}

const authResponseInterceptor = async (err: AxiosError) => {
  const status = err?.response?.status
  const config: RetryableConfig = (err?.config as RetryableConfig) ?? {}

  // not 401 or already retried: pass to upper layer
  if (status !== 401 || config._retry) return Promise.reject(err)

  // auth routes: no auto refresh, pass to upper layer
  if (isAuthPath(config.url)) return Promise.reject(err)

  // non-auth request: try refresh, but must have refresh_token
  const rt = getRefreshToken()
  if (!rt) {
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    return Promise.reject(err)
  }

  if (!isRefreshing) {
    isRefreshing = true
    try {
      const { data } = await axios.post(
        '/api/auth/refresh',
        { refresh_token: rt },
        { baseURL: api.defaults.baseURL }
      )
      if (!data?.access_token) throw new Error('bad refresh response')

      setAccessToken(data.access_token)
      if (data.refresh_token) {
        localStorage.setItem('refreshToken', data.refresh_token)
      }
      onRefreshed(data.access_token)
    } catch (e) {
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')
      return Promise.reject(e)
    } finally {
      isRefreshing = false
    }
  }

  // suspend current request, retry after refresh
  return new Promise((resolve) => {
    subscribers.push((newToken: string) => {
      if (err.config) {
        Object.assign(err.config.headers, { Authorization: `Bearer ${newToken}` })
        ;(err.config as InternalAxiosRequestConfig & { _retry?: boolean })._retry = true
        resolve(api(err.config))
      } else {
        resolve(Promise.reject(err))
      }
    })
  })
}

api.interceptors.response.use((res) => res, authResponseInterceptor)

// Also apply to global axios instance (used by orval-generated code)
axios.interceptors.response.use((res) => res, authResponseInterceptor)

export default api
