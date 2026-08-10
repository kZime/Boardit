// src/pages/Editor.tsx
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";

// Tailwind styles
import "@mdxeditor/editor/style.css";

// MDXEditor
import {
  MDXEditor,
  type MDXEditorMethods,
  toolbarPlugin,
  headingsPlugin,
  listsPlugin,
  linkPlugin,
  quotePlugin,
  codeBlockPlugin,
  markdownShortcutPlugin,
  frontmatterPlugin,
  InsertFrontmatter,
  // toolbar items
  BoldItalicUnderlineToggles,
  BlockTypeSelect,
  ListsToggle,
  CreateLink,
  CodeToggle,
  UndoRedo,
  Separator,
} from "@mdxeditor/editor";

import ReactMarkdown from "react-markdown";
import { Menu, LogOut, Plus, Trash2, X, Check, FileText, Pencil, FolderPlus, Folder as FolderIcon, ChevronRight, ChevronDown } from "lucide-react";
import { parseFrontmatterAndBody } from "../utils/markdownCover";
import PostHeaderCard from "../components/PostHeaderCard";

// Orval generated hooks
import {
  useListNotes,
  useCreateNote,
  useUpdateNote,
  useDeleteNote,
  useListFolders,
  useCreateFolder,
} from "../api/gen/client";
import type { Note } from "../api/gen/models/note";
import type { Folder } from "../api/gen/models/folder";
import type { CreateNoteRequest } from "../api/gen/models/createNoteRequest";
import type { UpdateNoteRequest } from "../api/gen/models/updateNoteRequest";

// Note row used in sidebar (both in folders and unfiled)
function NoteRow({
  note,
  currentNoteId,
  movingNoteId,
  folders,
  onSelect,
  onDelete,
  onMoveStart,
  onMove,
}: {
  note: Note;
  currentNoteId: number | null;
  movingNoteId: number | null;
  folders: Folder[];
  onSelect: (n: Note) => void;
  onDelete: (n: Note) => void;
  onMoveStart: (id: number | null) => void;
  onMove: (noteId: number, folderId: number | null) => void;
}) {
  return (
    <div className="group relative flex items-center justify-between gap-2 px-2 py-1.5 rounded-lg hover:bg-gray-200/80 dark:hover:bg-gray-700/80 transition-colors">
      <button
        className={`text-left flex-1 truncate px-2 py-1 rounded-md text-sm transition-colors ${
          currentNoteId === note.id
            ? "bg-blue-100 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200 font-medium"
            : "hover:bg-gray-100 dark:hover:bg-gray-700"
        }`}
        onClick={() => onSelect(note)}
        title={note.title}
      >
        {note.title || "(Untitled)"}
      </button>
      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-all">
        {/* Move to folder button */}
        <div className="relative">
          <button
            onClick={(e) => { e.stopPropagation(); onMoveStart(movingNoteId === note.id ? null : note.id); }}
            className="p-1.5 rounded-md text-gray-400 hover:bg-gray-300/80 hover:text-gray-600 dark:hover:bg-gray-600 dark:hover:text-gray-200 transition-colors"
            title="Move to folder"
          >
            <FolderIcon className="w-3.5 h-3.5" />
          </button>
          {movingNoteId === note.id && (
            <div className="absolute right-0 top-8 z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 min-w-[150px]">
              <button
                onClick={() => { onMove(note.id, null); }}
                className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors ${
                  !note.folder_id ? "text-blue-600 dark:text-blue-400 font-medium" : "text-gray-700 dark:text-gray-300"
                }`}
              >
                Unfiled
              </button>
              {folders.map((f: Folder) => (
                <button
                  key={f.id}
                  onClick={() => { onMove(note.id, f.id); }}
                  className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors flex items-center gap-1.5 ${
                    note.folder_id === f.id ? "text-blue-600 dark:text-blue-400 font-medium" : "text-gray-700 dark:text-gray-300"
                  }`}
                >
                  <FolderIcon className="w-3 h-3 shrink-0" />
                  <span className="truncate">{f.name}</span>
                </button>
              ))}
            </div>
          )}
        </div>
        <button
          onClick={(e) => { e.stopPropagation(); onDelete(note); }}
          className="p-1.5 rounded-md text-gray-400 hover:bg-red-100 hover:text-red-600 dark:hover:bg-red-900/40 dark:hover:text-red-400 transition-all"
          title="Delete"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}

export default function Editor() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const noteIdFromUrl = searchParams.get("noteId");
  const { logout } = useAuth();
  const editorRef = useRef<MDXEditorMethods>(null);

  // Sidebar toggle state
  const [open, setOpen] = useState(true);
  // Current note being edited
  const [currentNoteId, setCurrentNoteId] = useState<number | null>(null);

  // Current markdown content being edited
  const defaultTitle = "Untitled Page";
  const defaultMd = [
    "",
    "# Headline here",
    "",
    "This is a paragraph.",
    "",
    "This is a list:",
    "- Item 1",
    "- Item 2",
    "- Item 3",
  ].join("\n");

  const [md, setMd] = useState(defaultMd);

  // Edit page details modal state
  const [showEditModal, setShowEditModal] = useState(false);
  const [pageDetails, setPageDetails] = useState({
    title: defaultTitle,
    coverUrl: "",
    description: "",
    tags: "",
    visibility: "private" as "private" | "public" | "unlisted",
  });

  // Save success notification state
  const [showSaveSuccess, setShowSaveSuccess] = useState(false);

  // Delete confirmation modal: note to delete or null
  const [noteToDelete, setNoteToDelete] = useState<Note | null>(null);

  // Folder management state
  const [expandedFolders, setExpandedFolders] = useState<Set<number>>(new Set());
  const [showNewFolderInput, setShowNewFolderInput] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [movingNoteId, setMovingNoteId] = useState<number | null>(null);

  // Edit vs Preview mode
  const [viewMode, setViewMode] = useState<"edit" | "preview">("edit");

  // Last saved snapshot for dirty check (refs so beforeunload can read)
  const lastSavedRef = useRef<{
    md: string;
    title: string;
    visibility: "private" | "public" | "unlisted";
  }>({ md: defaultMd, title: defaultTitle, visibility: "private" });
  const isDirtyRef = useRef(false);
  useEffect(() => {
    if (!currentNoteId) {
      isDirtyRef.current = false;
      return;
    }
    isDirtyRef.current =
      md !== lastSavedRef.current.md ||
      pageDetails.title !== lastSavedRef.current.title ||
      pageDetails.visibility !== lastSavedRef.current.visibility;
  }, [currentNoteId, md, pageDetails.title, pageDetails.visibility]);

  // Load notes list
  const {
    data,
    isLoading,
    isError,
    refetch: refetchNotes,
  } = useListNotes({ limit: 50, offset: 0 });

  // Load current note details (currently unused but available for future use)
  // const { data: currentNote, isLoading: isLoadingNote } = useGetNote(
  //   currentNoteId!,
  //   { enabled: !!currentNoteId }
  // )

  // Open note from URL ?noteId= when notes have loaded
  const items = useMemo(() => data?.data?.items ?? [], [data?.data?.items]);
  useEffect(() => {
    if (!noteIdFromUrl || isLoading || items.length === 0) return;
    const id = parseInt(noteIdFromUrl, 10);
    if (Number.isNaN(id)) return;
    const note = items.find((n: Note) => n.id === id);
    if (note) {
      const title = note.title || "Untitled";
      const vis = (note.visibility as "private" | "public" | "unlisted") || "private";
      setCurrentNoteId(note.id);
      setMd(note.content_md || "");
      setPageDetails({
        title,
        coverUrl: note.cover_url || "",
        description: "",
        tags: "",
        visibility: vis,
      });
      lastSavedRef.current = { md: note.content_md || "", title, visibility: vis };
    }
  }, [noteIdFromUrl, isLoading, items]);

  // Ref for save handler so keydown effect can call latest handleSave without deps
  const saveHandlerRef = useRef<() => Promise<void>>(() => Promise.resolve());

  // ESC closes sidebar; Cmd/Ctrl+S saves
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        void saveHandlerRef.current();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // beforeunload when there are unsaved changes
  useEffect(() => {
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (isDirtyRef.current) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, []);

  // Mutation hooks
  const createNoteMutation = useCreateNote();
  const updateNoteMutation = useUpdateNote();
  const deleteNoteMutation = useDeleteNote();
  const createFolderMutation = useCreateFolder();

  // Folder list
  const { data: foldersData, refetch: refetchFolders } = useListFolders();
  const folders = useMemo(() => foldersData?.data?.items ?? [], [foldersData?.data?.items]);

  // ==== Actions ====
  const handleNew = async () => {
    try {
      const newNoteData: CreateNoteRequest = {
        title: defaultTitle,
        content_md: defaultMd,
      };

      const result = await createNoteMutation.mutateAsync({
        data: newNoteData,
      });
      if (result.data) {
        const title = result.data.title || defaultTitle;
        const vis = result.data.visibility || "private";
        setCurrentNoteId(result.data.id);
        setMd(result.data.content_md || "");
        setPageDetails({
          title,
          coverUrl: "",
          description: "",
          tags: "",
          visibility: vis,
        });
        lastSavedRef.current = { md: result.data.content_md || "", title, visibility: vis };
        refetchNotes(); // Refresh the notes list
      }
    } catch (error) {
      console.error("Failed to create note:", error);
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteNoteMutation.mutateAsync({ id });
      if (currentNoteId === id) {
        setCurrentNoteId(null);
        setMd(defaultMd);
        setPageDetails({
          title: defaultTitle,
          coverUrl: "",
          description: "",
          tags: "",
          visibility: "private",
        });
      }
      refetchNotes();
      setNoteToDelete(null);
    } catch (error) {
      console.error("Failed to delete note:", error);
    }
  };

  const handleSave = async () => {
    if (!currentNoteId) {
      // Create new note if no current note
      await handleNew();
      return;
    }

    try {
      const updateData: UpdateNoteRequest = {
        title: pageDetails.title,
        cover_url: pageDetails.coverUrl,
        content_md: md,
        is_published: pageDetails.visibility !== "private",
        visibility: pageDetails.visibility as "private" | "public" | "unlisted",
      };

      await updateNoteMutation.mutateAsync({
        id: currentNoteId,
        data: updateData,
      });
      refetchNotes(); // Refresh the notes list
      lastSavedRef.current = {
        md,
        title: pageDetails.title,
        visibility: pageDetails.visibility,
      };
      isDirtyRef.current = false;

      // Show success notification
      setShowSaveSuccess(true);
      setTimeout(() => {
        setShowSaveSuccess(false);
      }, 3000); // Hide after 3 seconds
    } catch (error) {
      console.error("Failed to save note:", error);
    }
  };
  saveHandlerRef.current = handleSave;

  const handleSelectNote = (note: Note) => {
    const title = note.title || "Untitled";
    const vis = (note.visibility as "private" | "public" | "unlisted") || "private";
    setCurrentNoteId(note.id);
    setMd(note.content_md || "");
    setPageDetails({
      title,
      coverUrl: note.cover_url || "",
      description: "",
      tags: "",
      visibility: vis,
    });
    lastSavedRef.current = { md: note.content_md || "", title, visibility: vis };

    // Force MDXEditor to update its content
    setTimeout(() => {
      if (editorRef.current) {
        editorRef.current.setMarkdown(note.content_md || "");
      }
    }, 0);
  };

  const handleCreateFolder = async () => {
    if (!newFolderName.trim()) return;
    try {
      await createFolderMutation.mutateAsync({ data: { name: newFolderName.trim() } });
      setNewFolderName("");
      setShowNewFolderInput(false);
      refetchFolders();
    } catch (error) {
      console.error("Failed to create folder:", error);
    }
  };

  const handleMoveNote = async (noteId: number, folderId: number | null) => {
    try {
      await updateNoteMutation.mutateAsync({
        id: noteId,
        data: { folder_id: folderId },
      });
      setMovingNoteId(null);
      refetchNotes();
    } catch (error) {
      console.error("Failed to move note:", error);
    }
  };

  // ==== edit page details ====
  const handleEditPage = () => {
    setShowEditModal(true);
  };

  const handleCloseModal = () => {
    setShowEditModal(false);
  };

  const handleSavePageDetails = async () => {
    if (!currentNoteId) {
      setShowEditModal(false);
      return;
    }
    try {
      const updateData: UpdateNoteRequest = {
        title: pageDetails.title,
        cover_url: pageDetails.coverUrl,
        content_md: md,
        is_published: pageDetails.visibility !== "private",
        visibility: pageDetails.visibility as "private" | "public" | "unlisted",
      };
      await updateNoteMutation.mutateAsync({
        id: currentNoteId,
        data: updateData,
      });
      refetchNotes();
      lastSavedRef.current = {
        md,
        title: pageDetails.title,
        visibility: pageDetails.visibility,
      };
      isDirtyRef.current = false;
      setShowSaveSuccess(true);
      setTimeout(() => setShowSaveSuccess(false), 3000);
      setShowEditModal(false);
    } catch (error) {
      console.error("Failed to save page details:", error);
    }
  };

  const handlePageDetailsChange = (field: string, value: string) => {
    setPageDetails((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  return (
    <div className="h-screen flex flex-col bg-gray-50 dark:bg-gray-900 transition-colors">
      {/* Header */}
      <header className="h-12 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 flex items-center justify-between px-4 transition-colors">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            aria-controls="sidebar"
            className="p-2 rounded-lg text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            title="Toggle sidebar"
          >
            <Menu className="w-5 h-5" />
          </button>
          <Link
            to="/"
            onClick={(e) => {
              if (isDirtyRef.current && !window.confirm("You have unsaved changes. Leave anyway?")) {
                e.preventDefault();
              }
            }}
            className="font-medium text-gray-800 dark:text-gray-200 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
          >
            Boardit
          </Link>
          <span className="text-gray-400 dark:text-gray-500 font-normal">/ Editor</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              if (isDirtyRef.current && !window.confirm("You have unsaved changes. Leave anyway?")) return;
              navigate("/");
              logout();
            }}
            className="inline-flex items-center gap-1.5 text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            Logout
          </button>
        </div>
      </header>

      {/* Main area: sidebar + editor */}
      <div className="flex-1 flex overflow-hidden bg-gray-50 dark:bg-gray-900">
        {/* Sidebar (fixed width on desktop, drawer on mobile) */}
        {/* Overlay (mobile only) */}
        {open && (
          <div
            className="fixed inset-0 bg-black/20 z-30 md:hidden"
            onClick={() => setOpen(false)}
          />
        )}

        {/* sidebar */}
        <aside
          id="sidebar"
          className={[
            "z-40 bg-gray-50 dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 overflow-y-auto transition-colors",
            // Mobile: fixed position, Desktop: static position
            "fixed h-full md:static",
            // Control width and visibility based on open state
            open
              ? "w-72 translate-x-0 md:translate-x-0 md:w-72"
              : "w-72 -translate-x-full md:translate-x-0 md:w-0",
          ].join(" ")}
          style={{ transition: "transform .2s ease, width .2s ease" }}
        >
          <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between gap-2">
            <h2 className="font-semibold text-gray-900 dark:text-gray-100">Notes</h2>
            <div className="flex items-center gap-1">
              <button
                onClick={() => { setShowNewFolderInput(true); setNewFolderName(""); }}
                className="p-1.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                title="New folder"
              >
                <FolderPlus className="w-4 h-4" />
              </button>
              <button
                onClick={handleNew}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 transition-colors"
              >
                <Plus className="w-4 h-4" />
                New
              </button>
            </div>
          </div>

          <div className="p-2">
            {/* New folder input */}
            {showNewFolderInput && (
              <div className="flex items-center gap-1 px-2 py-1.5 mb-1">
                <input
                  autoFocus
                  type="text"
                  value={newFolderName}
                  onChange={(e) => setNewFolderName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") void handleCreateFolder();
                    if (e.key === "Escape") { setShowNewFolderInput(false); setNewFolderName(""); }
                  }}
                  placeholder="Folder name…"
                  className="flex-1 text-sm px-2 py-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 outline-none focus:ring-1 focus:ring-blue-500"
                />
                <button
                  onClick={() => void handleCreateFolder()}
                  disabled={!newFolderName.trim() || createFolderMutation.isPending}
                  className="p-1 rounded-md text-blue-600 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-900/40 disabled:opacity-40 transition-colors"
                  title="Create"
                >
                  <Check className="w-4 h-4" />
                </button>
                <button
                  onClick={() => { setShowNewFolderInput(false); setNewFolderName(""); }}
                  className="p-1 rounded-md text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                  title="Cancel"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            )}

            {isLoading && (
              <div className="text-sm text-gray-500 dark:text-gray-400 p-2">Loading…</div>
            )}
            {isError && (
              <div className="text-sm text-red-600 dark:text-red-400 p-2">
                Failed to load notes
              </div>
            )}

            {/* Folder groups */}
            {folders.map((folder: Folder) => {
              const folderNotes = items.filter((n: Note) => n.folder_id === folder.id);
              const isExpanded = expandedFolders.has(folder.id);
              return (
                <div key={folder.id} className="mb-1">
                  <button
                    onClick={() => setExpandedFolders((prev) => {
                      const next = new Set(prev);
                      if (next.has(folder.id)) next.delete(folder.id); else next.add(folder.id);
                      return next;
                    })}
                    className="w-full flex items-center gap-1.5 px-2 py-1.5 rounded-lg text-gray-600 dark:text-gray-300 hover:bg-gray-200/80 dark:hover:bg-gray-700/80 transition-colors text-sm font-medium"
                  >
                    {isExpanded
                      ? <ChevronDown className="w-3.5 h-3.5 shrink-0 text-gray-400" />
                      : <ChevronRight className="w-3.5 h-3.5 shrink-0 text-gray-400" />}
                    <FolderIcon className="w-3.5 h-3.5 shrink-0 text-gray-400 dark:text-gray-500" />
                    <span className="truncate flex-1 text-left">{folder.name}</span>
                    <span className="text-xs text-gray-400 dark:text-gray-500">{folderNotes.length}</span>
                  </button>
                  {isExpanded && (
                    <ul className="space-y-0.5 ml-4 mt-0.5">
                      {folderNotes.length === 0 && (
                        <li className="text-xs text-gray-400 dark:text-gray-500 px-3 py-1">Empty</li>
                      )}
                      {folderNotes.map((n: Note) => (
                        <li key={n.id}>
                          <NoteRow
                            note={n}
                            currentNoteId={currentNoteId}
                            movingNoteId={movingNoteId}
                            folders={folders}
                            onSelect={handleSelectNote}
                            onDelete={setNoteToDelete}
                            onMoveStart={setMovingNoteId}
                            onMove={handleMoveNote}
                          />
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              );
            })}

            {/* Unfiled notes */}
            {(() => {
              const unfiledNotes = items.filter((n: Note) => !n.folder_id);
              return (
                <>
                  {folders.length > 0 && unfiledNotes.length > 0 && (
                    <div className="px-2 py-1 mt-2 mb-0.5">
                      <span className="text-xs font-medium text-gray-400 dark:text-gray-500 uppercase tracking-wide">Unfiled</span>
                    </div>
                  )}
                  <ul className="space-y-1">
                    {unfiledNotes.map((n: Note) => (
                      <li key={n.id}>
                        <NoteRow
                          note={n}
                          currentNoteId={currentNoteId}
                          movingNoteId={movingNoteId}
                          folders={folders}
                          onSelect={handleSelectNote}
                          onDelete={setNoteToDelete}
                          onMoveStart={setMovingNoteId}
                          onMove={handleMoveNote}
                        />
                      </li>
                    ))}
                    {!isLoading && items.length === 0 && (
                      <li className="text-sm text-gray-500 dark:text-gray-400 p-2">No notes</li>
                    )}
                  </ul>
                </>
              );
            })()}
          </div>
        </aside>

        {/* editor area */}
        <main className="flex-1 overflow-auto p-4 md:p-6">
          {currentNoteId ? (
            // Show editor when a note is selected
            <>
              {/* Page Title + visibility badge */}
              <div className="mb-4 flex flex-wrap items-center gap-2">
                <input
                  type="text"
                  value={pageDetails.title}
                  onChange={(e) =>
                    handlePageDetailsChange("title", e.target.value)
                  }
                  className="flex-1 min-w-0 text-2xl font-bold bg-transparent border-none outline-none text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:bg-white dark:focus:bg-gray-800 focus:border focus:border-gray-300 dark:focus:border-gray-600 focus:rounded-lg focus:px-3 focus:py-2 transition-all duration-200"
                  placeholder="Enter page title..."
                />
                <span
                  className={`inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium shrink-0 ${pageDetails.visibility === "public"
                    ? "bg-green-100 dark:bg-green-900/40 text-green-800 dark:text-green-200"
                    : pageDetails.visibility === "unlisted"
                      ? "bg-blue-100 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200"
                      : "bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300"
                    }`}
                >
                  {pageDetails.visibility === "public"
                    ? "Public"
                    : pageDetails.visibility === "unlisted"
                      ? "Unlisted"
                      : "Private"}
                </span>
              </div>

              {/* Edit / Preview toggle */}
              <div className="flex gap-2 mb-4">
                <button
                  type="button"
                  onClick={() => setViewMode("edit")}
                  className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${viewMode === "edit"
                    ? "bg-gray-800 dark:bg-gray-600 text-white"
                    : "bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600"
                    }`}
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode("preview")}
                  className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${viewMode === "preview"
                    ? "bg-gray-800 dark:bg-gray-600 text-white"
                    : "bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600"
                    }`}
                >
                  Preview
                </button>
              </div>

              {viewMode === "edit" ? (
                <>
                  <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm dark:shadow-none border border-gray-200/80 dark:border-gray-700 transition-colors">
                    <MDXEditor
                      key={currentNoteId || "new"}
                      ref={editorRef}
                      className="min-h-[70vh]"
                      contentEditableClassName="prose"
                      markdown={md}
                      onChange={setMd}
                      plugins={[
                        frontmatterPlugin(),
                        toolbarPlugin({
                          toolbarContents: () => (
                            <>
                              <UndoRedo />
                              <Separator />
                              <BlockTypeSelect />
                              <Separator />
                              <BoldItalicUnderlineToggles />
                              <Separator />
                              <ListsToggle />
                              <Separator />
                              <InsertFrontmatter />
                              <Separator />
                              <CreateLink />
                              <Separator />
                              <CodeToggle />
                            </>
                          ),
                        }),
                        headingsPlugin(),
                        listsPlugin(),
                        linkPlugin(),
                        quotePlugin(),
                        codeBlockPlugin(),
                        markdownShortcutPlugin(), // Place last
                      ]}
                    />
                  </div>

                  <div className="mt-6 flex justify-end gap-3">
                    <button
                      className="px-4 py-2 rounded-lg bg-gray-500 dark:bg-gray-600 text-white hover:bg-gray-600 dark:hover:bg-gray-500 transition-colors"
                      onClick={handleEditPage}
                    >
                      Edit Details
                    </button>
                    <button
                      className="px-4 py-2 rounded-lg bg-blue-500 dark:bg-blue-600 text-white hover:bg-blue-600 dark:hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                      onClick={handleSave}
                      disabled={
                        updateNoteMutation.isPending || createNoteMutation.isPending
                      }
                    >
                      {updateNoteMutation.isPending || createNoteMutation.isPending
                        ? "Saving..."
                        : "Save"}
                    </button>
                  </div>
                </>
              ) : (
                <div className="space-y-4">
                  <PostHeaderCard
                    title={pageDetails.title || "Untitled"}
                    coverUrl={pageDetails.coverUrl || null}
                  />
                  <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm dark:shadow-none border border-gray-200/80 dark:border-gray-700 min-h-[40vh] transition-colors">
                    <div className="prose prose-gray dark:prose-invert max-w-none">
                      <ReactMarkdown>
                        {parseFrontmatterAndBody(md || "").body}
                      </ReactMarkdown>
                    </div>
                  </div>
                </div>
              )}
            </>
          ) : (
            // Show welcome message when no note is selected (Phase 3: empty state)
            <div className="flex items-center justify-center h-full min-h-[400px]">
              <div className="text-center max-w-md px-4">
                <div className="mb-8">
                  <div className="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-gray-100 dark:bg-gray-700 mb-6 transition-colors">
                    <FileText className="w-10 h-10 text-gray-500 dark:text-gray-400" strokeWidth={1.5} />
                  </div>
                  <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-gray-100 mb-2 tracking-tight">
                    Welcome to Boardit
                  </h2>
                  <p className="text-gray-600 dark:text-gray-400 mb-8">
                    Start writing by creating a new note or selecting an existing one from the sidebar.
                  </p>
                </div>

                <div className="space-y-4 text-left">
                  <div className="flex items-start gap-4 p-3 rounded-xl hover:bg-gray-100/80 dark:hover:bg-gray-800/80 transition-colors">
                    <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/40 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span className="text-blue-600 dark:text-blue-300 font-semibold text-sm">1</span>
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-900 dark:text-gray-100">Create a new note</h3>
                      <p className="text-gray-600 dark:text-gray-400 text-sm">
                        Click &quot;New&quot; in the sidebar to start a new article.
                      </p>
                    </div>
                  </div>

                  <div className="flex items-start gap-4 p-3 rounded-xl hover:bg-gray-100/80 dark:hover:bg-gray-800/80 transition-colors">
                    <div className="w-8 h-8 bg-green-100 dark:bg-green-900/40 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span className="text-green-600 dark:text-green-300 font-semibold text-sm">2</span>
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-900 dark:text-gray-100">Select existing note</h3>
                      <p className="text-gray-600 dark:text-gray-400 text-sm">
                        Click any note in the sidebar to continue editing.
                      </p>
                    </div>
                  </div>

                  <div className="flex items-start gap-4 p-3 rounded-xl hover:bg-gray-100/80 dark:hover:bg-gray-800/80 transition-colors">
                    <div className="w-8 h-8 bg-purple-100 dark:bg-purple-900/40 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5">
                      <Pencil className="w-4 h-4 text-purple-600 dark:text-purple-300 mt-0.5" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-900 dark:text-gray-100">Start writing</h3>
                      <p className="text-gray-600 dark:text-gray-400 text-sm">
                        Use the rich text editor with Markdown support.
                      </p>
                    </div>
                  </div>
                </div>

                <div className="mt-8">
                  <button
                    onClick={handleNew}
                    className="inline-flex items-center gap-2 px-6 py-3 bg-blue-500 dark:bg-blue-600 text-white rounded-lg hover:bg-blue-600 dark:hover:bg-blue-500 hover:shadow-md active:scale-[0.98] transition-all font-medium"
                  >
                    <Plus className="w-4 h-4" />
                    Create Your First Note
                  </button>
                </div>
              </div>
            </div>
          )}
        </main>
      </div>

      {/* Save Success Notification */}
      {showSaveSuccess && (
        <div className="fixed top-4 right-4 z-50">
          <div className="bg-green-500 dark:bg-green-600 text-white px-6 py-3 rounded-xl shadow-lg flex items-center gap-2 border border-green-400/20">
            <Check className="w-5 h-5 shrink-0" />
            <span className="font-medium">Saved successfully!</span>
          </div>
        </div>
      )}

      {/* Edit Page Details Modal */}
      {showEditModal && (
        <div
          className="fixed inset-0 bg-black/40 dark:bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={handleCloseModal}
        >
          <div
            className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md border border-gray-200 dark:border-gray-700"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Edit Page Details</h3>
              <button
                onClick={handleCloseModal}
                className="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                aria-label="Close"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Title
                </label>
                <input
                  type="text"
                  value={pageDetails.title}
                  onChange={(e) =>
                    handlePageDetailsChange("title", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter page title"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Cover Image URL
                </label>
                <input
                  type="url"
                  value={pageDetails.coverUrl}
                  onChange={(e) =>
                    handlePageDetailsChange("coverUrl", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="https://example.com/cover.jpg"
                />
                {pageDetails.coverUrl && (
                  <div className="mt-2 rounded-lg overflow-hidden aspect-[16/9] bg-gray-100 dark:bg-gray-900">
                    <img
                      src={pageDetails.coverUrl}
                      alt="Cover preview"
                      className="w-full h-full object-cover"
                      onError={(e) => {
                        (e.target as HTMLImageElement).style.display = "none";
                      }}
                    />
                  </div>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Description
                </label>
                <textarea
                  value={pageDetails.description}
                  onChange={(e) =>
                    handlePageDetailsChange("description", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  rows={3}
                  placeholder="Enter page description"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Tags
                </label>
                <input
                  type="text"
                  value={pageDetails.tags}
                  onChange={(e) =>
                    handlePageDetailsChange("tags", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter tags (comma separated)"
                />
              </div>

              <div>
                <label htmlFor="page-visibility" className="block text-sm font-medium text-gray-700 mb-1">
                  Visibility
                </label>
                <select
                  id="page-visibility"
                  value={pageDetails.visibility}
                  onChange={(e) =>
                    handlePageDetailsChange("visibility", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="private">Private</option>
                  <option value="public">Public</option>
                  <option value="unlisted">Unlisted</option>
                </select>
              </div>
            </div>

            <div className="flex justify-end gap-3 p-6 border-t">
              <button
                onClick={handleCloseModal}
                className="px-4 py-2 text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleSavePageDetails}
                className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
              >
                Save Changes
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete note confirmation modal */}
      {noteToDelete && (
        <div
          className="fixed inset-0 bg-black/40 dark:bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={() => setNoteToDelete(null)}
        >
          <div
            className="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md border border-gray-200 dark:border-gray-700"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Delete note?</h3>
              <button
                onClick={() => setNoteToDelete(null)}
                className="p-2 rounded-lg text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                aria-label="Close"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <p className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
              {noteToDelete.title || "Untitled"} will be permanently deleted. This cannot be undone.
            </p>
            <div className="flex justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700">
              <button
                onClick={() => setNoteToDelete(null)}
                className="px-4 py-2 text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => void handleDelete(noteToDelete.id)}
                disabled={deleteNoteMutation.isPending}
                className="px-4 py-2 bg-red-500 dark:bg-red-600 text-white rounded-lg hover:bg-red-600 dark:hover:bg-red-500 disabled:opacity-50 transition-colors"
              >
                {deleteNoteMutation.isPending ? "Deleting…" : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
