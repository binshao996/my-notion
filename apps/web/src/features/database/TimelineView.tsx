import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import type { Property, Record as DatabaseRecord, View } from "../../types/database";

interface TimelineViewProps {
  properties: Property[];
  records: DatabaseRecord[];
  activeView?: View;
  titlePropertyId?: number;
  onUpdateRecord: (recordId: number, propertyId: number, value: any) => Promise<void>;
}

type ZoomLevel = "week" | "month" | "quarter";

const MONTH_NAMES = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];

const BAR_COLORS = [
  "bg-blue-400", "bg-green-400", "bg-purple-400", "bg-orange-400",
  "bg-pink-400", "bg-teal-400", "bg-yellow-400", "bg-red-400",
];

function getRecordTitle(record: DatabaseRecord, titlePropertyId?: number): string {
  if (!titlePropertyId) return `Record #${record.id}`;
  const pv = record.property_values?.find((v) => v.property_id === titlePropertyId);
  if (!pv) return `Record #${record.id}`;
  const val = typeof pv.value === "string" ? pv.value : pv.value?.text ?? pv.value?.title ?? "";
  return String(val) || `Record #${record.id}`;
}

function parseDateValue(value: any): Date | null {
  if (!value) return null;
  if (typeof value === "string") return new Date(value);
  if (value.date) return new Date(value.date);
  return null;
}

function getRecordDate(record: DatabaseRecord, propertyId: number): Date | null {
  const pv = record.property_values?.find((v) => v.property_id === propertyId);
  if (!pv) return null;
  return parseDateValue(pv.value);
}

function formatDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function daysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate();
}

export default function TimelineView({
  properties,
  records,
  activeView,
  titlePropertyId,
  onUpdateRecord: _onUpdateRecord,
}: TimelineViewProps) {
  const navigate = useNavigate();
  const [zoom, setZoom] = useState<ZoomLevel>("month");
  const [currentDate, setCurrentDate] = useState(new Date());

  const config = activeView?.config;
  const startPropId = (config as any)?.timeline_start as number | undefined;
  const endPropId = (config as any)?.timeline_end as number | undefined;

  const startProperty = startPropId ? properties.find((p) => p.id === startPropId) : undefined;

  // Compute date range based on zoom and current date
  const { dateRange, headerCells } = useMemo(() => {
    const year = currentDate.getFullYear();
    const month = currentDate.getMonth();

    let start: Date;
    let end: Date;
    const cells: { label: string; days: number; start: Date }[] = [];

    if (zoom === "week") {
      // Week containing currentDate
      const day = currentDate.getDay();
      start = new Date(year, month, currentDate.getDate() - day);
      end = new Date(start);
      end.setDate(end.getDate() + 7);
      for (let i = 0; i < 7; i++) {
        const d = new Date(start);
        d.setDate(d.getDate() + i);
        cells.push({
          label: d.toLocaleDateString("en-US", { weekday: "short", day: "numeric" }),
          days: 1,
          start: d,
        });
      }
    } else if (zoom === "month") {
      start = new Date(year, month, 1);
      const totalDays = daysInMonth(year, month);
      end = new Date(year, month, totalDays);
      // Split into ~7-day chunks for column headers
      for (let d = 1; d <= totalDays; d += 1) {
        const cellDate = new Date(year, month, d);
        cells.push({
          label: String(d),
          days: 1,
          start: cellDate,
        });
      }
    } else {
      // quarter
      const qStart = Math.floor(month / 3) * 3;
      start = new Date(year, qStart, 1);
      end = new Date(year, qStart + 3, 0);
      for (let m = qStart; m < qStart + 3; m++) {
        const dim = daysInMonth(year, m);
        for (let d = 1; d <= dim; d += Math.ceil(dim / 4)) {
          cells.push({
            label: `${MONTH_NAMES[m].slice(0, 3)} ${d}`,
            days: Math.ceil(dim / 4),
            start: new Date(year, m, d),
          });
        }
      }
    }

    return { dateRange: { start, end }, headerCells: cells };
  }, [currentDate, zoom]);

  // Filter records that have a start date
  const recordsWithDates = useMemo(() => {
    if (!startPropId) return [];
    return records
      .map((record) => {
        const startDate = getRecordDate(record, startPropId);
        const endDate = endPropId ? getRecordDate(record, endPropId) : null;
        return { record, startDate, endDate };
      })
      .filter((r) => r.startDate !== null);
  }, [records, startPropId, endPropId]);

  // Navigation
  const navBack = () => {
    const d = new Date(currentDate);
    if (zoom === "week") d.setDate(d.getDate() - 7);
    else if (zoom === "month") d.setMonth(d.getMonth() - 1);
    else d.setMonth(d.getMonth() - 3);
    setCurrentDate(d);
  };

  const navForward = () => {
    const d = new Date(currentDate);
    if (zoom === "week") d.setDate(d.getDate() + 7);
    else if (zoom === "month") d.setMonth(d.getMonth() + 1);
    else d.setMonth(d.getMonth() + 3);
    setCurrentDate(d);
  };

  const today = () => setCurrentDate(new Date());

  // No timeline configured
  if (!startPropId) {
    return (
      <div className="flex items-center justify-center h-64 text-sm text-gray-400">
        Configure a date property in the view settings to use the timeline view.
      </div>
    );
  }

  // Compute bar positions
  const totalDays = Math.ceil(
    (dateRange.end.getTime() - dateRange.start.getTime()) / (1000 * 60 * 60 * 24)
  );

  const getBarPosition = (date: Date) => {
    const diffDays =
      (date.getTime() - dateRange.start.getTime()) / (1000 * 60 * 60 * 24);
    return (diffDays / totalDays) * 100;
  };

  const getBarWidth = (start: Date, end: Date) => {
    const diffDays =
      (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24);
    return Math.max((diffDays / totalDays) * 100, 2);
  };

  const dayWidth = 100 / totalDays;

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-gray-200 px-4 py-2">
        <div className="flex items-center gap-2">
          <button onClick={navBack} className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
            ◂
          </button>
          <button onClick={today} className="text-sm text-gray-600 hover:text-gray-900">
            Today
          </button>
          <button onClick={navForward} className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
            ▸
          </button>
          <span className="text-sm font-medium text-gray-700">
            {zoom === "week"
              ? `${MONTH_NAMES[dateRange.start.getMonth()]} ${dateRange.start.getDate()}, ${dateRange.start.getFullYear()}`
              : zoom === "month"
                ? `${MONTH_NAMES[currentDate.getMonth()]} ${currentDate.getFullYear()}`
                : `Q${Math.floor(currentDate.getMonth() / 3) + 1} ${currentDate.getFullYear()}`}
          </span>
        </div>
        <div className="flex items-center gap-1 rounded border border-gray-200 bg-gray-50 p-0.5">
          {(["week", "month", "quarter"] as ZoomLevel[]).map((z) => (
            <button
              key={z}
              className={`rounded px-2 py-0.5 text-xs ${
                zoom === z ? "bg-white text-gray-900 shadow-sm" : "text-gray-500 hover:text-gray-700"
              }`}
              onClick={() => setZoom(z)}
            >
              {z.charAt(0).toUpperCase() + z.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Timeline body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left sidebar — record names */}
        <div className="flex-shrink-0 w-48 border-r border-gray-200 overflow-y-auto bg-gray-50">
          {/* Header spacer */}
          <div className="border-b border-gray-200 px-3 py-2">
            <span className="text-xs font-medium text-gray-500">
              {startProperty?.name ?? "Records"}
            </span>
          </div>
          {recordsWithDates.map(({ record }) => (
            <div
              key={record.id}
              className="border-b border-gray-100 px-3 py-2 hover:bg-gray-100 cursor-pointer"
              onClick={() => navigate(`/page/${record.page_id}`)}
            >
              <div className="text-sm text-gray-700 truncate">
                {getRecordTitle(record, titlePropertyId)}
              </div>
            </div>
          ))}
          {recordsWithDates.length === 0 && (
            <div className="px-3 py-4 text-xs text-gray-400 text-center">
              No records with dates
            </div>
          )}
        </div>

        {/* Timeline grid */}
        <div className="flex-1 overflow-auto">
          <div style={{ minWidth: totalDays * 30 }}>
            {/* Column headers */}
            <div className="flex border-b border-gray-200 sticky top-0 bg-white z-10">
              {zoom === "month" &&
                headerCells.map((cell, i) => (
                  <div
                    key={i}
                    className="flex-shrink-0 border-r border-gray-100 px-1 py-1 text-center"
                    style={{ width: `${dayWidth}%`, minWidth: 30 }}
                  >
                    <span className="text-xs text-gray-400">{cell.label}</span>
                  </div>
                ))}
              {zoom === "week" &&
                headerCells.map((cell, i) => (
                  <div
                    key={i}
                    className="flex-shrink-0 border-r border-gray-100 px-2 py-1 text-center"
                    style={{ width: `${100 / 7}%`, minWidth: 100 }}
                  >
                    <span className="text-xs text-gray-500">{cell.label}</span>
                  </div>
                ))}
              {zoom === "quarter" &&
                headerCells.map((cell, i) => (
                  <div
                    key={i}
                    className="flex-shrink-0 border-r border-gray-100 px-2 py-1 text-center"
                    style={{ width: `${100 / headerCells.length}%`, minWidth: 60 }}
                  >
                    <span className="text-xs text-gray-500">{cell.label}</span>
                  </div>
                ))}
            </div>

            {/* Rows with bars */}
            {recordsWithDates.map(({ record, startDate, endDate }, i) => {
              const left = getBarPosition(startDate!);
              const barEnd = endDate ?? startDate!;
              // Ensure barEnd is at least 1 day after start
              if (barEnd.getTime() <= startDate!.getTime()) {
                barEnd.setTime(startDate!.getTime() + 86400000);
              }
              const width = getBarWidth(startDate!, barEnd);
              const color = BAR_COLORS[i % BAR_COLORS.length];

              return (
                <div
                  key={record.id}
                  className="relative border-b border-gray-50"
                  style={{ height: 40 }}
                >
                  {/* Grid lines (thin columns) */}
                  {zoom === "month" && (
                    <div className="flex absolute inset-0 pointer-events-none">
                      {headerCells.map((_, j) => (
                        <div
                          key={j}
                          className="border-r border-gray-50"
                          style={{ width: `${dayWidth}%`, minWidth: 30 }}
                        />
                      ))}
                    </div>
                  )}

                  {/* The bar */}
                  <div
                    className={`absolute top-1/2 -translate-y-1/2 h-6 rounded ${color} opacity-80 hover:opacity-100 cursor-pointer flex items-center px-2 min-w-[4px]`}
                    style={{
                      left: `${left}%`,
                      width: `${width}%`,
                    }}
                    title={`${getRecordTitle(record, titlePropertyId)}: ${formatDate(startDate!)}${
                      endDate ? ` - ${formatDate(endDate)}` : ""
                    }`}
                    onClick={() => navigate(`/page/${record.page_id}`)}
                  >
                    {width > 10 && (
                      <span className="text-xs text-white truncate">
                        {getRecordTitle(record, titlePropertyId)}
                      </span>
                    )}
                  </div>
                </div>
              );
            })}

            {recordsWithDates.length === 0 && (
              <div className="flex items-center justify-center h-48 text-sm text-gray-400">
                No records to display in this time range.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
