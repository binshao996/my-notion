import { useEffect, useState } from "react";
import { useCommentsStore } from "../../stores/comments";
import CommentItem from "./CommentItem";
import CommentInput from "./CommentInput";

interface CommentThreadProps {
  pageId: number;
}

export default function CommentThread({ pageId }: CommentThreadProps) {
  const { comments, loading, loadComments, addComment, resolveComment, deleteComment } = useCommentsStore();
  const [replyTo, setReplyTo] = useState<number | null>(null);

  useEffect(() => {
    loadComments(pageId);
  }, [pageId, loadComments]);

  const topLevel = comments.filter(c => !c.parent_id);
  const replies = (parentId: number) => comments.filter(c => c.parent_id === parentId);

  const handleAdd = (text: string) => {
    addComment(pageId, text, replyTo ?? undefined);
    setReplyTo(null);
  };

  if (loading) return <div className="text-xs text-gray-400 py-4 text-center">Loading comments...</div>;

  return (
    <div className="space-y-1">
      {/* Comment list */}
      {topLevel.length === 0 && !loading && (
        <div className="text-xs text-gray-400 py-4 text-center">No comments yet.</div>
      )}

      {topLevel.map(comment => (
        <div key={comment.id}>
          <CommentItem
            comment={comment}
            onReply={setReplyTo}
            onResolve={resolveComment}
            onDelete={deleteComment}
          />
          {/* Replies */}
          {replies(comment.id).map(reply => (
            <div key={reply.id} className="ml-6 border-l-2 border-gray-100 pl-3">
              <CommentItem
                comment={reply}
                onReply={setReplyTo}
                onResolve={resolveComment}
                onDelete={deleteComment}
              />
            </div>
          ))}
          {/* Reply input */}
          {replyTo === comment.id && (
            <div className="ml-6 pl-3 py-1">
              <CommentInput
                placeholder="Write a reply..."
                onSubmit={(text) => {
                  addComment(pageId, text, comment.id);
                  setReplyTo(null);
                }}
              />
            </div>
          )}
        </div>
      ))}

      {/* New comment input */}
      {replyTo === null && (
        <div className="pt-2">
          <CommentInput onSubmit={handleAdd} />
        </div>
      )}
    </div>
  );
}
