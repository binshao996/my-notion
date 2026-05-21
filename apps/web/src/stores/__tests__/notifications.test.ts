import { describe, it, expect, beforeEach, vi } from "vitest";
import { useNotificationsStore } from "../notifications";

describe("useNotificationsStore", () => {
  beforeEach(() => {
    useNotificationsStore.setState({ notifications: [], unreadCount: 0, loading: false });
    vi.restoreAllMocks();
  });

  it("starts with empty notifications, unreadCount=0", () => {
    const state = useNotificationsStore.getState();
    expect(state.notifications).toEqual([]);
    expect(state.unreadCount).toBe(0);
  });

  it("loadNotifications() fetches and sets notifications + unreadCount", async () => {
    const mockData = {
      notifications: [
        { id: 1, user_id: 1, type: "mention", actor_id: 2, target_page_id: 10, read: false, created_at: "2024-01-01" },
        { id: 2, user_id: 1, type: "comment", actor_id: 3, target_page_id: 10, read: false, created_at: "2024-01-02" },
      ],
      unread_count: 2,
    };

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockData), { status: 200 })
    );

    await useNotificationsStore.getState().loadNotifications();

    const state = useNotificationsStore.getState();
    expect(state.loading).toBe(false);
    expect(state.notifications).toEqual(mockData.notifications);
    expect(state.unreadCount).toBe(2);
  });

  it("markRead(id) marks single notification read, decrements unreadCount", async () => {
    useNotificationsStore.setState({
      notifications: [
        { id: 1, user_id: 1, type: "mention", actor_id: 2, target_page_id: 10, read: false, created_at: "2024-01-01" },
      ],
      unreadCount: 1,
    });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );

    await useNotificationsStore.getState().markRead(1);

    const state = useNotificationsStore.getState();
    expect(state.notifications[0].read).toBe(true);
    expect(state.unreadCount).toBe(0);
  });

  it("markAllRead() marks all notifications read, sets unreadCount=0", async () => {
    useNotificationsStore.setState({
      notifications: [
        { id: 1, user_id: 1, type: "mention", actor_id: 2, target_page_id: 10, read: false, created_at: "2024-01-01" },
        { id: 2, user_id: 1, type: "comment", actor_id: 3, target_page_id: 10, read: false, created_at: "2024-01-02" },
      ],
      unreadCount: 2,
    });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );

    await useNotificationsStore.getState().markAllRead();

    const state = useNotificationsStore.getState();
    expect(state.notifications.every((n) => n.read)).toBe(true);
    expect(state.unreadCount).toBe(0);
  });

  it("markRead on unreadCount 0 doesn't go below 0", async () => {
    useNotificationsStore.setState({
      notifications: [
        { id: 1, user_id: 1, type: "mention", actor_id: 2, target_page_id: 10, read: true, created_at: "2024-01-01" },
      ],
      unreadCount: 0,
    });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );

    await useNotificationsStore.getState().markRead(1);

    expect(useNotificationsStore.getState().unreadCount).toBe(0);
  });
});
