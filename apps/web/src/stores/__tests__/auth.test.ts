import { describe, it, expect, beforeEach, vi } from "vitest";
import { useAuthStore } from "../auth";

describe("useAuthStore", () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, token: null, loading: false });
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("starts with null user and token", () => {
    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.token).toBeNull();
  });

  it("logout clears user, token, and localStorage", () => {
    useAuthStore.setState({
      user: { id: 1, email: "a@b.com", name: "Alice", avatar_url: "" },
      token: "test-token",
    });
    localStorage.setItem("token", "test-token");

    useAuthStore.getState().logout();

    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.token).toBeNull();
    expect(localStorage.getItem("token")).toBeNull();
  });

  it("loadFromStorage restores token from localStorage", () => {
    localStorage.setItem("token", "stored-token");

    useAuthStore.getState().loadFromStorage();

    const state = useAuthStore.getState();
    expect(state.token).toBe("stored-token");
    expect(state.user).toBeNull(); // user not restored from storage
  });

  it("login sets loading true then false on success", async () => {
    const mockUser = { id: 1, email: "a@b.com", name: "Alice", avatar_url: "" };
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ token: "jwt", user: mockUser }), { status: 200 })
    );

    await useAuthStore.getState().login("a@b.com", "password");

    const state = useAuthStore.getState();
    expect(state.loading).toBe(false);
    expect(state.user).toEqual(mockUser);
    expect(state.token).toBe("jwt");
  });

  it("login throws on failure and sets loading false", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "bad credentials" }), { status: 401 })
    );

    const state = useAuthStore.getState();
    await expect(state.login("a@b.com", "wrong")).rejects.toThrow();

    expect(useAuthStore.getState().loading).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
  });
});
