import { useState } from "react";

interface CommentInputProps {
  placeholder?: string;
  onSubmit: (text: string) => void;
}

export default function CommentInput({ placeholder = "Write a comment...", onSubmit }: CommentInputProps) {
  const [text, setText] = useState("");

  const handleSubmit = () => {
    if (!text.trim()) return;
    onSubmit(text.trim());
    setText("");
  };

  return (
    <div className="flex gap-2">
      <input
        type="text"
        className="flex-1 rounded border border-gray-200 px-3 py-1.5 text-sm outline-none focus:border-blue-400"
        placeholder={placeholder}
        value={text}
        onChange={e => setText(e.target.value)}
        onKeyDown={e => {
          if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            handleSubmit();
          }
        }}
      />
      <button
        className="rounded bg-blue-600 px-3 py-1.5 text-xs text-white hover:bg-blue-700 disabled:opacity-50"
        onClick={handleSubmit}
        disabled={!text.trim()}
      >
        Send
      </button>
    </div>
  );
}
