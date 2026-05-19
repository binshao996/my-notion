import { useAuthStore } from "../stores/auth";

export default function Workspace() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  return (
    <div className="flex min-h-screen bg-white">
      <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50 p-4">
        <div className="mb-4 text-sm font-semibold text-gray-900">
          {user?.name || "User"}'s Notion
        </div>
        <div className="flex-1" />
        <button
          onClick={logout}
          className="text-left text-sm text-gray-500 hover:text-gray-700"
        >
          Sign out
        </button>
      </aside>

      <main className="flex flex-1 items-center justify-center">
        <div className="text-center">
          <h2 className="text-xl font-medium text-gray-700">Welcome to My Notion</h2>
          <p className="mt-2 text-sm text-gray-500">
            M0 complete. Editor and pages coming in M1.
          </p>
        </div>
      </main>
    </div>
  );
}
