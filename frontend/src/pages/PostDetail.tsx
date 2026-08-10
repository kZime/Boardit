import { Link, useParams } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import { useAuth } from "../contexts/AuthContext";
import { useGetCurrentUser, useGetPublicNote } from "../api/gen/client";
import SiteHeader from "../components/SiteHeader";
import PostHeaderCard from "../components/PostHeaderCard";
import { parseFrontmatterAndBody } from "../utils/markdownCover";

export default function PostDetail() {
  const { username, slug } = useParams<{ username: string; slug: string }>();
  const { accessToken } = useAuth();
  const hasValidPath = Boolean(username && slug);
  const {
    data: noteResponse,
    isLoading,
    isError,
  } = useGetPublicNote(username ?? "", slug ?? "", {
    query: { enabled: hasValidPath },
  });
  const { data: userResponse } = useGetCurrentUser({
    query: { enabled: Boolean(accessToken), retry: false },
  });
  const note = noteResponse?.data;
  const currentUserId = userResponse?.data.id ?? null;
  const error = !hasValidPath ? "Invalid URL" : isError ? "Post not found" : null;

  const formatDate = (s: string) => {
    try {
      return new Date(s).toLocaleDateString(undefined, {
        year: "numeric",
        month: "long",
        day: "numeric",
      });
    } catch {
      return s;
    }
  };

  const isAuthor = note && currentUserId !== null && note.user_id === currentUserId;
  const coverUrl = note ? (note.cover_url || null) : null;
  const bodyMarkdown = note ? parseFrontmatterAndBody(note.content_md || "").body : "";

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 transition-colors">
      <SiteHeader
        showPostsLink
        editLink={
          isAuthor && note
            ? { to: `/editor?noteId=${note.id}`, label: "Edit" }
            : null
        }
      />

      <main className="max-w-3xl mx-auto px-4 py-8">
        {isLoading && (
          <p className="text-gray-500 dark:text-gray-400">Loading…</p>
        )}
        {error && (
          <p className="text-red-600 dark:text-red-400">
            {error}{" "}
            <Link to="/" className="underline hover:no-underline">
              Back to posts
            </Link>
          </p>
        )}

        {note && !isLoading && (
          <>
            <PostHeaderCard title={note.title || "(Untitled)"} coverUrl={coverUrl} />
            <article className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 md:p-8 shadow-sm dark:shadow-none">
              <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
                {note.author_username} · {formatDate(note.updated_at)}
              </p>
              <div className="prose prose-gray dark:prose-invert max-w-none prose-headings:text-gray-900 dark:prose-headings:text-gray-100 prose-p:text-gray-700 dark:prose-p:text-gray-300">
                <ReactMarkdown>{bodyMarkdown}</ReactMarkdown>
              </div>
            </article>
          </>
        )}
      </main>
    </div>
  );
}
