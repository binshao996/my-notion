import { create } from "zustand";
import { api } from "../lib/api";
import type { View, ViewConfig } from "../types/database";

interface ViewsState {
  views: View[];
  activeViewId: number | null;
  loading: boolean;

  setViews: (views: View[]) => void;
  setActiveView: (id: number) => void;
  createView: (databaseId: number, name: string, type: string) => Promise<View>;
  updateView: (id: number, updates: { name?: string; config?: ViewConfig }) => Promise<void>;
  deleteView: (id: number) => Promise<void>;
}

export const useViewsStore = create<ViewsState>((set, get) => ({
  views: [],
  activeViewId: null,
  loading: false,

  setViews: (views) => {
    set({ views });
    // Auto-select first view if none active
    if (!get().activeViewId && views.length > 0) {
      set({ activeViewId: views[0].id });
    }
  },

  setActiveView: (id) => set({ activeViewId: id }),

  createView: async (databaseId, name, type) => {
    const view = await api.post<View>(`/databases/${databaseId}/views`, { name, type });
    set((s) => ({ views: [...s.views, view], activeViewId: view.id }));
    return view;
  },

  updateView: async (id, updates) => {
    await api.patch(`/views/${id}`, updates);
    set((s) => ({
      views: s.views.map((v) =>
        v.id === id ? { ...v, ...updates, config: updates.config || v.config } : v
      ),
    }));
  },

  deleteView: async (id) => {
    await api.delete(`/views/${id}`);
    set((s) => {
      const newViews = s.views.filter((v) => v.id !== id);
      return {
        views: newViews,
        activeViewId: s.activeViewId === id
          ? (newViews[0]?.id || null)
          : s.activeViewId,
      };
    });
  },
}));
