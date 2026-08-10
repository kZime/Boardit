import React, { useState, useCallback, useEffect } from "react";
import { X } from "lucide-react";
import { useAuth } from "../contexts/AuthContext";
import { useAuthModal } from "../contexts/AuthModalContext";
import type { AuthModalMode } from "../contexts/AuthModalContext";
import { authErrorMessage, validateLogin, validateRegistration } from "../auth/form";
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
    const validation = validateLogin({ email, password });
    if (!validation.ok) {
      setErr(validation.error);
      return;
    }
    setIsSubmitting(true);
    try {
      await login(validation.value.email, validation.value.password);
      closeAuthModal();
    } catch (e: unknown) {
      setErr(authErrorMessage(e, "LOGIN FAILED"));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    const validation = validateRegistration({ username, email, password });
    if (!validation.ok) {
      setErr(validation.error);
      return;
    }
    setIsSubmitting(true);
    try {
      await register(validation.value.username, validation.value.email, validation.value.password);
      closeAuthModal();
    } catch (e: unknown) {
      setErr(authErrorMessage(e, "REGISTER FAILED"));
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
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 dark:bg-black/50"
      onClick={handleClose}
      role="dialog"
      aria-modal="true"
      aria-label={isLogin ? "Login" : "Register"}
    >
      <div
        className="bg-white dark:bg-gray-800 rounded-xl shadow-xl border border-gray-200 dark:border-gray-700 w-full max-w-sm overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-6 pt-6 pb-4 border-b border-gray-100 dark:border-gray-700">
          <div>
            <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Boardit</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
              {isLogin ? "Sign in to your account" : "Create an account"}
            </p>
          </div>
          <button
            type="button"
            onClick={closeAuthModal}
            className="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            aria-label="Close"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {isLogin ? (
          <form onSubmit={handleLogin} className="p-6">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Login</h2>
            {err && <div className="text-red-500 dark:text-red-400 mb-2 text-sm">{err}</div>}
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full mb-2 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 transition-colors"
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full mb-4 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
            />
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full p-2.5 bg-blue-500 dark:bg-blue-600 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-blue-600 dark:hover:bg-blue-500 transition-colors font-medium"
            >
              {isSubmitting ? "Logging in…" : "LOGIN"}
            </button>
            {isMockMode && (
              <button
                type="button"
                onClick={handleSkipLogin}
                className="w-full mt-2 p-2 bg-orange-500 text-white rounded-lg text-sm hover:bg-orange-600 transition-colors"
              >
                DEV: Skip Login (Mock Mode)
              </button>
            )}
            <p className="mt-4 text-sm text-center text-gray-600 dark:text-gray-400">
              No account?{" "}
              <button type="button" onClick={openRegisterModal} className="text-blue-600 dark:text-blue-400 hover:underline">
                Register
              </button>
            </p>
          </form>
        ) : (
          <form onSubmit={handleRegister} className="p-6">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Register</h2>
            {err && <div className="text-red-500 dark:text-red-400 mb-2 text-sm">{err}</div>}
            <input
              placeholder="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full mb-2 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
            />
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full mb-2 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full mb-4 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
            />
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full p-2.5 bg-green-500 dark:bg-green-600 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-green-600 dark:hover:bg-green-500 transition-colors font-medium"
            >
              {isSubmitting ? "Creating account…" : "REGISTER"}
            </button>
            <p className="mt-4 text-sm text-center text-gray-600 dark:text-gray-400">
              Already have an account?{" "}
              <button type="button" onClick={openLoginModal} className="text-blue-600 dark:text-blue-400 hover:underline">
                Login
              </button>
            </p>
          </form>
        )}
      </div>
    </div>
  );
}
