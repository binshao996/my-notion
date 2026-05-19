import { create } from "zustand";
import { api } from "../lib/api";

interface Page {
  id: number;
  workspace_id: number;
  parent_page_id: number | null;
  title: string;
  icon: string;
  cover: string;
  created_by: number;
  created_at: string;
  updated_at: string;
  archived: boolean;
}

interface PagesState {
  pages: Page[];
  expandedIds: Set<number>;
  loading: boolean;
  loadTree: (workspaceId: number) => Promise<void>;
  createPage: (workspaceId: number, title: string, parentId?: number) => Promise<Page>;
  loadChildren: (parentId: number) => Promise<Page[]>;
  toggleExpanded: (id: number) => void;
}

export const usePagesStore = create<PagesState>((set, get) => ({
  pages: [],
  expandedIds: new Set<number>(),
  loading: false,

  loadTree: async (workspaceId: number) => {
    set({ loading: true });
    try {
      const pages = await api.get<Page[]>(`/workspaces/${workspaceId}/tree`);
      set({ pages: Array.isArray(pages) ? pages : [], loading: false });
    } catch {
      set({ loading: false });
    }
  },

  createPage: async (workspaceId: number, title: string, parentId?: number) => {
    const page = await api.post<Page>("/pages", {
      workspace_id: workspaceId,
      title,
      parent_page_id: parentId || null,
    });
    if (parentId) {
      get().toggleExpanded(parentId);
    }
    return page;
  },

  loadChildren: async (parentId: number) => {
    const children = await api.get<Page[]>(`/pages/${parentId}/children`);
    return Array.isArray(children) ? children : [];
  },

  toggleExpanded: (id: number) => {
    const next = new Set(get().expandedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    set({ expandedIds: next });
  },
}));

export type { Page };
