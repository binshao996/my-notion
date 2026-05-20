import { Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Workspace from "./pages/Workspace";
import PageView from "./pages/PageView";
import DatabasePage from "./pages/DatabasePage";
import RecordDetailPage from "./pages/RecordDetailPage";
import SharedPage from "./pages/SharedPage";
import ProtectedRoute from "./components/ProtectedRoute";

export default function App() {
  return (
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
  );
}
