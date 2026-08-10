import { AlertTriangle } from 'lucide-react'

import type { Note } from '../../api/gen/models/note'
import type { PageDetails } from './types'

interface SaveConflictDialogProps {
  serverNote: Note
  localTitle: string
  localMarkdown: string
  localCoverUrl: string
  localVisibility: PageDetails['visibility']
  isSaving: boolean
  onKeepEditing: () => void
  onLoadServer: () => void
  onOverwrite: () => void
}

export default function SaveConflictDialog({
  serverNote,
  localTitle,
  localMarkdown,
  localCoverUrl,
  localVisibility,
  isSaving,
  onKeepEditing,
  onLoadServer,
  onOverwrite,
}: SaveConflictDialogProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        className="w-full max-w-2xl rounded-xl border border-amber-200 bg-white shadow-xl dark:border-amber-900 dark:bg-gray-800"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="save-conflict-title"
        aria-describedby="save-conflict-description"
      >
        <div className="flex items-start gap-3 border-b border-gray-200 p-6 dark:border-gray-700">
          <AlertTriangle className="mt-0.5 h-6 w-6 shrink-0 text-amber-500" />
          <div>
            <h3 id="save-conflict-title" className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              This note changed elsewhere
            </h3>
            <p id="save-conflict-description" className="mt-1 text-sm text-gray-600 dark:text-gray-300">
              Your draft is still in the editor. Compare both versions before choosing which one to keep.
            </p>
          </div>
        </div>

        <div className="grid gap-4 p-6 md:grid-cols-2">
          <section className="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-900 dark:bg-blue-950/30">
            <h4 className="font-medium text-gray-900 dark:text-gray-100">Your draft</h4>
            <p className="mt-2 truncate text-sm font-medium text-gray-700 dark:text-gray-200">{localTitle || 'Untitled'}</p>
            <p className="mt-1 line-clamp-5 whitespace-pre-wrap text-xs text-gray-600 dark:text-gray-400">
              {localMarkdown || 'Empty note'}
            </p>
            <dl className="mt-3 space-y-1 border-t border-blue-200 pt-3 text-xs text-gray-600 dark:border-blue-900 dark:text-gray-300">
              <div><dt className="inline font-medium">Visibility:</dt> <dd className="inline">{localVisibility}</dd></div>
              <div><dt className="inline font-medium">Publishing:</dt> <dd className="inline">{localVisibility === 'private' ? 'unpublished' : 'published'}</dd></div>
              <div className="truncate" title={localCoverUrl || 'No cover'}>
                <dt className="inline font-medium">Cover:</dt> <dd className="inline">{localCoverUrl || 'No cover'}</dd>
              </div>
            </dl>
          </section>
          <section className="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-900/40">
            <h4 className="font-medium text-gray-900 dark:text-gray-100">Latest server version</h4>
            <p className="mt-2 truncate text-sm font-medium text-gray-700 dark:text-gray-200">
              {serverNote.title || 'Untitled'}
            </p>
            <p className="mt-1 line-clamp-5 whitespace-pre-wrap text-xs text-gray-600 dark:text-gray-400">
              {serverNote.content_md || 'Empty note'}
            </p>
            <dl className="mt-3 space-y-1 border-t border-gray-200 pt-3 text-xs text-gray-600 dark:border-gray-700 dark:text-gray-300">
              <div><dt className="inline font-medium">Visibility:</dt> <dd className="inline">{serverNote.visibility}</dd></div>
              <div><dt className="inline font-medium">Publishing:</dt> <dd className="inline">{serverNote.is_published ? 'published' : 'unpublished'}</dd></div>
              <div className="truncate" title={serverNote.cover_url || 'No cover'}>
                <dt className="inline font-medium">Cover:</dt> <dd className="inline">{serverNote.cover_url || 'No cover'}</dd>
              </div>
            </dl>
          </section>
        </div>

        <div className="flex flex-wrap justify-end gap-3 border-t border-gray-200 p-6 dark:border-gray-700">
          <button
            type="button"
            onClick={onKeepEditing}
            disabled={isSaving}
            className="rounded-lg border border-gray-300 px-4 py-2 text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-200 dark:hover:bg-gray-700"
          >
            Keep editing
          </button>
          <button
            type="button"
            onClick={onLoadServer}
            disabled={isSaving}
            className="rounded-lg border border-amber-400 px-4 py-2 text-amber-700 hover:bg-amber-50 disabled:opacity-50 dark:text-amber-300 dark:hover:bg-amber-950/30"
          >
            Load server version
          </button>
          <button
            type="button"
            onClick={onOverwrite}
            disabled={isSaving}
            className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {isSaving ? 'Saving…' : 'Overwrite with my draft'}
          </button>
        </div>
      </div>
    </div>
  )
}
