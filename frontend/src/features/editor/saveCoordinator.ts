import type { UpdateNoteRequest } from '../../api/gen/models/updateNoteRequest'
import type { Note } from '../../api/gen/models/note'
import type { PageDetails } from './types'

interface SaveInput {
  noteID: number
  markdown: string
  details: PageDetails
  version: number
}

interface SaveDependencies {
  updateNote: (noteID: number, request: UpdateNoteRequest) => Promise<Note>
}

export interface SavedSnapshot {
  md: string
  title: string
  visibility: PageDetails['visibility']
}

export function snapshotOf(markdown: string, details: PageDetails): SavedSnapshot {
  return {
    md: markdown,
    title: details.title,
    visibility: details.visibility,
  }
}

export function isDirty(markdown: string, details: PageDetails, saved: SavedSnapshot): boolean {
  return (
    markdown !== saved.md ||
    details.title !== saved.title ||
    details.visibility !== saved.visibility
  )
}

export function buildUpdateRequest(markdown: string, details: PageDetails, version: number): UpdateNoteRequest {
  return {
    title: details.title,
    cover_url: details.coverUrl,
    content_md: markdown,
    is_published: details.visibility !== 'private',
    visibility: details.visibility,
    version,
  }
}

export async function saveExistingNote(
  dependencies: SaveDependencies,
  input: SaveInput,
): Promise<{ snapshot: SavedSnapshot; version: number }> {
  const saved = await dependencies.updateNote(
    input.noteID,
    buildUpdateRequest(input.markdown, input.details, input.version),
  )
  return { snapshot: snapshotOf(input.markdown, input.details), version: saved.version }
}
