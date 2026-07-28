import { useEffect, useState, useCallback, useMemo } from "react";
import { marked } from "marked";

async function api(path) {
  const res = await fetch(path);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${text}`);
  }
  return res.json();
}

function transformInterlinks(html) {
  const container = document.createElement("div");
  container.innerHTML = html;
  container.querySelectorAll('a[href$=".md"]').forEach((a) => {
    const href = a.getAttribute("href");
    a.setAttribute("href", "#page=" + encodeURIComponent(href));
  });
  return container.innerHTML;
}

function pageFromHash() {
  const hash = window.location.hash || "";
  const prefix = "#page=";
  if (!hash.startsWith(prefix)) return null;
  return decodeURIComponent(hash.slice(prefix.length));
}

export default function App() {
  const [pages, setPages] = useState([]);
  const [currentPage, setCurrentPage] = useState(null);
  const [page, setPage] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api("/api/pages")
      .then((data) => setPages(data.pages || []))
      .catch((e) => setError(e.message));
  }, []);

  const loadPage = useCallback(async (name) => {
    setLoading(true);
    setError(null);
    try {
      const data = await api("/api/pages/" + encodeURIComponent(name));
      setPage(data);
      setCurrentPage(name);
    } catch (e) {
      setError(e.message);
      setPage(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const onHash = () => {
      const name = pageFromHash();
      if (name) loadPage(name);
    };
    window.addEventListener("hashchange", onHash);
    onHash();
    return () => window.removeEventListener("hashchange", onHash);
  }, [loadPage]);

  if (error && pages.length === 0) {
    return (
      <div className="min-h-screen flex items-center justify-center text-red-600">
        Error: {error}
      </div>
    );
  }

  const renderedHtml = useMemo(
    () => (page ? transformInterlinks(marked.parse(page.markdown || "", { breaks: true })) : ""),
    [page]
  );

  return (
    <div className="min-h-screen flex">
      <aside className="w-72 bg-white border-r border-slate-200 p-4 overflow-y-auto">
        <h1 className="text-lg font-bold mb-4 text-blue-700">mocho</h1>
        <h2 className="text-xs uppercase tracking-wide text-slate-500 mb-2">
          Pages
        </h2>
        {pages.length === 0 ? (
          <p className="text-sm text-slate-500">
            No pages yet. Ingest a source to populate the wiki.
          </p>
        ) : (
          <ul className="space-y-1">
            {pages.map((p) => (
              <li key={p.name}>
                <a
                  href={"#page=" + encodeURIComponent(p.name)}
                  onClick={() => loadPage(p.name)}
                  className={
                    "block text-left w-full rounded px-2 py-1 hover:bg-slate-100 " +
                    (currentPage === p.name ? "bg-blue-50 text-blue-700 font-medium" : "")
                  }
                >
                  <span className="block text-sm">{p.title}</span>
                  {p.summary && (
                    <span className="block text-xs text-slate-500 truncate">
                      {p.summary}
                    </span>
                  )}
                </a>
              </li>
            ))}
          </ul>
        )}
      </aside>

      <main className="flex-1 p-8 overflow-y-auto">
        {loading && <p className="text-slate-500">Loading…</p>}
        {!loading && !page && (
          <div className="text-slate-500">
            <p>Select a page from the sidebar to start reading.</p>
          </div>
        )}
        {!loading && page && (
          <>
            <div className="text-xs text-slate-400 mb-2 font-mono">{page.name}</div>
            <div
              className="prose-mocho max-w-3xl"
              dangerouslySetInnerHTML={{ __html: renderedHtml }}
            />
          </>
        )}
        {error && page === null && !loading && (
          <p className="text-red-600 mt-4">{error}</p>
        )}
      </main>
    </div>
  );
}