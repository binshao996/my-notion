import { create } from "zustand";
import { api } from "../lib/api";

interface Notification {
  id: number;
  user_id: number;
  type: string;
  actor_id: number;
  target_page_id: number;
  comment_id?: number;
  read: boolean;
  created_at: string;
}

interface NotificationsState {
  notifications: Notification[];
  unreadCount: number;
  loading: boolean;

  loadNotifications: () => Promise<void>;
  markRead: (id: number) => Promise<void>;
  markAllRead: () => Promise<void>;
}

export const useNotificationsStore = create<NotificationsState>((set) => ({
  notifications: [],
  unreadCount: 0,
  loading: false,

  loadNotifications: async () => {
    set({ loading: true });
    try {
      const data = await api.get<{ notifications: Notification[]; unread_count: number }>("/notifications");
      set({
        notifications: data.notifications || [],
        unreadCount: data.unread_count || 0,
        loading: false,
      });
    } catch {
      set({ loading: false });
    }
  },

  markRead: async (id) => {
    try {
      await api.patch(`/notifications/${id}/read`, {});
      set((s) => ({
        notifications: s.notifications.map((n) => (n.id === id ? { ...n, read: true } : n)),
        unreadCount: Math.max(0, s.unreadCount - 1),
      }));
    } catch {}
  },

  markAllRead: async () => {
    try {
      await api.patch("/notifications/read-all", {});
      set((s) => ({
        notifications: s.notifications.map((n) => ({ ...n, read: true })),
        unreadCount: 0,
      }));
    } catch {}
  },
}));
