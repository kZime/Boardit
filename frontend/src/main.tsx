import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { AuthProvider } from './contexts/AuthContext';
import { BrowserRouter } from 'react-router-dom';




import './index.css';
import '@mdxeditor/editor/style.css';  // 再引入 MDXEditor 样式（覆盖前面的重置）



import { QueryClient, QueryClientProvider } from '@tanstack/react-query'


// DEBUG: monitor history navigation
if (import.meta.env.DEV) {
  // 1) record history API used by React Router
  const _push = history.pushState.bind(history);
  const _replace = history.replaceState.bind(history);
  history.pushState = (...args: any[]) => {
    console.trace('[history.pushState]', ...args);
    return _push(...args as Parameters<typeof _push>);
  };
  history.replaceState = (...args: any[]) => {
    console.trace('[history.replaceState]', ...args);
    return _replace(...args as Parameters<typeof _replace>);
  };
  window.addEventListener('popstate', () => console.trace('[popstate]'));

  // 2) record page unload
  window.addEventListener('beforeunload', () => {
    console.log('[beforeunload] page is unloading (reload or hard nav)');
  });

  // 3) print navigation type (reload / navigate / back_forward)
  const nav = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined;
  console.log('[navigation.type]', nav?.type);
}


// Enable MSW in development
if (import.meta.env.DEV && import.meta.env.VITE_USE_MSW === 'true') {
  await import('./mocks/browser').then(({ worker }) =>
    worker.start({ onUnhandledRequest: 'bypass' })
  )
}

const queryClient = new QueryClient()


ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AuthProvider>
      <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
      </QueryClientProvider>
    </AuthProvider>
  </React.StrictMode>
);
