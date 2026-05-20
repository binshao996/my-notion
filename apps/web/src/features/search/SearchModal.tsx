import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useSearchStore } from "../../stores/search";
import { useSearch } from "./useSearch";
import type { SearchResults } from "./types";

interface SearchResultItem {
  type: "page" | "block" | "record";
  label: string;
  sublabel: string;
  navigateTo: string;
}

function flattenResults(r: SearchResults): SearchResultItem[] {
  const items: SearchResultItem[] = [];
  for (const p of r.pages) {
    items.push({ type: "page", label: p.title, sublabel: "Page", navigateTo: `/page/${p.id}` });
  }
  for (const b of r.blocks) {
    items.push({ type: "block", label: b.text.slice(0, 80), sublabel: "Block", navigateTo: `/page/${b.page_id}` });
  }
  for (const r2 of r.records) {
    items.push({ type: "record", label: r2.title, sublabel: "Record", navigateTo: `/record/${r2.id}` });
  }
  return items;
}

const iconMap: Record<string, string> = {
  page: "📄",
  block: "📝",
  record: "🗂",
};

export default function SearchModal() {
  const isOpen = useSearchStore((s) => s.isOpen);
  const close = useSearchStore((s) => s.close);
  const workspaceId = useSearchStore((s) => s.activeWorkspaceId);
  const navigate = useNavigate();
  const { results, loading, search } = useSearch();

  const [query, setQuery] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const items = results ? flattenResults(results) : [];

  useEffect(() => {
    if (isOpen) {
      setQuery("");
      setSelectedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [isOpen]);

  useEffect(() => {
    if (workspaceId != null) {
      search(query, workspaceId);
    }
  }, [query, workspaceId, search]);

  useEffect(() => {
    setSelectedIndex(0);
  }, [results]);

  const goTo = useCallback(
    (item: SearchResultItem) => {
      close();
      navigate(item.navigateTo);
    },
    [close, navigate]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case "Escape":
          close();
          break;
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((i) => Math.min(i + 1, items.length - 1));
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((i) => Math.max(i - 1, 0));
          break;
        case "Enter":
          e.preventDefault();
          if (items[selectedIndex]) {
            goTo(items[selectedIndex]);
          }
          break;
      }
    },
    [close, items, selectedIndex, goTo]
  );

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/40"
      onClick={close}
    >
      <div
        className="w-full max-w-xl bg-white rounded-xl shadow-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center px-4 border-b">
          <span className="text-gray-400 mr-2">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          </span>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search pages, blocks, records..."
            className="w-full py-3 outline-none text-sm"
          />
          <kbd className="ml-2 text-xs text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded hidden sm:inline">
            Esc
          </kbd>
        </div>

        <div className="max-h-80 overflow-y-auto">
          {loading && (
            <div className="px-4 py-8 text-center text-sm text-gray-400">
              Searching...
            </div>
          )}

          {!loading && query && results && items.length === 0 && (
            <div className="px-4 py-8 text-center text-sm text-gray-400">
              No results for "{query}"
            </div>
          )}

          {!loading && !query && (
            <div className="px-4 py-8 text-center text-sm text-gray-400">
              Type to search across pages, blocks, and database records
            </div>
          )}

          {!loading && items.length > 0 && (
            <ul className="py-2">
              {items.map((item, i) => (
                <li
                  key={`${item.type}-${i}`}
                  className={`px-4 py-2 flex items-center gap-3 cursor-pointer text-sm ${
                    i === selectedIndex ? "bg-blue-50" : ""
                  }`}
                  onClick={() => goTo(item)}
                  onMouseEnter={() => setSelectedIndex(i)}
                >
                  <span className="text-base">{iconMap[item.type]}</span>
                  <div className="flex-1 min-w-0">
                    <div className="truncate">{item.label || "Untitled"}</div>
                    <div className="text-xs text-gray-400">{item.sublabel}</div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        {items.length > 0 && (
          <div className="px-4 py-1.5 border-t text-xs text-gray-400 flex gap-4">
            <span>↑↓ Navigate</span>
            <span>↵ Open</span>
            <span>Esc Close</span>
          </div>
        )}
      </div>
    </div>
  );
}
