import { useEffect, useState, useCallback } from "react";
import { useParams } from "react-router-dom";
import { useAuthStore } from "../stores/auth";
import { useDatabaseStore } from "../stores/database";
import { useViewsStore } from "../stores/views";
import PageTree from "../features/sidebar/PageTree";
import TableView from "../features/database/TableView";
import type { ViewType } from "../types/database";

export default function DatabasePage() {
  const { dbId } = useParams<{ dbId: string }>();
  const dbIdNum = Number(dbId);

  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  const {
    database,
    properties,
    records,
    loading,
    error,
    loadDatabase,
    renameDatabase,
    updateRecord,
    deleteRecord,
    createRecord,
  } = useDatabaseStore();

  const {
    views,
    activeViewId,
    setViews,
    setActiveView,
    createView,
  } = useViewsStore();

  const [renaming, setRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState("");
  const [showAddView, setShowAddView] = useState(false);
  const [newViewName, setNewViewName] = useState("");

  useEffect(() => {
    if (dbIdNum) {
      loadDatabase(dbIdNum);
    }
  }, [dbIdNum, loadDatabase]);

  useEffect(() => {
    if (views.length > 0) {
      setViews(views);
    }
  }, [views, setViews]);

  const activeView = views.find((v) => v.id === activeViewId);

  const titleProperty = properties.find((p) => p.type === "title");

  const handleRename = useCallback(() => {
    if (renameValue.trim() && renameValue !== database?.name) {
      renameDatabase(dbIdNum, renameValue.trim());
    }
    setRenaming(false);
  }, [renameValue, database?.name, renameDatabase, dbIdNum]);

  const handleNewRecord = useCallback(async () => {
    await createRecord(dbIdNum, {});
    loadDatabase(dbIdNum);
  }, [createRecord, dbIdNum, loadDatabase]);

  const handleAddView = useCallback(
    async (type: ViewType) => {
      const name = newViewName.trim() || `${type.charAt(0).toUpperCase() + type.slice(1)} view`;
      await createView(dbIdNum, name, type);
      setNewViewName("");
      setShowAddView(false);
    },
    [createView, dbIdNum, newViewName]
  );

  const handleUpdateRecord = useCallback(
    (recordId: number, propertyId: number, value: any) =>
      updateRecord(recordId, { [String(propertyId)]: value }),
    [updateRecord]
  );

  // ---------- Loading state ----------
  if (loading) {
    return (
      <div className="flex min-h-screen bg-white">
        <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <span className="text-sm font-semibold text-gray-900">
              {user?.name || "User"}'s Notion
            </span>
            <button
              onClick={logout}
              className="text-xs text-gray-400 hover:text-gray-600"
            >
              Sign out
            </button>
          </div>
          <div className="flex-1 overflow-y-auto">
            <PageTree />
          </div>
        </aside>
        <main className="flex flex-1 items-center justify-center">
          <div className="text-sm text-gray-400">Loading database...</div>
        </main>
      </div>
    );
  }

  // ---------- Error state ----------
  if (error) {
    return (
      <div className="flex min-h-screen bg-white">
        <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <span className="text-sm font-semibold text-gray-900">
              {user?.name || "User"}'s Notion
            </span>
            <button
              onClick={logout}
              className="text-xs text-gray-400 hover:text-gray-600"
            >
              Sign out
            </button>
          </div>
          <div className="flex-1 overflow-y-auto">
            <PageTree />
          </div>
        </aside>
        <main className="flex flex-1 items-center justify-center">
          <div className="text-center">
            <p className="text-red-500">{error}</p>
            <button
              onClick={() => dbIdNum && loadDatabase(dbIdNum)}
              className="mt-2 text-sm text-blue-600 hover:underline"
            >
              Retry
            </button>
          </div>
        </main>
      </div>
    );
  }

  // ---------- Normal state ----------
  return (
    <div className="flex min-h-screen bg-white">
      {/* Sidebar */}
      <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
        <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
          <span className="text-sm font-semibold text-gray-900">
            {user?.name || "User"}'s Notion
          </span>
          <button
            onClick={logout}
            className="text-xs text-gray-400 hover:text-gray-600"
          >
            Sign out
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          <PageTree />
        </div>
      </aside>

      {/* Main content */}
      <main className="flex flex-1 flex-col overflow-hidden">
        {/* Header bar */}
        <div className="flex items-center gap-3 border-b border-gray-200 px-6 py-3">
          {renaming ? (
            <input
              autoFocus
              className="rounded border border-blue-400 px-2 py-1 text-sm font-semibold text-gray-900 outline-none"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onBlur={handleRename}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRename();
                if (e.key === "Escape") setRenaming(false);
              }}
            />
          ) : (
            <button
              className="text-sm font-semibold text-gray-900 hover:bg-gray-100 rounded px-2 py-1"
              onClick={() => {
                setRenameValue(database?.name || "");
                setRenaming(true);
              }}
            >
              {database?.name || "Untitled"}
            </button>
          )}

          <div className="flex-1" />

          <button className="rounded px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100">
            Filter
          </button>
          <button className="rounded px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100">
            Sort
          </button>
          <button className="rounded px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100">
            Group
          </button>

          <button
            onClick={handleNewRecord}
            className="rounded bg-blue-600 px-3 py-1.5 text-xs text-white hover:bg-blue-700"
          >
            New Record
          </button>

          <div className="relative">
            <button
              onClick={() => setShowAddView(!showAddView)}
              className="rounded border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50"
            >
              Add View
            </button>
            {showAddView && (
              <div className="absolute right-0 top-full z-10 mt-1 w-48 rounded border border-gray-200 bg-white py-1 shadow-lg">
                <input
                  autoFocus
                  className="w-full border-b border-gray-100 px-3 py-1.5 text-xs outline-none"
                  placeholder="View name (optional)"
                  value={newViewName}
                  onChange={(e) => setNewViewName(e.target.value)}
                />
                {(
                  ["table", "board", "calendar", "list", "gallery"] as ViewType[]
                ).map((type) => (
                  <button
                    key={type}
                    className="w-full px-3 py-1.5 text-left text-xs text-gray-700 hover:bg-gray-50"
                    onClick={() => handleAddView(type)}
                  >
                    {type.charAt(0).toUpperCase() + type.slice(1)}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* View tabs */}
        <div className="flex items-center gap-0.5 border-b border-gray-200 px-6">
          {views.map((v) => (
            <button
              key={v.id}
              onClick={() => setActiveView(v.id)}
              className={`border-b-2 px-3 py-2 text-xs ${
                v.id === activeViewId
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-gray-500 hover:text-gray-700"
              }`}
            >
              {v.name}
            </button>
          ))}
          <button
            onClick={() => setShowAddView(true)}
            className="px-2 py-2 text-xs text-gray-400 hover:text-gray-600"
          >
            +
          </button>
        </div>

        {/* Active view area */}
        <div className="flex-1 overflow-auto">
          {activeView && activeView.type === "table" ? (
            <TableView
              properties={properties}
              records={records}
              activeView={activeView || undefined}
              onUpdateRecord={handleUpdateRecord}
              onDeleteRecord={deleteRecord}
              onCreateRecord={handleNewRecord}
              titlePropertyId={titleProperty?.id}
            />
          ) : !activeView ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-sm text-gray-400">
                No view selected. Create a view to get started.
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full">
              <div className="text-sm text-gray-400">
                {activeView.type.charAt(0).toUpperCase() +
                  activeView.type.slice(1)}{" "}
                view is being built...
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
