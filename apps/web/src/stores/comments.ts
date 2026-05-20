import { create } from "zustand";
import { api } from "../lib/api";

interface Comment {
  id: number;
  page_id: number;
  block_id: number | null;
  author_id: number;
  content: string; // JSONB string from backend
  resolved: boolean;
  parent_id: number | null;
  created_at: string;
  updated_at: string;
  Author?: { id: number; name: string; avatar_url: string };
}

interface CommentsState {
  comments: Comment[];
  loading: boolean;
  error: string | null;

  loadComments: (pageId: number) => Promise<void>;
  addComment: (pageId: number, content: string, parentId?: number, blockId?: number) => Promise<void>;
  resolveComment: (id: number) => Promise<void>;
  deleteComment: (id: number) => Promise<void>;
}

export const useCommentsStore = create<CommentsState>((set) => ({
  comments: [],
  loading: false,
  error: null,

  loadComments: async (pageId) => {
    set({ loading: true, error: null });
    try {
      const data = await api.get<Comment[]>(`/pages/${pageId}/comments`);
      set({ comments: data || [], loading: false });
    } catch (e) {
      set({ loading: false, error: e instanceof Error ? e.message : "Failed to load comments" });
    }
  },

  addComment: async (pageId, content, parentId, blockId) => {
    set({ error: null });
    try {
      const body: Record<string, any> = { content: JSON.stringify({ text: content }) };
      if (parentId) body.parent_id = parentId;
      if (blockId) body.block_id = blockId;
      const comment = await api.post<Comment>(`/pages/${pageId}/comments`, body);
      set((s) => ({ comments: [...s.comments, comment] }));
    } catch (e) {
      set({ error: e instanceof Error ? e.message : "Failed to add comment" });
    }
  },

  resolveComment: async (id) => {
    try {
      await api.patch(`/comments/${id}`, { resolved: true });
      set((s) => ({
        comments: s.comments.map((c) => c.id === id ? { ...c, resolved: !c.resolved } : c),
      }));
    } catch (e) {
      set({ error: e instanceof Error ? e.message : "Failed to resolve comment" });
    }
  },

  deleteComment: async (id) => {
    try {
      await api.delete(`/comments/${id}`);
      set((s) => ({ comments: s.comments.filter((c) => c.id !== id && c.parent_id !== id) }));
    } catch (e) {
      set({ error: e instanceof Error ? e.message : "Failed to delete comment" });
    }
  },
}));
