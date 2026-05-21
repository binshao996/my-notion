import { useState, useCallback } from "react";
import type { Editor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";

// ---------- constants ----------

const PRESET_COLORS = [
  { name: "Default", value: "" },
  { name: "Gray", value: "#9CA3AF" },
  { name: "Red", value: "#EF4444" },
  { name: "Orange", value: "#F97316" },
  { name: "Green", value: "#22C55E" },
  { name: "Blue", value: "#3B82F6" },
  { name: "Purple", value: "#A855F7" },
  { name: "Pink", value: "#EC4899" },
];

// ---------- types ----------

interface FormatToolbarProps {
  editor: Editor;
}

// ---------- Link input sub-component ----------

function LinkInput({
  editor,
  onClose,
}: {
  editor: Editor;
  onClose: () => void;
}) {
  const [url, setUrl] = useState("");

  const apply = useCallback(() => {
    if (!url.trim()) {
      editor.chain().focus().unsetLink().run();
      onClose();
      return;
    }
    // Prepend https:// if no protocol
    let href = url.trim();
    if (!/^https?:\/\//i.test(href)) {
      href = "https://" + href;
    }
    editor.chain().focus().setLink({ href }).run();
    onClose();
  }, [editor, url, onClose]);

  return (
    <div className="flex items-center gap-1 rounded-md border border-gray-200 bg-white p-1 shadow-lg">
      <input
        autoFocus
        className="w-40 rounded border border-gray-200 px-2 py-0.5 text-xs outline-none focus:border-blue-400"
        placeholder="Paste link..."
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") apply();
          if (e.key === "Escape") onClose();
        }}
      />
      <button
        className="rounded px-2 py-0.5 text-xs text-blue-600 hover:bg-blue-50"
        onClick={apply}
      >
        Apply
      </button>
    </div>
  );
}

// ---------- color picker sub-component ----------

function ColorPicker({
  editor,
  onClose,
}: {
  editor: Editor;
  onClose: () => void;
}) {
  const currentColor = editor.getAttributes("textStyle").color || "";
  const [showPicker, setShowPicker] = useState(true);

  return (
    showPicker && (
      <div className="flex flex-wrap gap-1 rounded-md border border-gray-200 bg-white p-2 shadow-lg">
        {PRESET_COLORS.map((c) => (
          <button
            key={c.value}
            className="flex h-5 w-5 items-center justify-center rounded-full border border-gray-200 hover:scale-110 transition-transform"
            style={{ backgroundColor: c.value || "inherit" }}
            title={c.name}
            onClick={() => {
              if (c.value) {
                editor.chain().focus().setColor(c.value).run();
              } else {
                editor.chain().focus().unsetColor().run();
              }
              setShowPicker(false);
              onClose();
            }}
          >
            {currentColor === c.value && (
              <span className="text-[8px] text-white drop-shadow">
                {c.value ? "✓" : "✕"}
              </span>
            )}
          </button>
        ))}
      </div>
    )
  );
}

// ---------- main toolbar ----------

export default function FormatToolbar({ editor }: FormatToolbarProps) {
  const [showLinkInput, setShowLinkInput] = useState(false);
  const [showColorPicker, setShowColorPicker] = useState(false);

  const toggleBold = useCallback(() => editor.chain().focus().toggleBold().run(), [editor]);
  const toggleItalic = useCallback(() => editor.chain().focus().toggleItalic().run(), [editor]);
  const toggleUnderline = useCallback(() => editor.chain().focus().toggleUnderline().run(), [editor]);
  const toggleStrike = useCallback(() => editor.chain().focus().toggleStrike().run(), [editor]);
  const toggleCode = useCallback(() => editor.chain().focus().toggleCode().run(), [editor]);

  const isMarkActive = useCallback(
    (mark: string, attrs?: Record<string, any>) =>
      editor.isActive(mark, attrs),
    [editor]
  );

  const isLinkActive = isMarkActive("link");
  const hasColor = !!editor.getAttributes("textStyle").color;

  const toggleLink = useCallback(() => {
    if (isLinkActive) {
      editor.chain().focus().unsetLink().run();
    } else {
      setShowLinkInput((v) => !v);
      setShowColorPicker(false);
    }
  }, [editor, isLinkActive]);

  const toggleColor = useCallback(() => {
    setShowColorPicker((v) => !v);
    setShowLinkInput(false);
  }, []);

  // ----- button style helpers -----

  const baseBtn =
    "flex h-6 w-6 items-center justify-center rounded text-xs transition-colors";
  const activeBtn = "bg-blue-100 text-blue-700";
  const inactiveBtn = "text-gray-500 hover:bg-gray-100 hover:text-gray-700";

  const btnClass = (active: boolean) => `${baseBtn} ${active ? activeBtn : inactiveBtn}`;

  return (
    <>
      <BubbleMenu
        editor={editor}
        options={{ placement: "top" }}
        className="flex items-center gap-0.5 rounded-lg border border-gray-200 bg-white p-1 shadow-lg"
      >
        {/* Bold */}
        <button
          className={btnClass(isMarkActive("bold"))}
          onClick={toggleBold}
          title="Bold (Ctrl+B)"
        >
          <span className="font-bold">B</span>
        </button>

        {/* Italic */}
        <button
          className={btnClass(isMarkActive("italic"))}
          onClick={toggleItalic}
          title="Italic (Ctrl+I)"
        >
          <span className="italic">I</span>
        </button>

        {/* Underline */}
        <button
          className={btnClass(isMarkActive("underline"))}
          onClick={toggleUnderline}
          title="Underline (Ctrl+U)"
        >
          <span className="underline">U</span>
        </button>

        {/* Strikethrough */}
        <button
          className={btnClass(isMarkActive("strike"))}
          onClick={toggleStrike}
          title="Strikethrough"
        >
          <span className="line-through">S</span>
        </button>

        {/* Code */}
        <button
          className={btnClass(isMarkActive("code"))}
          onClick={toggleCode}
          title="Code"
        >
          <span className="font-mono">&lt;/&gt;</span>
        </button>

        {/* Separator */}
        <div className="mx-0.5 h-4 w-px bg-gray-200" />

        {/* Link */}
        <button
          className={btnClass(isLinkActive)}
          onClick={toggleLink}
          title="Link"
        >
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
          </svg>
        </button>

        {/* Text Color */}
        <div className="relative">
          <button
            className={`${baseBtn} ${hasColor ? "bg-blue-100 text-blue-700" : inactiveBtn}`}
            onClick={toggleColor}
            title="Text Color"
          >
            <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
            </svg>
          </button>

          {showColorPicker && (
            <div className="absolute left-0 top-full mt-1">
              <ColorPicker editor={editor} onClose={() => setShowColorPicker(false)} />
            </div>
          )}
        </div>

        {showLinkInput && (
          <div className="absolute left-0 top-full mt-1">
            <LinkInput editor={editor} onClose={() => setShowLinkInput(false)} />
          </div>
        )}
      </BubbleMenu>
    </>
  );
}

export { PRESET_COLORS };
