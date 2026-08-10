import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import SaveConflictDialog from './SaveConflictDialog'

const serverNote = {
  id: 1,
  user_id: 1,
  title: 'Server title',
  cover_url: 'https://example.com/server-cover.png',
  content_md: 'Server body',
  content_html: '<p>Server body</p>',
  is_published: false,
  visibility: 'private' as const,
  sort_order: 0,
  version: 4,
  created_at: '2026-08-10T10:00:00Z',
  updated_at: '2026-08-10T11:00:00Z',
}

describe('SaveConflictDialog', () => {
  it('preserves both choices and requires an explicit resolution', async () => {
    const user = userEvent.setup()
    const onKeepEditing = vi.fn()
    const onLoadServer = vi.fn()
    const onOverwrite = vi.fn()

    render(
      <SaveConflictDialog
        serverNote={serverNote}
        localTitle="Local title"
        localMarkdown="Local body"
        localCoverUrl="https://example.com/local-cover.png"
        localVisibility="unlisted"
        isSaving={false}
        onKeepEditing={onKeepEditing}
        onLoadServer={onLoadServer}
        onOverwrite={onOverwrite}
      />,
    )

    expect(screen.getByText('Local body')).toBeInTheDocument()
    expect(screen.getByText('Server body')).toBeInTheDocument()
    expect(screen.getByTitle('https://example.com/local-cover.png')).toBeInTheDocument()
    expect(screen.getByTitle('https://example.com/server-cover.png')).toBeInTheDocument()
    expect(screen.getByText('unlisted')).toBeInTheDocument()
    expect(screen.getByText('private')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Keep editing' }))
    await user.click(screen.getByRole('button', { name: 'Load server version' }))
    await user.click(screen.getByRole('button', { name: 'Overwrite with my draft' }))

    expect(onKeepEditing).toHaveBeenCalledOnce()
    expect(onLoadServer).toHaveBeenCalledOnce()
    expect(onOverwrite).toHaveBeenCalledOnce()
  })
})
