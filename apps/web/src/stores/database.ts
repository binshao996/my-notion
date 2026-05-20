import { create } from "zustand";
import { api } from "../lib/api";
import type {
  Database,
  Property,
  Record as DatabaseRecord,
  RecordValue,
  View,
  DatabaseGetResponse,
  RecordsListResponse,
} from "../types/database";

interface DatabaseState {
  database: Database | null;
  properties: Property[];
  records: DatabaseRecord[];
  views: View[];
  loading: boolean;
  error: string | null;

  loadDatabase: (id: number) => Promise<void>;
  createDatabase: (workspaceId: number, name: string) => Promise<Database>;
  renameDatabase: (id: number, name: string) => Promise<void>;
  deleteDatabase: (id: number) => Promise<void>;

  addProperty: (databaseId: number, name: string, type: string, config?: string) => Promise<Property>;
  updateProperty: (id: number, updates: Record<string, any>) => Promise<void>;
  deleteProperty: (id: number) => Promise<void>;

  createRecord: (databaseId: number, values: Record<string, any>) => Promise<DatabaseRecord>;
  updateRecord: (id: number, values: Record<string, any>) => Promise<void>;
  deleteRecord: (id: number) => Promise<void>;
  loadRecords: (databaseId: number, viewId?: number, page?: number) => Promise<void>;
  loadRecord: (id: number) => Promise<{ record: DatabaseRecord; properties: Property[] }>;
}

export const useDatabaseStore = create<DatabaseState>((set) => ({
  database: null,
  properties: [],
  records: [],
  views: [],
  loading: false,
  error: null,

  loadDatabase: async (id) => {
    set({ loading: true, error: null });
    try {
      const data = await api.get<DatabaseGetResponse>(`/databases/${id}`);
      set({
        database: data.database,
        properties: data.properties,
        views: data.views || [],
        records: data.records || [],
        loading: false,
      });
    } catch (e) {
      set({
        loading: false,
        error: e instanceof Error ? e.message : "Failed to load database",
      });
    }
  },

  createDatabase: async (workspaceId, name) => {
    const data = await api.post<Database>("/databases", {
      workspace_id: workspaceId,
      name,
    });
    return data;
  },

  renameDatabase: async (id, name) => {
    await api.patch(`/databases/${id}`, { name });
    set((s) => ({
      database: s.database ? { ...s.database, name } : null,
    }));
  },

  deleteDatabase: async (id) => {
    await api.delete(`/databases/${id}`);
    set({ database: null, properties: [], records: [], views: [] });
  },

  addProperty: async (databaseId, name, type, config) => {
    const prop = await api.post<Property>(`/databases/${databaseId}/properties`, {
      name,
      type,
      config: config || "{}",
    });
    set((s) => ({ properties: [...s.properties, prop] }));
    return prop;
  },

  updateProperty: async (id, updates) => {
    await api.patch(`/properties/${id}`, updates);
    set((s) => ({
      properties: s.properties.map((p) =>
        p.id === id ? { ...p, ...updates } : p
      ),
    }));
  },

  deleteProperty: async (id) => {
    await api.delete(`/properties/${id}`);
    set((s) => ({
      properties: s.properties.filter((p) => p.id !== id),
    }));
  },

  createRecord: async (databaseId, values) => {
    const record = await api.post<DatabaseRecord>(`/databases/${databaseId}/records`, {
      property_values: values,
    });
    set((s) => ({ records: [...s.records, record] }));
    return record;
  },

  updateRecord: async (id, values) => {
    await api.patch(`/records/${id}`, { property_values: values });
    // Refetch to get updated property_values
    set((s) => ({
      records: s.records.map((r) => (r.id === id ? { ...r } : r)),
    }));
  },

  deleteRecord: async (id) => {
    await api.delete(`/records/${id}`);
    set((s) => ({
      records: s.records.filter((r) => r.id !== id),
    }));
  },

  loadRecords: async (databaseId, viewId, page = 1) => {
    set({ loading: true, error: null });
    try {
      const endpoint = viewId
        ? `/databases/${databaseId}/views/${viewId}/records?page=${page}&limit=50`
        : `/databases/${databaseId}/records?page=${page}&limit=50`;
      const data = await api.get<RecordsListResponse>(endpoint);
      set({ records: data.records || [], loading: false });
    } catch (e) {
      set({
        loading: false,
        error: e instanceof Error ? e.message : "Failed to load records",
      });
    }
  },

  loadRecord: async (id) => {
    const data = await api.get<{ record: DatabaseRecord; property_values: RecordValue[]; properties: Property[] }>(`/records/${id}`);
    // Parse property values from string JSONB to objects
    const record = {
      ...data.record,
      property_values: (data.property_values || []).map(pv => ({
        ...pv,
        value: typeof pv.value === 'string' ? JSON.parse(pv.value) : pv.value,
      })),
    };
    return { record, properties: data.properties };
  },
}));
