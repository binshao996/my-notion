import { useState } from "react";
import { useNavigate } from "react-router-dom";
import type {
  Property,
  View,
  Record as DatabaseRecord,
} from "../../types/database";
import CellEditor from "./CellEditor";

interface TableViewProps {
  properties: Property[];
  records: DatabaseRecord[];
  activeView?: View;
  onUpdateRecord: (
    recordId: number,
    propertyId: number,
    value: any
  ) => Promise<void>;
  onDeleteRecord: (recordId: number) => Promise<void>;
  onCreateRecord: () => Promise<void>;
  titlePropertyId?: number;
}

function getRecordValue(record: DatabaseRecord, propertyId: number): any {
  const pv = record.property_values?.find((v) => v.property_id === propertyId);
  if (!pv) return null;
  return pv.value;
}

type SortDirection = "asc" | "desc" | null;

export default function TableView({
  properties,
  records,
  activeView,
  onUpdateRecord,
  onDeleteRecord,
  onCreateRecord,
  titlePropertyId,
}: TableViewProps) {
  const navigate = useNavigate();
  const [sortPropId, setSortPropId] = useState<number | null>(
    activeView?.config?.sorts?.[0]?.property_id ?? null
  );
  const [sortDir, setSortDir] = useState<SortDirection>(
    activeView?.config?.sorts?.[0]?.direction ?? null
  );
  const [hoveredRecordId, setHoveredRecordId] = useState<number | null>(null);

  // Filter out hidden properties
  const hiddenIds = activeView?.config?.hidden_properties ?? [];
  const visibleProperties = properties.filter((p) => !hiddenIds.includes(p.id));

  // Sort records
  const sortedRecords = [...records].sort((a, b) => {
    if (!sortPropId || !sortDir) return 0;
    const aVal = getRecordValue(a, sortPropId);
    const bVal = getRecordValue(b, sortPropId);

    const aText = aVal?.text ?? aVal?.number ?? aVal?.select ?? aVal?.date ?? "";
    const bText = bVal?.text ?? bVal?.number ?? bVal?.select ?? bVal?.date ?? "";

    if (aText < bText) return sortDir === "asc" ? -1 : 1;
    if (aText > bText) return sortDir === "asc" ? 1 : -1;
    return 0;
  });

  const handleSortClick = (propertyId: number) => {
    if (sortPropId === propertyId) {
      // Cycle: asc -> desc -> none
      if (sortDir === "asc") {
        setSortDir("desc");
      } else if (sortDir === "desc") {
        setSortDir(null);
        setSortPropId(null);
      }
    } else {
      setSortPropId(propertyId);
      setSortDir("asc");
    }
  };

  const sortIndicator = (propertyId: number) => {
    if (sortPropId !== propertyId) return null;
    return <span className="ml-1 text-gray-400">{sortDir === "asc" ? "▲" : "▼"}</span>;
  };

  return (
    <div className="overflow-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="border-b border-gray-200">
            {visibleProperties.map((prop) => (
              <th
                key={prop.id}
                className="sticky top-0 bg-gray-50 px-3 py-2 text-left text-xs font-medium text-gray-500 cursor-pointer select-none hover:bg-gray-100"
                onClick={() => handleSortClick(prop.id)}
              >
                <span className="inline-flex items-center">
                  {prop.name}
                  {sortIndicator(prop.id)}
                </span>
              </th>
            ))}
            <th className="sticky top-0 bg-gray-50 w-10 px-2 py-2" />
          </tr>
        </thead>

        <tbody>
          {sortedRecords.map((record) => {
            const isHovered = hoveredRecordId === record.id;

            return (
              <tr
                key={record.id}
                className="border-b border-gray-100 hover:bg-gray-50"
                onMouseEnter={() => setHoveredRecordId(record.id)}
                onMouseLeave={() => setHoveredRecordId(null)}
              >
                {visibleProperties.map((prop) => {
                  const rawValue = getRecordValue(record, prop.id);

                  // Title column: render as link to page
                  if (prop.id === titlePropertyId) {
                    return (
                      <td key={prop.id} className="px-3 py-1.5">
                        <button
                          onClick={() => navigate(`/page/${record.page_id}`)}
                          className="text-sm text-gray-900 hover:text-blue-600 hover:underline text-left"
                        >
                          {rawValue?.text || rawValue?.title || "Untitled"}
                        </button>
                      </td>
                    );
                  }

                  return (
                    <td key={prop.id} className="px-3 py-1.5">
                      <CellEditor
                        property={prop}
                        value={rawValue}
                        onChange={(value) =>
                          onUpdateRecord(record.id, prop.id, value)
                        }
                      />
                    </td>
                  );
                })}

                {/* Hover action: delete */}
                <td className="px-2 py-1.5">
                  {isHovered && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteRecord(record.id);
                      }}
                      className="rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-red-500"
                      title="Delete record"
                    >
                      <svg
                        className="h-3.5 w-3.5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </button>
                  )}
                </td>
              </tr>
            );
          })}

          {/* Empty state */}
          {records.length === 0 && (
            <tr>
              <td
                colSpan={visibleProperties.length + 1}
                className="px-6 py-8 text-center text-sm text-gray-400"
              >
                No records yet. Click "New Record" to add one.
              </td>
            </tr>
          )}
        </tbody>

        {/* New record row */}
        <tfoot>
          <tr className="border-t border-gray-200">
            <td
              colSpan={visibleProperties.length + 1}
              className="px-6 py-2"
            >
              <button
                onClick={onCreateRecord}
                className="text-sm text-gray-400 hover:text-gray-600"
              >
                + New
              </button>
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
  );
}
