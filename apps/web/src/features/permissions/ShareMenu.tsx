import { useState } from "react";
import ShareDialog from "./ShareDialog";

interface ShareMenuProps {
  pageId: number;
}

export default function ShareMenu({ pageId }: ShareMenuProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        className="rounded bg-blue-600 px-3 py-1.5 text-xs text-white hover:bg-blue-700"
        onClick={() => setOpen(true)}
      >
        Share
      </button>
      <ShareDialog pageId={pageId} open={open} onClose={() => setOpen(false)} />
    </>
  );
}
