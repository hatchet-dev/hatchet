/* eslint-disable import/no-extraneous-dependencies */
'use client';

import { useState } from 'react';

export default function Home() {
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamContent, setStreamContent] = useState('');

  const startStreaming = async () => {
    if (isStreaming) return;

    setIsStreaming(true);
    setStreamContent('');

    const response = await fetch('/api/stream');
    const reader = response.body?.getReader();
    const decoder = new TextDecoder();

    if (reader) {
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value, { stream: true });
          setStreamContent((prev) => prev + chunk);
        }
      } finally {
        reader.releaseLock();
        setIsStreaming(false);
      }
    }
  };

  return (
    <div style={{ minHeight: '100vh', backgroundColor: '#f9fafb', padding: '3rem 0' }}>
      <div style={{ maxWidth: '64rem', margin: '0 auto', padding: '0 1.5rem' }}>
        <div style={{ textAlign: 'center', marginBottom: '3rem' }}>
          <h1
            style={{
              fontSize: '2.25rem',
              fontWeight: 'bold',
              color: '#111827',
              marginBottom: '1rem',
            }}
          >
            Hatchet Streaming Demo
          </h1>
          <p style={{ fontSize: '1.25rem', color: '#6b7280' }}>
            Real-time workflow streaming with TypeScript SDK
          </p>
        </div>

        <div
          style={{
            backgroundColor: 'white',
            borderRadius: '0.75rem',
            boxShadow: '0 1px 3px 0 rgb(0 0 0 / 0.1)',
            border: '1px solid #e5e7eb',
            overflow: 'hidden',
          }}
        >
          <div style={{ padding: '2rem', borderBottom: '1px solid #e5e7eb' }}>
            <div style={{ display: 'flex', justifyContent: 'center' }}>
              <button
                onClick={startStreaming}
                disabled={isStreaming}
                style={{
                  padding: '0.75rem 2rem',
                  backgroundColor: isStreaming ? '#9ca3af' : '#2563eb',
                  color: 'white',
                  fontWeight: '500',
                  borderRadius: '0.5rem',
                  border: 'none',
                  cursor: isStreaming ? 'not-allowed' : 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.75rem',
                  fontSize: '1rem',
                }}
                onMouseEnter={(e) => {
                  if (!isStreaming) {
                    e.currentTarget.style.backgroundColor = '#1d4ed8';
                  }
                }}
                onMouseLeave={(e) => {
                  if (!isStreaming) {
                    e.currentTarget.style.backgroundColor = '#2563eb';
                  }
                }}
              >
                {isStreaming && (
                  <div
                    style={{
                      width: '1rem',
                      height: '1rem',
                      border: '2px solid white',
                      borderTop: '2px solid transparent',
                      borderRadius: '50%',
                      animation: 'spin 1s linear infinite',
                    }}
                  />
                )}
                {isStreaming ? 'Streaming...' : 'Start Stream'}
              </button>
            </div>
          </div>

          <div style={{ padding: '2rem' }}>
            <div
              style={{
                backgroundColor: '#111827',
                borderRadius: '0.5rem',
                padding: '1.5rem',
                minHeight: '25rem',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.5rem',
                  marginBottom: '1rem',
                }}
              >
                <div style={{ display: 'flex', gap: '0.375rem' }}>
                  <div
                    style={{
                      width: '0.75rem',
                      height: '0.75rem',
                      backgroundColor: '#ef4444',
                      borderRadius: '50%',
                    }}
                  />
                  <div
                    style={{
                      width: '0.75rem',
                      height: '0.75rem',
                      backgroundColor: '#eab308',
                      borderRadius: '50%',
                    }}
                  />
                  <div
                    style={{
                      width: '0.75rem',
                      height: '0.75rem',
                      backgroundColor: '#22c55e',
                      borderRadius: '50%',
                    }}
                  />
                </div>
                <span style={{ color: '#9ca3af', fontSize: '0.875rem', fontFamily: 'monospace' }}>
                  Output
                </span>
              </div>

              <div
                style={{
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                  color: '#4ade80',
                  whiteSpace: 'pre-wrap',
                  lineHeight: '1.6',
                }}
              >
                {streamContent || (
                  <span style={{ color: '#6b7280' }}>
                    {isStreaming ? 'Initializing stream...' : 'Click "Start Stream" to begin'}
                  </span>
                )}
                {isStreaming && <span style={{ color: '#86efac' }}>█</span>}
              </div>
            </div>
          </div>
        </div>

        <div style={{ textAlign: 'center', marginTop: '2rem' }}>
          <p style={{ fontSize: '0.875rem', color: '#6b7280' }}>
            Powered by Hatchet workflow orchestration
          </p>
        </div>
      </div>

      <style jsx>{`
        @keyframes spin {
          to {
            transform: rotate(360deg);
          }
        }
      `}</style>
    </div>
  );
}
