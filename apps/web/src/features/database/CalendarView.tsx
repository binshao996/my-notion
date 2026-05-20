import { useState } from "react";
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

/**
 * Parse a date value from a record property. Supports string dates and objects.
 */
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

function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

/**
 * Get all days in a month as a flat array, padded to start on Sunday.
 */
function getMonthGrid(year: number, month: number): Date[] {
  const firstDay = new Date(year, month, 1);
  const start = new Date(firstDay);
  start.setDate(start.getDate() - start.getDay()); // Go back to Sunday

  const days: Date[] = [];
  for (let i = 0; i < 42; i++) {
    const d = new Date(start);
    d.setDate(d.getDate() + i);
    days.push(d);
  }
  return days;
}

export default function CalendarView({
  properties,
  records,
  activeView,
  titlePropertyId,
  onUpdateRecord,
}: CalendarViewProps) {
  const navigate = useNavigate();
  const groupByPropertyId = activeView?.config?.groupBy?.property_id;

  const groupByProperty = groupByPropertyId
    ? properties.find((p) => p.id === groupByPropertyId)
    : undefined;

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

  const [currentMonth, setCurrentMonth] = useState(new Date());
  const year = currentMonth.getFullYear();
  const month = currentMonth.getMonth();

  const days = getMonthGrid(year, month);

  const prevMonth = () => {
    setCurrentMonth(new Date(year, month - 1, 1));
  };

  const nextMonth = () => {
    setCurrentMonth(new Date(year, month + 1, 1));
  };

  // Build a map of day-key -> records for that day
  const recordsByDay = new Map<string, DatabaseRecord[]>();

  for (const record of records) {
    const pv = record.property_values?.find(
      (v) => v.property_id === groupByPropertyId
    );
    if (!pv) continue;
    const date = parseRecordDate(pv.value);
    if (!date) continue;
    const key = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
    const existing = recordsByDay.get(key);
    if (existing) {
      existing.push(record);
    } else {
      recordsByDay.set(key, [record]);
    }
  }

  const isCurrentMonth = (d: Date) => d.getMonth() === month;

  return (
    <div className="p-4 h-full flex flex-col">
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
      <div className="grid grid-cols-7 flex-1">
        {days.map((day, i) => {
          const key = `${day.getFullYear()}-${day.getMonth()}-${day.getDate()}`;
          const dayRecords = recordsByDay.get(key) || [];
          const inMonth = isCurrentMonth(day);
          const isToday = isSameDay(day, new Date());

          return (
            <div
              key={i}
              className={`min-h-[100px] border border-gray-200 p-1 text-sm ${
                inMonth ? "bg-white" : "bg-gray-50"
              }`}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault();
                const recordId = e.dataTransfer.getData("recordId");
                const id = Number(recordId);
                if (!id || !groupByPropertyId) return;
                // Build date value for this day cell
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
              {dayRecords.map((record) => (
                <div
                  key={record.id}
                  className="text-xs truncate rounded bg-blue-50 px-1 py-0.5 mb-0.5 cursor-pointer hover:bg-blue-100"
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.setData("recordId", String(record.id));
                  }}
                  onClick={() => navigate(`/record/${record.id}`)}
                >
                  {getRecordTitle(record, titlePropertyId)}
                </div>
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}
