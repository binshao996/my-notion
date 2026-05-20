import { useNavigate } from "react-router-dom";
import type { Property, Record as DatabaseRecord, View } from "../../types/database";

interface BoardViewProps {
  properties: Property[];
  records: DatabaseRecord[];
  activeView?: View;
  titlePropertyId?: number;
  onUpdateRecord: (recordId: number, propertyId: number, value: any) => Promise<void>;
}

const optionColors: { [key: string]: string } = {
  red: "bg-red-100 text-red-700",
  green: "bg-green-100 text-green-700",
  blue: "bg-blue-100 text-blue-700",
  yellow: "bg-yellow-100 text-yellow-700",
  purple: "bg-purple-100 text-purple-700",
  gray: "bg-gray-100 text-gray-700",
  default: "bg-gray-100 text-gray-700",
};

function getOptionColor(color?: string): string {
  return optionColors[color || ""] || optionColors.default;
}

/**
 * Extract the display value for a record's title property.
 */
function getRecordTitle(
  record: DatabaseRecord,
  titlePropertyId?: number
): string {
  if (!titlePropertyId) return `Record #${record.id}`;
  const pv = record.property_values?.find((v) => v.property_id === titlePropertyId);
  if (!pv) return `Record #${record.id}`;
  const val = typeof pv.value === "string" ? pv.value : pv.value?.text ?? pv.value?.title ?? "";
  return String(val) || `Record #${record.id}`;
}

/**
 * Group records by a select/status property value.
 * Records without a value for the groupBy property go into "No Status".
 */
function groupRecords(
  records: DatabaseRecord[],
  groupByProperty: Property
): Map<string, DatabaseRecord[]> {
  const groups = new Map<string, DatabaseRecord[]>();

  for (const record of records) {
    const pv = record.property_values?.find((v) => v.property_id === groupByProperty.id);
    let key: string;
    if (!pv || pv.value == null || pv.value === "") {
      key = "No Status";
    } else {
      const val = typeof pv.value === "string" ? pv.value : pv.value.select ?? "";
      // Look up the option name
      const option = groupByProperty.config?.options?.find(
        (o) => o.id === val || o.name === val
      );
      key = option?.name ?? (String(val) || "No Status");
    }

    const existing = groups.get(key);
    if (existing) {
      existing.push(record);
    } else {
      groups.set(key, [record]);
    }
  }

  return groups;
}

export default function BoardView({
  properties,
  records,
  activeView,
  titlePropertyId,
  onUpdateRecord,
}: BoardViewProps) {
  const navigate = useNavigate();
  const groupByPropertyId = activeView?.config?.groupBy?.property_id;

  const groupByProperty = groupByPropertyId
    ? properties.find((p) => p.id === groupByPropertyId)
    : undefined;

  // If no groupBy set, show guidance message
  if (!groupByProperty) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-gray-400">Select a property to group by</div>
      </div>
    );
  }

  // Verify the property type is groupable (select, status)
  if (groupByProperty.type !== "select" && groupByProperty.type !== "status") {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-gray-400">
          Group by property must be a Select or Status type
        </div>
      </div>
    );
  }

  const groups = groupRecords(records, groupByProperty);

  // Collect all options from the groupBy property for ordering
  const options = groupByProperty.config?.options || [];

  // Build ordered list of group keys (follow option order, then "No Status" last)
  const orderedKeys: string[] = [];
  for (const opt of options) {
    if (groups.has(opt.name)) {
      orderedKeys.push(opt.name);
    }
  }
  // Add any keys not matching an option
  for (const key of groups.keys()) {
    if (!orderedKeys.includes(key)) {
      orderedKeys.push(key);
    }
  }
  // Ensure "No Status" is last
  if (orderedKeys.includes("No Status")) {
    orderedKeys.splice(orderedKeys.indexOf("No Status"), 1);
    orderedKeys.push("No Status");
  }

  return (
    <div className="flex gap-4 overflow-x-auto p-4 h-full">
      {orderedKeys.map((groupKey) => {
        const groupRecords = groups.get(groupKey) || [];
        const option = options.find((o) => o.name === groupKey);

        return (
          <div
            key={groupKey}
            className="w-64 flex-shrink-0 rounded-lg bg-gray-100 p-3"
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              const recordId = e.dataTransfer.getData("recordId");
              const id = Number(recordId);
              if (!id || !groupByPropertyId) return;

              // Find the option id for this group
              const opt = groupByProperty.config?.options?.find(
                (o) => o.name === groupKey
              );
              const value = opt ? { select: opt.id } : { select: groupKey };
              onUpdateRecord(id, groupByPropertyId, value);
            }}
          >
            {/* Column header */}
            <div className="flex items-center gap-2 mb-3">
              {option && (
                <span
                  className={`inline-block h-2.5 w-2.5 rounded-full ${getOptionColor(option.color).split(" ")[0]}`}
                />
              )}
              <span className="text-xs font-medium text-gray-700">{groupKey}</span>
              <span className="text-xs text-gray-400">{groupRecords.length}</span>
            </div>

            {/* Cards */}
            {groupRecords.map((record) => (
              <div
                key={record.id}
                className="rounded border border-gray-200 bg-white p-3 mb-2 shadow-sm cursor-pointer hover:shadow-md"
                draggable
                onDragStart={(e) => {
                  e.dataTransfer.setData("recordId", String(record.id));
                }}
                onClick={() => navigate(`/record/${record.id}`)}
              >
                <span className="text-sm text-gray-900">
                  {getRecordTitle(record, titlePropertyId)}
                </span>
              </div>
            ))}

            {groupRecords.length === 0 && (
              <div className="text-xs text-gray-300 text-center py-4">
                No records
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
