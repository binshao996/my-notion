import { useState, useEffect, useCallback } from "react";
import { api } from "../../lib/api";

interface Permission {
  id: number;
  page_id: number;
  subject_type: string;
  subject_id: number;
  role: string;
}

interface ShareToken {
  id: number;
  token: string;
  page_id: number;
  role: string;
  expires_at: string | null;
}

interface ShareDialogProps {
  pageId: number;
  open: boolean;
  onClose: () => void;
}

export default function ShareDialog({ pageId, open, onClose }: ShareDialogProps) {
  const [tab, setTab] = useState<"invite" | "link">("invite");
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [tokens, setTokens] = useState<ShareToken[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("viewer");
  const [linkRole, setLinkRole] = useState("viewer");
  const [error, setError] = useState("");

  const loadData = useCallback(() => {
    api.get<Permission[]>(`/pages/${pageId}/permissions`).then(setPermissions).catch(() => {});
    api.get<ShareToken[]>(`/pages/${pageId}/share-tokens`).then(setTokens).catch(() => {});
  }, [pageId]);

  useEffect(() => {
    if (open) loadData();
  }, [open, loadData]);

  const addPermission = async () => {
    if (!email.trim()) return;
    setError("");
    try {
      await api.post(`/pages/${pageId}/permissions`, {
        subject_type: "user",
        subject_id: 0,
        role,
      });
      setEmail("");
      loadData();
    } catch (e: any) {
      setError(e.message || "Failed to add permission");
    }
  };

  const removePermission = async (permId: number) => {
    await api.delete(`/pages/${pageId}/permissions/${permId}`);
    loadData();
  };

  const createShareLink = async () => {
    try {
      await api.post(`/pages/${pageId}/share-tokens`, { role: linkRole });
      loadData();
    } catch (e: any) {
      setError(e.message || "Failed to create share link");
    }
  };

  const revokeToken = async (tokenId: number) => {
    await api.delete(`/pages/${pageId}/share-tokens/${tokenId}`);
    loadData();
  };

  const copyToClipboard = (token: string) => {
    const url = `${window.location.origin}/shared/${token}`;
    navigator.clipboard.writeText(url);
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-full max-w-md rounded-lg bg-white shadow-xl" onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
          <h2 className="text-sm font-semibold text-gray-900">Share</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">✕</button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-200">
          <button
            className={`flex-1 px-4 py-2 text-xs font-medium ${tab === "invite" ? "border-b-2 border-blue-600 text-blue-600" : "text-gray-500"}`}
            onClick={() => setTab("invite")}
          >
            Invite
          </button>
          <button
            className={`flex-1 px-4 py-2 text-xs font-medium ${tab === "link" ? "border-b-2 border-blue-600 text-blue-600" : "text-gray-500"}`}
            onClick={() => setTab("link")}
          >
            Share link
          </button>
        </div>

        <div className="p-4">
          {error && <div className="mb-2 text-xs text-red-500">{error}</div>}

          {tab === "invite" ? (
            <div className="space-y-3">
              <div className="flex gap-2">
                <input
                  type="text"
                  className="flex-1 rounded border border-gray-200 px-2 py-1.5 text-sm outline-none focus:border-blue-400"
                  placeholder="Email address"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  onKeyDown={e => e.key === "Enter" && addPermission()}
                />
                <select
                  className="rounded border border-gray-200 px-2 py-1.5 text-sm outline-none focus:border-blue-400"
                  value={role}
                  onChange={e => setRole(e.target.value)}
                >
                  <option value="editor">Can edit</option>
                  <option value="commenter">Can comment</option>
                  <option value="viewer">Can view</option>
                </select>
                <button
                  className="rounded bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700"
                  onClick={addPermission}
                >
                  Invite
                </button>
              </div>

              {/* Existing permissions */}
              {permissions.length > 0 && (
                <div className="space-y-1">
                  <div className="text-xs font-medium text-gray-500">People with access</div>
                  {permissions.map(p => (
                    <div key={p.id} className="flex items-center justify-between rounded bg-gray-50 px-3 py-1.5">
                      <div>
                        <div className="text-sm text-gray-700">User #{p.subject_id}</div>
                        <div className="text-xs text-gray-400">{p.role}</div>
                      </div>
                      <button
                        className="text-xs text-gray-400 hover:text-red-500"
                        onClick={() => removePermission(p.id)}
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <div className="space-y-3">
              <div className="flex gap-2">
                <select
                  className="rounded border border-gray-200 px-2 py-1.5 text-sm outline-none focus:border-blue-400"
                  value={linkRole}
                  onChange={e => setLinkRole(e.target.value)}
                >
                  <option value="editor">Can edit</option>
                  <option value="commenter">Can comment</option>
                  <option value="viewer">Can view</option>
                </select>
                <button
                  className="rounded bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700"
                  onClick={createShareLink}
                >
                  Create link
                </button>
              </div>

              {tokens.length > 0 && (
                <div className="space-y-1">
                  <div className="text-xs font-medium text-gray-500">Share links</div>
                  {tokens.map(t => (
                    <div key={t.id} className="flex items-center justify-between rounded bg-gray-50 px-3 py-1.5">
                      <div className="flex-1 truncate">
                        <div className="text-sm text-gray-700 truncate">
                          /shared/{t.token.slice(0, 8)}...
                        </div>
                        <div className="text-xs text-gray-400">{t.role}</div>
                      </div>
                      <div className="flex gap-1">
                        <button
                          className="rounded px-2 py-0.5 text-xs text-blue-600 hover:bg-blue-50"
                          onClick={() => copyToClipboard(t.token)}
                        >
                          Copy
                        </button>
                        <button
                          className="rounded px-2 py-0.5 text-xs text-red-500 hover:bg-red-50"
                          onClick={() => revokeToken(t.id)}
                        >
                          Revoke
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
