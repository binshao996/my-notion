import { useState, useEffect } from "react";
import { api } from "../../lib/api";
import type { Record as DatabaseRecord } from "../../types/database";

interface RelationPickerProps {
  databaseId: number;       // target database to search
  value: number[];           // selected record IDs
  onChange: (value: any) => void;  // call with {"relation": [...]}
}

export default function RelationPicker({ databaseId, value, onChange }: RelationPickerProps) {
  const [records, setRecords] = useState<DatabaseRecord[]>([]);
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    // Load records from the target database
    setLoading(true);
    api.get<{ records: DatabaseRecord[] }>(`/databases/${databaseId}/records?limit=200`)
      .then(data => setRecords(data.records || []))
      .catch(() => setRecords([]))
      .finally(() => setLoading(false));
  }, [databaseId]);

  const selected = value || [];
  const filtered = records.filter(r => {
    if (!search) return true;
    // Search in property values for title-like text
    const title = getRecordTitle(r);
    return title.toLowerCase().includes(search.toLowerCase());
  });

  const toggle = (recordId: number) => {
    if (selected.includes(recordId)) {
      onChange({ relation: selected.filter(id => id !== recordId) });
    } else {
      onChange({ relation: [...selected, recordId] });
    }
  };

  const removeId = (recordId: number) => {
    onChange({ relation: selected.filter(id => id !== recordId) });
  };

  return (
    <div className="relative">
      {/* Selected chips */}
      <div className="flex flex-wrap gap-1 mb-1">
        {selected.map(id => {
          const rec = records.find(r => r.id === id);
          return (
            <span key={id} className="inline-flex items-center gap-1 rounded bg-blue-100 px-2 py-0.5 text-xs text-blue-700">
              {rec ? getRecordTitle(rec) : `Record #${id}`}
              <button className="text-blue-400 hover:text-blue-600" onClick={() => removeId(id)}>×</button>
            </span>
          );
        })}
      </div>

      {/* Search input */}
      <input
        type="text"
        className="w-full rounded border border-gray-200 px-2 py-1 text-sm outline-none focus:border-blue-400"
        placeholder="Search records..."
        value={search}
        onChange={e => { setSearch(e.target.value); setOpen(true); }}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 200)}
      />

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1 max-h-48 w-full overflow-y-auto rounded border border-gray-200 bg-white shadow-lg">
          {loading && <div className="px-3 py-2 text-xs text-gray-400">Loading...</div>}
          {!loading && filtered.length === 0 && (
            <div className="px-3 py-2 text-xs text-gray-400">No records found</div>
          )}
          {filtered.map(rec => {
            const isSelected = selected.includes(rec.id);
            return (
              <button
                key={rec.id}
                className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-gray-50 ${isSelected ? "bg-blue-50" : ""}`}
                onMouseDown={e => { e.preventDefault(); toggle(rec.id); }}
              >
                <input type="checkbox" checked={isSelected} readOnly className="h-3.5 w-3.5" />
                <span>{getRecordTitle(rec)}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

/** Extract a display title from a record's property values */
function getRecordTitle(record: DatabaseRecord): string {
  const pvs = record.property_values || [];
  if (pvs.length === 0) return `Record #${record.id}`;
  // Use the first text-like value as title
  for (const pv of pvs) {
    if (pv.value && typeof pv.value === "object" && pv.value.text) {
      return pv.value.text;
    }
  }
  return `Record #${record.id}`;
}
