import { useEffect, useState } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Workspace from "./pages/Workspace";
import PageView from "./pages/PageView";
import DatabasePage from "./pages/DatabasePage";
import RecordDetailPage from "./pages/RecordDetailPage";
import SharedPage from "./pages/SharedPage";
import ProtectedRoute from "./components/ProtectedRoute";
import SearchModal from "./features/search/SearchModal";
import AIQAModal from "./features/ai/AIQAModal";
import { useSearchStore } from "./stores/search";

export default function App() {
  const toggle = useSearchStore((s) => s.toggle);
  const activeWorkspaceId = useSearchStore((s) => s.activeWorkspaceId);
  const [showAIQA, setShowAIQA] = useState(false);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        toggle();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [toggle]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "j") {
        e.preventDefault();
        setShowAIQA((v) => !v);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  return (
    <>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/workspace/:id"
          element={
            <ProtectedRoute>
              <Workspace />
            </ProtectedRoute>
          }
        />
        <Route
          path="/page/:pageId"
          element={
            <ProtectedRoute>
              <PageView />
            </ProtectedRoute>
          }
        />
        <Route
          path="/database/:dbId"
          element={
            <ProtectedRoute>
              <DatabasePage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/record/:recordId"
          element={
            <ProtectedRoute>
              <RecordDetailPage />
            </ProtectedRoute>
          }
        />
        <Route path="/shared/:token" element={<SharedPage />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
      <SearchModal />
      {showAIQA && (
        <AIQAModal isOpen={showAIQA} onClose={() => setShowAIQA(false)} workspaceId={activeWorkspaceId} />
      )}
    </>
  );
}
