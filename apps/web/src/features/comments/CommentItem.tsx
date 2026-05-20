interface CommentData {
  id: number;
  author_id: number;
  content: string;
  resolved: boolean;
  parent_id: number | null;
  created_at: string;
  Author?: { id: number; name: string; avatar_url: string };
}

interface CommentItemProps {
  comment: CommentData;
  onReply: (id: number) => void;
  onResolve: (id: number) => void;
  onDelete: (id: number) => void;
}

function parseContent(content: string): string {
  try {
    const parsed = JSON.parse(content);
    return parsed.text || content;
  } catch {
    return content;
  }
}

function formatTime(dateStr: string): string {
  const d = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return d.toLocaleDateString();
}

export default function CommentItem({ comment, onReply, onResolve, onDelete }: CommentItemProps) {
  const authorName = comment.Author?.name || `User #${comment.author_id}`;
  const text = parseContent(comment.content);

  return (
    <div className={`group rounded px-3 py-2 ${comment.resolved ? "opacity-50" : ""}`}>
      <div className="flex items-start gap-2">
        {/* Avatar placeholder */}
        <div className="flex h-6 w-6 items-center justify-center rounded-full bg-gray-200 text-xs font-medium text-gray-600">
          {authorName.charAt(0).toUpperCase()}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-900">{authorName}</span>
            <span className="text-xs text-gray-400">{formatTime(comment.created_at)}</span>
            {comment.resolved && <span className="text-xs text-green-600">Resolved</span>}
          </div>
          <div className="mt-0.5 text-sm text-gray-700">{text}</div>
          <div className="mt-1 flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
            <button className="text-xs text-gray-400 hover:text-gray-600" onClick={() => onReply(comment.id)}>
              Reply
            </button>
            <button className="text-xs text-gray-400 hover:text-gray-600" onClick={() => onResolve(comment.id)}>
              {comment.resolved ? "Reopen" : "Resolve"}
            </button>
            <button className="text-xs text-gray-400 hover:text-red-500" onClick={() => onDelete(comment.id)}>
              Delete
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
