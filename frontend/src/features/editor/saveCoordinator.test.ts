import { describe, expect, it, vi } from 'vitest'

import { getVersionConflict, isDirty, saveExistingNote, versionForMove } from './saveCoordinator'

const details = {
  title: 'Published note',
  coverUrl: 'https://example.com/cover.png',
  description: '',
  tags: '',
  visibility: 'unlisted' as const,
}

describe('editor save coordinator', () => {
  it('persists the complete editor state and returns its saved snapshot', async () => {
    const updateNote = vi.fn().mockResolvedValue({ version: 8 })

    const snapshot = await saveExistingNote(
      { updateNote },
      { noteID: 42, markdown: '# body', details, version: 7 },
    )

    expect(updateNote).toHaveBeenCalledWith(42, {
      title: 'Published note',
      cover_url: 'https://example.com/cover.png',
      content_md: '# body',
      is_published: true,
      visibility: 'unlisted',
      version: 7,
    })
    expect(snapshot).toEqual({
      snapshot: {
        md: '# body',
        title: 'Published note',
        coverUrl: 'https://example.com/cover.png',
        visibility: 'unlisted',
      },
      version: 8,
    })
  })

  it('detects changes relative to the saved snapshot', () => {
    const saved = {
      md: '# body',
      title: 'Published note',
      coverUrl: 'https://example.com/cover.png',
      visibility: 'unlisted' as const,
    }
    expect(isDirty('# body', details, saved)).toBe(false)
    expect(isDirty('# changed', details, saved)).toBe(true)
    expect(isDirty('# body', { ...details, coverUrl: 'https://example.com/new.png' }, saved)).toBe(true)
  })

  it('extracts version conflicts from Axios errors', () => {
    const serverSnapshot = { id: 42, version: 8 }
    const conflict = {
      error: 'VERSION_CONFLICT',
      message: 'stale version',
      server_updated_at: '2026-08-10T10:00:00Z',
      server_snapshot: serverSnapshot,
    }
    const error = {
      isAxiosError: true,
      response: { status: 409, data: conflict },
    }

    expect(getVersionConflict(error)).toEqual(conflict)
    expect(getVersionConflict({ ...error, response: { status: 500, data: conflict } })).toBeNull()
  })

  it('uses the live editor version when moving the open note', () => {
    expect(versionForMove(42, 42, 9, 7)).toBe(9)
    expect(versionForMove(7, 42, 9, 3)).toBe(3)
    expect(() => versionForMove(7, 42, 9)).toThrow('current version')
  })
})
