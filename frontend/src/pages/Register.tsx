import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate, Link } from 'react-router-dom';

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const USERNAME_RE = /^[A-Za-z0-9][A-Za-z0-9_-]{2,31}$/;
const MIN_PASSWORD_LENGTH = 8;

export default function Register() {
  const { register } = useAuth();
  const nav = useNavigate();
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    const trimmedUsername = username.trim();
    const trimmedEmail = email.trim();
    const trimmedPassword = password.trim();
    if (!trimmedUsername) {
      setErr('Username is required');
      return;
    }
    if (!USERNAME_RE.test(trimmedUsername)) {
      setErr('Username must be 3-32 letters, numbers, underscores, or hyphens');
      return;
    }
    if (!trimmedEmail) {
      setErr('Email is required');
      return;
    }
    if (!EMAIL_RE.test(trimmedEmail)) {
      setErr('Please enter a valid email address');
      return;
    }
    if (!trimmedPassword) {
      setErr('Password is required');
      return;
    }
    if (trimmedPassword.length < MIN_PASSWORD_LENGTH) {
      setErr(`Password must be at least ${MIN_PASSWORD_LENGTH} characters`);
      return;
    }
    setIsSubmitting(true);
    try {
      await register(trimmedUsername, trimmedEmail, trimmedPassword);
      nav('/editor');
    } catch (e: unknown) {
      if (typeof e === 'object' && e !== null && 'response' in e) {
        const error = e as { response?: { data?: { error?: string } } };
        setErr(error.response?.data?.error || 'REGISTER FAILED');
      } else {
        setErr('REGISTER FAILED');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center p-4 transition-colors">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-md border border-gray-200 dark:border-gray-700 w-full max-w-sm overflow-hidden">
        <div className="px-6 pt-6 pb-4 text-center border-b border-gray-100 dark:border-gray-700">
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Boardit</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">Create an account</p>
        </div>
        <form onSubmit={onSubmit} className="p-6">
          <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Register</h2>
          {err && <div className="text-red-500 dark:text-red-400 mb-2 text-sm">{err}</div>}
          <input
            placeholder="Username"
            value={username}
            onChange={e => setUsername(e.target.value)}
            className="w-full mb-2 p-2.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"
          />
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
            className="w-full p-2.5 bg-green-500 dark:bg-green-600 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-green-600 dark:hover:bg-green-500 transition-colors font-medium"
          >
            {isSubmitting ? 'Creating account…' : 'REGISTER'}
          </button>
          <p className="mt-4 text-sm text-center text-gray-600 dark:text-gray-400">
            Already have an account? <Link to="/login" className="text-blue-600 dark:text-blue-400 hover:underline">Login</Link>
          </p>
        </form>
      </div>
    </div>
  );
}
