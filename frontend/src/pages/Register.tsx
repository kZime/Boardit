import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate, Link } from 'react-router-dom';

export default function Register() {
  const { register } = useAuth();
  const nav = useNavigate();
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await register(username, email, password);
      nav('/editor');
    } catch (e: unknown) {
      if (typeof e === 'object' && e !== null && 'response' in e) {
        const error = e as { response?: { data?: { error?: string } } };
        setErr(error.response?.data?.error || 'REGISTER FAILED');
      } else {
        setErr('REGISTER FAILED');
      }
    }
  };

  return (
    <form onSubmit={onSubmit} className="max-w-sm mx-auto p-4">
      <h2 className="text-xl mb-4">REGISTER</h2>
      {err && <div className="text-red-500 mb-2">{err}</div>}
      <input
        placeholder="用户名"
        value={username}
        onChange={e => setUsername(e.target.value)}
        className="w-full mb-2 p-2 border rounded"
      />
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
      <button type="submit" className="w-full p-2 bg-green-500 text-white rounded">
        REGISTER
      </button>
      <p className="mt-2 text-sm">
        Already have an account? <Link to="/login" className="text-blue-400">LOGIN</Link>
      </p>
    </form>
  );
}
