import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import { useAuth } from "../contexts/AuthContext";
import api from "../api/axios";

interface PublicNote {
  id: number;
  user_id: number;
  title: string;
  slug: string;
  content_md: string;
  content_html: string;
  author_username: string;
  created_at: string;
  updated_at: string;
}

export default function PostDetail() {
  const { username, slug } = useParams<{ username: string; slug: string }>();
  const { accessToken } = useAuth();
  const [note, setNote] = useState<PublicNote | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentUserId, setCurrentUserId] = useState<number | null>(null);

  useEffect(() => {
    if (!username || !slug) {
      setError("Invalid URL");
      setLoading(false);
      return;
    }
    api
      .get<PublicNote>(`/api/v1/public/notes/${username}/${slug}`)
      .then((res) => setNote(res.data))
      .catch(() => setError("Post not found"))
      .finally(() => setLoading(false));
  }, [username, slug]);

  useEffect(() => {
    if (!accessToken) {
      setCurrentUserId(null);
      return;
    }
    api
      .get<{ id: number }>("/api/user")
      .then((res) => setCurrentUserId(res.data.id))
      .catch(() => setCurrentUserId(null));
  }, [accessToken]);

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

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b bg-white">
        <div className="max-w-3xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/" className="font-semibold text-gray-800">
            Blogedit
          </Link>
          <nav className="flex items-center gap-4">
            <Link to="/" className="text-gray-600 hover:underline">
              Posts
            </Link>
            {accessToken ? (
              <>
                <Link to="/editor" className="text-blue-600 hover:underline">
                  Write
                </Link>
                {isAuthor && (
                  <Link
                    to={`/editor?noteId=${note?.id}`}
                    className="text-blue-600 hover:underline"
                  >
                    Edit
                  </Link>
                )}
              </>
            ) : (
              <>
                <Link to="/login" className="text-gray-600 hover:underline">
                  Login
                </Link>
                <Link to="/register" className="text-gray-600 hover:underline">
                  Register
                </Link>
              </>
            )}
          </nav>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-4 py-8">
        {loading && <p className="text-gray-500">Loading…</p>}
        {error && (
          <p className="text-red-600">
            {error}{" "}
            <Link to="/" className="underline">
              Back to posts
            </Link>
          </p>
        )}

        {note && !loading && (
          <article className="bg-white rounded-lg border border-gray-200 p-6 md:p-8">
            <h1 className="text-3xl font-bold text-gray-800 mb-2">
              {note.title || "(Untitled)"}
            </h1>
            <p className="text-gray-500 text-sm mb-6">
              {note.author_username} · {formatDate(note.updated_at)}
            </p>
            <div className="prose prose-gray max-w-none">
              <ReactMarkdown>{note.content_md || ""}</ReactMarkdown>
            </div>
          </article>
        )}
      </main>
    </div>
  );
}
