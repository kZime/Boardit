// src/pages/Editor.tsx
import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
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
  // toolbar items
  BoldItalicUnderlineToggles,
  BlockTypeSelect,
  ListsToggle,
  CreateLink,
  CodeToggle,
  UndoRedo,
  Separator,
} from "@mdxeditor/editor";

// Orval generated hooks
import {
  useListNotes,
  useCreateNote,
  useUpdateNote,
  useDeleteNote,
} from "../api/gen/client";
import type { Note } from "../api/gen/models/note";
import type { CreateNoteRequest } from "../api/gen/models/createNoteRequest";
import type { UpdateNoteRequest } from "../api/gen/models/updateNoteRequest";

export default function Editor() {
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
  const defaultMd =
    "# Headline here \n \n This is a paragraph \n \n This is a list \n - Item 1 \n - Item 2 \n - Item 3";

  const [md, setMd] = useState(defaultMd);

  // Edit page details modal state
  const [showEditModal, setShowEditModal] = useState(false);
  const [pageDetails, setPageDetails] = useState({
    title: defaultTitle,
    description: "",
    tags: "",
    visibility: "private" as "private" | "public" | "unlisted",
  });

  // Save success notification state
  const [showSaveSuccess, setShowSaveSuccess] = useState(false);

  // Delete confirmation modal: note to delete or null
  const [noteToDelete, setNoteToDelete] = useState<Note | null>(null);

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
        setMd(
          "# Headline here \n \n This is a paragraph \n \n This is a list \n - Item 1 \n - Item 2 \n - Item 3"
        );
        setPageDetails({
          title: "Untitled Page",
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
        content_md: md,
        is_published: pageDetails.visibility === "public",
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
        content_md: md,
        is_published: pageDetails.visibility === "public",
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
      setShowSaveSuccess(true);
      setTimeout(() => setShowSaveSuccess(false), 3000);
      setShowEditModal(false);
    } catch (error) {
      console.error("Failed to save page details:", error);
    }
  };

  const handlePageDetailsChange = async (field: string, value: string) => {
    setPageDetails((prev) => ({
      ...prev,
      [field]: value,
    }));

    // If we're changing the title and have a current note, save it immediately
    if (field === "title" && currentNoteId) {
      try {
        const updateData: UpdateNoteRequest = {
          title: value,
          content_md: md,
          is_published: pageDetails.visibility === "public",
          visibility: pageDetails.visibility as
            | "private"
            | "public"
            | "unlisted",
        };

        await updateNoteMutation.mutateAsync({
          id: currentNoteId,
          data: updateData,
        });
        refetchNotes(); // Refresh the notes list to update sidebar
      } catch (error) {
        console.error("Failed to save title:", error);
      }
    }
  };

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <header className="h-12 border-b bg-white flex items-center justify-between px-3">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            aria-controls="sidebar"
            className="px-2 py-1 rounded hover:bg-gray-100"
            title="Toggle sidebar"
          >
            ☰
          </button>
          <span className="font-medium">Editor</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              if (isDirtyRef.current && !window.confirm("You have unsaved changes. Leave anyway?")) return;
              logout();
            }}
            className="text-red-500 hover:underline"
          >
            Logout
          </button>
        </div>
      </header>

      {/* Main area: sidebar + editor */}
      <div className="flex-1 flex overflow-hidden bg-gray-50">
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
            "z-40 bg-white border-r overflow-y-auto",
            // Mobile: fixed position, Desktop: static position
            "fixed h-full md:static",
            // Control width and visibility based on open state
            open
              ? "w-72 translate-x-0 md:translate-x-0 md:w-72"
              : "w-72 -translate-x-full md:translate-x-0 md:w-0",
          ].join(" ")}
          style={{ transition: "transform .2s ease, width .2s ease" }}
        >
          <div className="p-3 border-b flex items-center justify-between">
            <h2 className="font-semibold">Notes</h2>
            <button
              onClick={handleNew}
              className="px-2 py-1 text-sm rounded bg-blue-600 text-white hover:bg-blue-700"
            >
              New
            </button>
          </div>

          <div className="p-2">
            {isLoading && (
              <div className="text-sm text-gray-500 p-2">Loading…</div>
            )}
            {isError && (
              <div className="text-sm text-red-600 p-2">
                Failed to load notes
              </div>
            )}
            <ul className="space-y-1">
              {(data?.data?.items ?? []).map((n: Note) => (
                <li key={n.id}>
                  <div className="group flex items-center justify-between gap-2 px-2 py-1 rounded hover:bg-gray-100">
                    <button
                      className={`text-left flex-1 truncate px-2 py-1 rounded ${
                        currentNoteId === n.id
                          ? "bg-blue-100 text-blue-800 font-medium"
                          : "hover:bg-gray-50"
                      }`}
                      onClick={() => handleSelectNote(n)}
                      title={n.title}
                    >
                      {n.title || "(Untitled)"}
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        setNoteToDelete(n);
                      }}
                      className="text-xs text-gray-500 px-1 py-0.5 rounded hover:bg-red-100 hover:text-red-600 opacity-0 group-hover:opacity-100 transition-opacity"
                      title="Delete"
                    >
                      ✕
                    </button>
                  </div>
                </li>
              ))}
              {!isLoading && (data?.data?.items?.length ?? 0) === 0 && (
                <li className="text-sm text-gray-500 p-2">No notes</li>
              )}
            </ul>
          </div>
        </aside>

        {/* editor area */}
        <main className="flex-1 overflow-auto p-4">
          {currentNoteId ? (
            // Show editor when a note is selected
            <>
              {/* Page Title */}
              <div className="mb-4">
                <input
                  type="text"
                  value={pageDetails.title}
                  onChange={(e) =>
                    handlePageDetailsChange("title", e.target.value)
                  }
                  className="w-full text-2xl font-bold bg-transparent border-none outline-none focus:bg-white focus:border focus:border-gray-300 focus:rounded-md focus:px-3 focus:py-2 transition-all duration-200"
                  placeholder="Enter page title..."
                />
              </div>

              <div className="bg-white rounded-xl p-4 shadow-sm">
                <MDXEditor
                  key={currentNoteId || "new"}
                  ref={editorRef}
                  className="min-h-[70vh]"
                  contentEditableClassName="prose"
                  markdown={md}
                  onChange={setMd}
                  plugins={[
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

              {/* Button container - aligned to the right with proper spacing */}
              <div className="mt-4 flex justify-end gap-5">
                <button
                  className="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600"
                  onClick={handleEditPage}
                >
                  Edit Details
                </button>
                <button
                  className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
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
            // Show welcome message when no note is selected
            <div className="flex items-center justify-center h-full">
              <div className="text-center max-w-md">
                <div className="mb-8">
                  <div className="text-6xl mb-4">📝</div>
                  <h2 className="text-2xl font-bold text-gray-800 mb-2">
                    Welcome to Blogedit
                  </h2>
                  <p className="text-gray-600 mb-6">
                    Start writing by creating a new note or selecting an
                    existing one from the sidebar.
                  </p>
                </div>

                <div className="space-y-4 text-left">
                  <div className="flex items-start gap-3">
                    <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span className="text-blue-600 font-semibold text-sm">
                        1
                      </span>
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-800">
                        Create a new note
                      </h3>
                      <p className="text-gray-600 text-sm">
                        Click the "New" button in the sidebar to start writing a
                        new article.
                      </p>
                    </div>
                  </div>

                  <div className="flex items-start gap-3">
                    <div className="w-8 h-8 bg-green-100 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span className="text-green-600 font-semibold text-sm">
                        2
                      </span>
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-800">
                        Select existing note
                      </h3>
                      <p className="text-gray-600 text-sm">
                        Click on any note in the sidebar to continue editing
                        where you left off.
                      </p>
                    </div>
                  </div>

                  <div className="flex items-start gap-3">
                    <div className="w-8 h-8 bg-purple-100 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span className="text-purple-600 font-semibold text-sm">
                        3
                      </span>
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-800">
                        Start writing
                      </h3>
                      <p className="text-gray-600 text-sm">
                        Use the rich text editor with markdown support to create
                        your content.
                      </p>
                    </div>
                  </div>
                </div>

                <div className="mt-8">
                  <button
                    onClick={handleNew}
                    className="px-6 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors font-medium"
                  >
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
          <div className="bg-green-500 text-white px-6 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-in slide-in-from-right duration-300">
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 13l4 4L19 7"
              />
            </svg>
            <span className="font-medium">Saved successfully!</span>
          </div>
        </div>
      )}

      {/* Edit Page Details Modal */}
      {showEditModal && (
        <div
          className="fixed inset-0 bg-gray-900/20 flex items-center justify-center z-50"
          onClick={handleCloseModal}
        >
          <div
            className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-6 border-b">
              <h3 className="text-lg font-semibold">Edit Page Details</h3>
              <button
                onClick={handleCloseModal}
                className="text-gray-400 hover:text-gray-600 text-xl"
              >
                ×
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
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter page title"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Description
                </label>
                <textarea
                  value={pageDetails.description}
                  onChange={(e) =>
                    handlePageDetailsChange("description", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  rows={3}
                  placeholder="Enter page description"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Tags
                </label>
                <input
                  type="text"
                  value={pageDetails.tags}
                  onChange={(e) =>
                    handlePageDetailsChange("tags", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter tags (comma separated)"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Visibility
                </label>
                <select
                  value={pageDetails.visibility}
                  onChange={(e) =>
                    handlePageDetailsChange("visibility", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
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
          className="fixed inset-0 bg-gray-900/20 flex items-center justify-center z-50"
          onClick={() => setNoteToDelete(null)}
        >
          <div
            className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-6 border-b">
              <h3 className="text-lg font-semibold text-gray-800">Delete note?</h3>
              <p className="text-sm text-gray-600 mt-1">
                “{noteToDelete.title || "Untitled"}” will be permanently deleted. This cannot be undone.
              </p>
            </div>
            <div className="flex justify-end gap-3 p-6 border-t">
              <button
                onClick={() => setNoteToDelete(null)}
                className="px-4 py-2 text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={() => void handleDelete(noteToDelete.id)}
                disabled={deleteNoteMutation.isPending}
                className="px-4 py-2 bg-red-500 text-white rounded-md hover:bg-red-600 disabled:opacity-50"
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
