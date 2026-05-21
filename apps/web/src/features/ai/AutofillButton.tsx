import { useState, useRef, useEffect, useCallback } from "react";
import { api, ApiError } from "../../lib/api";
import type { AutofillJob } from "./types";

interface AutofillButtonProps {
  databaseId: number;
  propertyId: number;
  sourcePropertyId: number;
  recordIds?: number[];
  onComplete?: () => void;
}

type Status = "idle" | "loading" | "running" | "done" | "error";

export default function AutofillButton({
  databaseId,
  propertyId,
  sourcePropertyId,
  recordIds,
  onComplete,
}: AutofillButtonProps) {
  const [status, setStatus] = useState<Status>("idle");
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [job, setJob] = useState<AutofillJob | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const dropdownRef = useRef<HTMLDivElement>(null);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Clear polling on unmount
  useEffect(() => {
    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
      }
    };
  }, []);

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    };
    if (dropdownOpen) {
      document.addEventListener("mousedown", handler);
      return () => document.removeEventListener("mousedown", handler);
    }
  }, [dropdownOpen]);

  // Poll job status while running/pending
  const pollJob = useCallback((jobId: string) => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
    }

    pollingRef.current = setInterval(async () => {
      try {
        const data = await api.get<AutofillJob>(`/ai/autofill/${jobId}`);
        setJob(data);

        if (data.status === "completed") {
          clearInterval(pollingRef.current!);
          pollingRef.current = null;
          setStatus("done");
          onComplete?.();
          // Reset to idle after brief checkmark display
          setTimeout(() => setStatus("idle"), 2000);
        } else if (data.status === "failed") {
          clearInterval(pollingRef.current!);
          pollingRef.current = null;
          setStatus("error");
          setErrorMessage(`Autofill failed: ${data.failed} of ${data.total} records failed`);
        }
      } catch {
        clearInterval(pollingRef.current!);
        pollingRef.current = null;
        setStatus("error");
        setErrorMessage("Failed to check job status");
      }
    }, 1000);
  }, [onComplete]);

  const triggerAutofill = useCallback(
    async (scope: "all" | "selected") => {
      setStatus("loading");
      setErrorMessage(null);
      setDropdownOpen(false);

      try {
        const body: Record<string, unknown> = {
          database_id: databaseId,
          property_id: propertyId,
          source_prop_id: sourcePropertyId,
        };

        if (scope === "selected" && recordIds) {
          body.record_ids = recordIds;
        }

        const data = await api.post<{ job: AutofillJob }>("/ai/autofill", body);
        const newJob = data.job;

        setJob(newJob);
        setStatus("running");

        if (newJob.status === "completed") {
          setStatus("done");
          onComplete?.();
          setTimeout(() => setStatus("idle"), 2000);
        } else if (newJob.status === "failed") {
          setStatus("error");
          setErrorMessage("Autofill job failed");
        } else {
          pollJob(newJob.id);
        }
      } catch (err) {
        setStatus("error");
        if (err instanceof ApiError) {
          setErrorMessage(err.message || "Autofill request failed");
        } else {
          setErrorMessage("Network error. Please try again.");
        }
      }
    },
    [databaseId, propertyId, sourcePropertyId, recordIds, pollJob, onComplete]
  );

  return (
    <div ref={dropdownRef} className="relative inline-flex items-center">
      <button
        onClick={() => {
          if (status === "idle" || status === "error") {
            setDropdownOpen(!dropdownOpen);
          }
        }}
        disabled={status === "loading" || status === "running"}
        className={`inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium transition-colors ${
          status === "running"
            ? "cursor-default bg-blue-50 text-blue-600"
            : status === "done"
              ? "cursor-default bg-green-50 text-green-600"
              : status === "error"
                ? "cursor-pointer bg-red-50 text-red-600"
                : "cursor-pointer text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        }`}
        title={
          status === "error" && errorMessage
            ? errorMessage
            : status === "running" && job
              ? `${job.completed} of ${job.total} records`
              : "AI Autofill"
        }
      >
        {status === "loading" && (
          <svg className="h-3.5 w-3.5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        )}
        {status === "running" && (
          <svg className="h-3.5 w-3.5 animate-pulse" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" />
          </svg>
        )}
        {status === "done" && (
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
        )}
        {status === "error" && (
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
        )}
        {(status === "idle" || status === "error") && (
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" />
          </svg>
        )}

        {status === "running" && job ? (
          <span>
            {job.completed}/{job.total}
          </span>
        ) : status === "done" ? (
          <span>Done</span>
        ) : (
          <span>AI</span>
        )}
      </button>

      {/* Dropdown menu */}
      {dropdownOpen && (status === "idle" || status === "error") && (
        <div className="absolute left-0 top-full z-50 mt-1 w-64 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
          <button
            onClick={() => triggerAutofill("all")}
            className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100"
          >
            <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" />
            </svg>
            Fill this property for all records
          </button>
          <button
            onClick={() => triggerAutofill("selected")}
            disabled={!recordIds || recordIds.length === 0}
            className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40"
          >
            <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12" />
            </svg>
            Fill this property for selected records
            {(!recordIds || recordIds.length === 0) && (
              <span className="text-xs text-gray-400">(none selected)</span>
            )}
          </button>

          {/* Show error message if in error state */}
          {status === "error" && errorMessage && (
            <div className="border-t border-gray-100 px-4 py-2">
              <p className="text-xs text-red-500">{errorMessage}</p>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setStatus("idle");
                  setDropdownOpen(true);
                }}
                className="mt-1 text-xs font-medium text-red-600 underline hover:no-underline"
              >
                Dismiss
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
