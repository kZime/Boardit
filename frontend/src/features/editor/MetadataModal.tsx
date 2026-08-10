import { X } from 'lucide-react'

import type { PageDetails, Visibility } from './types'

interface MetadataModalProps {
  details: PageDetails
  onChange: (field: keyof PageDetails, value: string) => void
  onClose: () => void
  onSave: () => void
}

export default function MetadataModal({
  details,
  onChange,
  onClose,
  onSave,
}: MetadataModalProps) {
  return (
    <div
      className="fixed inset-0 bg-black/40 dark:bg-black/50 flex items-center justify-center z-50 p-4"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md border border-gray-200 dark:border-gray-700"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="metadata-title"
      >
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <h3 id="metadata-title" className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Edit Page Details
          </h3>
          <button
            onClick={onClose}
            className="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            aria-label="Close"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label htmlFor="page-title" className="block text-sm font-medium text-gray-700 mb-1">Title</label>
            <input
              id="page-title"
              type="text"
              value={details.title}
              onChange={(event) => onChange('title', event.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter page title"
            />
          </div>

          <div>
            <label htmlFor="page-cover" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Cover Image URL
            </label>
            <input
              id="page-cover"
              type="url"
              value={details.coverUrl}
              onChange={(event) => onChange('coverUrl', event.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="https://example.com/cover.jpg"
            />
            {details.coverUrl && (
              <div className="mt-2 rounded-lg overflow-hidden aspect-[16/9] bg-gray-100 dark:bg-gray-900">
                <img
                  src={details.coverUrl}
                  alt="Cover preview"
                  className="w-full h-full object-cover"
                  onError={(event) => {
                    event.currentTarget.style.display = 'none'
                  }}
                />
              </div>
            )}
          </div>

          <div>
            <label htmlFor="page-description" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Description
            </label>
            <textarea
              id="page-description"
              value={details.description}
              onChange={(event) => onChange('description', event.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
              placeholder="Enter page description"
            />
          </div>

          <div>
            <label htmlFor="page-tags" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Tags</label>
            <input
              id="page-tags"
              type="text"
              value={details.tags}
              onChange={(event) => onChange('tags', event.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter tags (comma separated)"
            />
          </div>

          <div>
            <label htmlFor="page-visibility" className="block text-sm font-medium text-gray-700 mb-1">Visibility</label>
            <select
              id="page-visibility"
              value={details.visibility}
              onChange={(event) => onChange('visibility', event.target.value as Visibility)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="private">Private</option>
              <option value="public">Public</option>
              <option value="unlisted">Unlisted</option>
            </select>
          </div>
        </div>

        <div className="flex justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700">
          <button onClick={onClose} className="px-4 py-2 text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700">
            Cancel
          </button>
          <button onClick={onSave} className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600">
            Save Changes
          </button>
        </div>
      </div>
    </div>
  )
}
