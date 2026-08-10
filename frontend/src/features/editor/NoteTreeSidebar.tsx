import { useState } from 'react'
import {
  Check,
  ChevronDown,
  ChevronRight,
  Folder as FolderIcon,
  FolderPlus,
  Plus,
  Trash2,
  X,
} from 'lucide-react'

import type { Folder } from '../../api/gen/models/folder'
import type { Note } from '../../api/gen/models/note'

interface NoteTreeSidebarProps {
  open: boolean
  notes: Note[]
  folders: Folder[]
  currentNoteID: number | null
  isLoading: boolean
  isError: boolean
  isCreatingFolder: boolean
  onClose: () => void
  onNewNote: () => void
  onSelectNote: (note: Note) => void
  onDeleteNote: (note: Note) => void
  onCreateFolder: (name: string) => Promise<void>
  onMoveNote: (noteID: number, folderID: number | null) => Promise<void>
}

function NoteRow({
  note,
  currentNoteID,
  movingNoteID,
  folders,
  onSelect,
  onDelete,
  onMoveStart,
  onMove,
}: {
  note: Note
  currentNoteID: number | null
  movingNoteID: number | null
  folders: Folder[]
  onSelect: (note: Note) => void
  onDelete: (note: Note) => void
  onMoveStart: (id: number | null) => void
  onMove: (noteID: number, folderID: number | null) => void
}) {
  return (
    <div className="group relative flex items-center justify-between gap-2 px-2 py-1.5 rounded-lg hover:bg-gray-200/80 dark:hover:bg-gray-700/80 transition-colors">
      <button
        className={`text-left flex-1 truncate px-2 py-1 rounded-md text-sm transition-colors ${
          currentNoteID === note.id
            ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200 font-medium'
            : 'hover:bg-gray-100 dark:hover:bg-gray-700'
        }`}
        onClick={() => onSelect(note)}
        title={note.title}
      >
        {note.title || '(Untitled)'}
      </button>
      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-all">
        <div className="relative">
          <button
            onClick={(event) => {
              event.stopPropagation()
              onMoveStart(movingNoteID === note.id ? null : note.id)
            }}
            className="p-1.5 rounded-md text-gray-400 hover:bg-gray-300/80 hover:text-gray-600 dark:hover:bg-gray-600 dark:hover:text-gray-200 transition-colors"
            title="Move to folder"
          >
            <FolderIcon className="w-3.5 h-3.5" />
          </button>
          {movingNoteID === note.id && (
            <div className="absolute right-0 top-8 z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 min-w-[150px]">
              <button
                onClick={() => onMove(note.id, null)}
                className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors ${
                  !note.folder_id ? 'text-blue-600 dark:text-blue-400 font-medium' : 'text-gray-700 dark:text-gray-300'
                }`}
              >
                Unfiled
              </button>
              {folders.map((folder) => (
                <button
                  key={folder.id}
                  onClick={() => onMove(note.id, folder.id)}
                  className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors flex items-center gap-1.5 ${
                    note.folder_id === folder.id
                      ? 'text-blue-600 dark:text-blue-400 font-medium'
                      : 'text-gray-700 dark:text-gray-300'
                  }`}
                >
                  <FolderIcon className="w-3 h-3 shrink-0" />
                  <span className="truncate">{folder.name}</span>
                </button>
              ))}
            </div>
          )}
        </div>
        <button
          onClick={(event) => {
            event.stopPropagation()
            onDelete(note)
          }}
          className="p-1.5 rounded-md text-gray-400 hover:bg-red-100 hover:text-red-600 dark:hover:bg-red-900/40 dark:hover:text-red-400 transition-all"
          title="Delete"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  )
}

export default function NoteTreeSidebar({
  open,
  notes,
  folders,
  currentNoteID,
  isLoading,
  isError,
  isCreatingFolder,
  onClose,
  onNewNote,
  onSelectNote,
  onDeleteNote,
  onCreateFolder,
  onMoveNote,
}: NoteTreeSidebarProps) {
  const [expandedFolders, setExpandedFolders] = useState<Set<number>>(new Set())
  const [showNewFolderInput, setShowNewFolderInput] = useState(false)
  const [newFolderName, setNewFolderName] = useState('')
  const [movingNoteID, setMovingNoteID] = useState<number | null>(null)

  const closeNewFolderInput = () => {
    setShowNewFolderInput(false)
    setNewFolderName('')
  }

  const createFolder = async () => {
    const name = newFolderName.trim()
    if (!name) return
    await onCreateFolder(name)
    closeNewFolderInput()
  }

  const moveNote = async (noteID: number, folderID: number | null) => {
    await onMoveNote(noteID, folderID)
    setMovingNoteID(null)
  }

  const noteRow = (note: Note) => (
    <NoteRow
      note={note}
      currentNoteID={currentNoteID}
      movingNoteID={movingNoteID}
      folders={folders}
      onSelect={onSelectNote}
      onDelete={onDeleteNote}
      onMoveStart={setMovingNoteID}
      onMove={(noteID, folderID) => void moveNote(noteID, folderID)}
    />
  )

  const unfiledNotes = notes.filter((note) => !note.folder_id)

  return (
    <>
      {open && <div className="fixed inset-0 bg-black/20 z-30 md:hidden" onClick={onClose} />}
      <aside
        id="sidebar"
        className={[
          'z-40 bg-gray-50 dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 overflow-y-auto transition-colors',
          'fixed h-full md:static',
          open
            ? 'w-72 translate-x-0 md:translate-x-0 md:w-72'
            : 'w-72 -translate-x-full md:translate-x-0 md:w-0',
        ].join(' ')}
        style={{ transition: 'transform .2s ease, width .2s ease' }}
      >
        <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between gap-2">
          <h2 className="font-semibold text-gray-900 dark:text-gray-100">Notes</h2>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setShowNewFolderInput(true)}
              className="p-1.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
              title="New folder"
            >
              <FolderPlus className="w-4 h-4" />
            </button>
            <button
              onClick={onNewNote}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 transition-colors"
            >
              <Plus className="w-4 h-4" />
              New
            </button>
          </div>
        </div>

        <div className="p-2">
          {showNewFolderInput && (
            <div className="flex items-center gap-1 px-2 py-1.5 mb-1">
              <input
                autoFocus
                type="text"
                value={newFolderName}
                onChange={(event) => setNewFolderName(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') void createFolder()
                  if (event.key === 'Escape') closeNewFolderInput()
                }}
                placeholder="Folder name…"
                className="flex-1 text-sm px-2 py-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 outline-none focus:ring-1 focus:ring-blue-500"
              />
              <button
                onClick={() => void createFolder()}
                disabled={!newFolderName.trim() || isCreatingFolder}
                className="p-1 rounded-md text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/40 disabled:opacity-40 transition-colors"
                title="Create"
              >
                <Check className="w-4 h-4" />
              </button>
              <button
                onClick={closeNewFolderInput}
                className="p-1 rounded-md text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                title="Cancel"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          )}

          {isLoading && <div className="text-sm text-gray-500 dark:text-gray-400 p-2">Loading…</div>}
          {isError && <div className="text-sm text-red-600 dark:text-red-400 p-2">Failed to load notes</div>}

          {folders.map((folder) => {
            const folderNotes = notes.filter((note) => note.folder_id === folder.id)
            const isExpanded = expandedFolders.has(folder.id)
            return (
              <div key={folder.id} className="mb-1">
                <button
                  onClick={() =>
                    setExpandedFolders((previous) => {
                      const next = new Set(previous)
                      if (next.has(folder.id)) next.delete(folder.id)
                      else next.add(folder.id)
                      return next
                    })
                  }
                  className="w-full flex items-center gap-1.5 px-2 py-1.5 rounded-lg text-gray-600 dark:text-gray-300 hover:bg-gray-200/80 dark:hover:bg-gray-700/80 transition-colors text-sm font-medium"
                >
                  {isExpanded ? (
                    <ChevronDown className="w-3.5 h-3.5 shrink-0 text-gray-400" />
                  ) : (
                    <ChevronRight className="w-3.5 h-3.5 shrink-0 text-gray-400" />
                  )}
                  <FolderIcon className="w-3.5 h-3.5 shrink-0 text-gray-400 dark:text-gray-500" />
                  <span className="truncate flex-1 text-left">{folder.name}</span>
                  <span className="text-xs text-gray-400 dark:text-gray-500">{folderNotes.length}</span>
                </button>
                {isExpanded && (
                  <ul className="space-y-0.5 ml-4 mt-0.5">
                    {folderNotes.length === 0 && (
                      <li className="text-xs text-gray-400 dark:text-gray-500 px-3 py-1">Empty</li>
                    )}
                    {folderNotes.map((note) => <li key={note.id}>{noteRow(note)}</li>)}
                  </ul>
                )}
              </div>
            )
          })}

          {folders.length > 0 && unfiledNotes.length > 0 && (
            <div className="px-2 py-1 mt-2 mb-0.5">
              <span className="text-xs font-medium text-gray-400 dark:text-gray-500 uppercase tracking-wide">Unfiled</span>
            </div>
          )}
          <ul className="space-y-1">
            {unfiledNotes.map((note) => <li key={note.id}>{noteRow(note)}</li>)}
            {!isLoading && notes.length === 0 && (
              <li className="text-sm text-gray-500 dark:text-gray-400 p-2">No notes</li>
            )}
          </ul>
        </div>
      </aside>
    </>
  )
}
