import { describe, it, expect, beforeEach, vi } from "vitest";
import { useCommentsStore } from "../comments";

describe("useCommentsStore", () => {
  beforeEach(() => {
    useCommentsStore.setState({ comments: [], loading: false, error: null });
    vi.restoreAllMocks();
  });

  it("starts with empty comments", () => {
    const state = useCommentsStore.getState();
    expect(state.comments).toEqual([]);
  });

  it("loadComments(pageId) fetches and sets comments", async () => {
    const mockComments = [
      { id: 1, page_id: 1, block_id: null, author_id: 1, content: "Hello", resolved: false, parent_id: null, created_at: "2024-01-01", updated_at: "2024-01-01" },
      { id: 2, page_id: 1, block_id: null, author_id: 2, content: "Reply", resolved: false, parent_id: 1, created_at: "2024-01-02", updated_at: "2024-01-02" },
    ];

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockComments), { status: 200 })
    );

    await useCommentsStore.getState().loadComments(1);

    const state = useCommentsStore.getState();
    expect(state.loading).toBe(false);
    expect(state.comments).toEqual(mockComments);
  });

  it("addComment makes POST and prepends comment to list", async () => {
    const newComment = { id: 3, page_id: 1, block_id: null, author_id: 1, content: "New comment", resolved: false, parent_id: null, created_at: "2024-01-03", updated_at: "2024-01-03" };

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(newComment), { status: 200 })
    );

    await useCommentsStore.getState().addComment(1, "New comment");

    const state = useCommentsStore.getState();
    expect(state.comments).toContainEqual(newComment);
  });

  it("resolveComment(id) toggles resolved state", async () => {
    useCommentsStore.setState({
      comments: [
        { id: 1, page_id: 1, block_id: null, author_id: 1, content: "Hello", resolved: false, parent_id: null, created_at: "2024-01-01", updated_at: "2024-01-01" },
      ],
    });

    // First call: false -> true
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );
    await useCommentsStore.getState().resolveComment(1);
    expect(useCommentsStore.getState().comments[0].resolved).toBe(true);

    // Second call: true -> false
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );
    await useCommentsStore.getState().resolveComment(1);
    expect(useCommentsStore.getState().comments[0].resolved).toBe(false);
  });

  it("deleteComment(id) removes comment and its replies (parent_id matches)", async () => {
    useCommentsStore.setState({
      comments: [
        { id: 1, page_id: 1, block_id: null, author_id: 1, content: "Parent", resolved: false, parent_id: null, created_at: "2024-01-01", updated_at: "2024-01-01" },
        { id: 2, page_id: 1, block_id: null, author_id: 2, content: "Reply", resolved: false, parent_id: 1, created_at: "2024-01-02", updated_at: "2024-01-02" },
        { id: 3, page_id: 1, block_id: null, author_id: 3, content: "Unrelated", resolved: false, parent_id: null, created_at: "2024-01-03", updated_at: "2024-01-03" },
      ],
    });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );

    await useCommentsStore.getState().deleteComment(1);

    const state = useCommentsStore.getState();
    // Comment 1 deleted, reply 2 deleted (parent_id=1), comment 3 remains
    expect(state.comments).toHaveLength(1);
    expect(state.comments[0].id).toBe(3);
  });
});
