import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";

interface BlockContentProps {
  type: string;
  content: string;
  placeholder?: string;
  onUpdate: (html: string, text: string) => void;
  onEnterPress?: () => void;
  readOnly?: boolean;
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
}: BlockContentProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: false, // we handle heading levels ourselves
      }),
      Placeholder.configure({
        placeholder: placeholder || placeholderMap[type] || "Type '/' for commands...",
      }),
    ],
    content: content,
    editable: !readOnly,
    onUpdate: ({ editor }) => {
      onUpdate(editor.getHTML(), editor.getText());
    },
  });

  if (!editor) return null;

  // Get the height class for headings
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
