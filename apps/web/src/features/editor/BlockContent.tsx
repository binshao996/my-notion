import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import { ySyncPlugin, yCursorPlugin } from "y-prosemirror";
import * as Y from "yjs";
import type { Awareness } from "../collaboration/websocket-provider";

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
}: BlockContentProps) {
  const extensions: any[] = [
    StarterKit.configure({
      heading: false,
    }),
    Placeholder.configure({
      placeholder: placeholder || placeholderMap[type] || "Type '/' for commands...",
    }),
  ];

  // Integrate Yjs collaborative editing
  if (ydoc && blockId) {
    const fragment = ydoc.getXmlFragment("block:" + blockId);
    extensions.push(ySyncPlugin(fragment));
    if (awareness) {
      extensions.push(yCursorPlugin(awareness as any));
    }
  }

  const editor = useEditor({
    extensions,
    content: ydoc && blockId ? undefined : content, // Yjs provides content, not initial prop
    editable: !readOnly,
    onUpdate: ({ editor }) => {
      onUpdate(editor.getHTML(), editor.getText());
    },
  });

  if (!editor) return null;

  const headingClass = type.startsWith("heading") ? {
    heading1: "text-3xl font-bold",
    heading2: "text-2xl font-semibold",
    heading3: "text-xl font-medium",
  }[type] || "" : "";

  return (
    <div className={`w-full outline-none ${headingClass}`}>
      <EditorContent editor={editor} />
    </div>
  );
}

export { placeholderMap };
