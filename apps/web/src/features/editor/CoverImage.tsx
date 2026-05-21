import { useRef, useState } from "react";
import { api } from "../../lib/api";

interface CoverImageProps {
  coverUrl: string | null;
  pageId: number;
  onChange: (url: string) => void;
  onRemove: () => void;
}

interface UploadUrlResponse {
  upload_url: string;
  public_url: string;
}

export default function CoverImage({
  coverUrl,
  pageId,
  onChange,
  onRemove,
}: CoverImageProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hovering, setHovering] = useState(false);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setError(null);
    setUploading(true);

    try {
      const data = await api.post<UploadUrlResponse>("/files/upload-url", {
        filename: file.name,
        content_type: file.type || "application/octet-stream",
      });

      const putResponse = await fetch(data.upload_url, {
        method: "PUT",
        body: file,
        headers: {
          "Content-Type": file.type || "application/octet-stream",
        },
      });

      if (!putResponse.ok) {
        throw new Error(`Upload failed with status ${putResponse.status}`);
      }

      onChange(data.public_url);
      await api.patch(`/pages/${pageId}`, { cover: data.public_url });
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Upload failed";
      setError(message);
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  const handleRemove = async () => {
    setError(null);
    try {
      await api.patch(`/pages/${pageId}`, { cover: "" });
      onRemove();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to remove cover";
      setError(message);
    }
  };

  return (
    <div
      className="relative w-full group"
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
    >
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        accept="image/*"
        onChange={handleFileSelect}
        disabled={uploading}
      />

      {coverUrl ? (
        <div className="relative w-full h-[200px] overflow-hidden">
          <img
            src={coverUrl}
            alt="Page cover"
            className="w-full h-full object-cover"
          />
          {hovering && (
            <div className="absolute bottom-3 right-4 flex items-center gap-2 transition-opacity duration-200">
              <button
                type="button"
                className="rounded bg-white/70 px-3 py-1 text-xs text-gray-700 backdrop-blur-sm hover:bg-white/90 transition-colors"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading}
              >
                {uploading ? "Uploading..." : "Change cover"}
              </button>
              <button
                type="button"
                className="rounded bg-white/70 px-3 py-1 text-xs text-gray-700 backdrop-blur-sm hover:bg-white/90 transition-colors"
                onClick={handleRemove}
              >
                Remove
              </button>
            </div>
          )}
        </div>
      ) : (
        <div className="relative w-full h-[30px]">
          {hovering && (
            <button
              type="button"
              className="absolute bottom-1 right-4 rounded bg-gray-100 px-3 py-1 text-xs text-gray-500 hover:bg-gray-200 transition-colors"
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
            >
              {uploading ? "Uploading..." : "Add cover"}
            </button>
          )}
        </div>
      )}

      {error && (
        <p className="absolute bottom-1 left-4 text-xs text-red-500">{error}</p>
      )}
    </div>
  );
}
