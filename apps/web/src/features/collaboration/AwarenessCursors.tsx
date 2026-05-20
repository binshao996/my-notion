import { useEffect, useState } from "react";
import { useCollaboration } from "./CollaborationProvider";

interface RemoteCursor {
  clientId: number;
  name: string;
  color: string;
  blockId: string;
  from: number;
  to: number;
}

export default function AwarenessCursors() {
  const collab = useCollaboration();
  const [cursors, setCursors] = useState<RemoteCursor[]>([]);

  useEffect(() => {
    if (!collab) return;

    const handler = () => {
      const states = collab.awareness.getStates() as Map<number, { name: string; color: string; cursor?: { blockId: string; from: number; to: number } | null }>;
      const remote: RemoteCursor[] = [];
      states.forEach((state, clientId) => {
        // Skip self
        if (clientId === collab.awareness.clientID) return;
        if (!state?.cursor) return;
        remote.push({
          clientId,
          name: state.name,
          color: state.color,
          blockId: state.cursor.blockId,
          from: state.cursor.from,
          to: state.cursor.to,
        });
      });
      setCursors(remote);
    };

    collab.awareness.onChange(handler);
    return () => {
      collab.awareness.offChange(handler);
    };
  }, [collab]);

  if (!collab || cursors.length === 0) return null;

  return (
    <>
      {cursors.map((c) => (
        <div
          key={c.clientId}
          className="fixed pointer-events-none z-50"
          style={{
            // Position is approximate — cursors rendered near the block they're in
            // For precise cursor positioning, we'd need ProseMirror decoration integration
            // which yCursorPlugin handles natively. This is a fallback indicator.
            top: "4rem",
            right: "1rem",
          }}
        >
          <div className="flex items-center gap-1">
            <div
              className="w-3 h-3 rounded-full"
              style={{ backgroundColor: c.color }}
            />
            <span className="text-xs text-gray-500">{c.name}</span>
          </div>
        </div>
      ))}
    </>
  );
}
