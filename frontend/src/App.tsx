import React, { lazy, Suspense } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "./contexts/AuthContext";
import { AuthModalProvider } from "./contexts/AuthModalContext";
import AuthModal from "./components/AuthModal";
import Login from "./pages/Login";
import Register from "./pages/Register";
import PostList from "./pages/PostList";
import PostDetail from "./pages/PostDetail";

const Editor = lazy(() => import("./pages/Editor"));

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { accessToken } = useAuth();
  return accessToken ? <>{children}</> : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <AuthModalProvider>
      <Routes>
        <Route path="/" element={<PostList />} />
        <Route path="/post/:username/:slug" element={<PostDetail />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/editor"
          element={
            <PrivateRoute>
              <Suspense fallback={<div className="p-4">Loading editor…</div>}>
                <Editor />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <AuthModal />
    </AuthModalProvider>
  );
}
