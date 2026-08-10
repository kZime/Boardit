import { X } from 'lucide-react'

import type { Note } from '../../api/gen/models/note'

interface DeleteNoteDialogProps {
  note: Note
  isDeleting: boolean
  onCancel: () => void
  onConfirm: (noteID: number) => void
}

export default function DeleteNoteDialog({ note, isDeleting, onCancel, onConfirm }: DeleteNoteDialogProps) {
  return (
    <div
      className="fixed inset-0 bg-black/40 dark:bg-black/50 flex items-center justify-center z-50 p-4"
      onClick={onCancel}
      role="presentation"
    >
      <div
        className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md border border-gray-200 dark:border-gray-700"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-note-title"
      >
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <h3 id="delete-note-title" className="text-lg font-semibold text-gray-900 dark:text-gray-100">Delete note?</h3>
          <button onClick={onCancel} className="p-2 rounded-lg text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700" aria-label="Close">
            <X className="w-5 h-5" />
          </button>
        </div>
        <p className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
          {note.title || 'Untitled'} will be permanently deleted. This cannot be undone.
        </p>
        <div className="flex justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700">
          <button onClick={onCancel} className="px-4 py-2 text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors">
            Cancel
          </button>
          <button
            onClick={() => onConfirm(note.id)}
            disabled={isDeleting}
            className="px-4 py-2 bg-red-500 dark:bg-red-600 text-white rounded-lg hover:bg-red-600 dark:hover:bg-red-500 disabled:opacity-50 transition-colors"
          >
            {isDeleting ? 'Deleting…' : 'Delete'}
          </button>
        </div>
      </div>
    </div>
  )
}
