import { describe, it, expect, beforeEach } from "vitest";
import { useSearchStore } from "../search";

describe("useSearchStore", () => {
  beforeEach(() => {
    useSearchStore.setState({ isOpen: false, activeWorkspaceId: null });
  });

  it("starts with isOpen=false, activeWorkspaceId=null", () => {
    const state = useSearchStore.getState();
    expect(state.isOpen).toBe(false);
    expect(state.activeWorkspaceId).toBeNull();
  });

  it("open() sets isOpen=true", () => {
    useSearchStore.getState().open();
    expect(useSearchStore.getState().isOpen).toBe(true);
  });

  it("close() sets isOpen=false", () => {
    useSearchStore.setState({ isOpen: true });
    useSearchStore.getState().close();
    expect(useSearchStore.getState().isOpen).toBe(false);
  });

  it("toggle() toggles isOpen from false to true to false", () => {
    expect(useSearchStore.getState().isOpen).toBe(false);

    useSearchStore.getState().toggle();
    expect(useSearchStore.getState().isOpen).toBe(true);

    useSearchStore.getState().toggle();
    expect(useSearchStore.getState().isOpen).toBe(false);
  });

  it("setActiveWorkspaceId(42) sets activeWorkspaceId=42", () => {
    useSearchStore.getState().setActiveWorkspaceId(42);
    expect(useSearchStore.getState().activeWorkspaceId).toBe(42);
  });
});
