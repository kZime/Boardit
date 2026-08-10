import axios from 'axios'

import type { UpdateNoteRequest } from '../../api/gen/models/updateNoteRequest'
import type { Note } from '../../api/gen/models/note'
import type { VersionConflictError } from '../../api/gen/models/versionConflictError'
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
  coverUrl: string
  visibility: PageDetails['visibility']
}

export function snapshotOf(markdown: string, details: PageDetails): SavedSnapshot {
  return {
    md: markdown,
    title: details.title,
    coverUrl: details.coverUrl,
    visibility: details.visibility,
  }
}

export function isDirty(markdown: string, details: PageDetails, saved: SavedSnapshot): boolean {
  return (
    markdown !== saved.md ||
    details.title !== saved.title ||
    details.coverUrl !== saved.coverUrl ||
    details.visibility !== saved.visibility
  )
}

export function shouldReplaceEditorContent(
  dirty: boolean,
  confirmDiscard: () => boolean,
): boolean {
  return !dirty || confirmDiscard()
}

export function getVersionConflict(error: unknown): VersionConflictError | null {
  if (!axios.isAxiosError<VersionConflictError>(error) || error.response?.status !== 409) {
    return null
  }
  const conflict = error.response.data
  if (conflict?.error !== 'VERSION_CONFLICT' || !conflict.server_snapshot) {
    return null
  }
  return conflict
}

export function versionForMove(
  noteID: number,
  currentNoteID: number | null,
  currentVersion: number | null,
  listedVersion?: number,
): number {
  const version = noteID === currentNoteID ? currentVersion : listedVersion
  if (version === null || version === undefined) {
    throw new Error('Cannot move a note without its current version')
  }
  return version
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
