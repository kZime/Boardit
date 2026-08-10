const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const USERNAME_RE = /^[A-Za-z0-9][A-Za-z0-9_-]{2,31}$/

export const MIN_PASSWORD_LENGTH = 8

export interface LoginFields {
  email: string
  password: string
}

export interface RegistrationFields extends LoginFields {
  username: string
}

type ValidationResult<T> =
  | { ok: true; value: T }
  | { ok: false; error: string }

function normalizeLogin(fields: LoginFields): LoginFields {
  return {
    email: fields.email.trim(),
    password: fields.password.trim(),
  }
}

export function validateLogin(fields: LoginFields): ValidationResult<LoginFields> {
  const value = normalizeLogin(fields)
  if (!value.email) return { ok: false, error: 'Email is required' }
  if (!EMAIL_RE.test(value.email)) return { ok: false, error: 'Please enter a valid email address' }
  if (!value.password) return { ok: false, error: 'Password is required' }
  return { ok: true, value }
}

export function validateRegistration(
  fields: RegistrationFields,
): ValidationResult<RegistrationFields> {
  const login = normalizeLogin(fields)
  const value = { ...login, username: fields.username.trim() }
  if (!value.username) return { ok: false, error: 'Username is required' }
  if (!USERNAME_RE.test(value.username)) {
    return { ok: false, error: 'Username must be 3-32 letters, numbers, underscores, or hyphens' }
  }
  const loginResult = validateLogin(value)
  if (!loginResult.ok) return loginResult
  if (value.password.length < MIN_PASSWORD_LENGTH) {
    return { ok: false, error: `Password must be at least ${MIN_PASSWORD_LENGTH} characters` }
  }
  return { ok: true, value }
}

export function authErrorMessage(error: unknown, fallback: string): string {
  if (typeof error !== 'object' || error === null || !('response' in error)) return fallback
  const responseError = error as { response?: { data?: { error?: unknown } } }
  return typeof responseError.response?.data?.error === 'string'
    ? responseError.response.data.error
    : fallback
}
