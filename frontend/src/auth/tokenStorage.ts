const ACCESS_TOKEN_KEY = 'accessToken'
const REFRESH_TOKEN_KEY = 'refreshToken'
const TOKEN_PAIR_KEY = 'authTokenPairV1'
const AUTH_STATE_LOCK = 'boardit-auth-state-v1'
const AUTH_STATE_CHANGED_EVENT = 'boardit:auth-state-changed'
const AUTH_LOCK_WAIT_MS = 15_000

interface TokenPair {
  accessToken: string
  refreshToken: string
}

let fallbackLock: Promise<void> = Promise.resolve()

function getTokenPair(): TokenPair | null {
  const serialized = localStorage.getItem(TOKEN_PAIR_KEY)
  if (serialized) {
    try {
      const parsed = JSON.parse(serialized) as Partial<TokenPair>
      if (typeof parsed.accessToken === 'string' && typeof parsed.refreshToken === 'string') {
        return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken }
      }
    } catch {
      // Fall through to the legacy keys so existing sessions can migrate.
    }
  }

  const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY)
  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
  return accessToken && refreshToken ? { accessToken, refreshToken } : null
}

export function getAccessToken(): string | null {
  return getTokenPair()?.accessToken ?? null
}

export function getRefreshToken(): string | null {
  return getTokenPair()?.refreshToken ?? null
}

export function setTokens(accessToken: string, refreshToken: string): void {
  // Readers prefer this single-key bundle, so they can never observe a mixed
  // access/refresh pair while a writer publishes a new session.
  localStorage.setItem(TOKEN_PAIR_KEY, JSON.stringify({ accessToken, refreshToken }))
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  window.dispatchEvent(new Event(AUTH_STATE_CHANGED_EVENT))
}

export function clearTokens(): void {
  // Remove legacy keys first while the authoritative bundle remains readable.
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(TOKEN_PAIR_KEY)
  window.dispatchEvent(new Event(AUTH_STATE_CHANGED_EVENT))
}

export function subscribeAuthState(listener: (accessToken: string | null) => void): () => void {
  const handleChange = () => listener(getAccessToken())
  const handleStorage = (event: StorageEvent) => {
    if (
      event.storageArea === localStorage &&
      [TOKEN_PAIR_KEY, ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY].includes(event.key ?? '')
    ) {
      handleChange()
    }
  }
  window.addEventListener(AUTH_STATE_CHANGED_EVENT, handleChange)
  window.addEventListener('storage', handleStorage)
  return () => {
    window.removeEventListener(AUTH_STATE_CHANGED_EVENT, handleChange)
    window.removeEventListener('storage', handleStorage)
  }
}

export async function withAuthStateLock<T>(operation: () => Promise<T> | T): Promise<T> {
  if (typeof navigator !== 'undefined' && navigator.locks) {
    const controller = new AbortController()
    const timeout = window.setTimeout(
      () => controller.abort(new Error('timed out waiting for authentication state lock')),
      AUTH_LOCK_WAIT_MS,
    )
    try {
      return await navigator.locks.request(
        AUTH_STATE_LOCK,
        { signal: controller.signal },
        async () => {
          // The acquisition timeout must not abort an operation after it owns
          // the lock. Network calls inside the callback have their own timeout.
          window.clearTimeout(timeout)
          return operation()
        },
      )
    } finally {
      window.clearTimeout(timeout)
    }
  }

  // Web Locks are available in supported browsers. This queue keeps tests and
  // non-browser runtimes deterministic, but cannot coordinate separate tabs.
  const previous = fallbackLock
  let release!: () => void
  fallbackLock = new Promise<void>((resolve) => {
    release = resolve
  })
  await previous
  try {
    return await operation()
  } finally {
    release()
  }
}
