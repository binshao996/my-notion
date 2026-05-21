import { describe, it, expect, beforeEach, vi } from "vitest";
import { useViewsStore } from "../views";

function makeView(overrides: Partial<ReturnType<typeof useViewsStore.getState>["views"][number]> = {}) {
  return {
    id: 1,
    name: "View",
    type: "table" as const,
    database_id: 1,
    config: {} as any,
    position: "a0",
    created_at: "2024-01-01",
    updated_at: "2024-01-01",
    ...overrides,
  };
}

describe("useViewsStore", () => {
  beforeEach(() => {
    useViewsStore.setState({ views: [], activeViewId: null, loading: false });
    vi.restoreAllMocks();
  });

  it("starts with empty views, activeViewId=null", () => {
    const state = useViewsStore.getState();
    expect(state.views).toEqual([]);
    expect(state.activeViewId).toBeNull();
  });

  it("setViews([v1, v2]) sets views and auto-selects first as active", () => {
    const v1 = makeView({ id: 1, name: "View 1" });
    const v2 = makeView({ id: 2, name: "View 2", type: "board" as const, position: "a1" });

    useViewsStore.getState().setViews([v1 as any, v2 as any]);

    const state = useViewsStore.getState();
    expect(state.views).toEqual([v1, v2]);
    expect(state.activeViewId).toBe(1);
  });

  it("setActiveView(id) changes activeViewId", () => {
    const v1 = makeView({ id: 1, name: "View 1" });
    const v2 = makeView({ id: 2, name: "View 2", type: "board" as const, position: "a1" });

    useViewsStore.setState({ views: [v1 as any, v2 as any], activeViewId: 1 });

    useViewsStore.getState().setActiveView(2);

    expect(useViewsStore.getState().activeViewId).toBe(2);
  });

  it("createView makes API call and adds to views", async () => {
    const newView = makeView({ id: 3, name: "New View", type: "list" as const });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(newView), { status: 200 })
    );

    const result = await useViewsStore.getState().createView(1, "New View", "list");

    expect(result).toEqual(newView);
    expect(useViewsStore.getState().views).toContainEqual(newView);
    expect(useViewsStore.getState().activeViewId).toBe(3);
  });

  it("deleteView(id) removes view, auto-selects next if deleted was active", async () => {
    const v1 = makeView({ id: 1, name: "View 1" });
    const v2 = makeView({ id: 2, name: "View 2", type: "board" as const, position: "a1" });

    useViewsStore.setState({ views: [v1 as any, v2 as any], activeViewId: 1 });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );

    await useViewsStore.getState().deleteView(1);

    const state = useViewsStore.getState();
    expect(state.views).toEqual([v2]);
    expect(state.activeViewId).toBe(2);
  });

  it("deleteView on last view sets activeViewId=null", async () => {
    const v1 = makeView({ id: 1, name: "View 1" });

    useViewsStore.setState({ views: [v1 as any], activeViewId: 1 });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );

    await useViewsStore.getState().deleteView(1);

    const state = useViewsStore.getState();
    expect(state.views).toEqual([]);
    expect(state.activeViewId).toBeNull();
  });
});
