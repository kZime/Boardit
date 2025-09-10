import { http, HttpResponse } from 'msw'
import type { Note } from '../api/gen/models/note'
import type { CreateNoteRequest } from '../api/gen/models/createNoteRequest'
import type { UpdateNoteRequest } from '../api/gen/models/updateNoteRequest'
import { NoteVisibility } from '../api/gen/models/noteVisibility'

// In-memory notes database and auto-increment ID
export const notesDb: Note[] = []
let nextNoteId = 1

const nowIso = () => new Date().toISOString()

// Seed a few example notes that match the API contract
;(() => {
  if (notesDb.length > 0) return
  const iso = (d: Date) => d.toISOString()
  const base = new Date()
  const add = (partial: Partial<Note>) => {
    const now = new Date()
    const note: Note = {
      id: nextNoteId++,
      user_id: 1,
      folder_id: partial.folder_id ?? null,
      title: partial.title ?? '',
      slug: partial.slug ?? null,
      content_md: partial.content_md ?? '',
      // For mock, mirror markdown as HTML; real API would render
      content_html: partial.content_html ?? partial.content_md ?? '',
      is_published: partial.is_published ?? false,
      visibility: (partial.visibility as Note['visibility']) ?? NoteVisibility.private,
      sort_order: partial.sort_order ?? notesDb.length,
      created_at: partial.created_at ?? iso(new Date(base.getTime() - 1000 * 60 * 60)),
      updated_at: partial.updated_at ?? iso(now),
    }
    notesDb.push(note)
  }
  add({
    title: 'Welcome to Blogedit',
    slug: 'welcome-to-blogedit',
    content_md: '# Welcome\n\nThis is your first note in Blogedit.\n\n- Edit the content\n- Change visibility\n- Save updates',
    is_published: false,
    visibility: NoteVisibility.private,
    sort_order: 0,
  })
  add({
    title: 'Public Page Example',
    slug: 'public-page-example',
    content_md: '# Public Page\n\nThis note is visible to everyone when published.',
    is_published: true,
    visibility: NoteVisibility.public,
    sort_order: 1,
  })
  add({
    title: 'Unlisted Sharing',
    slug: 'unlisted-sharing',
    content_md: '# Unlisted\n\nAnyone with the link can view this note.',
    is_published: true,
    visibility: NoteVisibility.unlisted,
    sort_order: 2,
  })
  add({
    title: 'Draft Ideas',
    slug: null,
    content_md: '# Draft\n\nScratchpad for ideas and todos.',
    is_published: false,
    visibility: NoteVisibility.private,
    sort_order: 3,
  })
})()

// Helpers
const findIndexById = (id: number) => notesDb.findIndex(n => n.id === id)

// GET /api/v1/notes?limit&offset — list
export const listNotesHandler = http.get('*/api/v1/notes', async ({ request }) => {
  const url = new URL(request.url)
  const limit = Math.max(0, Math.min(200, parseInt(url.searchParams.get('limit') || '20', 10)))
  const offset = Math.max(0, parseInt(url.searchParams.get('offset') || '0', 10))

  const items = notesDb.slice(offset, offset + limit)
  return HttpResponse.json({
    items,
    total: notesDb.length,
    limit,
    offset,
  })
})

// POST /api/v1/notes — create
export const createNoteHandler = http.post('*/api/v1/notes', async ({ request }) => {
  const body = (await request.json()) as CreateNoteRequest
  const ts = nowIso()

  const note: Note = {
    id: nextNoteId++,
    user_id: 1,
    folder_id: body.folder_id ?? null,
    title: body.title ?? '',
    slug: body.slug ?? null,
    content_md: body.content_md ?? '',
    // For mock, mirror markdown as HTML; real API would render
    content_html: body.content_md ?? '',
    is_published: false,
    visibility: NoteVisibility.private,
    sort_order: notesDb.length,
    created_at: ts,
    updated_at: ts,
  }

  notesDb.push(note)
  return new HttpResponse(JSON.stringify(note), {
    status: 201,
    headers: { 'Content-Type': 'application/json' },
  })
})

// GET /api/v1/notes/:id — get by id
export const getNoteHandler = http.get('*/api/v1/notes/:id', async ({ params }) => {
  const id = Number(params.id)
  const note = notesDb.find(n => n.id === id)
  if (!note) return new HttpResponse('Not Found', { status: 404 })
  return HttpResponse.json(note)
})

// PATCH /api/v1/notes/:id — update
export const updateNoteHandler = http.patch('*/api/v1/notes/:id', async ({ params, request }) => {
  const id = Number(params.id)
  const idx = findIndexById(id)
  if (idx === -1) return new HttpResponse('Not Found', { status: 404 })

  const patch = (await request.json()) as UpdateNoteRequest
  const prev = notesDb[idx]
  const updated: Note = {
    ...prev,
    title: patch.title ?? prev.title,
    folder_id: patch.folder_id ?? prev.folder_id,
    content_md: patch.content_md ?? prev.content_md,
    content_html: patch.content_md ?? prev.content_html,
    is_published: patch.is_published ?? prev.is_published,
    visibility: (patch.visibility as Note['visibility']) ?? prev.visibility,
    slug: patch.slug ?? prev.slug,
    updated_at: nowIso(),
  }

  notesDb[idx] = updated
  return HttpResponse.json(updated)
})

// DELETE /api/v1/notes/:id — delete
export const deleteNoteHandler = http.delete('*/api/v1/notes/:id', async ({ params }) => {
  const id = Number(params.id)
  const idx = findIndexById(id)
  if (idx === -1) return new HttpResponse('Not Found', { status: 404 })
  notesDb.splice(idx, 1)
  return new HttpResponse(null, { status: 204 })
})

export const noteMockHandlers = [
  listNotesHandler,
  createNoteHandler,
  getNoteHandler,
  updateNoteHandler,
  deleteNoteHandler,
]

