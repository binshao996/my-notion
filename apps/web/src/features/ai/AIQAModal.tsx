import { useState, useRef, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { api, ApiError } from "../../lib/api";
import type { Citation, ChatMessage, QAResponse } from "./types";

interface AIQAModalProps {
  isOpen: boolean;
  onClose: () => void;
  workspaceId: number | null;
}

const exampleQuestions = [
  "What are the key projects we discussed last week?",
  "Summarize the product requirements document",
  "Find meeting notes related to the Q4 planning",
  "What decisions were made about the new feature launch?",
];

export default function AIQAModal({ isOpen, onClose, workspaceId }: AIQAModalProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const navigate = useNavigate();

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  // Focus input when opened
  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen]);

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

  // Register Cmd+J shortcut to open the modal
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "j") {
        e.preventDefault();
        if (!isOpen) {
          // The parent should open the modal; we just listen here to avoid
          // the default browser behavior (downloads on some systems)
        }
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [isOpen]);

  const handleSend = useCallback(async () => {
    const question = input.trim();
    if (!question || !workspaceId) return;

    const userMessage: ChatMessage = { role: "user", content: question };
    setMessages((prev) => [...prev, userMessage]);
    setInput("");
    setLoading(true);
    setError(null);

    try {
      const data = await api.post<QAResponse>("/ai/ask", {
        question,
        workspace_id: workspaceId,
      });

      const assistantMessage: ChatMessage = {
        role: "assistant",
        content: data.answer,
        citations: data.citations,
      };
      setMessages((prev) => [...prev, assistantMessage]);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || "Failed to get answer");
      } else {
        setError("Network error. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  }, [input, workspaceId]);

  const handleExampleClick = useCallback((question: string) => {
    setInput(question);
    // Auto-send after a brief delay for visual feedback
    setTimeout(() => {
      // Need to capture the question since we're in a closure
    }, 100);
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend]
  );

  const handleRetry = useCallback(() => {
    // Remove the last user message and retry
    setMessages((prev) => {
      const lastUserIdx = prev.map((m) => m.role).lastIndexOf("user");
      if (lastUserIdx >= 0) {
        const question = prev[lastUserIdx].content;
        setInput(question);
        return prev.slice(0, -1); // Remove the last (error'd) assistant message
      }
      return prev;
    });
    setError(null);
  }, []);

  // Render citation markers like [1], [2] as styled badges
  const renderAnswer = useCallback(
    (content: string, citations?: Citation[]) => {
      if (!citations || citations.length === 0) {
        return <p className="whitespace-pre-wrap text-sm text-gray-700">{content}</p>;
      }

      // Split on citation markers [N]
      const parts = content.split(/(\[\d+\])/g);
      const rendered = parts.map((part, i) => {
        const match = part.match(/^\[(\d+)\]$/);
        if (match) {
          const num = parseInt(match[1], 10);
          const citation = citations[num - 1];
          if (citation) {
            return (
              <span
                key={i}
                className="inline-flex cursor-pointer items-center gap-1 rounded bg-blue-50 px-1.5 py-0.5 text-xs font-medium text-blue-600 hover:bg-blue-100"
                onClick={() => navigate(`/page/${citation.page_id}`)}
                title={`${citation.title}: ${citation.snippet}`}
              >
                [{num}]
              </span>
            );
          }
          return part;
        }
        return <span key={i} className="whitespace-pre-wrap">{part}</span>;
      });

      return <p className="text-sm text-gray-700">{rendered}</p>;
    },
    [navigate]
  );

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="flex h-[600px] max-h-[90vh] w-full max-w-2xl flex-col rounded-xl border border-gray-200 bg-white shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-100 px-5 py-3">
          <div className="flex items-center gap-2">
            <svg className="h-5 w-5 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" />
            </svg>
            <h2 className="text-sm font-semibold text-gray-800">Ask AI</h2>
          </div>
          <button
            onClick={onClose}
            className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Messages area */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {messages.length === 0 && !loading && (
            <div className="flex h-full flex-col items-center justify-center text-center">
              <svg className="mb-3 h-10 w-10 text-purple-200" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" />
              </svg>
              <p className="mb-1 text-sm font-medium text-gray-700">Ask anything about your workspace</p>
              <p className="mb-4 text-xs text-gray-400">AI searches across all your pages and databases</p>
              <div className="flex flex-wrap justify-center gap-2">
                {exampleQuestions.map((q, i) => (
                  <button
                    key={i}
                    onClick={() => handleExampleClick(q)}
                    className="rounded-full border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50 hover:text-gray-900 transition-colors"
                  >
                    {q}
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((msg, i) => (
            <div
              key={i}
              className={`mb-4 flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
            >
              <div
                className={`max-w-[85%] rounded-xl px-4 py-2.5 ${
                  msg.role === "user"
                    ? "bg-blue-600 text-white"
                    : "bg-gray-100 text-gray-800"
                }`}
              >
                {msg.role === "assistant" ? (
                  <div>
                    {renderAnswer(msg.content, msg.citations)}
                    {msg.citations && msg.citations.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1.5 border-t border-gray-200 pt-2">
                        {msg.citations.map((citation, ci) => (
                          <button
                            key={ci}
                            onClick={() => navigate(`/page/${citation.page_id}`)}
                            className="inline-flex items-center gap-1 rounded bg-white px-2 py-1 text-xs text-gray-600 shadow-sm hover:bg-blue-50 hover:text-blue-600 transition-colors border border-gray-200"
                            title={citation.snippet}
                          >
                            <span className="font-medium text-blue-500">[{ci + 1}]</span>
                            <span className="truncate max-w-[160px]">{citation.title}</span>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                ) : (
                  <p className="whitespace-pre-wrap text-sm">{msg.content}</p>
                )}
              </div>
            </div>
          ))}

          {/* Loading indicator */}
          {loading && (
            <div className="mb-4 flex justify-start">
              <div className="rounded-xl bg-gray-100 px-4 py-3">
                <div className="flex items-center gap-1">
                  <span className="h-2 w-2 animate-bounce rounded-full bg-gray-400" style={{ animationDelay: "0ms" }} />
                  <span className="h-2 w-2 animate-bounce rounded-full bg-gray-400" style={{ animationDelay: "150ms" }} />
                  <span className="h-2 w-2 animate-bounce rounded-full bg-gray-400" style={{ animationDelay: "300ms" }} />
                </div>
                <p className="mt-1 text-xs text-gray-500">Searching workspace...</p>
              </div>
            </div>
          )}

          {/* Error state */}
          {error && (
            <div className="mb-4 flex justify-start">
              <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-2.5">
                <p className="text-sm text-red-600">{error}</p>
                <button
                  onClick={handleRetry}
                  className="mt-1 text-xs font-medium text-red-700 underline hover:no-underline"
                >
                  Retry
                </button>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Input area */}
        <div className="border-t border-gray-100 px-5 py-3">
          <div className="flex items-end gap-2">
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={
                workspaceId
                  ? "Ask a question about your workspace..."
                  : "Select a workspace to ask questions..."
              }
              disabled={!workspaceId || loading}
              rows={2}
              className="flex-1 resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700 placeholder-gray-400 focus:border-purple-500 focus:outline-none focus:ring-1 focus:ring-purple-500 disabled:cursor-not-allowed disabled:opacity-50"
            />
            <button
              onClick={handleSend}
              disabled={!input.trim() || !workspaceId || loading}
              className="shrink-0 rounded-lg bg-purple-600 p-2 text-white transition-colors hover:bg-purple-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? (
                <svg className="h-5 w-5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              ) : (
                <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" />
                </svg>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
