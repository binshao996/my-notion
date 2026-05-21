import { useState, useRef, useEffect, useCallback } from "react";
import { api, ApiError } from "../../lib/api";
import type { AIBlock, WritingResponse } from "./types";

interface AIWritingPanelProps {
  isOpen: boolean;
  onClose: () => void;
  selectedText: string;
  pageContext?: string;
  onInsertBlocks: (blocks: AIBlock[]) => void;
  position?: { top: number; left: number };
}

const actions = [
  { value: "generate", label: "Generate", placeholder: "What do you want to generate?" },
  { value: "rewrite", label: "Rewrite", placeholder: "How should this be rewritten?" },
  { value: "summarize", label: "Summarize", placeholder: "Any specific focus for the summary?" },
  { value: "expand", label: "Expand", placeholder: "What direction should the expansion take?" },
  { value: "translate", label: "Translate", placeholder: "Any special instructions?" },
  { value: "proofread", label: "Proofread", placeholder: "Any specific concerns?" },
] as const;

const tones = [
  { value: "", label: "Default" },
  { value: "professional", label: "Professional" },
  { value: "casual", label: "Casual" },
  { value: "confident", label: "Confident" },
  { value: "polite", label: "Polite" },
];

const languages = [
  { value: "", label: "Auto" },
  { value: "english", label: "English" },
  { value: "chinese", label: "Chinese" },
];

/** Parse AI response text into blocks, matching the server-side parseBlocks logic. */
function parseBlocks(content: string): AIBlock[] {
  let text = content.trim();

  // Strip markdown code fences if present
  if (text.startsWith("```")) {
    const nlIdx = text.indexOf("\n");
    if (nlIdx !== -1) text = text.slice(nlIdx + 1);
    const endIdx = text.lastIndexOf("```");
    if (endIdx !== -1) text = text.slice(0, endIdx);
    text = text.trim();
  }

  try {
    const blocks = JSON.parse(text) as AIBlock[];
    return blocks.map((b) => ({
      type: b.type || "paragraph",
      content: b.content || "",
    }));
  } catch {
    // Fallback: wrap raw content in a single paragraph block
    return [{ type: "paragraph", content: text }];
  }
}

export default function AIWritingPanel({
  isOpen,
  onClose,
  selectedText,
  pageContext,
  onInsertBlocks,
  position,
}: AIWritingPanelProps) {
  const [action, setAction] = useState<string>("generate");
  const [prompt, setPrompt] = useState("");
  const [tone, setTone] = useState("");
  const [lang, setLang] = useState("");
  const [loading, setLoading] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [streamedText, setStreamedText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [results, setResults] = useState<AIBlock[] | null>(null);

  const panelRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const currentAction = actions.find((a) => a.value === action);

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen, action]);

  // Close on Escape
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    if (isOpen) {
      window.addEventListener("keydown", handler);
      return () => window.removeEventListener("keydown", handler);
    }
  }, [isOpen, onClose]);

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    if (isOpen) {
      // Delay to avoid the triggering click from immediately closing
      setTimeout(() => document.addEventListener("mousedown", handler), 0);
      return () => document.removeEventListener("mousedown", handler);
    }
  }, [isOpen, onClose]);

  /** Build the request body for both streaming and non-streaming. */
  const buildBody = useCallback((): Record<string, string> => {
    const context = pageContext
      ? `${pageContext}\n\nSelected text:\n${selectedText}`
      : selectedText;

    const body: Record<string, string> = {
      action,
      context,
      prompt: prompt.trim() || action,
    };
    if (tone) body.tone = tone;
    if (lang && action === "translate") body.lang = lang;
    return body;
  }, [prompt, selectedText, pageContext, action, tone, lang]);

  /** Stream the AI response via SSE from ?stream=true. Throws on failure. */
  const handleRunStreaming = useCallback(async () => {
    setStreaming(true);
    setStreamedText("");
    setError(null);
    setResults(null);

    const token = localStorage.getItem("token");
    const body = buildBody();

    const response = await fetch("/api/v1/ai/write?stream=true", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errBody = await response.json().catch(() => ({}));
      throw new ApiError(errBody.error || "AI writing request failed", response.status);
    }

    const reader = response.body?.getReader();
    if (!reader) throw new Error("Browser does not support streaming");

    const decoder = new TextDecoder();
    let buffer = "";
    let fullText = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (!line.startsWith("data: ")) continue;
        const data = line.slice(6);

        if (data === "[DONE]") {
          const blocks = parseBlocks(fullText);
          setResults(blocks);
          setStreaming(false);
          return;
        }

        try {
          const parsed = JSON.parse(data);
          if (parsed.error) {
            throw new ApiError(parsed.error, 500);
          }
          if (parsed.delta) {
            fullText += parsed.delta;
            setStreamedText(fullText);
          }
        } catch (e) {
          if (e instanceof ApiError) throw e;
          // Ignore unparseable individual lines
        }
      }
    }

    // Reached end-of-stream without [DONE] — parse what we have
    const blocks = parseBlocks(fullText);
    setResults(blocks);
    setStreaming(false);
  }, [buildBody]);

  /** Fallback: use the standard (non-streaming) POST and get full response. */
  const handleRunNonStreaming = useCallback(async () => {
    const body = buildBody();
    const data = await api.post<WritingResponse>("/ai/write", body);
    setResults(data.blocks);
  }, [buildBody]);

  const handleRun = useCallback(async () => {
    if (!prompt.trim() && !selectedText.trim()) return;

    setLoading(true);
    setError(null);
    setResults(null);

    try {
      await handleRunStreaming();
    } catch {
      // Streaming failed; fall back to non-streaming
      setStreaming(false);
      try {
        await handleRunNonStreaming();
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message || "AI writing request failed");
        } else {
          setError("Network error. Please try again.");
        }
      }
    } finally {
      setLoading(false);
    }
  }, [prompt, selectedText, handleRunStreaming, handleRunNonStreaming]);

  const handleInsertBlock = useCallback(
    (block: AIBlock) => {
      onInsertBlocks([block]);
    },
    [onInsertBlocks]
  );

  const handleInsertAll = useCallback(() => {
    if (results) {
      onInsertBlocks(results);
      onClose();
    }
  }, [results, onInsertBlocks, onClose]);

  if (!isOpen) return null;

  const panelStyle: React.CSSProperties = position
    ? { position: "fixed", top: position.top, left: position.left, zIndex: 100 }
    : { position: "fixed", top: "50%", left: "50%", transform: "translate(-50%, -50%)", zIndex: 100 };

  return (
    <div
      ref={panelRef}
      style={panelStyle}
      className="w-full max-w-md rounded-lg border border-gray-200 bg-white shadow-lg"
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
        <h3 className="text-sm font-semibold text-gray-800">AI Writing</h3>
        <button
          onClick={onClose}
          className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        >
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div className="p-4 space-y-3">
        {/* Action selector */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">Action</label>
          <div className="flex flex-wrap gap-1">
            {actions.map((a) => (
              <button
                key={a.value}
                onClick={() => {
                  setAction(a.value);
                  setResults(null);
                  setError(null);
                  setStreamedText("");
                  setStreaming(false);
                }}
                className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
                  action === a.value
                    ? "bg-blue-600 text-white"
                    : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                }`}
              >
                {a.label}
              </button>
            ))}
          </div>
        </div>

        {/* Context preview */}
        {selectedText && (
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">Context</label>
            <div className="max-h-20 overflow-y-auto rounded border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600">
              {selectedText.slice(0, 300)}
              {selectedText.length > 300 && "..."}
            </div>
          </div>
        )}

        {/* Prompt input */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">
            {currentAction?.placeholder || "Prompt"}
          </label>
          <textarea
            ref={inputRef}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                handleRun();
              }
            }}
            placeholder={currentAction?.placeholder}
            rows={2}
            className="w-full resize-none rounded border border-gray-300 px-3 py-2 text-sm text-gray-700 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        {/* Tone selector (for rewrite) */}
        {action === "rewrite" && (
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">Tone</label>
            <select
              value={tone}
              onChange={(e) => setTone(e.target.value)}
              className="w-full rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              {tones.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>
        )}

        {/* Language selector (for translate) */}
        {action === "translate" && (
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">Target Language</label>
            <select
              value={lang}
              onChange={(e) => setLang(e.target.value)}
              className="w-full rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              {languages.map((l) => (
                <option key={l.value} value={l.value}>
                  {l.label}
                </option>
              ))}
            </select>
          </div>
        )}

        {/* Run button */}
        <button
          onClick={handleRun}
          disabled={loading || (!prompt.trim() && !selectedText.trim())}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading ? (
            <>
              <Spinner />
              {streaming ? "Streaming..." : "AI is thinking..."}
            </>
          ) : (
            "Run"
          )}
        </button>

        {/* Streaming output — real-time text with blinking cursor */}
        {streaming && streamedText && (
          <div className="rounded border border-blue-200 bg-blue-50 px-3 py-3">
            <div className="whitespace-pre-wrap text-sm text-gray-700">
              {streamedText}
              <span className="ml-0.5 inline-block h-4 w-0.5 animate-pulse bg-blue-500 align-middle" />
            </div>
          </div>
        )}

        {/* Error state */}
        {error && (
          <div className="rounded border border-red-200 bg-red-50 px-3 py-2">
            <p className="text-xs text-red-600">{error}</p>
            <button
              onClick={handleRun}
              className="mt-1 text-xs font-medium text-red-700 underline hover:no-underline"
            >
              Retry
            </button>
          </div>
        )}

        {/* Results */}
        {results && results.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-gray-500">Results</span>
              <button
                onClick={handleInsertAll}
                className="text-xs font-medium text-blue-600 hover:text-blue-800"
              >
                Insert All
              </button>
            </div>
            <div className="max-h-64 space-y-2 overflow-y-auto">
              {results.map((block, i) => (
                <div
                  key={i}
                  className="flex items-start justify-between gap-2 rounded border border-gray-200 bg-gray-50 p-3"
                >
                  <div className="flex-1 min-w-0">
                    <span className="mb-1 inline-block rounded bg-gray-200 px-1.5 py-0.5 text-[10px] font-medium uppercase text-gray-500">
                      {block.type}
                    </span>
                    <p className="whitespace-pre-wrap text-sm text-gray-700">{block.content}</p>
                  </div>
                  <button
                    onClick={() => handleInsertBlock(block)}
                    className="shrink-0 rounded px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-50"
                  >
                    Insert
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Empty results */}
        {results && results.length === 0 && (
          <div className="rounded border border-gray-200 bg-gray-50 px-3 py-4 text-center">
            <p className="text-sm text-gray-500">No results generated. Try a different prompt.</p>
          </div>
        )}
      </div>
    </div>
  );
}

function Spinner() {
  return (
    <svg
      className="h-4 w-4 animate-spin text-white"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
  );
}
