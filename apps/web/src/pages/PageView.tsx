import { useEffect, useCallback, useState } from "react";
import { useParams } from "react-router-dom";
import { useEditorStore } from "../features/editor/useEditorStore";
import { useAuthStore } from "../stores/auth";
import { useSearchStore } from "../stores/search";
import { api } from "../lib/api";
import BlockShell from "../features/editor/BlockShell";
import CommandPalette from "../features/editor/CommandPalette";
import PageTree from "../features/sidebar/PageTree";
import ShareMenu from "../features/permissions/ShareMenu";
import CommentThread from "../features/comments/CommentThread";
import NotificationPopover from "../features/notifications/NotificationPopover";
import CollaborationProvider, { useCollaboration } from "../features/collaboration/CollaborationProvider";
import AwarenessCursors from "../features/collaboration/AwarenessCursors";
import FileUploadButton from "../features/editor/FileUploadButton";
import AIWritingPanel from "../features/ai/AIWritingPanel";
import AIQAModal from "../features/ai/AIQAModal";
import type { AIBlock } from "../features/ai/types";

export default function PageView() {
  const { pageId } = useParams<{ pageId: string }>();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const setWorkspaceId = useSearchStore((s) => s.setActiveWorkspaceId);
  const activeWorkspaceId = useSearchStore((s) => s.activeWorkspaceId);
  const openSearch = useSearchStore((s) => s.open);
  const {
    blocks,
    loading,
    saving,
    dirty,
    error,
    commandPaletteIndex,
    focusedBlockIndex,
    selectedBlockIndices,
    loadPage,
    addBlock,
    updateBlock,
    updateBlockType,
    deleteBlock,
    deleteSelectedBlocks,
    duplicateBlock,
    moveBlock,
    focusBlock,
    indentBlock,
    outdentBlock,
    openCommandPalette,
    closeCommandPalette,
    toggleBlockSelect,
    clearBlockSelection,
  } = useEditorStore();

  useEffect(() => {
    if (pageId) {
      loadPage(Number(pageId));
      api.get<{ workspace_id: number }>(`/pages/${pageId}`).then((p) => {
        if (p?.workspace_id) setWorkspaceId(p.workspace_id);
      }).catch(() => {});
    }
  }, [pageId, loadPage, setWorkspaceId]);

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
  const [showAIQA, setShowAIQA] = useState(false);

  if (loading) {
    return (
      <div className="flex min-h-screen bg-white">
        <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
            <span className="text-sm font-semibold text-gray-900">{user?.name || "User"}'s Notion</span>
            <div className="flex items-center gap-1">
              <NotificationPopover />
              <button onClick={() => setShowAIQA(true)} className="text-xs text-purple-400 hover:text-purple-600" title="Ask AI (Cmd+J)">
                ✨ AI
              </button>
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
              <button onClick={() => setShowAIQA(true)} className="text-xs text-purple-400 hover:text-purple-600" title="Ask AI (Cmd+J)">
                ✨ AI
              </button>
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
            <button onClick={() => setShowAIQA(true)} className="text-xs text-purple-400 hover:text-purple-600" title="Ask AI (Cmd+J)">
              ✨ AI
            </button>
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
          deleteSelectedBlocks={deleteSelectedBlocks}
          duplicateBlock={duplicateBlock}
          handleAddAfter={handleAddAfter}
          indentBlock={indentBlock}
          outdentBlock={outdentBlock}
          commandPaletteIndex={commandPaletteIndex}
          handleCommandSelect={handleCommandSelect}
          closeCommandPalette={closeCommandPalette}
          addBlock={addBlock}
          updateBlock={updateBlock}
          openSearch={openSearch}
          moveBlock={moveBlock}
          focusBlock={focusBlock}
          focusedBlockIndex={focusedBlockIndex}
          selectedBlockIndices={selectedBlockIndices}
          toggleBlockSelect={toggleBlockSelect}
          clearBlockSelection={clearBlockSelection}
          workspaceId={activeWorkspaceId}
        />
      </CollaborationProvider>
      {showAIQA && (
        <AIQAModal isOpen={showAIQA} onClose={() => setShowAIQA(false)} workspaceId={activeWorkspaceId} />
      )}
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
  deleteSelectedBlocks,
  duplicateBlock,
  handleAddAfter,
  indentBlock,
  outdentBlock,
  commandPaletteIndex,
  handleCommandSelect,
  closeCommandPalette,
  addBlock,
  updateBlock,
  openSearch,
  moveBlock,
  focusBlock,
  focusedBlockIndex,
  selectedBlockIndices,
  toggleBlockSelect,
  clearBlockSelection,
  workspaceId,
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
  deleteSelectedBlocks: () => void;
  duplicateBlock: (i: number) => void;
  handleAddAfter: (i: number) => void;
  indentBlock: (i: number) => void;
  outdentBlock: (i: number) => void;
  commandPaletteIndex: number | null;
  handleCommandSelect: (t: string) => void;
  closeCommandPalette: () => void;
  addBlock: (i: number, t?: string) => void;
  updateBlock: (i: number, props: Record<string, any>) => void;
  openSearch: () => void;
  moveBlock: (from: number, to: number) => void;
  focusBlock: (index: number) => void;
  focusedBlockIndex: number;
  selectedBlockIndices: number[];
  toggleBlockSelect: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
  clearBlockSelection: () => void;
  workspaceId: number | null;
}) {
  const collab = useCollaboration();

  const [showAIWrite, setShowAIWrite] = useState(false);
  const [selectedText, setSelectedText] = useState("");
  const [aiPanelPos, setAiPanelPos] = useState<{top: number; left: number} | null>(null);

  const handleFileUploaded = useCallback(
    (url: string, fileName: string) => {
      const isImage = /\.(png|jpe?g|gif|svg|webp|bmp)$/i.test(fileName);
      const blockType = isImage ? "image" : "file";
      const blockProps = isImage ? { url } : { url, name: fileName };
      // addBlock inserts a new block after the focused index and sets dirty
      addBlock(focusedBlockIndex, blockType);
      // The new block is at index focusedBlockIndex + 1
      updateBlock(focusedBlockIndex + 1, blockProps);
    },
    [addBlock, focusedBlockIndex, updateBlock]
  );

  // Keyboard handlers: Ctrl+Shift+Arrow to move blocks, Delete for multi-select
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // Delete key: delete selected blocks if multi-select is active
      if (e.key === "Delete" || (e.key === "Backspace" && (e.ctrlKey || e.metaKey))) {
        if (selectedBlockIndices.length > 0) {
          e.preventDefault();
          deleteSelectedBlocks();
          return;
        }
      }

      if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
        if (e.key === "ArrowUp" && focusedBlockIndex > 0) {
          e.preventDefault();
          moveBlock(focusedBlockIndex, focusedBlockIndex - 1);
        } else if (e.key === "ArrowDown" && focusedBlockIndex < blocks.length - 1) {
          e.preventDefault();
          moveBlock(focusedBlockIndex, focusedBlockIndex + 1);
        }
      }
    },
    [focusedBlockIndex, blocks.length, moveBlock, selectedBlockIndices, deleteSelectedBlocks]
  );

  // Click on empty container space to clear selection
  const handleContainerClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === e.currentTarget) {
        clearBlockSelection();
      }
    },
    [clearBlockSelection]
  );

  // Capture text selection for AI writing
  const handleMouseUp = useCallback(
    () => {
      setTimeout(() => {
        const selection = window.getSelection();
        const text = selection?.toString().trim();
        if (text && text.length > 0) {
          setSelectedText(text);
          const range = selection?.getRangeAt(0);
          const rect = range?.getBoundingClientRect();
          if (rect) {
            setAiPanelPos({
              top: rect.bottom + 6,
              left: rect.left + rect.width / 2 - 28,
            });
          }
        }
      }, 0);
    },
    []
  );

  // Clear selection float when clicking elsewhere
  useEffect(() => {
    if (!selectedText) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest(".ai-float-btn")) {
        setSelectedText("");
        setAiPanelPos(null);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [selectedText]);

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
          <button
            className="rounded px-2 py-0.5 text-xs text-gray-400 hover:text-gray-600"
            onClick={openSearch}
          >
            Search
          </button>
          <FileUploadButton onUploaded={handleFileUploaded} />
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
      <div
        className="flex-1 overflow-y-auto px-24 py-8 relative"
        onKeyDown={handleKeyDown}
        onClick={handleContainerClick}
        onMouseUp={handleMouseUp}
      >
        <AwarenessCursors />
        <div className="mx-auto max-w-3xl">
          {blocks.map((block, i) => {
            const isSelected = selectedBlockIndices.includes(i);
            return (
              <div
                key={block.tempId || block.id || i}
                className="relative"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleBlockSelect(i, e.shiftKey, e.ctrlKey || e.metaKey);
                }}
                onDragOver={(e) => {
                  e.preventDefault();
                  e.dataTransfer.dropEffect = "move";
                }}
                onDrop={(e) => {
                  e.preventDefault();
                  const fromIndex = parseInt(e.dataTransfer.getData("text/plain"), 10);
                  if (!isNaN(fromIndex) && fromIndex !== i) {
                    moveBlock(fromIndex, i);
                  }
                }}
              >
                <BlockShell
                  block={block}
                  index={i}
                  isSelected={isSelected}
                  onUpdate={handleUpdate}
                  onChangeType={updateBlockType}
                  onDelete={deleteBlock}
                  onDuplicate={duplicateBlock}
                  onAddAfter={handleAddAfter}
                  onIndent={indentBlock}
                  onOutdent={outdentBlock}
                  onFocus={focusBlock}
                  ydoc={collab?.ydoc ?? null}
                  awareness={collab?.awareness ?? null}
                  workspaceId={workspaceId ?? null}
                />
                {commandPaletteIndex === i && (
                  <CommandPalette
                    isOpen={true}
                    onSelect={handleCommandSelect}
                    onClose={closeCommandPalette}
                  />
                )}
              </div>
            );
          })}

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

      {/* Floating AI sparkle button near text selection */}
      {selectedText && aiPanelPos && !showAIWrite && (
        <button
          className="ai-float-btn fixed z-50 rounded-full bg-purple-600 px-3 py-1 text-xs font-medium text-white shadow-lg hover:bg-purple-700 transition-colors"
          style={{ top: aiPanelPos.top, left: aiPanelPos.left }}
          onClick={() => setShowAIWrite(true)}
        >
          ✨ AI
        </button>
      )}

      {/* AI Writing Panel */}
      {showAIWrite && (
        <AIWritingPanel
          isOpen={showAIWrite}
          onClose={() => {
            setShowAIWrite(false);
            setSelectedText("");
            setAiPanelPos(null);
          }}
          selectedText={selectedText}
          onInsertBlocks={(blocks: AIBlock[]) => {
            blocks.forEach((b) => {
              addBlock(focusedBlockIndex, b.type || "paragraph");
              updateBlock(focusedBlockIndex + 1, { text: b.content });
            });
            setShowAIWrite(false);
            setSelectedText("");
            setAiPanelPos(null);
          }}
          position={aiPanelPos ?? undefined}
        />
      )}

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
