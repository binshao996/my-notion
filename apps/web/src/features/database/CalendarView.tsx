import { useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import type { Property, Record as DatabaseRecord, View } from "../../types/database";

interface CalendarViewProps {
  properties: Property[];
  records: DatabaseRecord[];
  activeView?: View;
  titlePropertyId?: number;
  onUpdateRecord: (recordId: number, propertyId: number, value: any) => Promise<void>;
}

const MONTH_NAMES = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];

const DAY_NAMES = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function getRecordTitle(record: DatabaseRecord, titlePropertyId?: number): string {
  if (!titlePropertyId) return `Record #${record.id}`;
  const pv = record.property_values?.find((v) => v.property_id === titlePropertyId);
  if (!pv) return `Record #${record.id}`;
  const val = typeof pv.value === "string" ? pv.value : pv.value?.text ?? pv.value?.title ?? "";
  return String(val) || `Record #${record.id}`;
}

function parseRecordDate(value: any): Date | null {
  if (!value) return null;
  if (value instanceof Date) return value;
  if (typeof value === "string") {
    const d = new Date(value);
    return isNaN(d.getTime()) ? null : d;
  }
  if (typeof value === "object") {
    const s = value.date ?? value.start ?? value.$date ?? value.value;
    if (s) {
      const d = new Date(s);
      return isNaN(d.getTime()) ? null : d;
    }
  }
  return null;
}

function parseRecordEndDate(value: any): Date | null {
  if (!value || typeof value !== "object") return null;
  const e = value.end;
  if (!e) return null;
  const d = new Date(e);
  return isNaN(d.getTime()) ? null : d;
}

function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

function getMonthGrid(year: number, month: number): Date[] {
  const firstDay = new Date(year, month, 1);
  const start = new Date(firstDay);
  start.setDate(start.getDate() - start.getDay());

  const days: Date[] = [];
  for (let i = 0; i < 42; i++) {
    const d = new Date(start);
    d.setDate(d.getDate() + i);
    days.push(d);
  }
  return days;
}

interface ResizeState {
  recordId: number;
  propertyId: number;
  startDate: string;
  endDate: string | null;
  anchorDay: Date; // the day cell the resize started from
}

export default function CalendarView({
  properties,
  records,
  activeView,
  titlePropertyId,
  onUpdateRecord,
}: CalendarViewProps) {
  const navigate = useNavigate();
  const gridRef = useRef<HTMLDivElement>(null);
  const groupByPropertyId = activeView?.config?.groupBy?.property_id;

  const groupByProperty = groupByPropertyId
    ? properties.find((p) => p.id === groupByPropertyId)
    : undefined;

  const [currentMonth, setCurrentMonth] = useState(new Date());
  const [resizing, setResizing] = useState<ResizeState | null>(null);

  const year = currentMonth.getFullYear();
  const month = currentMonth.getMonth();
  const days = getMonthGrid(year, month);

  const prevMonth = () => setCurrentMonth(new Date(year, month - 1, 1));
  const nextMonth = () => setCurrentMonth(new Date(year, month + 1, 1));

  const recordsByDay = new Map<string, { record: DatabaseRecord; isFirst: boolean; isLast: boolean }[]>();

  for (const record of records) {
    const pv = record.property_values?.find((v) => v.property_id === groupByPropertyId);
    if (!pv) continue;
    const startDate = parseRecordDate(pv.value);
    if (!startDate) continue;
    const endDate = parseRecordEndDate(pv.value);

    if (endDate && !isSameDay(startDate, endDate)) {
      // Multi-day: add to each day in range
      const cur = new Date(startDate);
      while (cur <= endDate) {
        const key = `${cur.getFullYear()}-${cur.getMonth()}-${cur.getDate()}`;
        const isFirst = isSameDay(cur, startDate);
        const isLast = isSameDay(cur, endDate);
        const existing = recordsByDay.get(key);
        if (existing) {
          existing.push({ record, isFirst, isLast });
        } else {
          recordsByDay.set(key, [{ record, isFirst, isLast }]);
        }
        cur.setDate(cur.getDate() + 1);
      }
    } else {
      // Single day
      const key = `${startDate.getFullYear()}-${startDate.getMonth()}-${startDate.getDate()}`;
      const existing = recordsByDay.get(key);
      if (existing) {
        existing.push({ record, isFirst: true, isLast: true });
      } else {
        recordsByDay.set(key, [{ record, isFirst: true, isLast: true }]);
      }
    }
  }

  const isCurrentMonth = (d: Date) => d.getMonth() === month;

  const commitResize = useCallback((targetDay: Date) => {
    if (!resizing || !groupByPropertyId) return;
    const dayStr = targetDay.toISOString().split("T")[0];

    if (resizing.endDate) {
      // Resizing end of range: update end_date
      const newEnd = dayStr;
      if (newEnd < resizing.startDate) {
        // Swap: new end is before start, make it the new start
        onUpdateRecord(resizing.recordId, resizing.propertyId, {
          start: newEnd,
          end: resizing.startDate,
        });
      } else {
        onUpdateRecord(resizing.recordId, resizing.propertyId, {
          start: resizing.startDate,
          end: newEnd,
        });
      }
    } else {
      // Single date: create a range with start and end
      const start = resizing.startDate;
      if (dayStr < start) {
        onUpdateRecord(resizing.recordId, resizing.propertyId, {
          start: dayStr,
          end: start,
        });
      } else if (dayStr === start) {
        // No change
      } else {
        onUpdateRecord(resizing.recordId, resizing.propertyId, {
          start: start,
          end: dayStr,
        });
      }
    }
  }, [resizing, groupByPropertyId, onUpdateRecord]);

  // Validate groupBy property
  if (!groupByProperty) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-gray-400">Select a date property to view calendar</div>
      </div>
    );
  }

  if (groupByProperty.type !== "date") {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-gray-400">Select a date property to view calendar</div>
      </div>
    );
  }

  return (
    <div className="p-4 h-full flex flex-col" ref={gridRef}>
      {/* Calendar header */}
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={prevMonth}
          className="rounded px-2 py-1 text-sm text-gray-500 hover:bg-gray-100"
        >
          &lt;
        </button>
        <span className="font-medium text-sm text-gray-900">
          {MONTH_NAMES[month]} {year}
        </span>
        <button
          onClick={nextMonth}
          className="rounded px-2 py-1 text-sm text-gray-500 hover:bg-gray-100"
        >
          &gt;
        </button>
      </div>

      {/* Day name headers */}
      <div className="grid grid-cols-7 border-b border-gray-200 pb-1">
        {DAY_NAMES.map((day) => (
          <div key={day} className="text-center text-xs font-medium text-gray-500 py-1">
            {day}
          </div>
        ))}
      </div>

      {/* Day cells */}
      {resizing && (
        <div
          className="fixed inset-0 z-40 cursor-col-resize"
          onMouseMove={(e) => {
            const target = document.elementFromPoint(e.clientX, e.clientY);
            const dayEl = target?.closest("[data-day]");
            if (dayEl) {
              const dayStr = dayEl.getAttribute("data-day");
              if (dayStr) {
                // Store target day for mouseup
                (dayEl as HTMLElement).dataset.resizeTarget = "true";
              }
            }
          }}
          onMouseUp={(e) => {
            const target = document.elementFromPoint(e.clientX, e.clientY);
            const dayEl = target?.closest("[data-day]") as HTMLElement | null;
            if (dayEl) {
              const dayStr = dayEl.getAttribute("data-day");
              if (dayStr) {
                commitResize(new Date(dayStr));
              }
            }
            setResizing(null);
          }}
          onMouseLeave={() => setResizing(null)}
        />
      )}
      <div className="grid grid-cols-7 flex-1">
        {days.map((day, i) => {
          const key = `${day.getFullYear()}-${day.getMonth()}-${day.getDate()}`;
          const dayEntries = recordsByDay.get(key) || [];
          const inMonth = isCurrentMonth(day);
          const isToday = isSameDay(day, new Date());

          return (
            <div
              key={i}
              data-day={day.toISOString().split("T")[0]}
              className={`min-h-[100px] border border-gray-200 p-1 text-sm ${
                inMonth ? "bg-white" : "bg-gray-50"
              }`}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault();
                const recordId = e.dataTransfer.getData("recordId");
                const id = Number(recordId);
                if (!id || !groupByPropertyId) return;
                const dateStr = day.toISOString().split("T")[0];
                onUpdateRecord(id, groupByPropertyId, { date: dateStr });
              }}
            >
              <div
                className={`text-xs mb-0.5 ${
                  inMonth
                    ? isToday
                      ? "text-blue-600 font-semibold"
                      : "text-gray-600"
                    : "text-gray-300"
                }`}
              >
                {day.getDate()}
              </div>
              {dayEntries.map(({ record, isFirst, isLast }) => {
                const pv = record.property_values?.find(
                  (v) => v.property_id === groupByPropertyId
                );
                const value = pv?.value;
                const endDate = value && typeof value === "object" ? parseRecordEndDate(value) : null;
                const isMultiDay = endDate && parseRecordDate(value) ? !isSameDay(parseRecordDate(value)!, endDate) : false;
                const showResize = isMultiDay ? isLast : true;

                return (
                  <div
                    key={record.id}
                    className={`text-xs truncate rounded px-1 py-0.5 mb-0.5 cursor-pointer hover:bg-blue-100 group relative flex items-center gap-0.5 ${
                      isMultiDay
                        ? isFirst
                          ? "bg-blue-100 border-l-2 border-blue-400 rounded-l"
                          : isLast
                          ? "bg-blue-100 border-r-2 border-blue-400 rounded-r"
                          : "bg-blue-50 border-l border-dashed border-blue-200 text-blue-300"
                        : "bg-blue-50"
                    }`}
                    draggable
                    onDragStart={(e) => {
                      if (resizing) return;
                      e.dataTransfer.setData("recordId", String(record.id));
                    }}
                    onClick={() => navigate(`/record/${record.id}`)}
                    title={isMultiDay && !isFirst && !isLast ? "" : getRecordTitle(record, titlePropertyId)}
                  >
                    <span className="flex-1 truncate">
                      {isFirst || !isMultiDay ? getRecordTitle(record, titlePropertyId) : ""}
                    </span>
                    {showResize && (
                      <span
                        className="hidden group-hover:inline-block ml-auto w-2 h-3 cursor-col-resize flex-shrink-0 rounded-r bg-gray-300 hover:bg-blue-400"
                        onMouseDown={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          const pv = record.property_values?.find(
                            (v) => v.property_id === groupByPropertyId
                          );
                          if (!pv || !groupByPropertyId) return;
                          const startDate = parseRecordDate(pv.value);
                          if (!startDate) return;
                          const end = parseRecordEndDate(pv.value);
                          setResizing({
                            recordId: record.id,
                            propertyId: groupByPropertyId,
                            startDate: startDate.toISOString().split("T")[0],
                            endDate: end ? end.toISOString().split("T")[0] : null,
                            anchorDay: day,
                          });
                        }}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}
