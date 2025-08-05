/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';



import api from '../api/axios';
import { jwtDecode } from 'jwt-decode';

interface AuthContextType {
  accessToken: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  register: (username: string, email: string, password: string) => Promise<void>;
}
const AuthContext = createContext<AuthContextType>({} as any);

// 供拦截器拿当前 token
export function getAccessToken(): string | null {
  return localStorage.getItem('accessToken');
}

export function getRefreshToken(): string | null {
  return localStorage.getItem('refreshToken');
}

export function setAccessToken(token: string) {
  localStorage.setItem('accessToken', token);
}

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [accessToken, setToken] = useState<string | null>(getAccessToken());

  const login = async (email: string, password: string) => {
    const { data } = await api.post('/api/auth/login', { email, password });
    setToken(data.access_token);
    localStorage.setItem('accessToken', data.access_token);
    localStorage.setItem('refreshToken', data.refresh_token);
  };

  const register = async (username: string, email: string, password: string) => {
    await api.post('/api/auth/register', { username, email, password });
    // 注册后可自动登录，或跳到登录页
  };

  const logout = () => {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    setToken(null);
    window.location.href = '/login';
  };

  // （可选）检查 token 过期并自动 logout
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
