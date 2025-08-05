import axios from 'axios';
import { getAccessToken, setAccessToken, getRefreshToken } from '../contexts/AuthContext';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
});

// 请求：自动附带 access token
api.interceptors.request.use(config => {
  const token = getAccessToken();
  if (token) config.headers!['Authorization'] = `Bearer ${token}`;
  return config;
});

// 响应：遇到 401，尝试刷新 token 并重发一次
let isRefreshing = false;
let subscribers: ((token: string) => void)[] = [];

function onRefreshed(token: string) {
  subscribers.forEach(cb => cb(token));
  subscribers = [];
}

api.interceptors.response.use(
  res => res,
  async err => {
    const status = err.response?.status;
    const config = err.config;
    if (status === 401 && !config._retry) {
      if (isRefreshing === false) {
        isRefreshing = true;
        try {
          const { data } = await axios.post(
            '/api/auth/refresh',
            { refresh_token: getRefreshToken() },
            { baseURL: api.defaults.baseURL }
          );
          setAccessToken(data.access_token);
          localStorage.setItem('refreshToken', data.refresh_token);
          onRefreshed(data.access_token);
        } catch (e) {
          // 刷新失败，跳转登录
          window.location.href = '/login';
          return Promise.reject(e);
        } finally {
          isRefreshing = false;
        }
      }
      // 将当前请求挂起，等刷新完成后再重发
      return new Promise(resolve => {
        subscribers.push((token: string) => {
          config.headers!['Authorization'] = `Bearer ${token}`;
          config._retry = true;
          resolve(api(config));
        });
      });
    }
    return Promise.reject(err);
  }
);

export default api;
