import { useNavigate } from "react-router-dom";
import type { Property, Record as DatabaseRecord, View } from "../../types/database";

interface ListViewProps {
  properties: Property[];
  records: DatabaseRecord[];
  activeView?: View;
  titlePropertyId?: number;
}

function getRecordTitle(record: DatabaseRecord, titlePropertyId?: number): string {
  if (!titlePropertyId) return `Record #${record.id}`;
  const pv = record.property_values?.find((v) => v.property_id === titlePropertyId);
  if (!pv) return `Record #${record.id}`;
  const val = typeof pv.value === "string" ? pv.value : pv.value?.text ?? pv.value?.title ?? "";
  return String(val) || `Record #${record.id}`;
}

function renderPropertyValue(record: DatabaseRecord, prop: Property): string {
  const pv = record.property_values?.find((v) => v.property_id === prop.id);
  if (!pv) return "";
  const val = pv.value;
  if (val == null) return "";

  switch (prop.type) {
    case "title":
    case "text":
      return typeof val === "string" ? val : val.text ?? val.title ?? "";
    case "number":
      return String(val.number ?? val);
    case "select":
    case "status": {
      const selectVal = typeof val === "string" ? val : val.select;
      if (!selectVal) return "";
      const opt = prop.config?.options?.find((o) => o.id === selectVal || o.name === selectVal);
      return opt?.name ?? selectVal;
    }
    case "date":
      return typeof val === "string" ? val : val.date ?? "";
    case "checkbox":
      return val.checked || val === true ? "Yes" : "No";
    case "url":
      return typeof val === "string" ? val : val.url ?? "";
    case "email":
      return typeof val === "string" ? val : val.email ?? "";
    case "phone":
      return typeof val === "string" ? val : val.phone ?? "";
    default:
      return "";
  }
}

export default function ListView({
  properties,
  records,
  activeView,
  titlePropertyId,
}: ListViewProps) {
  const navigate = useNavigate();

  // Visible properties: exclude title and hidden (from active view config)
  const hiddenIds = activeView?.config?.hidden_properties || [];
  const visibleProps = properties.filter(
    (p) => p.id !== titlePropertyId && p.type !== "title" && !hiddenIds.includes(p.id)
  );
  const previewProps = visibleProps.slice(0, 3);

  return (
    <div className="flex flex-col">
      {records.map((record) => (
        <div
          key={record.id}
          className="flex items-center gap-4 px-4 py-2 border-b border-gray-100 hover:bg-gray-50 cursor-pointer"
          onClick={() => navigate(`/record/${record.id}`)}
        >
          {/* Title */}
          <span className="font-medium text-sm text-gray-900 flex-shrink-0 w-48 truncate">
            {getRecordTitle(record, titlePropertyId)}
          </span>

          {/* Inline property values */}
          {previewProps.map((prop) => {
            const val = renderPropertyValue(record, prop);
            if (!val) return null;
            return (
              <span key={prop.id} className="text-xs text-gray-500 truncate">
                {val}
              </span>
            );
          })}
        </div>
      ))}

      {records.length === 0 && (
        <div className="flex items-center justify-center py-12">
          <div className="text-sm text-gray-400">No records yet</div>
        </div>
      )}
    </div>
  );
}
