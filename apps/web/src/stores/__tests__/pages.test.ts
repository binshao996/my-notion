import { describe, it, expect, beforeEach, vi } from "vitest";
import { usePagesStore } from "../pages";

describe("usePagesStore", () => {
  beforeEach(() => {
    usePagesStore.setState({ pages: [], expandedIds: new Set(), loading: false });
    vi.restoreAllMocks();
  });

  it("starts with empty pages, loading=false", () => {
    const state = usePagesStore.getState();
    expect(state.pages).toEqual([]);
    expect(state.loading).toBe(false);
  });

  it("loadTree(workspaceId) fetches and sets pages", async () => {
    const mockPages = [
      { id: 1, workspace_id: 1, parent_page_id: null, title: "Home", icon: "", cover: "", created_by: 1, created_at: "2024-01-01", updated_at: "2024-01-01", archived: false },
      { id: 2, workspace_id: 1, parent_page_id: 1, title: "Child", icon: "", cover: "", created_by: 1, created_at: "2024-01-02", updated_at: "2024-01-02", archived: false },
    ];

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockPages), { status: 200 })
    );

    await usePagesStore.getState().loadTree(1);

    const state = usePagesStore.getState();
    expect(state.loading).toBe(false);
    expect(state.pages).toEqual(mockPages);
  });

  it("loadTree handles API error gracefully (loading goes false, pages stays [])", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "Server error" }), { status: 500 })
    );

    await usePagesStore.getState().loadTree(1);

    const state = usePagesStore.getState();
    expect(state.loading).toBe(false);
    expect(state.pages).toEqual([]);
  });

  it("toggleExpanded(id) adds id to expandedIds set", () => {
    usePagesStore.getState().toggleExpanded(5);
    expect(usePagesStore.getState().expandedIds.has(5)).toBe(true);
  });

  it("toggleExpanded(id) called twice removes id from expandedIds", () => {
    usePagesStore.getState().toggleExpanded(5);
    usePagesStore.getState().toggleExpanded(5);
    expect(usePagesStore.getState().expandedIds.has(5)).toBe(false);
  });

  it("createPage makes POST and returns page", async () => {
    const newPage = {
      id: 3, workspace_id: 1, parent_page_id: null, title: "New Page",
      icon: "", cover: "", created_by: 1, created_at: "2024-01-03", updated_at: "2024-01-03", archived: false,
    };

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(newPage), { status: 200 })
    );

    const result = await usePagesStore.getState().createPage(1, "New Page");

    expect(result).toEqual(newPage);
  });

  it("loadChildren fetches children pages", async () => {
    const mockChildren = [
      { id: 3, workspace_id: 1, parent_page_id: 2, title: "Grandchild", icon: "", cover: "", created_by: 1, created_at: "2024-01-03", updated_at: "2024-01-03", archived: false },
    ];

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockChildren), { status: 200 })
    );

    const children = await usePagesStore.getState().loadChildren(2);

    expect(children).toEqual(mockChildren);
  });
});
