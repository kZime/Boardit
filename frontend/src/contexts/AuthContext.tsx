/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import api from '../api/axios';
import { clearTokens, getAccessToken, setTokens } from '../auth/tokenStorage';
import { jwtDecode } from 'jwt-decode';

interface AuthContextType {
  accessToken: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  register: (username: string, email: string, password: string) => Promise<void>;
}
const AuthContext = createContext<AuthContextType>({} as any);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [accessToken, setToken] = useState<string | null>(getAccessToken());

  const login = async (email: string, password: string) => {
    const { data } = await api.post('/api/auth/login', { email, password });
    setToken(data.access_token);
    setTokens(data.access_token, data.refresh_token);
  };

  const register = async (username: string, email: string, password: string) => {
    await api.post('/api/auth/register', { username, email, password });
    await login(email, password);
  };

  const logout = () => {
    clearTokens();
    setToken(null);
    // SPA: PrivateRoute will redirect to /login when token is null
  };

  // (optional) check token expiration and auto logout
  useEffect(() => {
    if (accessToken) {
      const { exp } = jwtDecode<{ exp: number }>(accessToken);
      if (Date.now() >= exp * 1000) logout();
    }
  }, [accessToken]);

  return (
    <AuthContext.Provider value={{ accessToken, login, logout, register }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
