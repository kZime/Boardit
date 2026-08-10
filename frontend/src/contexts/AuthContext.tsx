/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable react-refresh/only-export-components */
import { createContext, Fragment, useContext, useState, useEffect, useMemo, useRef } from 'react';
import type { ReactNode } from 'react';
import api, { AUTH_HTTP_TIMEOUT_MS } from '../api/axios';
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
  subscribeAuthState,
  withAuthStateLock,
} from '../auth/tokenStorage';
import { jwtDecode } from 'jwt-decode';
import { useQueryClient } from '@tanstack/react-query';

interface AuthContextType {
  accessToken: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
}
const AuthContext = createContext<AuthContextType>({} as any);

const getTokenSubject = (token: string | null): string | null => {
  if (!token) return null;
  try {
    const subject = jwtDecode<{ sub?: string | number }>(token).sub;
    return subject === undefined ? null : String(subject);
  } catch {
    return null;
  }
};

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const queryClient = useQueryClient();
  const [accessToken, setToken] = useState<string | null>(getAccessToken());
  const authSubject = useMemo(() => getTokenSubject(accessToken), [accessToken]);
  const authSubjectRef = useRef(authSubject);

  useEffect(() => subscribeAuthState((nextAccessToken) => {
    const nextSubject = getTokenSubject(nextAccessToken);
    if (nextSubject !== authSubjectRef.current) {
      queryClient.clear();
      authSubjectRef.current = nextSubject;
    }
    setToken(nextAccessToken);
  }), [queryClient]);

  const login = async (email: string, password: string) => {
    await withAuthStateLock(async () => {
      const response = await api.post(
        '/api/auth/login',
        { email, password },
        { timeout: AUTH_HTTP_TIMEOUT_MS },
      );
      setTokens(response.data.access_token, response.data.refresh_token);
    });
  };

  const register = async (username: string, email: string, password: string) => {
    await api.post(
      '/api/auth/register',
      { username, email, password },
      { timeout: AUTH_HTTP_TIMEOUT_MS },
    );
    await login(email, password);
  };

  const logout = async () => {
    const refreshToken = await withAuthStateLock(() => {
      const currentRefreshToken = getRefreshToken();
      clearTokens();
      return currentRefreshToken;
    });
    if (refreshToken) {
      void api.post(
        '/api/auth/logout',
        { refresh_token: refreshToken },
        { timeout: AUTH_HTTP_TIMEOUT_MS },
      ).catch(() => undefined);
    }
    // SPA: PrivateRoute will redirect to /login when token is null
  };

  // (optional) check token expiration and auto logout
  useEffect(() => {
    if (accessToken) {
      const { exp } = jwtDecode<{ exp: number }>(accessToken);
      if (Date.now() >= exp * 1000) void logout();
    }
  }, [accessToken]);

  return (
    <AuthContext.Provider value={{ accessToken, login, logout, register }}>
      <Fragment key={authSubject ?? 'anonymous'}>{children}</Fragment>
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
