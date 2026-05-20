import { useCallback } from "react";
import type { Property, GroupConfig as GroupConfigType } from "../../types/database";

interface GroupConfigProps {
  properties: Property[];
  groupBy?: GroupConfigType;
  onChange: (group: GroupConfigType | undefined) => void;
}

/**
 * Only certain property types support grouping.
 */
const GROUPABLE_TYPES = new Set(["select", "status", "date"]);

export default function GroupConfig({
  properties,
  groupBy,
  onChange,
}: GroupConfigProps) {
  const groupableProps = properties.filter((p) => GROUPABLE_TYPES.has(p.type));

  const handleSelect = useCallback(
    (propertyId: number | null) => {
      if (propertyId === null) {
        onChange(undefined);
      } else {
        onChange({ property_id: propertyId });
      }
    },
    [onChange]
  );

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-semibold text-gray-700">Group</span>
        {groupBy && (
          <button
            onClick={() => onChange(undefined)}
            className="text-xs text-gray-400 hover:text-red-500"
          >
            Clear
          </button>
        )}
      </div>

      {groupableProps.length === 0 && (
        <div className="text-xs text-gray-400 py-1">
          No groupable properties (Select, Status, or Date).
        </div>
      )}

      {/* None option */}
      <button
        onClick={() => handleSelect(null)}
        className={`w-full rounded px-3 py-1.5 text-left text-xs mb-1 ${
          !groupBy
            ? "bg-blue-50 text-blue-700"
            : "text-gray-600 hover:bg-gray-100"
        }`}
      >
        None
      </button>

      {/* Groupable properties */}
      {groupableProps.map((prop) => {
        const isSelected = groupBy?.property_id === prop.id;
        return (
          <button
            key={prop.id}
            onClick={() => handleSelect(prop.id)}
            className={`w-full rounded px-3 py-1.5 text-left text-xs ${
              isSelected
                ? "bg-blue-50 text-blue-700"
                : "text-gray-600 hover:bg-gray-100"
            }`}
          >
            {prop.name}
            <span className="ml-2 text-gray-400">({prop.type})</span>
          </button>
        );
      })}
    </div>
  );
}
