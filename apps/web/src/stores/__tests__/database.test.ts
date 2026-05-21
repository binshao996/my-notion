import { describe, it, expect, beforeEach, vi } from "vitest";
import { useDatabaseStore } from "../database";

const mockDatabase = {
  id: 1,
  workspace_id: 1,
  page_id: 10,
  name: "Test DB",
  created_at: "",
  updated_at: "",
};

const mockProperty = {
  id: 1,
  database_id: 1,
  name: "Status",
  type: "status" as const,
  config: {} as any,
  position: "a0",
  created_at: "",
};

const mockProperty2 = {
  id: 2,
  database_id: 1,
  name: "Priority",
  type: "select" as const,
  config: {} as any,
  position: "a1",
  created_at: "",
};

const mockRecord = {
  id: 1,
  database_id: 1,
  page_id: 11,
  position: "a0",
  property_values: [],
  created_at: "",
};

const mockRecord2 = {
  id: 2,
  database_id: 1,
  page_id: 12,
  position: "a1",
  property_values: [],
  created_at: "",
};

const mockView = {
  id: 1,
  database_id: 1,
  name: "Table View",
  type: "table" as const,
  config: {} as any,
  position: "a0",
  created_at: "",
  updated_at: "",
};

describe("useDatabaseStore", () => {
  beforeEach(() => {
    useDatabaseStore.setState({
      database: null,
      properties: [],
      records: [],
      views: [],
      loading: false,
      error: null,
    });
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("has correct initial state with all fields null or empty", () => {
    const state = useDatabaseStore.getState();
    expect(state.database).toBeNull();
    expect(state.properties).toEqual([]);
    expect(state.records).toEqual([]);
    expect(state.views).toEqual([]);
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("loadDatabase fetches and sets all fields on success", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          database: mockDatabase,
          properties: [mockProperty as any],
          views: [mockView],
          records: [mockRecord],
        }),
        { status: 200 },
      ),
    );

    await useDatabaseStore.getState().loadDatabase(1);

    const state = useDatabaseStore.getState();
    expect(state.database).toEqual(mockDatabase);
    expect(state.properties).toEqual([mockProperty]);
    expect(state.views).toEqual([mockView]);
    expect(state.records).toEqual([mockRecord]);
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("loadDatabase sets error message on 500 response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "Internal server error" }), { status: 500 }),
    );

    await useDatabaseStore.getState().loadDatabase(1);

    const state = useDatabaseStore.getState();
    expect(state.loading).toBe(false);
    expect(state.error).toBe("Internal server error");
    expect(state.database).toBeNull();
  });

  it("createDatabase posts and returns the created database", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockDatabase), { status: 201 }),
    );

    const result = await useDatabaseStore.getState().createDatabase(1, "New DB");

    expect(result).toEqual(mockDatabase);
    // createDatabase does not modify store state
    expect(useDatabaseStore.getState().database).toBeNull();
  });

  it("renameDatabase patches and updates database.name in state", async () => {
    useDatabaseStore.setState({ database: { ...mockDatabase } });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("{}", { status: 200 }),
    );

    await useDatabaseStore.getState().renameDatabase(1, "Renamed DB");

    const state = useDatabaseStore.getState();
    expect(state.database?.name).toBe("Renamed DB");
    expect(state.database?.id).toBe(1);
  });

  it("deleteDatabase calls DELETE and clears database, properties, records, views", async () => {
    useDatabaseStore.setState({
      database: mockDatabase,
      properties: [mockProperty as any],
      records: [mockRecord],
      views: [mockView],
    });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("{}", { status: 200 }),
    );

    await useDatabaseStore.getState().deleteDatabase(1);

    const state = useDatabaseStore.getState();
    expect(state.database).toBeNull();
    expect(state.properties).toEqual([]);
    expect(state.records).toEqual([]);
    expect(state.views).toEqual([]);
  });

  it("addProperty posts and appends the property to the list", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockProperty), { status: 201 }),
    );

    const result = await useDatabaseStore.getState().addProperty(1, "Status", "status");

    expect(result).toEqual(mockProperty);
    expect(useDatabaseStore.getState().properties).toEqual([mockProperty]);
  });

  it("updateProperty patches and updates the matching property in state", async () => {
    useDatabaseStore.setState({ properties: [{ ...mockProperty }, { ...mockProperty2 }] });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("{}", { status: 200 }),
    );

    await useDatabaseStore.getState().updateProperty(1, { name: "Updated Status" });

    const state = useDatabaseStore.getState();
    expect(state.properties).toHaveLength(2);
    expect(state.properties[0].name).toBe("Updated Status");
    expect(state.properties[0].id).toBe(1);
    expect(state.properties[1]).toEqual(mockProperty2);
  });

  it("deleteProperty calls DELETE and removes the property from the list", async () => {
    useDatabaseStore.setState({ properties: [{ ...mockProperty }, { ...mockProperty2 }] });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("{}", { status: 200 }),
    );

    await useDatabaseStore.getState().deleteProperty(1);

    const state = useDatabaseStore.getState();
    expect(state.properties).toHaveLength(1);
    expect(state.properties[0].id).toBe(2);
  });

  it("createRecord posts and appends the record to the list", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockRecord), { status: 201 }),
    );

    const result = await useDatabaseStore.getState().createRecord(1, { title: "Hello" });

    expect(result).toEqual(mockRecord);
    expect(useDatabaseStore.getState().records).toEqual([mockRecord]);
  });

  it("updateRecord patches and keeps the record in the list", async () => {
    useDatabaseStore.setState({ records: [{ ...mockRecord }, { ...mockRecord2 }] });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("{}", { status: 200 }),
    );

    await useDatabaseStore.getState().updateRecord(1, { title: "Updated" });

    const state = useDatabaseStore.getState();
    expect(state.records).toHaveLength(2);
    expect(state.records[0].id).toBe(1);
    expect(state.records[1].id).toBe(2);
  });

  it("deleteRecord calls DELETE and removes the record from the list", async () => {
    useDatabaseStore.setState({ records: [{ ...mockRecord }, { ...mockRecord2 }] });

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("{}", { status: 200 }),
    );

    await useDatabaseStore.getState().deleteRecord(1);

    const state = useDatabaseStore.getState();
    expect(state.records).toHaveLength(1);
    expect(state.records[0].id).toBe(2);
  });

  it("loadRecords fetches and sets records with loading cleared", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          records: [mockRecord, mockRecord2],
          total: 2,
          page: 1,
          limit: 50,
        }),
        { status: 200 },
      ),
    );

    await useDatabaseStore.getState().loadRecords(1);

    const state = useDatabaseStore.getState();
    expect(state.records).toEqual([mockRecord, mockRecord2]);
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("loadRecords sets error on 500 response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "Not found" }), { status: 500 }),
    );

    await useDatabaseStore.getState().loadRecords(1);

    const state = useDatabaseStore.getState();
    expect(state.loading).toBe(false);
    expect(state.error).toBe("Not found");
    expect(state.records).toEqual([]);
  });

  it("loadRecord parses string property_values using JSON.parse", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          record: mockRecord,
          property_values: [
            { record_id: 1, property_id: 1, value: '{"title":"Hello World"}' },
            { record_id: 1, property_id: 2, value: "42" },
            { record_id: 1, property_id: 3, value: true },
          ],
          properties: [mockProperty as any],
        }),
        { status: 200 },
      ),
    );

    const parseSpy = vi.spyOn(JSON, "parse");
    const result = await useDatabaseStore.getState().loadRecord(1);

    // String JSON values should be parsed into objects/numbers
    expect(result.record.property_values![0].value).toEqual({ title: "Hello World" });
    expect(result.record.property_values![1].value).toBe(42);
    // Non-string values pass through unchanged
    expect(result.record.property_values![2].value).toBe(true);
    expect(result.properties).toEqual([mockProperty]);

    // JSON.parse must have been called with the string values
    const stringArgs = parseSpy.mock.calls.map((call) => call[0]);
    expect(stringArgs).toContain('{"title":"Hello World"}');
    expect(stringArgs).toContain("42");
  });
});
