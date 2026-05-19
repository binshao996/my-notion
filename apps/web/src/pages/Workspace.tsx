import { useAuthStore } from "../stores/auth";
import PageTree from "../features/sidebar/PageTree";

export default function Workspace() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  return (
    <div className="flex min-h-screen bg-white">
      {/* Sidebar */}
      <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50">
        <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
          <div className="text-sm font-semibold text-gray-900">
            {user?.name || "User"}'s Notion
          </div>
          <button
            onClick={logout}
            className="text-xs text-gray-400 hover:text-gray-600"
          >
            Sign out
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          <PageTree />
        </div>
      </aside>

      {/* Main content */}
      <main className="flex flex-1 items-center justify-center">
        <div className="text-center">
          <h2 className="text-xl font-medium text-gray-700">Welcome to My Notion</h2>
          <p className="mt-2 text-sm text-gray-500">
            Create a page or select one from the sidebar.
          </p>
        </div>
      </main>
    </div>
  );
}
