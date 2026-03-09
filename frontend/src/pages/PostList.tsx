import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import api from "../api/axios";
import SiteHeader from "../components/SiteHeader";

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
    <div className="min-h-screen bg-gray-100/80">
      <SiteHeader />

      <main className="max-w-2xl mx-auto px-4 py-10 sm:py-12">
        <h1 className="text-sm font-medium uppercase tracking-wider text-gray-400 mb-8">
          Public posts
        </h1>

        {loading && (
          <p className="text-gray-500 text-sm">Loading…</p>
        )}
        {error && (
          <p className="text-red-600 text-sm">{error}</p>
        )}

        {data && !loading && (
          <ul className="space-y-4">
            {data.items.length === 0 ? (
              <li className="text-center py-12 text-gray-500 text-sm">
                No public posts yet.
              </li>
            ) : (
              data.items.map((post) => (
                <li key={post.id}>
                  <Link
                    to={`/post/${post.author_username}/${post.slug}`}
                    className="group block p-5 sm:p-6 rounded-xl bg-white border border-gray-200/80 hover:border-gray-300 hover:shadow-lg hover:shadow-gray-200/50 transition-all duration-200"
                  >
                    <h2 className="text-lg sm:text-xl font-bold text-gray-900 tracking-tight mb-1.5 group-hover:text-blue-600 transition-colors">
                      {post.title || "(Untitled)"}
                    </h2>
                    <p className="text-xs text-gray-500 mb-3 flex items-center gap-1.5">
                      <span className="font-medium text-gray-600">{post.author_username}</span>
                      <span aria-hidden className="text-gray-300">·</span>
                      <span>{formatDate(post.updated_at)}</span>
                    </p>
                    {post.excerpt && (
                      <div className="line-clamp-2 overflow-hidden">
                        <div className="prose prose-sm max-w-none prose-p:my-0.5 prose-p:text-gray-600 prose-headings:my-0.5 prose-headings:text-sm prose-headings:font-semibold prose-headings:text-gray-700">
                          <ReactMarkdown>{post.excerpt}</ReactMarkdown>
                        </div>
                      </div>
                    )}
                    <p className="mt-3 text-xs text-blue-600 font-medium opacity-0 group-hover:opacity-100 transition-opacity">
                      Read more →
                    </p>
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
