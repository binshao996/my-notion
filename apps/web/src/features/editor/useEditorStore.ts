import { create } from "zustand";
import { api, ApiError } from "../../lib/api";

export interface BlockData {
  id: number;
  type: string;
  props: Record<string, any>;
  parent_block_id: number | null;
  tempId?: string;
}

interface EditorState {
  blocks: BlockData[];
  pageId: number | null;
  loading: boolean;
  saving: boolean;
  dirty: boolean;
  error: string | null;
  commandPaletteIndex: number | null; // which block shows the / menu
  focusedBlockIndex: number;

  loadPage: (pageId: number) => Promise<void>;
  addBlock: (afterIndex: number, type?: string) => void;
  updateBlock: (index: number, props: Record<string, any>) => void;
  updateBlockType: (index: number, newType: string) => void;
  deleteBlock: (index: number) => void;
  duplicateBlock: (index: number) => void;
  moveBlock: (fromIndex: number, toIndex: number) => void;
  indentBlock: (index: number) => void;
  outdentBlock: (index: number) => void;
  save: () => Promise<void>;
  openCommandPalette: (index: number) => void;
  closeCommandPalette: () => void;
  focusBlock: (index: number) => void;
}

let tempIdCounter = 0;
function tempId() {
  return `temp-${++tempIdCounter}`;
}

function newBlock(type: string = "paragraph"): BlockData {
  return {
    id: 0,
    type,
    props: { text: "" },
    parent_block_id: null,
    tempId: tempId(),
  };
}

export const useEditorStore = create<EditorState>((set, get) => ({
  blocks: [],
  pageId: null,
  loading: false,
  saving: false,
  dirty: false,
  error: null,
  commandPaletteIndex: null,
  focusedBlockIndex: 0,

  loadPage: async (pageId: number) => {
    set({ loading: true, error: null });
    try {
      const blocks = await api.get<any[]>(`/pages/${pageId}/blocks`);
      const parsed = Array.isArray(blocks)
        ? blocks.map((b: any) => ({
            ...b,
            props: typeof b.props === "string" ? JSON.parse(b.props) : b.props,
          }))
        : [];
      set({
        blocks: parsed.length > 0 ? parsed : [newBlock("paragraph")],
        pageId,
        loading: false,
        dirty: false,
        focusedBlockIndex: 0,
      });
    } catch (e) {
      set({ loading: false, error: e instanceof Error ? e.message : "Failed to load page" });
    }
  },

  addBlock: (afterIndex, type = "paragraph") => {
    const { blocks } = get();
    const newBlocks = [...blocks];
    newBlocks.splice(afterIndex + 1, 0, newBlock(type));
    set({ blocks: newBlocks, dirty: true, focusedBlockIndex: afterIndex + 1 });
  },

  updateBlock: (index, props) => {
    const { blocks } = get();
    if (index >= blocks.length) return;
    const newBlocks = [...blocks];
    newBlocks[index] = { ...newBlocks[index], props: { ...newBlocks[index].props, ...props } };
    set({ blocks: newBlocks, dirty: true });
  },

  updateBlockType: (index, newType) => {
    const { blocks } = get();
    if (index >= blocks.length) return;
    const newBlocks = [...blocks];
    newBlocks[index] = { ...newBlocks[index], type: newType };
    // Reset props for divider
    if (newType === "divider") {
      newBlocks[index].props = {};
    }
    set({ blocks: newBlocks, dirty: true });
  },

  deleteBlock: (index) => {
    const { blocks } = get();
    if (blocks.length <= 1) return; // keep at least one block
    const newBlocks = blocks.filter((_, i) => i !== index);
    const newFocus = Math.min(index, newBlocks.length - 1);
    set({ blocks: newBlocks, dirty: true, focusedBlockIndex: newFocus });
  },

  duplicateBlock: (index) => {
    const { blocks } = get();
    const newBlocks = [...blocks];
    const copy = { ...blocks[index], id: 0, tempId: tempId() };
    newBlocks.splice(index + 1, 0, copy);
    set({ blocks: newBlocks, dirty: true });
  },

  moveBlock: (fromIndex, toIndex) => {
    const { blocks } = get();
    const newBlocks = [...blocks];
    const [moved] = newBlocks.splice(fromIndex, 1);
    newBlocks.splice(toIndex, 0, moved);
    set({ blocks: newBlocks, dirty: true, focusedBlockIndex: toIndex });
  },

  indentBlock: (index) => {
    const { blocks } = get();
    if (index === 0) return;
    const newBlocks = [...blocks];
    newBlocks[index] = {
      ...newBlocks[index],
      parent_block_id: newBlocks[index - 1].id || 0,
    };
    set({ blocks: newBlocks, dirty: true });
  },

  outdentBlock: (index) => {
    const { blocks } = get();
    if (!blocks[index].parent_block_id) return;
    const newBlocks = [...blocks];
    newBlocks[index] = { ...newBlocks[index], parent_block_id: null };
    set({ blocks: newBlocks, dirty: true });
  },

  save: async () => {
    const { blocks, pageId } = get();
    if (!pageId) return;
    set({ saving: true, error: null });
    try {
      const toSave = blocks.map((b) => ({
        id: b.id || 0,
        type: b.type,
        props: JSON.stringify(b.props),
        parent_block_id: b.parent_block_id,
      }));
      const saved = await api.put<any[]>(`/pages/${pageId}/blocks`, toSave);
      const parsed = Array.isArray(saved)
        ? saved.map((b: any) => ({
            ...b,
            props: typeof b.props === "string" ? JSON.parse(b.props) : b.props,
          }))
        : [];
      set({ blocks: parsed, saving: false, dirty: false });
    } catch (e) {
      set({
        saving: false,
        error: e instanceof ApiError ? e.message : "Failed to save",
      });
    }
  },

  openCommandPalette: (index) => set({ commandPaletteIndex: index }),
  closeCommandPalette: () => set({ commandPaletteIndex: null }),
  focusBlock: (index) => set({ focusedBlockIndex: index }),
}));

// Auto-save: save when dirty after 1 second of no changes
let saveTimeout: ReturnType<typeof setTimeout>;

useEditorStore.subscribe((state) => {
  if (state.dirty && !state.saving) {
    clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      useEditorStore.getState().save();
    }, 1000);
  }
});
