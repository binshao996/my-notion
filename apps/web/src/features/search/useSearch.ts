import { useState, useRef, useCallback } from "react";
import { api } from "../../lib/api";
import type { SearchResults } from "./types";

export function useSearch() {
  const [results, setResults] = useState<SearchResults | null>(null);
  const [loading, setLoading] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>();

  const search = useCallback((query: string, workspaceId: number) => {
    clearTimeout(timerRef.current);
    if (!query.trim()) {
      setResults(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    timerRef.current = setTimeout(async () => {
      try {
        const res = await api.get<SearchResults>(
          `/search?q=${encodeURIComponent(query)}&workspace_id=${workspaceId}`
        );
        setResults(res);
      } catch {
        setResults(null);
      } finally {
        setLoading(false);
      }
    }, 200);
  }, []);

  return { results, loading, search };
}
