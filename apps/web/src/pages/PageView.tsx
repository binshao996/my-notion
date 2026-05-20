import { useEffect, useCallback, useState } from "react";
import { useParams } from "react-router-dom";
import { useEditorStore } from "../features/editor/useEditorStore";
import { useAuthStore } from "../stores/auth";
import BlockShell from "../features/editor/BlockShell";
import CommandPalette from "../features/editor/CommandPalette";
import PageTree from "../features/sidebar/PageTree";
import ShareMenu from "../features/permissions/ShareMenu";
import CommentThread from "../features/comments/CommentThread";
import NotificationPopover from "../features/notifications/NotificationPopover";
import CollaborationProvider, { useCollaboration } from "../features/collaboration/CollaborationProvider";
import AwarenessCursors from "../features/collaboration/AwarenessCursors";

export default function PageView() {
  const { pageId } = useParams<{ pageId: string }>();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const {
    blocks,
    loading,
    saving,
    dirty,
    error,
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

  useEffect(() => {
    if (pageId) {
      loadPage(Number(pageId));
    }
  }, [pageId, loadPage]);

  const handleCommandSelect = useCallback(
    (type: string) => {
      if (commandPaletteIndex !== null) {
        updateBlockType(commandPaletteIndex, type);
      }
    },
    [commandPaletteIndex, updateBlockType]
  );

  // Detect / command: if block text is just "/", open the command palette
  const handleUpdate = useCallback(
    (index: number, props: Record<string, any>) => {
      if (props.text === "/") {
        openCommandPalette(index);
        // Clear the "/" character from the block
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

  const [showComments, setShowComments] = useState(false);

  if (loading) {
    return (
      <div className="flex min-h-screen bg-white">
        <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <span className="text-sm font-semibold text-gray-900">{user?.name || "User"}'s Notion</span>
            <div className="flex items-center gap-1">
              <NotificationPopover />
              <button onClick={logout} className="text-xs text-gray-400 hover:text-gray-600">
                Sign out
              </button>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto">
            <PageTree />
          </div>
        </aside>
        <main className="flex flex-1 items-center justify-center">
          <div className="text-sm text-gray-400">Loading page...</div>
        </main>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-screen bg-white">
        <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <span className="text-sm font-semibold text-gray-900">{user?.name || "User"}'s Notion</span>
            <div className="flex items-center gap-1">
              <NotificationPopover />
              <button onClick={logout} className="text-xs text-gray-400 hover:text-gray-600">
                Sign out
              </button>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto">
            <PageTree />
          </div>
        </aside>
        <main className="flex flex-1 items-center justify-center">
          <div className="text-center">
            <p className="text-red-500">{error}</p>
            <button
              onClick={() => pageId && loadPage(Number(pageId))}
              className="mt-2 text-sm text-blue-600 hover:underline"
            >
              Retry
            </button>
          </div>
        </main>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen bg-white">
      {/* Sidebar */}
      <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
        <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
          <span className="text-sm font-semibold text-gray-900">{user?.name || "User"}'s Notion</span>
          <div className="flex items-center gap-1">
            <NotificationPopover />
            <button onClick={logout} className="text-xs text-gray-400 hover:text-gray-600">
              Sign out
            </button>
          </div>
        </div>
        <div className="flex-1 overflow-y-auto">
          <PageTree />
        </div>
      </aside>

      {/* Editor */}
      <CollaborationProvider pageId={Number(pageId)}>
        <EditorArea
          pageId={Number(pageId)}
          blocks={blocks}
          showComments={showComments}
          setShowComments={setShowComments}
          saving={saving}
          dirty={dirty}
          handleUpdate={handleUpdate}
          updateBlockType={updateBlockType}
          deleteBlock={deleteBlock}
          duplicateBlock={duplicateBlock}
          handleAddAfter={handleAddAfter}
          indentBlock={indentBlock}
          outdentBlock={outdentBlock}
          commandPaletteIndex={commandPaletteIndex}
          handleCommandSelect={handleCommandSelect}
          closeCommandPalette={closeCommandPalette}
          addBlock={addBlock}
        />
      </CollaborationProvider>
    </div>
  );
}

// Inner component that reads collaboration context and renders the editor
function EditorArea({
  pageId,
  blocks,
  showComments,
  setShowComments,
  saving,
  dirty,
  handleUpdate,
  updateBlockType,
  deleteBlock,
  duplicateBlock,
  handleAddAfter,
  indentBlock,
  outdentBlock,
  commandPaletteIndex,
  handleCommandSelect,
  closeCommandPalette,
  addBlock,
}: {
  pageId: number;
  blocks: any[];
  showComments: boolean;
  setShowComments: (v: boolean) => void;
  saving: boolean;
  dirty: boolean;
  handleUpdate: (i: number, props: Record<string, any>) => void;
  updateBlockType: (i: number, t: string) => void;
  deleteBlock: (i: number) => void;
  duplicateBlock: (i: number) => void;
  handleAddAfter: (i: number) => void;
  indentBlock: (i: number) => void;
  outdentBlock: (i: number) => void;
  commandPaletteIndex: number | null;
  handleCommandSelect: (t: string) => void;
  closeCommandPalette: () => void;
  addBlock: (i: number, t?: string) => void;
}) {
  const collab = useCollaboration();

  return (
    <main className="flex flex-1 flex-col">
      {/* Save status bar */}
      <div className="flex items-center justify-between border-b border-gray-100 px-6 py-1">
        <div className="flex items-center gap-2">
          <ShareMenu pageId={pageId} />
          <button
            className={`rounded px-2 py-0.5 text-xs ${showComments ? "bg-blue-100 text-blue-700" : "text-gray-400 hover:text-gray-600"}`}
            onClick={() => setShowComments(!showComments)}
          >
            Comments
          </button>
          {collab && (
            <span className={`text-xs ${collab.connected ? "text-green-500" : "text-red-400"}`}>
              {collab.connected ? "Live" : "Offline"}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {saving && <span className="text-xs text-gray-400">Saving...</span>}
          {dirty && !saving && <span className="text-xs text-gray-300">Unsaved</span>}
          {!dirty && !saving && <span className="text-xs text-gray-300">Saved</span>}
        </div>
      </div>

      {/* Block editor */}
      <div className="flex-1 overflow-y-auto px-24 py-8 relative">
        <AwarenessCursors />
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
                ydoc={collab?.ydoc ?? null}
                awareness={collab?.awareness ?? null}
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

      {showComments && (
        <div className="border-t border-gray-200 px-6 py-4">
          <div className="mx-auto max-w-3xl">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold text-gray-500">Comments</span>
              <button className="text-xs text-gray-400 hover:text-gray-600" onClick={() => setShowComments(false)}>
                ✕
              </button>
            </div>
            <CommentThread pageId={pageId} />
          </div>
        </div>
      )}
    </main>
  );
}
