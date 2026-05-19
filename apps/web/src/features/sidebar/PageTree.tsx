import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { usePagesStore, Page } from "../../stores/pages";

function PageTreeItem({ page, depth = 0 }: { page: Page; depth?: number }) {
  const navigate = useNavigate();
  const { expandedIds, toggleExpanded, loadChildren, createPage } = usePagesStore();
  const [children, setChildren] = useState<Page[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [showNewInput, setShowNewInput] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const { pageId } = useParams();

  const isExpanded = expandedIds.has(page.id);
  const isActive = pageId === String(page.id);

  useEffect(() => {
    if (isExpanded && !loaded) {
      loadChildren(page.id).then((c) => {
        setChildren(c);
        setLoaded(true);
      });
    }
  }, [isExpanded, loaded, page.id, loadChildren]);

  const handleNewPage = async () => {
    if (!newTitle.trim()) {
      setShowNewInput(false);
      return;
    }
    await createPage(page.workspace_id, newTitle.trim(), page.id);
    setNewTitle("");
    setShowNewInput(false);
    setLoaded(false); // force reload children
    if (!isExpanded) {
      toggleExpanded(page.id);
    }
  };

  return (
    <div className="select-none">
      <div
        className={`group flex cursor-pointer items-center gap-1 rounded px-2 py-1 text-sm text-gray-700 hover:bg-gray-200 ${
          isActive ? "bg-gray-200 font-medium" : ""
        }`}
        style={{ paddingLeft: `${8 + depth * 16}px` }}
      >
        <button
          onClick={(e) => { e.stopPropagation(); toggleExpanded(page.id); }}
          className="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:text-gray-600"
        >
          {isExpanded ? "▾" : "▸"}
        </button>
        <span
          className="flex-1 truncate"
          onClick={() => navigate(`/page/${page.id}`)}
        >
          {page.icon && <span className="mr-1">{page.icon}</span>}
          {page.title || "Untitled"}
        </span>
        <button
          onClick={(e) => { e.stopPropagation(); setShowNewInput(!showNewInput); }}
          className="hidden text-gray-400 hover:text-gray-600 group-hover:inline-block"
          title="Add subpage"
        >
          +
        </button>
      </div>

      {showNewInput && (
        <div style={{ paddingLeft: `${8 + (depth + 1) * 16}px` }} className="px-2 py-1">
          <input
            autoFocus
            className="w-full rounded border border-blue-400 px-2 py-1 text-sm outline-none"
            placeholder="New page name"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleNewPage();
              if (e.key === "Escape") setShowNewInput(false);
            }}
            onBlur={handleNewPage}
          />
        </div>
      )}

      {isExpanded && children.map((child) => (
        <PageTreeItem key={child.id} page={child} depth={depth + 1} />
      ))}
    </div>
  );
}

export default function PageTree() {
  const { pages, loading, loadTree, createPage } = usePagesStore();
  const [showNewInput, setShowNewInput] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    loadTree(1); // TODO: dynamic workspace ID
  }, [loadTree]);

  const handleNewRootPage = async () => {
    if (!newTitle.trim()) {
      setShowNewInput(false);
      return;
    }
    const page = await createPage(1, newTitle.trim());
    setNewTitle("");
    setShowNewInput(false);
    navigate(`/page/${page.id}`);
  };

  if (loading) {
    return <div className="p-4 text-sm text-gray-400">Loading...</div>;
  }

  return (
    <div className="py-2">
      {pages.map((page) => (
        <PageTreeItem key={page.id} page={page} />
      ))}

      {showNewInput ? (
        <div className="px-3 py-1">
          <input
            autoFocus
            className="w-full rounded border border-blue-400 px-2 py-1 text-sm outline-none"
            placeholder="Page name"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleNewRootPage();
              if (e.key === "Escape") setShowNewInput(false);
            }}
            onBlur={handleNewRootPage}
          />
        </div>
      ) : (
        <button
          onClick={() => setShowNewInput(true)}
          className="w-full px-3 py-1 text-left text-sm text-gray-500 hover:bg-gray-100"
        >
          + New page
        </button>
      )}
    </div>
  );
}
