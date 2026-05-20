import { useState, useEffect, useRef, useCallback } from "react";

interface CommandItem {
  type: string;
  label: string;
  icon: string;
  category: string;
}

const commands: CommandItem[] = [
  // Basic
  { type: "paragraph", label: "Text", icon: "¶", category: "Basic" },
  { type: "heading1", label: "Heading 1", icon: "H1", category: "Basic" },
  { type: "heading2", label: "Heading 2", icon: "H2", category: "Basic" },
  { type: "heading3", label: "Heading 3", icon: "H3", category: "Basic" },
  { type: "bulleted_list_item", label: "Bulleted List", icon: "•", category: "Basic" },
  { type: "numbered_list_item", label: "Numbered List", icon: "1.", category: "Basic" },
  { type: "todo", label: "To-do", icon: "☐", category: "Basic" },
  { type: "toggle", label: "Toggle", icon: "▸", category: "Basic" },
  { type: "quote", label: "Quote", icon: "\"", category: "Basic" },
  { type: "divider", label: "Divider", icon: "—", category: "Basic" },
  { type: "callout", label: "Callout", icon: "💡", category: "Basic" },
  { type: "code", label: "Code", icon: "</>", category: "Basic" },
  // Media
  { type: "image", label: "Image", icon: "🖼", category: "Media" },
];

interface CommandPaletteProps {
  isOpen: boolean;
  onSelect: (type: string) => void;
  onClose: () => void;
}

export default function CommandPalette({ isOpen, onSelect, onClose }: CommandPaletteProps) {
  const [search, setSearch] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen) {
      setSearch("");
      setSelectedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 10);
    }
  }, [isOpen]);

  const filtered = commands.filter(
    (c) =>
      c.label.toLowerCase().includes(search.toLowerCase()) ||
      c.type.toLowerCase().includes(search.toLowerCase())
  );

  const handleSelect = useCallback(
    (type: string) => {
      onSelect(type);
      onClose();
    },
    [onSelect, onClose]
  );

  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setSelectedIndex((i) => (i + 1) % filtered.length);
        break;
      case "ArrowUp":
        e.preventDefault();
        setSelectedIndex((i) => (i - 1 + filtered.length) % filtered.length);
        break;
      case "Enter":
        e.preventDefault();
        if (filtered[selectedIndex]) {
          handleSelect(filtered[selectedIndex].type);
        }
        break;
      case "Escape":
        onClose();
        break;
    }
  };

  if (!isOpen) return null;

  // Group by category
  const grouped: Record<string, CommandItem[]> = {};
  filtered.forEach((c) => {
    if (!grouped[c.category]) grouped[c.category] = [];
    grouped[c.category].push(c);
  });

  let globalIndex = 0;

  return (
    <div className="absolute z-50 w-64 rounded-lg border border-gray-200 bg-white shadow-xl">
      <div className="border-b border-gray-100 p-2">
        <input
          ref={inputRef}
          className="w-full rounded px-2 py-1 text-sm outline-none"
          placeholder="Filter..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setSelectedIndex(0); }}
          onKeyDown={handleKeyDown}
        />
      </div>
      <div className="max-h-64 overflow-y-auto py-1">
        {Object.entries(grouped).map(([category, items]) => (
          <div key={category}>
            <div className="px-3 pt-1 text-xs font-medium text-gray-400 uppercase">
              {category}
            </div>
            {items.map((item) => {
              const idx = globalIndex++;
              return (
                <button
                  key={item.type}
                  className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm ${
                    idx === selectedIndex
                      ? "bg-blue-50 text-blue-700"
                      : "text-gray-700 hover:bg-gray-50"
                  }`}
                  onClick={() => handleSelect(item.type)}
                >
                  <span className="w-5 text-center text-xs">{item.icon}</span>
                  <span>{item.label}</span>
                </button>
              );
            })}
          </div>
        ))}
        {filtered.length === 0 && (
          <div className="px-3 py-2 text-sm text-gray-400">No results</div>
        )}
      </div>
    </div>
  );
}

export { commands };
