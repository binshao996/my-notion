import { useEffect, useState, useCallback } from "react";
import { useParams } from "react-router-dom";
import { useEditorStore } from "../features/editor/useEditorStore";
import { useAuthStore } from "../stores/auth";
import { useDatabaseStore } from "../stores/database";
import PageTree from "../features/sidebar/PageTree";
import BlockShell from "../features/editor/BlockShell";
import CommandPalette from "../features/editor/CommandPalette";
import type { Property } from "../types/database";

export default function RecordDetailPage() {
  const { recordId } = useParams<{ recordId: string }>();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const loadRecord = useDatabaseStore((s) => s.loadRecord);

  const {
    blocks,
    loading,
    saving,
    dirty,
    error: editorError,
    commandPaletteIndex,
    loadPage,
    addBlock,
    updateBlock,
    updateBlockType,
    deleteBlock,
    duplicateBlock,
    indentBlock,
    outdentBlock,
    openCommandPalette,
    closeCommandPalette,
  } = useEditorStore();

  const [pageLoading, setPageLoading] = useState(true);
  const [pageId, setPageId] = useState<number | null>(null);
  const [recordProps, setRecordProps] = useState<Property[]>([]);
  const [recordValues, setRecordValues] = useState<Record<number, any>>({});
  const [pageError, setPageError] = useState<string | null>(null);

  useEffect(() => {
    if (!recordId) return;
    const id = Number(recordId);
    setPageLoading(true);
    setPageError(null);

    loadRecord(id)
      .then(({ record: rec, properties: props }) => {
        setRecordProps(props || []);
        const vals: Record<number, any> = {};
        (rec.property_values || []).forEach((pv: { property_id: number; value: any }) => {
          vals[pv.property_id] = pv.value;
        });
        setRecordValues(vals);
        setPageId(rec.page_id);
        return loadPage(rec.page_id);
      })
      .catch((e) => {
        setPageError(e instanceof Error ? e.message : "Record not found");
        setPageLoading(false);
      });
  }, [recordId]);

  // Watch for editor load completion
  useEffect(() => {
    if (!loading && pageLoading && blocks.length >= 0) {
      setPageLoading(false);
    }
  }, [loading, blocks]);

  const handleCommandSelect = useCallback(
    (type: string) => {
      if (commandPaletteIndex !== null) {
        updateBlockType(commandPaletteIndex, type);
      }
    },
    [commandPaletteIndex, updateBlockType]
  );

  const handleUpdate = useCallback(
    (index: number, props: Record<string, any>) => {
      if (props.text === "/") {
        openCommandPalette(index);
        updateBlock(index, { text: "" });
        return;
      }
      updateBlock(index, props);
    },
    [updateBlock, openCommandPalette]
  );

  const handleAddAfter = useCallback(
    (index: number) => {
      addBlock(index, "paragraph");
    },
    [addBlock]
  );

  // Helper to render a property value for display
  const renderValue = (prop: Property) => {
    const val = recordValues[prop.id];
    if (val == null || val === "") {
      return <span className="text-gray-300">Empty</span>;
    }

    switch (prop.type) {
      case "title":
      case "text":
        return <span>{typeof val === "string" ? val : val.text ?? ""}</span>;
      case "number":
        return <span>{val.number ?? val}</span>;
      case "select":
      case "status": {
        const selectVal = typeof val === "string" ? val : val.select;
        if (!selectVal) return <span className="text-gray-300">Empty</span>;
        const opt = prop.config?.options?.find((o) => o.id === selectVal);
        return <span>{opt?.name || selectVal}</span>;
      }
      case "date":
        return <span>{typeof val === "string" ? val : val.date ?? ""}</span>;
      case "checkbox":
        return <span>{val.checked || val === true ? "Yes" : "No"}</span>;
      case "url":
        return <span className="text-blue-600">{typeof val === "string" ? val : val.url ?? ""}</span>;
      case "email":
        return <span>{typeof val === "string" ? val : val.email ?? ""}</span>;
      case "phone":
        return <span>{typeof val === "string" ? val : val.phone ?? ""}</span>;
      default:
        return <span className="text-gray-400">--</span>;
    }
  };

  // ---------- Loading state ----------
  if (pageLoading || loading) {
    return (
      <div className="flex min-h-screen bg-white">
        <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <span className="text-sm font-semibold text-gray-900">
              {user?.name || "User"}'s Notion
            </span>
            <button onClick={logout} className="text-xs text-gray-400 hover:text-gray-600">
              Sign out
            </button>
          </div>
          <div className="flex-1 overflow-y-auto">
            <PageTree />
          </div>
        </aside>
        <main className="flex flex-1 items-center justify-center">
          <div className="text-sm text-gray-400">Loading record...</div>
        </main>
      </div>
    );
  }

  // ---------- Error state ----------
  if (pageError || editorError) {
    return (
      <div className="flex min-h-screen bg-white">
        <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <span className="text-sm font-semibold text-gray-900">
              {user?.name || "User"}'s Notion
            </span>
            <button onClick={logout} className="text-xs text-gray-400 hover:text-gray-600">
              Sign out
            </button>
          </div>
          <div className="flex-1 overflow-y-auto">
            <PageTree />
          </div>
        </aside>
        <main className="flex flex-1 items-center justify-center">
          <div className="text-center">
            <p className="text-red-500">{pageError || editorError}</p>
            <button
              onClick={() => pageId && loadPage(pageId)}
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
          <button onClick={logout} className="text-xs text-gray-400 hover:text-gray-600">
            Sign out
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          <PageTree />
        </div>
      </aside>

      <main className="flex flex-1 flex-col">
        {/* Save status bar */}
        <div className="flex items-center justify-end gap-2 border-b border-gray-100 px-6 py-1">
          {saving && <span className="text-xs text-gray-400">Saving...</span>}
          {dirty && !saving && <span className="text-xs text-gray-300">Unsaved</span>}
          {!dirty && !saving && <span className="text-xs text-gray-300">Saved</span>}
        </div>

        {/* Properties section */}
        {recordProps.length > 0 && (
          <div className="border-b border-gray-100 px-24 py-4">
            <div className="mx-auto max-w-3xl space-y-2">
              <h2 className="text-xs font-medium uppercase text-gray-400">Properties</h2>
              {recordProps.map((prop) => (
                <div key={prop.id} className="flex items-center gap-3 text-sm">
                  <span className="w-32 flex-shrink-0 text-gray-500">{prop.name}</span>
                  <span className="text-gray-900">{renderValue(prop)}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Block editor */}
        <div className="flex-1 overflow-y-auto px-24 py-8">
          <div className="mx-auto max-w-3xl">
            {blocks.map((block, i) => (
              <div key={block.tempId || block.id || i} className="relative">
                <BlockShell
                  block={block}
                  index={i}
                  onUpdate={handleUpdate}
                  onChangeType={updateBlockType}
                  onDelete={deleteBlock}
                  onDuplicate={duplicateBlock}
                  onAddAfter={handleAddAfter}
                  onIndent={indentBlock}
                  onOutdent={outdentBlock}
                />
                {commandPaletteIndex === i && (
                  <CommandPalette
                    isOpen={true}
                    onSelect={handleCommandSelect}
                    onClose={closeCommandPalette}
                  />
                )}
              </div>
            ))}

            {blocks.length === 0 && (
              <button
                onClick={() => addBlock(-1, "paragraph")}
                className="text-sm text-gray-400 hover:text-gray-600"
              >
                Click to add a block, or type /
              </button>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
