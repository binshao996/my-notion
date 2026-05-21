import { useState, useCallback, useRef, useEffect } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Underline from "@tiptap/extension-underline";
import LinkExtension from "@tiptap/extension-link";
import Color from "@tiptap/extension-color";
import { TextStyle } from "@tiptap/extension-text-style";
import { ySyncPlugin, yCursorPlugin } from "y-prosemirror";
import * as Y from "yjs";
import type { Awareness } from "../collaboration/websocket-provider";
import FormatToolbar from "./FormatToolbar";
import MentionPicker from "./MentionPicker";
import type { MentionItem } from "./MentionPicker";

interface BlockContentProps {
  type: string;
  content: string;
  placeholder?: string;
  onUpdate: (html: string, text: string) => void;
  onEnterPress?: () => void;
  readOnly?: boolean;
  ydoc?: Y.Doc | null;
  awareness?: Awareness | null;
  blockId?: string;
  workspaceId?: number | null;
}

const placeholderMap: Record<string, string> = {
  paragraph: "Type '/' for commands...",
  heading1: "Heading 1",
  heading2: "Heading 2",
  heading3: "Heading 3",
  bulleted_list_item: "List item",
  numbered_list_item: "List item",
  todo: "To-do",
  toggle: "Toggle",
  quote: "Quote",
  callout: "Type something...",
  code: "Write code...",
  equation: "e = mc²",
  file: "File name",
  bookmark: "Paste a link...",
  image: "Paste an image URL...",
  table_of_contents: "",
  columns: "",
};

export default function BlockContent({
  type,
  content,
  placeholder,
  onUpdate,
  onEnterPress: _onEnterPress,
  readOnly,
  ydoc,
  awareness,
  blockId,
  workspaceId,
}: BlockContentProps) {
  // ----- mention picker state -----
  const [showMention, setShowMention] = useState(false);
  const [mentionPos, setMentionPos] = useState<{ top: number; left: number } | null>(null);
  const mentionQueryRef = useRef<string>("");
  const editorContainerRef = useRef<HTMLDivElement>(null);

  const extensions: any[] = [
    StarterKit.configure({
      heading: false,
    }),
    Placeholder.configure({
      placeholder: placeholder || placeholderMap[type] || "Type '/' for commands...",
    }),
    Underline,
    LinkExtension.configure({
      openOnClick: true,
      HTMLAttributes: {
        class: "text-blue-600 underline cursor-pointer hover:text-blue-800",
      },
    }),
    TextStyle,
    Color,
  ];

  // Integrate Yjs collaborative editing
  if (ydoc && blockId) {
    const fragment = ydoc.getXmlFragment("block:" + blockId);
    extensions.push(ySyncPlugin(fragment));
    if (awareness) {
      extensions.push(yCursorPlugin(awareness as any));
    }
  }

  // ----- mention: insert selected item -----
  const handleMentionSelect = useCallback(
    (item: MentionItem) => {
      if (!editorRef.current) return;

      const editor = editorRef.current;
      const { state } = editor;
      const { from } = state.selection;

      let mentionText = "";

      if (item.category === "pages") {
        const page = item.data as { title: string; id: number };
        mentionText = `@[${page.title || "Untitled"}]`;
      } else if (item.category === "people") {
        const user = item.data as { name: string };
        mentionText = `@${user.name || "Unknown"}`;
      } else if (item.category === "date") {
        mentionText = `@${typeof item.data === "string" ? item.data : item.label}`;
      }

      // Remove the @ character and insert the mention text
      // Find the position of the last @ before the cursor
      const textBefore = state.doc.textBetween(0, from, "\0", "\0");
      const atIndex = textBefore.lastIndexOf("@");

      if (atIndex >= 0) {
        const absoluteAtPos = atIndex;
        editor
          .chain()
          .focus()
          .deleteRange({ from: absoluteAtPos, to: from })
          .insertContent(mentionText + " ")
          .run();
      } else {
        editor.chain().focus().insertContent(mentionText + " ").run();
      }

      setShowMention(false);
      setMentionPos(null);
    },
    []
  );

  const handleMentionClose = useCallback(() => {
    setShowMention(false);
    setMentionPos(null);
  }, []);

  // ----- update cursor position for mention picker -----
  const updateMentionPosition = useCallback((editor: NonNullable<ReturnType<typeof useEditor>>) => {
    const { view } = editor;
    const { state } = view;
    const { from } = state.selection;

    // Get coordinates at cursor
    const start = view.coordsAtPos(from);
    const editorRect = editorContainerRef.current?.getBoundingClientRect();

    if (editorRect) {
      setMentionPos({
        top: start.bottom + 4,
        left: start.left,
      });
    }
  }, []);

  // We need a ref to the editor instance so we can use it in callbacks
  // without stale closures
  const editorRef = useRef<ReturnType<typeof useEditor> | null>(null);

  const editor = useEditor({
    extensions,
    content: ydoc && blockId ? undefined : content,
    editable: !readOnly,
    onUpdate: ({ editor }) => {
      (editorRef as any).current = editor as any;

      const text = editor.getText();
      const html = editor.getHTML();

      // ----- @ mention detection -----
      const { state } = editor;
      const { from } = state.selection;
      const textBefore = state.doc.textBetween(0, from, "\0", "\0");

      // Check if the last typed character or the text buffer ends with @
      // that is not preceded by a word character (making it a true trigger)
      const atMatch = textBefore.match(/@([\w]*)$/);
      if (atMatch) {
        const beforeAt = textBefore.slice(0, atMatch.index);
        // Only trigger if @ is at a word boundary
        const charBeforeAt = beforeAt[beforeAt.length - 1] || "";
        const isWordBoundary = !charBeforeAt || /[\s\p{P}]/u.test(charBeforeAt);

        if (isWordBoundary && !showMention) {
          setShowMention(true);
          mentionQueryRef.current = atMatch[1];
          // Position after a microtask so the DOM has updated
          setTimeout(() => updateMentionPosition(editor as any), 10);
        } else if (isWordBoundary && showMention) {
          // Update query
          mentionQueryRef.current = atMatch[1];
        }
      } else if (showMention) {
        setShowMention(false);
        setMentionPos(null);
      }

      onUpdate(html, text);
    },
    onSelectionUpdate: ({ editor }) => {
      // If mention picker is open and selection changes, update position
      if (showMention) {
        (editorRef as any).current = editor as any;
        setTimeout(() => updateMentionPosition(editor as any), 10);
      }
    },
  });

  // Keep the ref updated
  useEffect(() => {
    if (editor) {
      (editorRef as any).current = editor as any;
    }
  }, [editor]);

  if (!editor) return null;

  const headingClass = type.startsWith("heading")
    ? {
        heading1: "text-3xl font-bold",
        heading2: "text-2xl font-semibold",
        heading3: "text-xl font-medium",
      }[type] || ""
    : "";

  return (
    <div ref={editorContainerRef} className={`relative w-full outline-none ${headingClass}`}>
      {/* Format toolbar (BubbleMenu) */}
      <FormatToolbar editor={editor as any} />

      {/* Mention picker */}
      {showMention && workspaceId && (
        <MentionPicker
          isOpen={showMention}
          workspaceId={workspaceId}
          position={mentionPos}
          onSelect={handleMentionSelect}
          onClose={handleMentionClose}
        />
      )}

      <EditorContent editor={editor} />
    </div>
  );
}

export { placeholderMap };
