import { useState, useEffect, useRef, useCallback } from "react";
import { api } from "../../lib/api";

// ---------- types ----------

interface Page {
  id: number;
  workspace_id: number;
  parent_page_id: number | null;
  title: string;
  icon: string;
  cover: string;
  created_at: string;
  updated_at: string;
}

interface User {
  id: number;
  email: string;
  name: string;
  avatar_url: string;
}

type MentionCategory = "pages" | "people" | "date";

interface MentionItem {
  id: string;
  label: string;
  category: MentionCategory;
  detail?: string;      // subtitle e.g. page path or email
  data: Page | User | string;  // the raw item
}

// ---------- constants ----------

const PRESET_COLORS: Record<string, { bg: string; text: string }> = {
  pages: { bg: "bg-blue-50", text: "text-blue-700" },
  people: { bg: "bg-green-50", text: "text-green-700" },
  date: { bg: "bg-amber-50", text: "text-amber-700" },
};

// ---------- helpers ----------

function formatDate(): string {
  const d = new Date();
  return d.toLocaleDateString("en-US", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function formatShortDate(): string {
  const d = new Date();
  return d.toISOString().slice(0, 10); // YYYY-MM-DD
}

// ---------- props ----------

interface MentionPickerProps {
  isOpen: boolean;
  workspaceId: number | null;
  position: { top: number; left: number } | null;
  onSelect: (item: MentionItem) => void;
  onClose: () => void;
}

// ---------- component ----------

export default function MentionPicker({
  isOpen,
  workspaceId,
  position,
  onSelect,
  onClose,
}: MentionPickerProps) {
  const [search, setSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<MentionCategory>("pages");
  const [selectedIndex, setSelectedIndex] = useState(0);

  const [pages, setPages] = useState<Page[]>([]);
  const [members, setMembers] = useState<User[]>([]);
  const [loadingPages, setLoadingPages] = useState(false);
  const [loadingMembers, setLoadingMembers] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // ----- fetch data on open -----

  useEffect(() => {
    if (!isOpen || !workspaceId) return;

    setSearch("");
    setSelectedIndex(0);
    setActiveCategory("pages");
    setError(null);

    // Fetch pages
    setLoadingPages(true);
    api
      .get<Page[]>(`/workspaces/${workspaceId}/tree`)
      .then((data) => {
        setPages(Array.isArray(data) ? data : []);
        setLoadingPages(false);
      })
      .catch(() => {
        setPages([]);
        setLoadingPages(false);
      });

    // Fetch workspace members
    setLoadingMembers(true);
    api
      .get<User[]>(`/workspaces/${workspaceId}/members`)
      .then((data) => {
        setMembers(Array.isArray(data) ? data : []);
        setLoadingMembers(false);
      })
      .catch(() => {
        setMembers([]);
        setLoadingMembers(false);
      });

    setTimeout(() => inputRef.current?.focus(), 10);
  }, [isOpen, workspaceId]);

  // ----- build filtered list -----

  const dateItems: MentionItem[] = [
    {
      id: "date-today",
      label: formatDate(),
      category: "date",
      detail: formatShortDate(),
      data: formatShortDate(),
    },
    {
      id: "date-now",
      label: new Date().toLocaleDateString("en-US", {
        month: "long",
        day: "numeric",
        year: "numeric",
      }),
      category: "date",
      detail: "Short format",
      data: new Date().toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
      }),
    },
  ];

  const filterItems = (items: MentionItem[]) =>
    items.filter(
      (item) =>
        item.label.toLowerCase().includes(search.toLowerCase()) ||
        (item.detail && item.detail.toLowerCase().includes(search.toLowerCase()))
    );

  const filteredPages: MentionItem[] = filterItems(
    pages.map((p) => ({
      id: `page-${p.id}`,
      label: p.title || "Untitled",
      category: "pages" as MentionCategory,
      detail: p.icon || undefined,
      data: p,
    }))
  );

  const filteredPeople: MentionItem[] = filterItems(
    members.map((u) => ({
      id: `user-${u.id}`,
      label: u.name || u.email,
      category: "people" as MentionCategory,
      detail: u.email,
      data: u,
    }))
  );

  const filteredDates = filterItems(dateItems);

  // ----- select active list -----

  const activeList: MentionItem[] =
    activeCategory === "pages"
      ? filteredPages
      : activeCategory === "people"
        ? filteredPeople
        : filteredDates;

  // Reset index when list changes
  useEffect(() => {
    setSelectedIndex(0);
  }, [search, activeCategory]);

  // ----- keyboard handling -----

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((i) => Math.min(i + 1, activeList.length - 1));
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((i) => Math.max(i - 1, 0));
          break;
        case "Enter":
          e.preventDefault();
          if (activeList[selectedIndex]) {
            handleSelect(activeList[selectedIndex]);
          }
          break;
        case "Escape":
          e.preventDefault();
          onClose();
          break;
        case "Tab":
          e.preventDefault();
          // cycle categories
          const cats: MentionCategory[] = ["pages", "people", "date"];
          const idx = cats.indexOf(activeCategory);
          setActiveCategory(cats[(idx + 1) % cats.length]);
          break;
      }
    },
    [activeList, selectedIndex, activeCategory, onClose]
  );

  // ----- selection -----

  const handleSelect = useCallback(
    (item: MentionItem) => {
      onSelect(item);
      onClose();
    },
    [onSelect, onClose]
  );

  // ----- render -----

  if (!isOpen || !position) return null;

  const categories: { key: MentionCategory; label: string; count: number }[] = [
    { key: "pages", label: "Pages", count: filteredPages.length },
    { key: "people", label: "People", count: filteredPeople.length },
    { key: "date", label: "Date", count: filteredDates.length },
  ];

  const isLoading = loadingPages || loadingMembers;

  return (
    <div
      ref={containerRef}
      className="fixed z-50 w-72 rounded-lg border border-gray-200 bg-white shadow-xl"
      style={{
        top: position.top,
        left: position.left,
      }}
    >
      {/* Search input */}
      <div className="border-b border-gray-100 p-2">
        <input
          ref={inputRef}
          className="w-full rounded px-2 py-1 text-sm outline-none placeholder-gray-300"
          placeholder={`Search ${activeCategory}...`}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={handleKeyDown}
        />
      </div>

      {/* Category tabs */}
      <div className="flex border-b border-gray-100">
        {categories.map((cat) => (
          <button
            key={cat.key}
            className={`flex-1 px-2 py-1.5 text-xs font-medium transition-colors ${
              activeCategory === cat.key
                ? "border-b-2 border-blue-500 text-blue-600"
                : "text-gray-400 hover:text-gray-600"
            }`}
            onClick={() => setActiveCategory(cat.key)}
          >
            {cat.label}
            {cat.count > 0 && (
              <span className="ml-1 text-[10px] opacity-60">{cat.count}</span>
            )}
          </button>
        ))}
      </div>

      {/* Item list */}
      <div className="max-h-56 overflow-y-auto py-1">
        {isLoading && activeList.length === 0 && (
          <div className="px-3 py-4 text-center text-sm text-gray-400">
            Loading...
          </div>
        )}

        {error && (
          <div className="px-3 py-4 text-center text-sm text-red-400">
            {error}
          </div>
        )}

        {!isLoading && !error && activeList.length === 0 && (
          <div className="px-3 py-4 text-center text-sm text-gray-400">
            {search ? "No results found" : `No ${activeCategory} available`}
          </div>
        )}

        {activeList.map((item, idx) => {
          const colors = PRESET_COLORS[item.category] || {
            bg: "bg-gray-50",
            text: "text-gray-700",
          };
          return (
            <button
              key={item.id}
              className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm ${
                idx === selectedIndex
                  ? "bg-blue-50 text-blue-700"
                  : "text-gray-700 hover:bg-gray-50"
              }`}
              onClick={() => handleSelect(item)}
              onMouseEnter={() => setSelectedIndex(idx)}
            >
              {/* Category badge */}
              <span
                className={`inline-flex h-5 w-5 shrink-0 items-center justify-center rounded text-[10px] font-bold ${colors.bg} ${colors.text}`}
              >
                {item.category === "pages" ? "P" : item.category === "people" ? "@" : "D"}
              </span>

              {/* Label + detail */}
              <div className="min-w-0 flex-1">
                <div className="truncate">{item.label}</div>
                {item.detail && (
                  <div className="truncate text-xs text-gray-400">{item.detail}</div>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

export type { MentionItem, Page, User, MentionCategory };
