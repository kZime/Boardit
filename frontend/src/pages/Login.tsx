import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate, Link } from 'react-router-dom';
import { authErrorMessage, validateLogin } from '../auth/form';

// Check if we're in mock mode
const isMockMode = import.meta.env.DEV && import.meta.env.VITE_USE_MSW === 'true';

export default function Login() {
  const { login } = useAuth();
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    const validation = validateLogin({ email, password });
    if (!validation.ok) {
      setErr(validation.error);
      return;
    }
    setIsSubmitting(true);
    try {
      await login(validation.value.email, validation.value.password);
      nav('/editor');
    } catch (e: unknown) {
      setErr(authErrorMessage(e, 'LOGIN FAILED'));
    } finally {
      setIsSubmitting(false);
    }
  };

  // Development skip login function
  const handleSkipLogin = async () => {
    if (!isMockMode) return;
    setErr('');
    setIsSubmitting(true);
    try {
      await login('dev@example.com', 'password');
      nav('/editor');
    } catch (e) {
      console.error('Skip login failed:', e);
      setErr('Skip login failed');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center p-4 transition-colors">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md border border-gray-200 dark:border-gray-700 w-full max-w-sm overflow-hidden">
        <div className="px-6 pt-6 pb-4 text-center border-b border-gray-100 dark:border-gray-700">
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Boardit</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">Sign in to your account</p>
        </div>
        <form onSubmit={onSubmit} className="p-6">
          <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Login</h2>
          {err && <div className="text-red-500 dark:text-red-400 mb-2 text-sm">{err}</div>}
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            className="w-full mb-2 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            className="w-full mb-4 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
          />
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full p-2.5 bg-blue-500 dark:bg-blue-600 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-blue-600 dark:hover:bg-blue-500 transition-colors font-medium"
          >
            {isSubmitting ? 'Logging in…' : 'LOGIN'}
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
            No account? <Link to="/register" className="text-blue-600 dark:text-blue-400 hover:underline">Register</Link>
          </p>
        </form>
      </div>
    </div>
  );
}
