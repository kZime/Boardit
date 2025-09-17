import { setupWorker } from 'msw/browser'
import { http, HttpResponse } from 'msw'
import { noteMockHandlers } from './notes'
import {
  getCreateFolderMockHandler,
  getUpdateFolderMockHandler,
  getDeleteFolderMockHandler,
  getReorderTreeMockHandler,
} from '../api/gen/client'
import { FAKE_ACCESS, FAKE_REFRESH } from './tokens'

// replace the real handlers with fake ones
// FOR DEBUG ONLY
const mocks = [
  // Custom notes handlers with in-memory DB
  ...noteMockHandlers,
  // Keep other API mocks as generated
  getCreateFolderMockHandler(),
  getUpdateFolderMockHandler(),
  getDeleteFolderMockHandler(),
  getReorderTreeMockHandler(),
]

// override auth: return "valid" JWT
const login = http.post('*/api/auth/login', async () =>
  HttpResponse.json({ access_token: FAKE_ACCESS, refresh_token: FAKE_REFRESH, expires_in: 3600 })
)
const refresh = http.post('*/api/auth/refresh', async () =>
  HttpResponse.json({ access_token: FAKE_ACCESS, refresh_token: FAKE_REFRESH, expires_in: 3600 })
)

// order matters: custom auth handlers at the end, highest priority
export const worker = setupWorker(...mocks, login, refresh)
