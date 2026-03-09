import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import api from "../api/axios";

interface PublicNoteItem {
  id: number;
  title: string;
  slug: string;
  user_id: number;
  author_username: string;
  excerpt: string;
  created_at: string;
  updated_at: string;
}

interface PublicNotesPage {
  items: PublicNoteItem[];
  total: number;
  limit: number;
  offset: number;
}

export default function PostList() {
  const { accessToken } = useAuth();
  const [data, setData] = useState<PublicNotesPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .get<PublicNotesPage>("/api/v1/public/notes", { params: { limit: 50, offset: 0 } })
      .then((res) => setData(res.data))
      .catch(() => setError("Failed to load posts"))
      .finally(() => setLoading(false));
  }, []);

  const formatDate = (s: string) => {
    try {
      return new Date(s).toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
    } catch {
      return s;
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b bg-white">
        <div className="max-w-3xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/" className="font-semibold text-gray-800">
            Blogedit
          </Link>
          <nav className="flex items-center gap-4">
            {accessToken ? (
              <>
                <Link to="/editor" className="text-blue-600 hover:underline">
                  Write
                </Link>
                <span className="text-gray-500 text-sm">Logged in</span>
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
        <h1 className="text-2xl font-bold text-gray-800 mb-6">Public posts</h1>

        {loading && <p className="text-gray-500">Loading…</p>}
        {error && <p className="text-red-600">{error}</p>}

        {data && !loading && (
          <ul className="space-y-6">
            {data.items.length === 0 ? (
              <li className="text-gray-500">No public posts yet.</li>
            ) : (
              data.items.map((post) => (
                <li key={post.id}>
                  <Link
                    to={`/post/${post.author_username}/${post.slug}`}
                    className="block p-4 rounded-lg bg-white border border-gray-200 hover:border-gray-300 hover:shadow-sm transition"
                  >
                    <h2 className="text-lg font-semibold text-gray-800 mb-1">
                      {post.title || "(Untitled)"}
                    </h2>
                    <p className="text-sm text-gray-500 mb-2">
                      {post.author_username} · {formatDate(post.updated_at)}
                    </p>
                    {post.excerpt && (
                      <p className="text-gray-600 text-sm line-clamp-2">
                        {post.excerpt}
                      </p>
                    )}
                  </Link>
                </li>
              ))
            )}
          </ul>
        )}
      </main>
    </div>
  );
}
