import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate, Link } from 'react-router-dom';
import { authErrorMessage, validateRegistration } from '../auth/form';

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
    const validation = validateRegistration({ username, email, password });
    if (!validation.ok) {
      setErr(validation.error);
      return;
    }
    setIsSubmitting(true);
    try {
      await register(validation.value.username, validation.value.email, validation.value.password);
      nav('/editor');
    } catch (e: unknown) {
      setErr(authErrorMessage(e, 'REGISTER FAILED'));
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
