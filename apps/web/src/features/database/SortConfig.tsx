import { useCallback } from "react";
import type { Property, SortConfig as SortConfigType } from "../../types/database";

interface SortConfigProps {
  properties: Property[];
  sorts: SortConfigType[];
  onChange: (sorts: SortConfigType[]) => void;
}

function emptySort(): SortConfigType {
  return { property_id: 0, direction: "asc" };
}

export default function SortConfig({
  properties,
  sorts,
  onChange,
}: SortConfigProps) {
  const handleAdd = useCallback(() => {
    onChange([...sorts, emptySort()]);
  }, [sorts, onChange]);

  const handleRemove = useCallback(
    (index: number) => {
      const next = sorts.filter((_, i) => i !== index);
      onChange(next);
    },
    [sorts, onChange]
  );

  const handleUpdate = useCallback(
    (index: number, updates: Partial<SortConfigType>) => {
      const next = sorts.map((s, i) => (i === index ? { ...s, ...updates } : s));
      onChange(next);
    },
    [sorts, onChange]
  );

  const handleClearAll = useCallback(() => {
    onChange([]);
  }, [onChange]);

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-semibold text-gray-700">Sort</span>
        {sorts.length > 0 && (
          <button
            onClick={handleClearAll}
            className="text-xs text-gray-400 hover:text-red-500"
          >
            Clear all
          </button>
        )}
      </div>

      {/* Sort rows */}
      {sorts.map((sort, i) => (
        <div key={i} className="flex items-center gap-2 mb-2">
          {/* Property dropdown */}
          <select
            className="rounded border border-gray-200 px-2 py-1 text-xs text-gray-700 outline-none"
            value={sort.property_id || ""}
            onChange={(e) =>
              handleUpdate(i, { property_id: Number(e.target.value) || 0 })
            }
          >
            <option value="" disabled>
              Property
            </option>
            {properties.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>

          {/* Direction toggle */}
          <button
            onClick={() =>
              handleUpdate(i, {
                direction: sort.direction === "asc" ? "desc" : "asc",
              })
            }
            className="rounded border border-gray-200 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50"
          >
            {sort.direction === "asc" ? "Asc" : "Desc"}
          </button>

          {/* Remove button */}
          <button
            onClick={() => handleRemove(i)}
            className="text-xs text-gray-400 hover:text-red-500"
            title="Remove sort"
          >
            &times;
          </button>
        </div>
      ))}

      {/* Add sort button */}
      <button
        onClick={handleAdd}
        className="text-xs text-blue-600 hover:underline"
      >
        + Add sort
      </button>
    </div>
  );
}
