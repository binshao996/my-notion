import { useNavigate } from "react-router-dom";
import type { Property, Record as DatabaseRecord, View } from "../../types/database";

interface GalleryViewProps {
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

export default function GalleryView({
  properties,
  records,
  titlePropertyId,
}: GalleryViewProps) {
  const navigate = useNavigate();

  // Determine which properties to show on cards (exclude title property, show up to 3 others)
  const visibleProps = properties.filter(
    (p) => p.id !== titlePropertyId && p.type !== "title"
  );
  const previewProps = visibleProps.slice(0, 3);

  return (
    <div className="p-4">
      <div className="grid grid-cols-3 gap-4">
        {records.map((record) => {
          return (
            <div
              key={record.id}
              className="rounded-lg border border-gray-200 overflow-hidden hover:shadow-md cursor-pointer"
              onClick={() => navigate(`/record/${record.id}`)}
            >
              {/* Cover image placeholder */}
              <div className="h-32 bg-gray-100 flex items-center justify-center">
                <span className="text-gray-300 text-2xl">&#9632;</span>
              </div>

              {/* Card body */}
              <div className="p-4">
                {/* Title */}
                <div className="font-medium text-sm text-gray-900 mb-2">
                  {getRecordTitle(record, titlePropertyId)}
                </div>

                {/* Property values */}
                {previewProps.length > 0 && (
                  <div className="space-y-1">
                    {previewProps.map((prop) => {
                      const val = renderPropertyValue(record, prop);
                      if (!val) return null;
                      return (
                        <div key={prop.id} className="flex gap-2 text-xs">
                          <span className="text-gray-400 flex-shrink-0">{prop.name}</span>
                          <span className="text-gray-600 truncate">{val}</span>
                        </div>
                      );
                    })}
                  </div>
                )}

                {/* Fallback if no visible properties */}
                {previewProps.length === 0 && (
                  <div className="text-xs text-gray-300">No properties</div>
                )}
              </div>
            </div>
          );
        })}

        {records.length === 0 && (
          <div className="col-span-full flex items-center justify-center py-12">
            <div className="text-sm text-gray-400">No records yet</div>
          </div>
        )}
      </div>
    </div>
  );
}
