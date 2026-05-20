import { create } from "zustand";

interface SearchState {
  isOpen: boolean;
  activeWorkspaceId: number | null;
  open: () => void;
  close: () => void;
  toggle: () => void;
  setActiveWorkspaceId: (id: number) => void;
}

export const useSearchStore = create<SearchState>((set) => ({
  isOpen: false,
  activeWorkspaceId: null,
  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  toggle: () => set((s) => ({ isOpen: !s.isOpen })),
  setActiveWorkspaceId: (id: number) => set({ activeWorkspaceId: id }),
}));
