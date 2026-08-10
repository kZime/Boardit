import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest'
import { setupServer } from 'msw/node'

import { notesDb, updateNoteHandler } from './notes'

const server = setupServer(updateNoteHandler)
const originalNotes = notesDb.map((note) => ({ ...note }))

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  notesDb.splice(0, notesDb.length, ...originalNotes.map((note) => ({ ...note })))
})
afterAll(() => server.close())

describe('mock update note contract', () => {
  it('rejects updates without a valid version', async () => {
    const response = await fetch('http://localhost/api/v1/notes/1', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'Unsafe mock update' }),
    })

    expect(response.status).toBe(400)
    await expect(response.json()).resolves.toMatchObject({ error: 'VALIDATION_ERROR' })
  })

  it('persists nullable folder and cover fields with optimistic concurrency', async () => {
    const version = notesDb[0].version
    const response = await fetch('http://localhost/api/v1/notes/1', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        version,
        folder_id: null,
        cover_url: 'https://example.com/updated-cover.png',
      }),
    })

    expect(response.status).toBe(200)
    await expect(response.json()).resolves.toMatchObject({
      folder_id: null,
      cover_url: 'https://example.com/updated-cover.png',
      version: version + 1,
    })
  })
})
