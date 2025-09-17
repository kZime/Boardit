import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate, Link } from 'react-router-dom';

// Check if we're in mock mode
const isMockMode = import.meta.env.DEV && import.meta.env.VITE_USE_MSW === 'true';

export default function Login() {
  const { login } = useAuth();
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await login(email, password);
      nav('/editor');
    } catch (e: unknown) {
      if (typeof e === 'object' && e !== null && 'response' in e) {
        // @ts-expect-error: e.response may exist on error objects from axios
        setErr(e.response?.data?.error || 'LOGIN FAILED');
      } else {
        setErr('LOGIN FAILED');
      }
    }
  };

  // Development skip login function
  const handleSkipLogin = async () => {
    if (!isMockMode) return;
    try {
      // Use fake credentials for mock mode
      await login('dev@example.com', 'password');
      nav('/editor');
    } catch (e) {
      console.error('Skip login failed:', e);
    }
  };

  return (
    <form onSubmit={onSubmit} className="max-w-sm mx-auto p-4">
      <h2 className="text-xl mb-4">LOGIN</h2>
      {err && <div className="text-red-500 mb-2">{err}</div>}
      <input
        type="email"
        placeholder="Email"
        value={email}
        onChange={e => setEmail(e.target.value)}
        className="w-full mb-2 p-2 border rounded"
      />
      <input
        type="password"
        placeholder="Password"
        value={password}
        onChange={e => setPassword(e.target.value)}
        className="w-full mb-4 p-2 border rounded"
      />
      <button type="submit" className="w-full p-2 bg-blue-500 text-white rounded">
        LOGIN
      </button>
      
      {/* Development Skip Login Button - Only visible in mock mode */}
      {isMockMode && (
        <button 
          type="button" 
          onClick={handleSkipLogin}
          className="w-full mt-2 p-2 bg-orange-500 text-white rounded text-sm"
        >
          🚀 DEV: Skip Login (Mock Mode)
        </button>
      )}
      
      <p className="mt-2 text-sm">
        No account? <Link to="/register" className="text-blue-400">REGISTER</Link>
      </p>
    </form>
  );
}
