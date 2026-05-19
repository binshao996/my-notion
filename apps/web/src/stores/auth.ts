import { create } from "zustand";
import { api } from "../lib/api";

interface User {
  id: number;
  email: string;
  name: string;
  avatar_url: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, name: string, password: string) => Promise<void>;
  logout: () => void;
  loadFromStorage: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  loading: false,

  login: async (email: string, password: string) => {
    set({ loading: true });
    try {
      const data = await api.post<{ token: string; user: User }>(
        "/auth/login",
        { email, password }
      );
      localStorage.setItem("token", data.token);
      set({ user: data.user, token: data.token, loading: false });
    } catch (e) {
      set({ loading: false });
      throw e;
    }
  },

  register: async (email: string, name: string, password: string) => {
    set({ loading: true });
    try {
      const data = await api.post<{ token: string; user: User }>(
        "/auth/register",
        { email, name, password }
      );
      localStorage.setItem("token", data.token);
      set({ user: data.user, token: data.token, loading: false });
    } catch (e) {
      set({ loading: false });
      throw e;
    }
  },

  logout: () => {
    localStorage.removeItem("token");
    set({ user: null, token: null });
  },

  loadFromStorage: () => {
    const token = localStorage.getItem("token");
    if (token) {
      set({ token });
    }
  },
}));
