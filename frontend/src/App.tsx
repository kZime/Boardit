import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './contexts/AuthContext';
import Login from './pages/Login';
import Register from './pages/Register';
import Editor from './pages/Editor';

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { accessToken } = useAuth();
  return accessToken ? <>{children}</> : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route
        path="/editor"
        element={
          <PrivateRoute>
            <Editor />
          </PrivateRoute>
        }
      />
      <Route path="*" element={<Navigate to="/editor" replace />} />
    </Routes>
  );
}
