import { describe, expect, it, vi } from 'vitest'

import type { Note } from '../../api/gen/models/note'
import { loadAllNotes } from './useAllNotes'

const makeNote = (id: number): Note => ({
  id,
  user_id: 1,
  folder_id: null,
  title: `Note ${id}`,
  slug: `note-${id}`,
  cover_url: null,
  content_md: '',
  content_html: '',
  is_published: false,
  visibility: 'private',
  sort_order: id,
  created_at: '2026-08-10T00:00:00Z',
  updated_at: '2026-08-10T00:00:00Z',
})

describe('loadAllNotes', () => {
  it('loads every page instead of truncating the editor tree', async () => {
    const notes = Array.from({ length: 450 }, (_, index) => makeNote(index + 1))
    const loader = vi.fn(async (limit: number, offset: number) => ({
      items: notes.slice(offset, offset + limit),
      total: notes.length,
      limit,
      offset,
    }))

    const result = await loadAllNotes(loader)

    expect(result.items).toHaveLength(450)
    expect(loader).toHaveBeenCalledTimes(3)
    expect(loader).toHaveBeenLastCalledWith(200, 400)
  })
})
