import { useState, useCallback, useRef } from "react";
import BlockContent from "./BlockContent";

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
  onUpdate: (index: number, props: Record<string, any>) => void;
  onChangeType: (index: number, newType: string) => void;
  onDelete: (index: number) => void;
  onDuplicate: (index: number) => void;
  onAddAfter: (index: number, type?: string) => void;
  onIndent: (index: number) => void;
  onOutdent: (index: number) => void;
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
];

export default function BlockShell({
  block,
  index,
  onUpdate,
  onChangeType,
  onDelete,
  onDuplicate,
  onAddAfter,
  onIndent,
  onOutdent,
}: BlockShellProps) {
  const [showMenu, setShowMenu] = useState(false);
  const [showTypeMenu, setShowTypeMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const handleUpdate = useCallback(
    (_html: string, text: string) => {
      onUpdate(index, { ...block.props, text });
    },
    [index, block.props, onUpdate]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
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
    [index, block, onAddAfter, onChangeType, onIndent, onOutdent]
  );

  // Divider is a special case
  if (block.type === "divider") {
    return (
      <div className="group relative my-4">
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

  const content = typeof block.props.text === "string" ? block.props.text : "";

  return (
    <div className="group relative flex w-full items-start" onKeyDown={handleKeyDown}>
      {/* Hover handle */}
      <div className="absolute -left-10 top-0 hidden pt-1 group-hover:flex">
        <button
          className="flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          onClick={() => setShowMenu(!showMenu)}
          title="Drag to reorder"
        >
          ⋮⋮
        </button>
      </div>

      {/* Block content */}
      <div className="flex-1 py-0.5">
        <BlockContent
          type={block.type}
          content={content}
          onUpdate={handleUpdate}
          onEnterPress={() => onAddAfter(index)}
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
