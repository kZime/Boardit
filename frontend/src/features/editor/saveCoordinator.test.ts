import { describe, expect, it, vi } from 'vitest'

import { isDirty, saveExistingNote } from './saveCoordinator'

const details = {
  title: 'Published note',
  coverUrl: 'https://example.com/cover.png',
  description: '',
  tags: '',
  visibility: 'unlisted' as const,
}

describe('editor save coordinator', () => {
  it('persists the complete editor state and returns its saved snapshot', async () => {
    const updateNote = vi.fn().mockResolvedValue(undefined)

    const snapshot = await saveExistingNote(
      { updateNote },
      { noteID: 42, markdown: '# body', details },
    )

    expect(updateNote).toHaveBeenCalledWith(42, {
      title: 'Published note',
      cover_url: 'https://example.com/cover.png',
      content_md: '# body',
      is_published: true,
      visibility: 'unlisted',
    })
    expect(snapshot).toEqual({ md: '# body', title: 'Published note', visibility: 'unlisted' })
  })

  it('detects changes relative to the saved snapshot', () => {
    const saved = { md: '# body', title: 'Published note', visibility: 'unlisted' as const }
    expect(isDirty('# body', details, saved)).toBe(false)
    expect(isDirty('# changed', details, saved)).toBe(true)
  })
})
