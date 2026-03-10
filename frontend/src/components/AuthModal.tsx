import React, { useState, useCallback, useEffect } from "react";
import { useAuth } from "../contexts/AuthContext";
import { useAuthModal } from "../contexts/AuthModalContext";
import type { AuthModalMode } from "../contexts/AuthModalContext";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const MIN_PASSWORD_LENGTH = 8;
const isMockMode = import.meta.env.DEV && import.meta.env.VITE_USE_MSW === "true";

export default function AuthModal() {
  const { authModal, closeAuthModal, openRegisterModal, openLoginModal } = useAuthModal();
  const { login, register } = useAuth();

  const [mode, setMode] = useState<AuthModalMode>(authModal);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    setMode(authModal);
    if (authModal) {
      setErr("");
      setUsername("");
      setEmail("");
      setPassword("");
    }
  }, [authModal]);

  const handleClose = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === e.currentTarget) closeAuthModal();
    },
    [closeAuthModal]
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeAuthModal();
    };
    if (authModal) {
      window.addEventListener("keydown", onKey);
      return () => window.removeEventListener("keydown", onKey);
    }
  }, [authModal, closeAuthModal]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    const trimmedEmail = email.trim();
    const trimmedPassword = password.trim();
    if (!trimmedEmail) {
      setErr("Email is required");
      return;
    }
    if (!EMAIL_RE.test(trimmedEmail)) {
      setErr("Please enter a valid email address");
      return;
    }
    if (!trimmedPassword) {
      setErr("Password is required");
      return;
    }
    setIsSubmitting(true);
    try {
      await login(trimmedEmail, trimmedPassword);
      closeAuthModal();
    } catch (e: unknown) {
      if (typeof e === "object" && e !== null && "response" in e) {
        const res = e as { response?: { data?: { error?: string } } };
        setErr(res.response?.data?.error || "LOGIN FAILED");
      } else {
        setErr("LOGIN FAILED");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    const trimmedUsername = username.trim();
    const trimmedEmail = email.trim();
    const trimmedPassword = password.trim();
    if (!trimmedUsername) {
      setErr("Username is required");
      return;
    }
    if (!trimmedEmail) {
      setErr("Email is required");
      return;
    }
    if (!EMAIL_RE.test(trimmedEmail)) {
      setErr("Please enter a valid email address");
      return;
    }
    if (!trimmedPassword) {
      setErr("Password is required");
      return;
    }
    if (trimmedPassword.length < MIN_PASSWORD_LENGTH) {
      setErr(`Password must be at least ${MIN_PASSWORD_LENGTH} characters`);
      return;
    }
    setIsSubmitting(true);
    try {
      await register(trimmedUsername, trimmedEmail, trimmedPassword);
      closeAuthModal();
    } catch (e: unknown) {
      if (typeof e === "object" && e !== null && "response" in e) {
        const res = e as { response?: { data?: { error?: string } } };
        setErr(res.response?.data?.error || "REGISTER FAILED");
      } else {
        setErr("REGISTER FAILED");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSkipLogin = async () => {
    if (!isMockMode) return;
    setErr("");
    setIsSubmitting(true);
    try {
      await login("dev@example.com", "password");
      closeAuthModal();
    } catch (e) {
      console.error("Skip login failed:", e);
      setErr("Skip login failed");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!authModal) return null;

  const isLogin = mode === "login";

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40"
      onClick={handleClose}
      role="dialog"
      aria-modal="true"
      aria-label={isLogin ? "Login" : "Register"}
    >
      <div
        className="bg-white rounded-xl shadow-xl border border-gray-200 w-full max-w-sm overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-6 pt-6 pb-2 border-b border-gray-100">
          <div>
            <h1 className="text-xl font-semibold text-gray-800">Blogedit</h1>
            <p className="text-sm text-gray-500 mt-0.5">
              {isLogin ? "Sign in to your account" : "Create an account"}
            </p>
          </div>
          <button
            type="button"
            onClick={closeAuthModal}
            className="p-1.5 rounded-md text-gray-400 hover:text-gray-600 hover:bg-gray-100"
            aria-label="Close"
          >
            <span className="text-xl leading-none">×</span>
          </button>
        </div>

        {isLogin ? (
          <form onSubmit={handleLogin} className="p-6">
            <h2 className="text-lg font-medium text-gray-800 mb-4">Login</h2>
            {err && <div className="text-red-500 mb-2 text-sm">{err}</div>}
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full mb-2 p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full mb-4 p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full p-2 bg-blue-500 text-white rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-blue-600"
            >
              {isSubmitting ? "Logging in…" : "LOGIN"}
            </button>
            {isMockMode && (
              <button
                type="button"
                onClick={handleSkipLogin}
                className="w-full mt-2 p-2 bg-orange-500 text-white rounded-md text-sm hover:bg-orange-600"
              >
                DEV: Skip Login (Mock Mode)
              </button>
            )}
            <p className="mt-4 text-sm text-center text-gray-600">
              No account?{" "}
              <button type="button" onClick={openRegisterModal} className="text-blue-600 hover:underline">
                Register
              </button>
            </p>
          </form>
        ) : (
          <form onSubmit={handleRegister} className="p-6">
            <h2 className="text-lg font-medium text-gray-800 mb-4">Register</h2>
            {err && <div className="text-red-500 mb-2 text-sm">{err}</div>}
            <input
              placeholder="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full mb-2 p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full mb-2 p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full mb-4 p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full p-2 bg-green-500 text-white rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-green-600"
            >
              {isSubmitting ? "Creating account…" : "REGISTER"}
            </button>
            <p className="mt-4 text-sm text-center text-gray-600">
              Already have an account?{" "}
              <button type="button" onClick={openLoginModal} className="text-blue-600 hover:underline">
                Login
              </button>
            </p>
          </form>
        )}
      </div>
    </div>
  );
}
