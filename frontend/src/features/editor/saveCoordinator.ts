import type { UpdateNoteRequest } from '../../api/gen/models/updateNoteRequest'
import type { PageDetails } from './types'

interface SaveInput {
  noteID: number
  markdown: string
  details: PageDetails
}

interface SaveDependencies {
  updateNote: (noteID: number, request: UpdateNoteRequest) => Promise<void>
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

export function buildUpdateRequest(markdown: string, details: PageDetails): UpdateNoteRequest {
  return {
    title: details.title,
    cover_url: details.coverUrl,
    content_md: markdown,
    is_published: details.visibility !== 'private',
    visibility: details.visibility,
  }
}

export async function saveExistingNote(
  dependencies: SaveDependencies,
  input: SaveInput,
): Promise<SavedSnapshot> {
  await dependencies.updateNote(input.noteID, buildUpdateRequest(input.markdown, input.details))
  return snapshotOf(input.markdown, input.details)
}
