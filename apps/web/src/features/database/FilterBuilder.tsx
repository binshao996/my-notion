import { useState, useCallback } from "react";
import type {
  Property,
  FilterGroup,
  FilterCondition,
  FilterOperator,
} from "../../types/database";

interface FilterBuilderProps {
  properties: Property[];
  filterGroup?: FilterGroup;
  onChange: (filters: FilterGroup | undefined) => void;
}

type FilterMode = "and" | "or";

const ALL_OPERATORS: { value: FilterOperator; label: string; needsValue: boolean }[] = [
  { value: "equals", label: "Equals", needsValue: true },
  { value: "not_equals", label: "Not equals", needsValue: true },
  { value: "contains", label: "Contains", needsValue: true },
  { value: "not_contains", label: "Does not contain", needsValue: true },
  { value: "starts_with", label: "Starts with", needsValue: true },
  { value: "ends_with", label: "Ends with", needsValue: true },
  { value: "greater_than", label: "Greater than", needsValue: true },
  { value: "less_than", label: "Less than", needsValue: true },
  { value: "greater_than_or_equal", label: "Greater or equal", needsValue: true },
  { value: "less_than_or_equal", label: "Less or equal", needsValue: true },
  { value: "is_empty", label: "Is empty", needsValue: false },
  { value: "is_not_empty", label: "Is not empty", needsValue: false },
];

function emptyCondition(): FilterCondition {
  return { property_id: 0, operator: "equals", value: "" };
}

/**
 * Update a single filter condition within a group.
 */
function updateConditionInGroup(
  group: FilterGroup,
  mode: FilterMode,
  index: number,
  updates: Partial<FilterCondition>
): FilterGroup {
  const conditions = [...(group[mode] || [])];
  conditions[index] = { ...conditions[index], ...updates };
  return { [mode]: conditions };
}

/**
 * Remove a condition from a group. If the group is now empty, return undefined.
 */
function removeConditionFromGroup(
  group: FilterGroup,
  mode: FilterMode,
  index: number
): FilterGroup | undefined {
  const conditions = (group[mode] || []).filter((_, i) => i !== index);
  if (conditions.length === 0) {
    // Remove this mode key entirely
    const trimmed = { ...group };
    delete trimmed[mode];
    if (Object.keys(trimmed).length === 0) return undefined;
    return trimmed;
  }
  return { ...group, [mode]: conditions };
}

export default function FilterBuilder({
  properties,
  filterGroup,
  onChange,
}: FilterBuilderProps) {
  // Determine current mode from existing filterGroup
  const currentMode: FilterMode = filterGroup?.and ? "and" : "or";
  const conditions: FilterCondition[] = filterGroup?.[currentMode] || [];
  const [mode, setMode] = useState<FilterMode>(currentMode);

  const handleAdd = useCallback(() => {
    const next = [...conditions, emptyCondition()];
    onChange({ [mode]: next });
  }, [conditions, mode, onChange]);

  const handleRemove = useCallback(
    (index: number) => {
      const result = removeConditionFromGroup(
        filterGroup || { [mode]: conditions },
        mode,
        index
      );
      onChange(result);
    },
    [filterGroup, mode, conditions, onChange]
  );

  const handleUpdate = useCallback(
    (index: number, updates: Partial<FilterCondition>) => {
      const result = updateConditionInGroup(
        filterGroup || { [mode]: conditions },
        mode,
        index,
        updates
      );
      onChange(result);
    },
    [filterGroup, mode, conditions, onChange]
  );

  const handleModeChange = useCallback(
    (newMode: FilterMode) => {
      setMode(newMode);
      if (conditions.length === 0) {
        onChange({ [newMode]: [] });
      } else {
        onChange({ [newMode]: [...conditions] });
      }
    },
    [conditions, onChange]
  );

  const handleClearAll = useCallback(() => {
    onChange(undefined);
  }, [onChange]);

  // Find the operator metadata for a given condition
  const getOperatorMeta = (op: FilterOperator) =>
    ALL_OPERATORS.find((o) => o.value === op) || ALL_OPERATORS[0];

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-semibold text-gray-700">Filters</span>
        {conditions.length > 0 && (
          <button
            onClick={handleClearAll}
            className="text-xs text-gray-400 hover:text-red-500"
          >
            Clear all
          </button>
        )}
      </div>

      {/* Mode toggle: AND / OR */}
      {conditions.length > 0 && (
        <div className="flex gap-1 mb-3">
          <button
            onClick={() => handleModeChange("and")}
            className={`rounded px-2 py-0.5 text-xs ${
              mode === "and"
                ? "bg-blue-100 text-blue-700"
                : "text-gray-500 hover:bg-gray-100"
            }`}
          >
            AND
          </button>
          <button
            onClick={() => handleModeChange("or")}
            className={`rounded px-2 py-0.5 text-xs ${
              mode === "or"
                ? "bg-blue-100 text-blue-700"
                : "text-gray-500 hover:bg-gray-100"
            }`}
          >
            OR
          </button>
        </div>
      )}

      {/* Filter rows */}
      {conditions.map((cond, i) => {
        const opMeta = getOperatorMeta(cond.operator);

        return (
          <div key={i} className="flex items-center gap-2 mb-2">
            {/* Property dropdown */}
            <select
              className="rounded border border-gray-200 px-2 py-1 text-xs text-gray-700 outline-none"
              value={cond.property_id || ""}
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

            {/* Operator dropdown */}
            <select
              className="rounded border border-gray-200 px-2 py-1 text-xs text-gray-700 outline-none"
              value={cond.operator}
              onChange={(e) =>
                handleUpdate(i, {
                  operator: e.target.value as FilterOperator,
                  ...(getOperatorMeta(e.target.value as FilterOperator).needsValue
                    ? {}
                    : { value: undefined }),
                })
              }
            >
              {ALL_OPERATORS.map((op) => (
                <option key={op.value} value={op.value}>
                  {op.label}
                </option>
              ))}
            </select>

            {/* Value input (only for operators that need it) */}
            {opMeta.needsValue && (
              <input
                className="flex-1 rounded border border-gray-200 px-2 py-1 text-xs text-gray-700 outline-none"
                placeholder="Value"
                value={cond.value ?? ""}
                onChange={(e) => handleUpdate(i, { value: e.target.value })}
              />
            )}

            {/* Remove button */}
            <button
              onClick={() => handleRemove(i)}
              className="text-xs text-gray-400 hover:text-red-500"
              title="Remove filter"
            >
              &times;
            </button>
          </div>
        );
      })}

      {/* Add filter button */}
      <button
        onClick={handleAdd}
        className="text-xs text-blue-600 hover:underline"
      >
        + Add filter
      </button>
    </div>
  );
}
