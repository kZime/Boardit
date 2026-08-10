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
import { Menu, LogOut, Plus, Check, FileText, Pencil } from "lucide-react";
import { parseFrontmatterAndBody } from "../utils/markdownCover";
import PostHeaderCard from "../components/PostHeaderCard";
import DeleteNoteDialog from "../features/editor/DeleteNoteDialog";
import MetadataModal from "../features/editor/MetadataModal";
import NoteTreeSidebar from "../features/editor/NoteTreeSidebar";
import SaveConflictDialog from "../features/editor/SaveConflictDialog";
import {
  getVersionConflict,
  isDirty,
  saveExistingNote,
  snapshotOf,
  type SavedSnapshot,
  versionForMove,
} from "../features/editor/saveCoordinator";

// Orval generated hooks
import {
  useCreateNote,
  useUpdateNote,
  useDeleteNote,
  useListFolders,
  useCreateFolder,
} from "../api/gen/client";
import { useAllNotes } from "../features/editor/useAllNotes";
import type { Note } from "../api/gen/models/note";
import type { CreateNoteRequest } from "../api/gen/models/createNoteRequest";
import type { VersionConflictError } from "../api/gen/models/versionConflictError";

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
  const [currentVersion, setCurrentVersion] = useState<number | null>(null);

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
  const [saveConflict, setSaveConflict] = useState<VersionConflictError | null>(null);

  // Delete confirmation modal: note to delete or null
  const [noteToDelete, setNoteToDelete] = useState<Note | null>(null);

  // Edit vs Preview mode
  const [viewMode, setViewMode] = useState<"edit" | "preview">("edit");

  // Last saved snapshot for dirty check (refs so beforeunload can read)
  const lastSavedRef = useRef<SavedSnapshot>({
    md: defaultMd,
    title: defaultTitle,
    coverUrl: "",
    visibility: "private",
  });
  const isDirtyRef = useRef(false);
  useEffect(() => {
    if (!currentNoteId) {
      isDirtyRef.current = false;
      return;
    }
    isDirtyRef.current = isDirty(md, pageDetails, lastSavedRef.current);
  }, [currentNoteId, md, pageDetails]);

  // Load notes list
  const {
    data,
    isLoading,
    isError,
    refetch: refetchNotes,
  } = useAllNotes();

  // Load current note details (currently unused but available for future use)
  // const { data: currentNote, isLoading: isLoadingNote } = useGetNote(
  //   currentNoteId!,
  //   { enabled: !!currentNoteId }
  // )

  // Open note from URL ?noteId= when notes have loaded
  const items = useMemo(() => data?.items ?? [], [data?.items]);
  useEffect(() => {
    if (!noteIdFromUrl || isLoading || items.length === 0) return;
    const id = parseInt(noteIdFromUrl, 10);
    if (Number.isNaN(id)) return;
    const note = items.find((n: Note) => n.id === id);
    if (note) {
      const title = note.title || "Untitled";
      const vis = (note.visibility as "private" | "public" | "unlisted") || "private";
      setCurrentNoteId(note.id);
      setCurrentVersion(note.version);
      setMd(note.content_md || "");
      const details = {
        title,
        coverUrl: note.cover_url || "",
        description: "",
        tags: "",
        visibility: vis,
      };
      setPageDetails(details);
      lastSavedRef.current = snapshotOf(note.content_md || "", details);
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
        setCurrentVersion(result.data.version);
        setMd(result.data.content_md || "");
        const details = {
          title,
          coverUrl: "",
          description: "",
          tags: "",
          visibility: vis,
        };
        setPageDetails(details);
        lastSavedRef.current = snapshotOf(result.data.content_md || "", details);
        setSaveConflict(null);
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
        setCurrentVersion(null);
        setMd(defaultMd);
        setPageDetails({
          title: defaultTitle,
          coverUrl: "",
          description: "",
          tags: "",
          visibility: "private",
        });
        setSaveConflict(null);
      }
      refetchNotes();
      setNoteToDelete(null);
    } catch (error) {
      console.error("Failed to delete note:", error);
    }
  };

  const handleSave = async (versionOverride?: number) => {
    if (!currentNoteId || currentVersion === null) {
      // Create new note if no current note
      await handleNew();
      return true;
    }

    try {
      const saved = await saveExistingNote({
        updateNote: async (noteID, request) => {
          const response = await updateNoteMutation.mutateAsync({ id: noteID, data: request });
          return response.data;
        },
      }, {
        noteID: currentNoteId,
        markdown: md,
        details: pageDetails,
        version: versionOverride ?? currentVersion,
      });
      refetchNotes(); // Refresh the notes list
      lastSavedRef.current = saved.snapshot;
      setCurrentVersion(saved.version);
      isDirtyRef.current = false;
      setSaveConflict(null);

      // Show success notification
      setShowSaveSuccess(true);
      setTimeout(() => {
        setShowSaveSuccess(false);
      }, 3000); // Hide after 3 seconds
      return true;
    } catch (error) {
      const conflict = getVersionConflict(error);
      if (conflict) {
        setSaveConflict(conflict);
        return false;
      }
      console.error("Failed to save note:", error);
      return false;
    }
  };
  saveHandlerRef.current = async () => {
    await handleSave();
  };

  const handleSelectNote = (note: Note) => {
    const title = note.title || "Untitled";
    const vis = (note.visibility as "private" | "public" | "unlisted") || "private";
    setCurrentNoteId(note.id);
    setCurrentVersion(note.version);
    setMd(note.content_md || "");
    const details = {
      title,
      coverUrl: note.cover_url || "",
      description: "",
      tags: "",
      visibility: vis,
    };
    setPageDetails(details);
    lastSavedRef.current = snapshotOf(note.content_md || "", details);
    setSaveConflict(null);

    // Force MDXEditor to update its content
    setTimeout(() => {
      if (editorRef.current) {
        editorRef.current.setMarkdown(note.content_md || "");
      }
    }, 0);
  };

  const handleCreateFolder = async (name: string) => {
    try {
      await createFolderMutation.mutateAsync({ data: { name } });
      refetchFolders();
    } catch (error) {
      console.error("Failed to create folder:", error);
      throw error;
    }
  };

  const handleMoveNote = async (noteId: number, folderId: number | null) => {
    try {
      const target = items.find((note) => note.id === noteId);
      const version = versionForMove(noteId, currentNoteId, currentVersion, target?.version);
      const response = await updateNoteMutation.mutateAsync({
        id: noteId,
        data: { folder_id: folderId, version },
      });
      if (noteId === currentNoteId) setCurrentVersion(response.data.version);
      refetchNotes();
    } catch (error) {
      console.error("Failed to move note:", error);
      throw error;
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
    if (await handleSave()) setShowEditModal(false);
  };

  const handlePageDetailsChange = (field: string, value: string) => {
    setPageDetails((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleLoadServerVersion = () => {
    if (!saveConflict) return;
    const note = saveConflict.server_snapshot;
    const details = {
      title: note.title || "Untitled",
      coverUrl: note.cover_url || "",
      description: "",
      tags: "",
      visibility: note.visibility,
    };
    setCurrentVersion(note.version);
    setMd(note.content_md || "");
    setPageDetails(details);
    lastSavedRef.current = snapshotOf(note.content_md || "", details);
    isDirtyRef.current = false;
    editorRef.current?.setMarkdown(note.content_md || "");
    setSaveConflict(null);
    setShowEditModal(false);
    void refetchNotes();
  };

  const handleOverwriteConflict = async () => {
    if (!saveConflict) return;
    await handleSave(saveConflict.server_snapshot.version);
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
        <NoteTreeSidebar
          open={open}
          notes={items}
          folders={folders}
          currentNoteID={currentNoteId}
          isLoading={isLoading}
          isError={isError}
          isCreatingFolder={createFolderMutation.isPending}
          onClose={() => setOpen(false)}
          onNewNote={() => void handleNew()}
          onSelectNote={handleSelectNote}
          onDeleteNote={setNoteToDelete}
          onCreateFolder={handleCreateFolder}
          onMoveNote={handleMoveNote}
        />

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
                      onClick={() => void handleSave()}
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
        <MetadataModal
          details={pageDetails}
          onChange={handlePageDetailsChange}
          onClose={handleCloseModal}
          onSave={() => void handleSavePageDetails()}
        />
      )}

      {/* Delete note confirmation modal */}
      {noteToDelete && (
        <DeleteNoteDialog
          note={noteToDelete}
          isDeleting={deleteNoteMutation.isPending}
          onCancel={() => setNoteToDelete(null)}
          onConfirm={(noteID) => void handleDelete(noteID)}
        />
      )}

      {saveConflict && (
        <SaveConflictDialog
          serverNote={saveConflict.server_snapshot}
          localTitle={pageDetails.title}
          localMarkdown={md}
          localCoverUrl={pageDetails.coverUrl}
          localVisibility={pageDetails.visibility}
          isSaving={updateNoteMutation.isPending}
          onKeepEditing={() => setSaveConflict(null)}
          onLoadServer={handleLoadServerVersion}
          onOverwrite={() => void handleOverwriteConflict()}
        />
      )}
    </div>
  );
}
