import { useState, useCallback, useRef } from "react";
import BlockContent from "./BlockContent";
import { useEditorStore } from "./useEditorStore";
import type * as Y from "yjs";

export interface BlockData {
  id: number;
  type: string;
  props: Record<string, any>;
  parent_block_id: number | null;
  tempId?: string; // client-side temp ID for new blocks
}

interface BlockShellProps {
  block: BlockData;
  index: number;
  isSelected?: boolean;
  onUpdate: (index: number, props: Record<string, any>) => void;
  onChangeType: (index: number, newType: string) => void;
  onDelete: (index: number) => void;
  onDuplicate: (index: number) => void;
  onAddAfter: (index: number, type?: string) => void;
  onIndent: (index: number) => void;
  onOutdent: (index: number) => void;
  onFocus: (index: number) => void;
  ydoc?: Y.Doc | null;
  awareness?: any | null;
  workspaceId?: number | null;
}

const blockTypes = [
  { type: "paragraph", label: "Text" },
  { type: "heading1", label: "Heading 1" },
  { type: "heading2", label: "Heading 2" },
  { type: "heading3", label: "Heading 3" },
  { type: "bulleted_list_item", label: "Bulleted List" },
  { type: "numbered_list_item", label: "Numbered List" },
  { type: "todo", label: "To-do" },
  { type: "toggle", label: "Toggle" },
  { type: "quote", label: "Quote" },
  { type: "divider", label: "Divider" },
  { type: "callout", label: "Callout" },
  { type: "code", label: "Code" },
  { type: "image", label: "Image" },
  { type: "file", label: "File" },
  { type: "bookmark", label: "Bookmark" },
  { type: "equation", label: "Equation" },
  { type: "table_of_contents", label: "Table of Contents" },
  { type: "columns", label: "Columns" },
];

export default function BlockShell({
  block,
  index,
  isSelected = false,
  onUpdate,
  onChangeType,
  onDelete,
  onDuplicate,
  onAddAfter,
  onIndent,
  onOutdent,
  onFocus,
  ydoc,
  awareness,
  workspaceId,
}: BlockShellProps) {
  const [showMenu, setShowMenu] = useState(false);
  const [showTypeMenu, setShowTypeMenu] = useState(false);
  const [isComposing, setIsComposing] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const dragOverIndex = useEditorStore((s) => s.dragOverIndex);
  const setDragOver = useEditorStore((s) => s.setDragOver);
  const findBlockIndex = useEditorStore((s) => s.findBlockIndex);
  const moveBlock = useEditorStore((s) => s.moveBlock);
  const blockId = String(block.tempId || block.id || index);

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      e.dataTransfer.dropEffect = "move";
      setDragOver(index);
    },
    [index, setDragOver]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const draggedId = e.dataTransfer.getData("text/plain");
      if (!draggedId) return;
      const fromIndex = findBlockIndex(draggedId);
      if (fromIndex === -1 || fromIndex === index) return;
      moveBlock(fromIndex, index);
      setDragOver(null);
    },
    [index, findBlockIndex, moveBlock, setDragOver]
  );

  const handleUpdate = useCallback(
    (_html: string, text: string) => {
      onUpdate(index, { ...block.props, text });
    },
    [index, block.props, onUpdate]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // Skip keyboard shortcuts during IME composition (e.g. CJK input)
      if (isComposing) return;

      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        onAddAfter(index);
      }
      if (e.key === "Backspace" && block.props.text === "" && block.type !== "paragraph") {
        e.preventDefault();
        onChangeType(index, "paragraph");
      }
      if (e.key === "Tab") {
        e.preventDefault();
        if (e.shiftKey) {
          onOutdent(index);
        } else {
          onIndent(index);
        }
      }
    },
    [index, block, onAddAfter, onChangeType, onIndent, onOutdent, isComposing]
  );

  const handleCompositionStart = useCallback(() => {
    setIsComposing(true);
  }, []);

  const handleCompositionEnd = useCallback(() => {
    setIsComposing(false);
  }, []);

  // Divider is a special case
  if (block.type === "divider") {
    return (
      <div
        className={`group relative my-4${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
        onClick={() => onFocus(index)}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <hr className="border-gray-200" />
        <div className="absolute -left-10 top-1/2 hidden -translate-y-1/2 group-hover:block">
          <BlockMenu
            onTurnInto={onChangeType}
            onDelete={onDelete}
            onDuplicate={onDuplicate}
            index={index}
          />
        </div>
      </div>
    );
  }

  // Image block
  if (block.type === "image") {
    const src = block.props?.url || block.props?.src || "";
    return (
      <div
        className={`group relative my-2${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
        onClick={() => onFocus(index)}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        {src ? (
          <img src={src} alt="" className="max-w-full rounded" />
        ) : (
          <div className="flex h-32 items-center justify-center rounded border-2 border-dashed border-gray-300 bg-gray-50 text-sm text-gray-400">
            Add an image URL
          </div>
        )}
        <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
          <button
            className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600 cursor-grab active:cursor-grabbing"
            draggable
            onClick={() => setShowMenu(!showMenu)}
            onDragStart={(e) => {
              e.dataTransfer.setData("text/plain", blockId);
              e.dataTransfer.effectAllowed = "move";
            }}
            onDragEnd={() => setDragOver(null)}
            title="Drag to reorder"
          >
            ⋮⋮
          </button>
        </div>
        {showMenu && (
          <div ref={menuRef} className="absolute -left-10 top-8 z-50 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => { onDuplicate(index); setShowMenu(false); }}>Duplicate</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => setShowTypeMenu(!showTypeMenu)}>Turn into ▸</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-red-600 hover:bg-gray-100" onClick={() => { onDelete(index); setShowMenu(false); }}>Delete</button>
            {showTypeMenu && (
              <div className="absolute -right-40 top-0 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {blockTypes.map((bt) => (
                  <button
                    key={bt.type}
                    className={`block w-full px-4 py-1 text-left text-sm hover:bg-gray-100 ${block.type === bt.type ? "font-medium text-blue-600" : "text-gray-700"}`}
                    onClick={() => { onChangeType(index, bt.type); setShowMenu(false); setShowTypeMenu(false); }}
                  >
                    {bt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  // File block
  if (block.type === "file") {
    const url = block.props?.url || "";
    const name = block.props?.name || block.props?.title || "File";
    return (
      <div
        className={`group relative my-2${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
        onClick={() => onFocus(index)}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <a
          href={url || "#"}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 rounded border border-gray-200 px-3 py-2 text-sm text-blue-600 hover:bg-blue-50"
        >
          <span>📎</span>
          <span>{name}</span>
        </a>
        <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
          <button
            className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600 cursor-grab active:cursor-grabbing"
            draggable
            onClick={() => setShowMenu(!showMenu)}
            onDragStart={(e) => {
              e.dataTransfer.setData("text/plain", blockId);
              e.dataTransfer.effectAllowed = "move";
            }}
            onDragEnd={() => setDragOver(null)}
            title="Drag to reorder"
          >
            ⋮⋮
          </button>
        </div>
        {showMenu && (
          <div ref={menuRef} className="absolute -left-10 top-8 z-50 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => { onDuplicate(index); setShowMenu(false); }}>Duplicate</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => setShowTypeMenu(!showTypeMenu)}>Turn into ▸</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-red-600 hover:bg-gray-100" onClick={() => { onDelete(index); setShowMenu(false); }}>Delete</button>
            {showTypeMenu && (
              <div className="absolute -right-40 top-0 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {blockTypes.map((bt) => (
                  <button
                    key={bt.type}
                    className={`block w-full px-4 py-1 text-left text-sm hover:bg-gray-100 ${block.type === bt.type ? "font-medium text-blue-600" : "text-gray-700"}`}
                    onClick={() => { onChangeType(index, bt.type); setShowMenu(false); setShowTypeMenu(false); }}
                  >
                    {bt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  // Bookmark block
  if (block.type === "bookmark") {
    const url = block.props?.url || "";
    const title = block.props?.title || "Bookmark";
    const desc = block.props?.description || "";
    return (
      <div
        className={`group relative my-2${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
        onClick={() => onFocus(index)}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <a
          href={url || "#"}
          target="_blank"
          rel="noopener noreferrer"
          className="block overflow-hidden rounded border border-gray-200 hover:bg-gray-50"
        >
          <div className="px-4 py-3">
            <div className="text-sm font-medium text-gray-900">{title}</div>
            {desc && <div className="mt-0.5 text-xs text-gray-500">{desc}</div>}
            {url && <div className="mt-0.5 text-xs text-gray-400 truncate">{url}</div>}
          </div>
        </a>
        <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
          <button
            className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600 cursor-grab active:cursor-grabbing"
            draggable
            onClick={() => setShowMenu(!showMenu)}
            onDragStart={(e) => {
              e.dataTransfer.setData("text/plain", blockId);
              e.dataTransfer.effectAllowed = "move";
            }}
            onDragEnd={() => setDragOver(null)}
            title="Drag to reorder"
          >
            ⋮⋮
          </button>
        </div>
        {showMenu && (
          <div ref={menuRef} className="absolute -left-10 top-8 z-50 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => { onDuplicate(index); setShowMenu(false); }}>Duplicate</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => setShowTypeMenu(!showTypeMenu)}>Turn into ▸</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-red-600 hover:bg-gray-100" onClick={() => { onDelete(index); setShowMenu(false); }}>Delete</button>
            {showTypeMenu && (
              <div className="absolute -right-40 top-0 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {blockTypes.map((bt) => (
                  <button
                    key={bt.type}
                    className={`block w-full px-4 py-1 text-left text-sm hover:bg-gray-100 ${block.type === bt.type ? "font-medium text-blue-600" : "text-gray-700"}`}
                    onClick={() => { onChangeType(index, bt.type); setShowMenu(false); setShowTypeMenu(false); }}
                  >
                    {bt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  // Equation block - rendered text with monospace styling
  if (block.type === "equation") {
    const eqContent = typeof block.props.text === "string" ? block.props.text : "";
    return (
      <div
        className={`group relative flex w-full items-start ${isSelected ? "border-l-2 border-blue-500 bg-blue-50 pl-1" : ""}${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
        onKeyDown={handleKeyDown}
        onCompositionStart={handleCompositionStart}
        onCompositionEnd={handleCompositionEnd}
        onClick={() => onFocus(index)}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
          <button
            className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600 cursor-grab active:cursor-grabbing"
            draggable
            onClick={() => setShowMenu(!showMenu)}
            onDragStart={(e) => {
              e.dataTransfer.setData("text/plain", blockId);
              e.dataTransfer.effectAllowed = "move";
            }}
            onDragEnd={() => setDragOver(null)}
            title="Drag to reorder"
          >
            ⋮⋮
          </button>
        </div>
        <div className="flex-1 py-0.5">
          <BlockContent
            type={block.type}
            content={eqContent}
            onUpdate={handleUpdate}
            onEnterPress={() => onAddAfter(index)}
            ydoc={ydoc ?? null}
            awareness={awareness ?? null}
            blockId={String(block.tempId || block.id || index)}
            workspaceId={workspaceId ?? null}
          />
          {showMenu && (
            <div ref={menuRef} className="absolute -left-10 top-8 z-50 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
              <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => { onDuplicate(index); setShowMenu(false); }}>Duplicate</button>
              <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => setShowTypeMenu(!showTypeMenu)}>Turn into ▸</button>
              <button className="block w-full px-4 py-1 text-left text-sm text-red-600 hover:bg-gray-100" onClick={() => { onDelete(index); setShowMenu(false); }}>Delete</button>
              {showTypeMenu && (
                <div className="absolute -right-40 top-0 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                  {blockTypes.map((bt) => (
                    <button
                      key={bt.type}
                      className={`block w-full px-4 py-1 text-left text-sm hover:bg-gray-100 ${block.type === bt.type ? "font-medium text-blue-600" : "text-gray-700"}`}
                      onClick={() => { onChangeType(index, bt.type); setShowMenu(false); setShowTypeMenu(false); }}
                    >
                      {bt.label}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    );
  }

  // Table of Contents block
  if (block.type === "table_of_contents") {
    return (
      <div
        className={`group relative my-2${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
        onClick={() => onFocus(index)}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <div className="rounded border border-gray-200 bg-gray-50 px-4 py-2 text-sm text-gray-500">
          Table of Contents
        </div>
        <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
          <button
            className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600 cursor-grab active:cursor-grabbing"
            draggable
            onClick={() => setShowMenu(!showMenu)}
            onDragStart={(e) => {
              e.dataTransfer.setData("text/plain", blockId);
              e.dataTransfer.effectAllowed = "move";
            }}
            onDragEnd={() => setDragOver(null)}
            title="Drag to reorder"
          >
            ⋮⋮
          </button>
        </div>
        {showMenu && (
          <div ref={menuRef} className="absolute -left-10 top-8 z-50 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => { onDuplicate(index); setShowMenu(false); }}>Duplicate</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => setShowTypeMenu(!showTypeMenu)}>Turn into ▸</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-red-600 hover:bg-gray-100" onClick={() => { onDelete(index); setShowMenu(false); }}>Delete</button>
            {showTypeMenu && (
              <div className="absolute -right-40 top-0 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {blockTypes.map((bt) => (
                  <button
                    key={bt.type}
                    className={`block w-full px-4 py-1 text-left text-sm hover:bg-gray-100 ${block.type === bt.type ? "font-medium text-blue-600" : "text-gray-700"}`}
                    onClick={() => { onChangeType(index, bt.type); setShowMenu(false); setShowTypeMenu(false); }}
                  >
                    {bt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  // Columns block - placeholder container
  if (block.type === "columns") {
    return (
      <div
        className={`group relative my-2${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
        onClick={() => onFocus(index)}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <div className="flex gap-4 rounded border-2 border-dashed border-gray-300 px-4 py-3">
          <div className="flex-1 rounded border border-dashed border-gray-200 bg-gray-50 p-3 text-center text-sm text-gray-400">
            Column
          </div>
          <div className="flex-1 rounded border border-dashed border-gray-200 bg-gray-50 p-3 text-center text-sm text-gray-400">
            Column
          </div>
        </div>
        <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
          <button
            className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600 cursor-grab active:cursor-grabbing"
            draggable
            onClick={() => setShowMenu(!showMenu)}
            onDragStart={(e) => {
              e.dataTransfer.setData("text/plain", blockId);
              e.dataTransfer.effectAllowed = "move";
            }}
            onDragEnd={() => setDragOver(null)}
            title="Drag to reorder"
          >
            ⋮⋮
          </button>
        </div>
        {showMenu && (
          <div ref={menuRef} className="absolute -left-10 top-8 z-50 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => { onDuplicate(index); setShowMenu(false); }}>Duplicate</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100" onClick={() => setShowTypeMenu(!showTypeMenu)}>Turn into ▸</button>
            <button className="block w-full px-4 py-1 text-left text-sm text-red-600 hover:bg-gray-100" onClick={() => { onDelete(index); setShowMenu(false); }}>Delete</button>
            {showTypeMenu && (
              <div className="absolute -right-40 top-0 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {blockTypes.map((bt) => (
                  <button
                    key={bt.type}
                    className={`block w-full px-4 py-1 text-left text-sm hover:bg-gray-100 ${block.type === bt.type ? "font-medium text-blue-600" : "text-gray-700"}`}
                    onClick={() => { onChangeType(index, bt.type); setShowMenu(false); setShowTypeMenu(false); }}
                  >
                    {bt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  const content = typeof block.props.text === "string" ? block.props.text : "";

  return (
    <div
      className={`group relative flex w-full items-start ${isSelected ? "border-l-2 border-blue-500 bg-blue-50 pl-1" : ""}${dragOverIndex === index ? " border-t-2 border-blue-500" : ""}`}
      onKeyDown={handleKeyDown}
      onCompositionStart={handleCompositionStart}
      onCompositionEnd={handleCompositionEnd}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {/* Hover handle */}
      <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
        <button
          className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600 cursor-grab active:cursor-grabbing"
          draggable
          onClick={() => setShowMenu(!showMenu)}
          onDragStart={(e) => {
            e.dataTransfer.setData("text/plain", blockId);
            e.dataTransfer.effectAllowed = "move";
          }}
          onDragEnd={() => setDragOver(null)}
          title="Drag to reorder"
        >
          ⋮⋮
        </button>
      </div>

      {/* Block content */}
      <div className="flex-1 py-0.5" onClick={() => onFocus(index)}>
        <BlockContent
          type={block.type}
          content={content}
          onUpdate={handleUpdate}
          onEnterPress={() => onAddAfter(index)}
          ydoc={ydoc ?? null}
          awareness={awareness ?? null}
          blockId={String(block.tempId || block.id || index)}
          workspaceId={workspaceId ?? null}
        />

        {/* Type change menu */}
        {showMenu && (
          <div ref={menuRef} className="absolute -left-10 top-8 z-50 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
            <button
              className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100"
              onClick={() => { onDuplicate(index); setShowMenu(false); }}
            >
              Duplicate
            </button>
            <button
              className="block w-full px-4 py-1 text-left text-sm text-gray-700 hover:bg-gray-100"
              onClick={() => setShowTypeMenu(!showTypeMenu)}
            >
              Turn into ▸
            </button>
            <button
              className="block w-full px-4 py-1 text-left text-sm text-red-600 hover:bg-gray-100"
              onClick={() => { onDelete(index); setShowMenu(false); }}
            >
              Delete
            </button>

            {showTypeMenu && (
              <div className="absolute -right-40 top-0 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {blockTypes.map((bt) => (
                  <button
                    key={bt.type}
                    className={`block w-full px-4 py-1 text-left text-sm hover:bg-gray-100 ${
                      block.type === bt.type ? "font-medium text-blue-600" : "text-gray-700"
                    }`}
                    onClick={() => { onChangeType(index, bt.type); setShowMenu(false); setShowTypeMenu(false); }}
                  >
                    {bt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// Mini menu for divider blocks
function BlockMenu({
  onDelete, onDuplicate, index,
}: {
  onTurnInto: (idx: number, type: string) => void;
  onDelete: (idx: number) => void;
  onDuplicate: (idx: number) => void;
  index: number;
}) {
  return (
    <div className="flex gap-1">
      <button
        className="rounded p-1 text-xs text-gray-400 hover:bg-gray-100"
        onClick={() => onDuplicate(index)}
      >
        ⧉
      </button>
      <button
        className="rounded p-1 text-xs text-gray-400 hover:bg-gray-100"
        onClick={() => onDelete(index)}
      >
        ✕
      </button>
    </div>
  );
}
