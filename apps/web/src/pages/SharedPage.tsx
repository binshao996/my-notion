import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import BlockContent from "../features/editor/BlockContent";
import { api } from "../lib/api";

interface SharedPageData {
  page: { id: number; title: string; icon: string; cover: string };
  blocks: { id: number; type: string; props: Record<string, any> }[];
  role: string;
}

export default function SharedPage() {
  const { token } = useParams<{ token: string }>();
  const [data, setData] = useState<SharedPageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    api.get<SharedPageData>(`/share/${token}`)
      .then(setData)
      .catch(e => setError(e.message || "Failed to load shared page"))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-white">
        <div className="text-sm text-gray-400">Loading shared page...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-white">
        <div className="text-center">
          <p className="text-lg font-semibold text-gray-700">Page not available</p>
          <p className="mt-1 text-sm text-gray-400">
            {error || "This share link may have expired or been revoked."}
          </p>
        </div>
      </div>
    );
  }

  const { page, blocks, role } = data;
  const isReadOnly = role !== "editor";

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="border-b border-gray-200 px-6 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {page.icon && <span className="text-lg">{page.icon}</span>}
            <h1 className="text-sm font-semibold text-gray-900">{page.title || "Untitled"}</h1>
          </div>
          <span className="text-xs text-gray-400 bg-gray-100 rounded px-2 py-0.5">
            {role === "editor" ? "Can edit" : role === "commenter" ? "Can comment" : "View only"}
          </span>
        </div>
      </header>

      {/* Blocks (read-only) */}
      <main className="mx-auto max-w-3xl px-24 py-8">
        {blocks.length === 0 && (
          <div className="text-sm text-gray-400 text-center py-8">This page is empty.</div>
        )}
        {blocks.map(block => (
          <div key={block.id} className="py-0.5">
            <BlockContent
              type={block.type}
              content={typeof block.props.text === "string" ? block.props.text : ""}
              onUpdate={() => {}}
              readOnly={isReadOnly}
            />
          </div>
        ))}
      </main>
    </div>
  );
}
