import { useState, useRef } from "react";
import type { Property, SelectOption } from "../../types/database";
import { api } from "../../lib/api";
import RelationPicker from "./RelationPicker";

interface CellEditorProps {
  property: Property;
  value: any;
  onChange: (value: any) => void;
  readOnly?: boolean;
  workspaceId?: number;
}

/** Extract the human-readable display string from a raw JSONB value */
function displayValue(value: any, property: Property): string {
  if (value === null || value === undefined) return "";

  switch (property.type) {
    case "title":
    case "text":
    case "url":
    case "email":
    case "phone":
      return value?.text ?? value?.title ?? "";
    case "number":
      return value?.number != null ? String(value.number) : "";
    case "select":
    case "status": {
      const opt = property.config?.options?.find(
        (o: SelectOption) => o.id === value?.select
      );
      return opt?.name ?? "";
    }
    case "multi_select": {
      const ids: string[] = value?.multi_select ?? [];
      if (ids.length === 0) return "";
      const names = ids
        .map(
          (id: string) =>
            property.config?.options?.find((o: SelectOption) => o.id === id)?.name ?? id
        )
        .filter(Boolean);
      return names.join(", ");
    }
    case "relation": {
      const ids: number[] = value?.relation ?? [];
      return ids.length === 0 ? "" : `${ids.length} record${ids.length > 1 ? "s" : ""}`;
    }
    case "rollup":
      return value?.number != null ? String(value.number) : (value?.text ?? "");
    case "formula":
      return value?.number != null ? String(value.number) : (value?.text ?? "");
    case "date":
      return value?.date ?? "";
    case "checkbox":
      return value?.checkbox ? "true" : "false";
    case "person": {
      const persons = value?.person;
      if (!persons) return "—";
      if (Array.isArray(persons)) {
        return persons.length === 0 ? "—" : persons.map((p: any) => p.name).join(", ");
      }
      return persons.name ?? "—";
    }
    case "files": {
      const files = value?.files;
      if (!files || !Array.isArray(files) || files.length === 0) return "";
      return files.map((f: any) => f.name || f.url).join(", ");
    }
    case "created_time":
    case "last_edited_time":
      return value?.date ? new Date(value.date).toLocaleDateString() : "";
    default:
      return String(value);
  }
}

/** Build the JSONB value to send back on commit */
function buildCommitValue(raw: string, property: Property): any {
  switch (property.type) {
    case "title":
    case "text":
    case "url":
    case "email":
    case "phone":
      return { text: raw };
    case "number": {
      const num = parseFloat(raw);
      return { number: isNaN(num) ? 0 : num };
    }
    case "select":
    case "status":
      return { select: raw };
    case "date":
      return { date: raw };
    case "checkbox":
      return { checkbox: raw === "true" || raw === "checked" };
    case "multi_select":
      // Splitting comma-separated IDs
      return { multi_select: raw.split(",").map((s) => s.trim()).filter(Boolean) };
    case "relation":
      return null; // relation values come from RelationPicker, not text input
    case "rollup":
      return null;
    case "formula":
      return null;
    case "person":
      return null; // person values come from a picker, not text input
    case "created_time":
    case "last_edited_time":
      return null;
    case "files":
      return null; // files editing handled by FilesCellEditor directly
    default:
      return { text: raw };
  }
}

/** Get the initial edit-text from the raw value */
function editInitialValue(value: any, property: Property): string {
  if (value === null || value === undefined) return "";

  switch (property.type) {
    case "title":
    case "text":
    case "url":
    case "email":
    case "phone":
      return value?.text ?? "";
    case "number":
      return value?.number != null ? String(value.number) : "";
    case "select":
    case "status":
      return value?.select ?? "";
    case "date":
      return value?.date ?? "";
    case "checkbox":
      return value?.checkbox ? "checked" : "";
    case "multi_select":
      return (value?.multi_select ?? []).join(", ");
    case "relation":
      return (value?.relation ?? []).map(String).join(", ");
    case "rollup":
      return value?.number != null ? String(value.number) : (value?.text ?? "");
    case "formula":
      return value?.number != null ? String(value.number) : (value?.text ?? "");
    default:
      return "";
  }
}

function PersonCellEditor({
  value,
  onChange,
  workspaceId,
}: {
  value: any;
  onChange: (value: any) => void;
  workspaceId?: number;
}) {
  const currentPerson: { id: number; name: string; avatar_url: string } | null =
    value?.person ?? null;
  const [editing, setEditing] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<any[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout>>();

  const handleSearch = (query: string) => {
    if (searchTimeoutRef.current) clearTimeout(searchTimeoutRef.current);
    if (!query.trim() || !workspaceId) {
      setSearchResults([]);
      setShowDropdown(false);
      return;
    }
    searchTimeoutRef.current = setTimeout(async () => {
      try {
        const members = await api.get<any[]>(
          `/workspaces/${workspaceId}/members?q=${encodeURIComponent(query)}`
        );
        setSearchResults(members || []);
        setShowDropdown(true);
      } catch {
        setSearchResults([]);
        setShowDropdown(false);
      }
    }, 300);
  };

  // Chip with remove button when person is selected
  if (currentPerson) {
    return (
      <div className="flex items-center gap-1 px-1 py-0.5">
        <span className="inline-flex items-center gap-1 rounded bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">
          <span className="inline-flex h-4 w-4 items-center justify-center rounded-full bg-blue-400 text-[10px] font-medium text-white">
            {currentPerson.name?.charAt(0).toUpperCase()}
          </span>
          {currentPerson.name}
          <button
            onClick={() => onChange({ person: null })}
            className="ml-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full text-blue-500 hover:bg-blue-200 hover:text-blue-800 text-xs leading-none"
            title="Remove"
          >
            &times;
          </button>
        </span>
      </div>
    );
  }

  // Not editing: show empty placeholder
  if (!editing) {
    return (
      <div
        className="text-sm text-gray-400 px-1 py-0.5 min-h-[1.5rem] cursor-pointer rounded hover:bg-blue-50"
        onClick={() => setEditing(true)}
      >
        —
      </div>
    );
  }

  // Editing: show search input with dropdown
  return (
    <div className="relative">
      <input
        autoFocus
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
        placeholder="Search members..."
        value={searchQuery}
        onChange={(e) => {
          setSearchQuery(e.target.value);
          handleSearch(e.target.value);
        }}
        onBlur={() => {
          setTimeout(() => {
            setShowDropdown(false);
            if (!value?.person) setEditing(false);
          }, 200);
        }}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            setEditing(false);
            setSearchQuery("");
            setSearchResults([]);
            setShowDropdown(false);
          }
        }}
      />
      {showDropdown && searchResults.length > 0 && (
        <div className="absolute left-0 top-full z-50 mt-1 w-56 rounded border border-gray-200 bg-white py-1 shadow-lg">
          {searchResults.map((member: any) => (
            <button
              key={member.id}
              className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
              onMouseDown={(e) => {
                e.preventDefault();
                onChange({
                  person: {
                    id: member.id,
                    name: member.name,
                    avatar_url: member.avatar_url,
                  },
                });
                setSearchQuery("");
                setSearchResults([]);
                setShowDropdown(false);
                setEditing(false);
              }}
            >
              <span className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-gray-300 text-[10px] font-medium text-white">
                {member.name?.charAt(0).toUpperCase()}
              </span>
              {member.name}
            </button>
          ))}
        </div>
      )}
      {showDropdown && searchQuery.trim() && searchResults.length === 0 && (
        <div className="absolute left-0 top-full z-50 mt-1 w-56 rounded border border-gray-200 bg-white py-2 px-3 text-xs text-gray-400 shadow-lg">
          No members found
        </div>
      )}
    </div>
  );
}

function FilesCellEditor({
  value,
  onChange,
  workspaceId,
}: {
  value: any;
  onChange: (value: any) => void;
  workspaceId?: number;
}) {
  const files: { url: string; name: string }[] = value?.files ?? [];
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setError(null);
    setUploading(true);

    try {
      const body: Record<string, unknown> = {
        filename: file.name,
        content_type: file.type || "application/octet-stream",
      };
      if (workspaceId) {
        body.workspace_id = workspaceId;
      }

      const data = await api.post<{ upload_url: string; public_url: string }>(
        "/files/upload-url",
        body
      );

      const putResponse = await fetch(data.upload_url, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type || "application/octet-stream" },
      });

      if (!putResponse.ok) {
        throw new Error(`Upload failed with status ${putResponse.status}`);
      }

      const newFile = { url: data.public_url, name: file.name };
      onChange({ files: [...files, newFile] });
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Upload failed";
      setError(message);
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  const removeFile = (index: number) => {
    const updated = files.filter((_, i) => i !== index);
    onChange({ files: updated.length > 0 ? updated : [] });
  };

  return (
    <div className="flex flex-col gap-1 py-0.5">
      {files.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {files.map((file, i) => (
            <span
              key={i}
              className="inline-flex items-center gap-1 rounded bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700"
            >
              {file.name || file.url}
              <button
                onClick={() => removeFile(i)}
                className="ml-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full text-blue-500 hover:bg-blue-200 hover:text-blue-800 text-xs leading-none"
                title="Remove"
              >
                &times;
              </button>
            </span>
          ))}
        </div>
      )}
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileSelect}
        disabled={uploading}
      />
      <button
        type="button"
        className="rounded px-2 py-0.5 text-xs text-gray-400 hover:text-gray-600 disabled:opacity-50 text-left"
        onClick={() => fileInputRef.current?.click()}
        disabled={uploading}
      >
        {uploading ? "Uploading..." : "+ Add file"}
      </button>
      {error && <span className="text-xs text-red-500">{error}</span>}
    </div>
  );
}

export default function CellEditor({
  property,
  value,
  onChange,
  readOnly,
  workspaceId,
}: CellEditorProps) {
  const [editing, setEditing] = useState(false);
  const [editText, setEditText] = useState("");

  if (readOnly) {
    return (
      <div className="text-sm text-gray-700 px-1 py-0.5 min-h-[1.5rem]">
        {displayValue(value, property) || "—"}
      </div>
    );
  }

  // ---------- select / status: dropdown ----------
  if (property.type === "select" || property.type === "status") {
    const options = property.config?.options ?? [];
    const currentId = value?.select ?? "";

    return (
      <select
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400 bg-white"
        value={currentId}
        onChange={(e) => {
          const id = e.target.value;
          onChange(id ? { select: id } : { select: null });
        }}
      >
        <option value="">--</option>
        {options.map((opt: SelectOption) => (
          <option key={opt.id} value={opt.id}>
            {opt.name}
          </option>
        ))}
      </select>
    );
  }

  // ---------- checkbox: inline toggle ----------
  if (property.type === "checkbox") {
    const checked = value?.checkbox === true;

    return (
      <label className="inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange({ checkbox: e.target.checked })}
          className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
        />
      </label>
    );
  }

  // ---------- date: date picker ----------
  if (property.type === "date") {
    const currentDate = value?.date ?? "";

    return (
      <input
        type="date"
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
        value={currentDate}
        onChange={(e) => onChange({ date: e.target.value })}
      />
    );
  }

  // ---------- relation: multi-record picker ----------
  if (property.type === "relation") {
    const targetDbId = property.config?.database_id;
    if (!targetDbId) {
      return (
        <div className="text-sm text-gray-400 px-1 py-0.5">No target database configured</div>
      );
    }
    const ids: number[] = value?.relation ?? [];
    return (
      <RelationPicker
        databaseId={targetDbId}
        value={ids}
        onChange={onChange}
      />
    );
  }

  // ---------- person: member picker ----------
  if (property.type === "person") {
    return (
      <PersonCellEditor
        value={value}
        onChange={onChange}
        workspaceId={workspaceId}
      />
    );
  }

  // ---------- files: multi-file upload ----------
  if (property.type === "files") {
    return (
      <FilesCellEditor
        value={value}
        onChange={onChange}
        workspaceId={workspaceId}
      />
    );
  }

  // ---------- created_time / last_edited_time / rollup / formula: read-only ----------
  if (property.type === "created_time" || property.type === "last_edited_time" || property.type === "rollup" || property.type === "formula") {
    return (
      <div className="text-sm text-gray-400 px-1 py-0.5 select-none">
        {displayValue(value, property) || "—"}
      </div>
    );
  }

  // ---------- multi_select: click to edit text ----------
  if (property.type === "multi_select") {
    const ids: string[] = value?.multi_select ?? [];
    const label =
      ids.length === 0
        ? ""
        : `${ids.length} selected`;

    if (!editing) {
      return (
        <div
          className="text-sm text-gray-700 px-1 py-0.5 min-h-[1.5rem] cursor-pointer rounded hover:bg-blue-50"
          onClick={() => {
            setEditText((value?.multi_select ?? []).join(", "));
            setEditing(true);
          }}
        >
          {label || "—"}
        </div>
      );
    }

    return (
      <input
        autoFocus
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
        value={editText}
        onChange={(e) => setEditText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === "Tab") {
            e.preventDefault();
            const ids = editText.split(",").map((s) => s.trim()).filter(Boolean);
            onChange({ multi_select: ids });
            setEditing(false);
          }
          if (e.key === "Escape") {
            setEditing(false);
          }
        }}
        onBlur={() => {
          const ids = editText.split(",").map((s) => s.trim()).filter(Boolean);
          onChange({ multi_select: ids });
          setEditing(false);
        }}
      />
    );
  }

  // ---------- number: number input ----------
  if (property.type === "number") {
    if (!editing) {
      return (
        <div
          className="text-sm text-gray-700 px-1 py-0.5 min-h-[1.5rem] cursor-pointer rounded hover:bg-blue-50"
          onClick={() => {
            setEditText(editInitialValue(value, property));
            setEditing(true);
          }}
        >
          {displayValue(value, property) || "—"}
        </div>
      );
    }

    return (
      <input
        autoFocus
        type="number"
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
        value={editText}
        onChange={(e) => setEditText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === "Tab") {
            e.preventDefault();
            onChange(buildCommitValue(editText, property));
            setEditing(false);
          }
          if (e.key === "Escape") {
            setEditing(false);
          }
        }}
        onBlur={() => {
          onChange(buildCommitValue(editText, property));
          setEditing(false);
        }}
      />
    );
  }

  // ---------- title / text / url / email / phone: text input ----------
  if (!editing) {
    return (
      <div
        className="text-sm text-gray-700 px-1 py-0.5 min-h-[1.5rem] cursor-pointer rounded hover:bg-blue-50"
        onClick={() => {
          setEditText(editInitialValue(value, property));
          setEditing(true);
        }}
      >
        {displayValue(value, property) || "—"}
      </div>
    );
  }

  return (
    <input
      autoFocus
      type="text"
      className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
      value={editText}
      onChange={(e) => setEditText(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === "Tab") {
          e.preventDefault();
          onChange(buildCommitValue(editText, property));
          setEditing(false);
        }
        if (e.key === "Escape") {
          setEditing(false);
        }
      }}
      onBlur={() => {
        onChange(buildCommitValue(editText, property));
        setEditing(false);
      }}
    />
  );
}
