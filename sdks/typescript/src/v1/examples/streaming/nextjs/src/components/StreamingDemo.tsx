'use client';

import { useState } from 'react';

export default function StreamingDemo() {
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamContent, setStreamContent] = useState('');
  const [error, setError] = useState<string | null>(null);

  const startStreaming = async () => {
    if (isStreaming) return;

    setIsStreaming(true);
    setStreamContent('');
    setError(null);

    try {
      const response = await fetch('/api/stream');

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error('Failed to get response reader');
      }

      const decoder = new TextDecoder();

      try {
        while (true) {
          const { done, value } = await reader.read();

          if (done) break;

          const chunk = decoder.decode(value, { stream: true });
          setStreamContent((prev) => prev + chunk);
        }
      } finally {
        reader.releaseLock();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setIsStreaming(false);
    }
  };

  const clearContent = () => {
    setStreamContent('');
    setError(null);
  };

  return (
    <div className="w-full max-w-2xl mx-auto">
      <div className="bg-white rounded-2xl shadow-xl border border-slate-200 overflow-hidden">
        <div className="bg-gradient-to-r from-blue-600 to-indigo-600 px-8 py-6">
          <h1 className="text-2xl font-bold text-white mb-2">Hatchet Streaming</h1>
          <p className="text-blue-100 text-sm">Real-time workflow streaming with Next.js</p>
        </div>

        <div className="p-8 space-y-6">
          <div className="flex gap-3 justify-center">
            <button
              onClick={startStreaming}
              disabled={isStreaming}
              className="px-8 py-3 bg-gradient-to-r from-blue-600 to-indigo-600 text-white font-semibold rounded-lg hover:from-blue-700 hover:to-indigo-700 disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed transition-all duration-200 shadow-lg hover:shadow-xl transform hover:-translate-y-0.5 disabled:transform-none flex items-center gap-2"
            >
              {isStreaming && (
                <div className="animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full"></div>
              )}
              {isStreaming ? 'Streaming...' : 'Start Stream'}
            </button>

            <button
              onClick={clearContent}
              disabled={isStreaming}
              className="px-6 py-3 bg-slate-500 text-white font-semibold rounded-lg hover:bg-slate-600 disabled:bg-gray-300 disabled:cursor-not-allowed transition-all duration-200"
            >
              Clear
            </button>
          </div>

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg">
              <div className="flex items-center gap-2">
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
                <span className="font-medium">Error:</span> {error}
              </div>
            </div>
          )}

          <div className="relative">
            <div className="bg-slate-900 rounded-lg p-6 min-h-[300px] overflow-hidden">
              <div className="flex items-center gap-2 mb-4">
                <div className="flex gap-1.5">
                  <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                  <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                  <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                </div>
                <span className="text-slate-400 text-sm font-mono">Terminal</span>
              </div>

              <div className="font-mono text-sm text-green-400 whitespace-pre-wrap leading-relaxed">
                {streamContent || (
                  <span className="text-slate-500">
                    {isStreaming ? (
                      <div className="flex items-center gap-2">
                        <div className="animate-pulse">▊</div>
                        Initializing stream...
                      </div>
                    ) : (
                      'Ready to stream. Click "Start Stream" to begin...'
                    )}
                  </span>
                )}
                {isStreaming && streamContent && (
                  <span className="animate-pulse text-green-300">▊</span>
                )}
              </div>
            </div>
          </div>

          {isStreaming && (
            <div className="flex items-center justify-center gap-3 text-sm text-slate-600">
              <div className="flex items-center gap-2">
                <div className="relative">
                  <div className="w-2 h-2 bg-green-500 rounded-full animate-ping"></div>
                  <div className="absolute inset-0 w-2 h-2 bg-green-500 rounded-full"></div>
                </div>
                <span>Live streaming from Hatchet workflow</span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
