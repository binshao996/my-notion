import { useState, useRef, useEffect } from "react";
import type { View, ViewType } from "../../types/database";

interface ViewSwitcherProps {
  views: View[];
  activeViewId: number | null;
  onSelect: (id: number) => void;
  onCreate: (name: string, type: ViewType) => void;
  onDelete: (id: number) => void;
  onRename: (id: number, name: string) => void;
}

const VIEW_TYPES: { type: ViewType; label: string }[] = [
  { type: "table", label: "Table" },
  { type: "board", label: "Board" },
  { type: "calendar", label: "Calendar" },
  { type: "list", label: "List" },
  { type: "gallery", label: "Gallery" },
];

export default function ViewSwitcher({
  views,
  activeViewId,
  onSelect,
  onCreate,
  onDelete,
  onRename,
}: ViewSwitcherProps) {
  const [openMenuFor, setOpenMenuFor] = useState<number | null>(null);
  const [showAddPopover, setShowAddPopover] = useState(false);
  const [renamingId, setRenamingId] = useState<number | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const menuRef = useRef<HTMLDivElement | null>(null);
  const addRef = useRef<HTMLDivElement | null>(null);

  // Close menus on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      const target = e.target as HTMLElement;
      if (menuRef.current && !menuRef.current.contains(target)) {
        setOpenMenuFor(null);
      }
      if (addRef.current && !addRef.current.contains(target)) {
        setShowAddPopover(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const handleStartRename = (id: number, currentName: string) => {
    setRenamingId(id);
    setRenameValue(currentName);
    setOpenMenuFor(null);
  };

  const handleSubmitRename = () => {
    if (renamingId !== null && renameValue.trim()) {
      onRename(renamingId, renameValue.trim());
    }
    setRenamingId(null);
    setRenameValue("");
  };

  return (
    <div className="flex items-center gap-1 border-b border-gray-200 px-4">
      {views.map((v) => (
        <div key={v.id} className="relative flex items-center">
          {renamingId === v.id ? (
            <input
              autoFocus
              className="border-b-2 border-blue-600 bg-transparent px-3 py-2 text-xs text-gray-700 outline-none"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onBlur={handleSubmitRename}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleSubmitRename();
                if (e.key === "Escape") {
                  setRenamingId(null);
                  setRenameValue("");
                }
              }}
            />
          ) : (
            <>
              <button
                onClick={() => onSelect(v.id)}
                onContextMenu={(e) => {
                  e.preventDefault();
                  setOpenMenuFor(v.id);
                }}
                className={`border-b-2 px-3 py-2 text-xs ${
                  v.id === activeViewId
                    ? "border-blue-600 text-blue-600"
                    : "border-transparent text-gray-500 hover:text-gray-700"
                }`}
              >
                {v.name}
              </button>
              {/* Dropdown arrow */}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setOpenMenuFor(openMenuFor === v.id ? null : v.id);
                }}
                className="ml-0.5 text-gray-400 hover:text-gray-600"
              >
                <svg className="h-2.5 w-2.5" viewBox="0 0 10 6" fill="currentColor">
                  <path d="M5 6L0 0h10z" />
                </svg>
              </button>
            </>
          )}

          {/* Dropdown menu */}
          {openMenuFor === v.id && (
            <div
              ref={menuRef}
              className="absolute left-0 top-full z-20 mt-1 w-32 rounded border border-gray-200 bg-white py-1 shadow-lg"
            >
              <button
                className="w-full px-3 py-1.5 text-left text-xs text-gray-700 hover:bg-gray-100"
                onClick={() => handleStartRename(v.id, v.name)}
              >
                Rename
              </button>
              <button
                className="w-full px-3 py-1.5 text-left text-xs text-red-500 hover:bg-gray-100"
                onClick={() => {
                  onDelete(v.id);
                  setOpenMenuFor(null);
                }}
              >
                Delete
              </button>
            </div>
          )}
        </div>
      ))}

      {/* + Button with popover */}
      <div ref={addRef} className="relative">
        <button
          onClick={() => setShowAddPopover(!showAddPopover)}
          className="px-2 py-2 text-xs text-gray-400 hover:text-gray-600"
        >
          +
        </button>
        {showAddPopover && (
          <div className="absolute left-0 top-full z-20 mt-1 w-36 rounded border border-gray-200 bg-white py-1 shadow-lg">
            {VIEW_TYPES.map(({ type, label }) => (
              <button
                key={type}
                className="w-full px-3 py-1.5 text-left text-xs text-gray-700 hover:bg-gray-100"
                onClick={() => {
                  onCreate(`New ${label}`, type);
                  setShowAddPopover(false);
                }}
              >
                {label}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
