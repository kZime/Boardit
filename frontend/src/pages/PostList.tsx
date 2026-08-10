import { Link } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import { ArrowRight } from "lucide-react";
import { useListPublicNotes } from "../api/gen/client";
import SiteHeader from "../components/SiteHeader";

export default function PostList() {
  const { data: response, isLoading, isError } = useListPublicNotes({ limit: 50, offset: 0 });
  const data = response?.data;

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
    <div className="min-h-screen bg-gray-100/80 dark:bg-gray-900 transition-colors">
      <SiteHeader />

      <main className="max-w-2xl mx-auto px-4 py-10 sm:py-12">
        <h1 className="text-sm font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500 mb-8">
          Public posts
        </h1>

        {isLoading && (
          <p className="text-gray-500 dark:text-gray-400 text-sm">Loading…</p>
        )}
        {isError && (
          <p className="text-red-600 dark:text-red-400 text-sm">Failed to load posts</p>
        )}

        {data && !isLoading && (
          <ul className="space-y-4">
            {data.items.length === 0 ? (
              <li className="text-center py-12 text-gray-500 dark:text-gray-400 text-sm">
                No public posts yet.
              </li>
            ) : (
              data.items.map((post) => {
                const coverUrl = post.cover_url || null;
                return (
                  <li key={post.id}>
                    <Link
                      to={`/post/${post.author_username}/${post.slug}`}
                      className="group block rounded-xl bg-white dark:bg-gray-800 border border-gray-200/80 dark:border-gray-700 shadow-sm hover:shadow-md dark:shadow-none dark:hover:border-gray-600 transition-all duration-200 hover:-translate-y-0.5 overflow-hidden"
                    >
                      {coverUrl && (
                        <div className="relative w-full aspect-[16/9] overflow-hidden bg-gray-100 dark:bg-gray-900">
                          <img
                            src={coverUrl}
                            alt={post.title || "(Untitled)"}
                            className="h-full w-full object-cover"
                          />
                        </div>
                      )}
                      <div className="p-6 sm:p-8">
                        <h2 className="text-lg sm:text-xl font-bold text-gray-900 dark:text-gray-100 tracking-tight mb-2 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                          {post.title || "(Untitled)"}
                        </h2>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mb-4 flex items-center gap-2">
                          <span className="font-medium text-gray-600 dark:text-gray-300">
                            {post.author_username}
                          </span>
                          <span aria-hidden className="text-gray-300 dark:text-gray-600">·</span>
                          <span>{formatDate(post.updated_at)}</span>
                        </p>
                        {post.excerpt && (
                          <div className="line-clamp-2 overflow-hidden">
                            <div className="prose prose-sm max-w-none prose-p:my-0.5 prose-p:text-gray-600 dark:prose-p:text-gray-400 prose-headings:my-0.5 prose-headings:text-sm prose-headings:font-semibold prose-headings:text-gray-700 dark:prose-headings:text-gray-200">
                              <ReactMarkdown>{post.excerpt}</ReactMarkdown>
                            </div>
                          </div>
                        )}
                        <p className="mt-4 text-xs text-blue-600 dark:text-blue-400 font-medium inline-flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                          Read more
                          <ArrowRight className="w-3.5 h-3.5" />
                        </p>
                      </div>
                    </Link>
                  </li>
                );
              })
            )}
          </ul>
        )}
      </main>
    </div>
  );
}
