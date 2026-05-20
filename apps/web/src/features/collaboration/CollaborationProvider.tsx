import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import * as Y from "yjs";
import { PageAwareness, Awareness } from "./websocket-provider";
import { useAuthStore } from "../../stores/auth";

interface CollaborationContextValue {
  ydoc: Y.Doc;
  awareness: Awareness;
  connected: boolean;
}

const CollaborationContext = createContext<CollaborationContextValue | null>(null);

export function useCollaboration(): CollaborationContextValue | null {
  return useContext(CollaborationContext);
}

interface Props {
  pageId: number;
  children: ReactNode;
}

export default function CollaborationProvider({ pageId, children }: Props) {
  const user = useAuthStore((s) => s.user);
  const providerRef = useRef<PageAwareness | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const provider = new PageAwareness();
    providerRef.current = provider;

    provider.onConnected = () => setConnected(true);
    provider.onDisconnected = () => setConnected(false);

    provider.connect(pageId, user?.name || "Anonymous");

    return () => {
      provider.destroy();
      providerRef.current = null;
    };
  }, [pageId, user?.name]);

  if (!providerRef.current) {
    return <>{children}</>;
  }

  return (
    <CollaborationContext.Provider
      value={{
        ydoc: providerRef.current.ydoc,
        awareness: providerRef.current.awareness,
        connected,
      }}
    >
      {children}
    </CollaborationContext.Provider>
  );
}

// Re-export Awareness type for convenience
export type { Awareness };
