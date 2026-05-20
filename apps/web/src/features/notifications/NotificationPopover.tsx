import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useNotificationsStore } from "../../stores/notifications";

function formatTime(dateStr: string): string {
  const d = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return d.toLocaleDateString();
}

function notificationLabel(type: string): string {
  switch (type) {
    case "comment": return "left a comment";
    case "mention": return "mentioned you";
    case "share": return "shared a page";
    case "invite": return "invited you";
    default: return type;
  }
}

export default function NotificationPopover() {
  const navigate = useNavigate();
  const { notifications, unreadCount, loadNotifications, markRead, markAllRead, loading } = useNotificationsStore();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    loadNotifications();
  }, [loadNotifications]);

  const handleClick = (notif: { id: number; target_page_id: number; read: boolean }) => {
    if (!notif.read) markRead(notif.id);
    navigate(`/page/${notif.target_page_id}`);
    setOpen(false);
  };

  return (
    <div className="relative">
      <button
        className="relative rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        onClick={() => { setOpen(!open); if (!open) loadNotifications(); }}
      >
        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        {unreadCount > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-0.5 text-[10px] font-medium text-white">
            {unreadCount > 9 ? "9+" : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full z-20 mt-1 w-80 rounded-lg border border-gray-200 bg-white shadow-lg">
            <div className="flex items-center justify-between border-b border-gray-100 px-4 py-2">
              <span className="text-xs font-semibold text-gray-900">Notifications</span>
              {unreadCount > 0 && (
                <button className="text-xs text-blue-600 hover:underline" onClick={markAllRead}>
                  Mark all read
                </button>
              )}
            </div>
            <div className="max-h-80 overflow-y-auto">
              {loading && (
                <div className="px-4 py-3 text-xs text-gray-400 text-center">Loading...</div>
              )}
              {!loading && notifications.length === 0 && (
                <div className="px-4 py-6 text-xs text-gray-400 text-center">No notifications</div>
              )}
              {notifications.map((n) => (
                <button
                  key={n.id}
                  className={`flex w-full items-start gap-3 px-4 py-2.5 text-left hover:bg-gray-50 ${!n.read ? "bg-blue-50/50" : ""}`}
                  onClick={() => handleClick(n)}
                >
                  <div className={`mt-0.5 h-2 w-2 flex-shrink-0 rounded-full ${n.read ? "bg-transparent" : "bg-blue-500"}`} />
                  <div className="min-w-0 flex-1">
                    <div className="text-xs text-gray-700">
                      <span className="font-medium">User #{n.actor_id}</span>{" "}
                      {notificationLabel(n.type)}
                    </div>
                    <div className="text-[10px] text-gray-400 mt-0.5">{formatTime(n.created_at)}</div>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
