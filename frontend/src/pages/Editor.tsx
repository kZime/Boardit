/* eslint-disable @typescript-eslint/no-empty-object-type */
// src/pages/Editor.tsx
import React, { useState, Suspense } from 'react';
import { useAuth } from '../contexts/AuthContext';

const CodeMirror = SuspenseFor(
  () => import('@uiw/react-codemirror'), 
  { fallback: <div>Loading editor…</div> }
);
const ReactMarkdown = SuspenseFor(
  () => import('react-markdown'), 
  { fallback: <div>Loading preview…</div> }
);

export default function Editor() {
  const [text, setText] = useState('# Hello Markdown');
  const { logout } = useAuth();

  return (
    <div className="h-screen grid grid-cols-2">
      <div className="p-4">
        <button onClick={logout} className="mb-4 text-red-500">退出</button>
        <Suspense fallback={<div>Loading editor…</div>}>
          <CodeMirror
            value={text}
            height="90%"
            extensions={[]}
            onChange={(v: string) => setText(v)}
          />
        </Suspense>
      </div>
      <div className="p-4 overflow-auto bg-gray-50">
        <Suspense fallback={<div>Loading preview…</div>}>
          <ReactMarkdown>{text}</ReactMarkdown>
        </Suspense>
      </div>
    </div>
  );
}

// 辅助：把懒加载封装为组件
function SuspenseFor<T extends {}>(
  importer: () => Promise<{ default: React.ComponentType<T> }>,
  options: { fallback: React.ReactNode }
) {
  const Component = React.lazy(importer);
  return (props: T) => (
    <Suspense fallback={options.fallback}>
      <Component {...props} />
    </Suspense>
  );
}
