import { useState } from "react";
import type { Property, SelectOption } from "../../types/database";
import RelationPicker from "./RelationPicker";

interface CellEditorProps {
  property: Property;
  value: any;
  onChange: (value: any) => void;
  readOnly?: boolean;
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
    case "person":
    case "files":
      return "—";
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
    case "created_time":
    case "last_edited_time":
      return null;
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

export default function CellEditor({
  property,
  value,
  onChange,
  readOnly,
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

  // ---------- person / files: read-only for now ----------
  if (property.type === "person" || property.type === "files" || property.type === "created_time" || property.type === "last_edited_time" || property.type === "rollup" || property.type === "formula") {
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
