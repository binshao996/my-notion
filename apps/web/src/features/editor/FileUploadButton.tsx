import { useRef, useState } from "react";
import { api, ApiError } from "../../lib/api";

interface FileUploadButtonProps {
  onUploaded: (url: string, fileName: string) => void;
  workspaceId?: number;
}

interface UploadUrlResponse {
  upload_url: string;
  public_url: string;
}

export default function FileUploadButton({
  onUploaded,
  workspaceId,
}: FileUploadButtonProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setError(null);
    setUploading(true);

    try {
      // 1. Get pre-signed upload URL from backend
      const body: Record<string, unknown> = {
        filename: file.name,
        content_type: file.type || "application/octet-stream",
      };
      if (workspaceId) {
        body.workspace_id = workspaceId;
      }

      const data = await api.post<UploadUrlResponse>(
        "/files/upload-url",
        body
      );

      // 2. Upload file to the pre-signed URL
      const putResponse = await fetch(data.upload_url, {
        method: "PUT",
        body: file,
        headers: {
          "Content-Type": file.type || "application/octet-stream",
        },
      });

      if (!putResponse.ok) {
        throw new Error(
          `Upload failed with status ${putResponse.status}`
        );
      }

      // 3. Call callback with the public URL and filename
      onUploaded(data.public_url, file.name);
    } catch (err) {
      const message =
        err instanceof ApiError
          ? err.message
          : err instanceof Error
          ? err.message
          : "Upload failed";
      setError(message);
    } finally {
      setUploading(false);
      // Reset file input so the same file can be re-selected
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  return (
    <div className="inline-flex items-center gap-1">
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileSelect}
        disabled={uploading}
      />
      <button
        type="button"
        className="rounded px-2 py-0.5 text-xs text-gray-400 hover:text-gray-600 disabled:opacity-50"
        onClick={() => fileInputRef.current?.click()}
        disabled={uploading}
      >
        {uploading ? "Uploading..." : "Upload"}
      </button>
      {error && (
        <span className="text-xs text-red-500" title={error}>
          {error}
        </span>
      )}
    </div>
  );
}
