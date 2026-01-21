import { TerminalTheme } from '../terminalThemes';
import { init, Terminal, FitAddon } from 'ghostty-web';
import { useEffect, useRef, RefObject, useState } from 'react';

export interface TerminalScrollCallbacks {
  onTopReached?: () => void;
  onBottomReached?: () => void;
  onInfiniteScroll?: (scrollMetrics: {
    scrollTop: number;
    scrollHeight: number;
    clientHeight: number;
  }) => void;
}

// Initialize WASM once globally, not per terminal instance
let wasmInitialized = false;
let wasmInitPromise: Promise<void> | null = null;

async function ensureWasmInit() {
  if (wasmInitialized) {
    return;
  }
  if (wasmInitPromise) {
    return wasmInitPromise;
  }
  wasmInitPromise = init().then(() => {
    wasmInitialized = true;
  });
  return wasmInitPromise;
}

export function useTerminal(
  containerRef: RefObject<HTMLDivElement>,
  logs: string,
  options?: {
    autoScroll?: boolean;
    callbacks?: TerminalScrollCallbacks;
    theme?: TerminalTheme;
    onInitError?: () => void;
  },
) {
  const [terminalInitialized, setTerminalInitialized] = useState(false);
  const terminalRef = useRef<any>(null);
  const fitAddonRef = useRef<any>(null);
  const initializedRef = useRef(false);
  const isWritingRef = useRef(false);
  const callbacksRef = useRef<TerminalScrollCallbacks | undefined>(
    options?.callbacks,
  );
  const lastTopCallRef = useRef<number>(0);
  const lastBottomCallRef = useRef<number>(0);
  const lastInfiniteScrollCallRef = useRef<number>(0);
  const lastWrittenLogsRef = useRef<string>('');
  const isFreshTerminalRef = useRef<boolean>(true);

  // Update callbacks ref when they change (without reinitializing terminal)
  useEffect(() => {
    callbacksRef.current = options?.callbacks;
  }, [options?.callbacks]);

  // Initialize terminal once and keep it alive
  useEffect(() => {
    if (!containerRef.current || initializedRef.current) {
      return;
    }

    const handleResize = () => {
      if (fitAddonRef.current && !isWritingRef.current) {
        fitAddonRef.current.fit();
      }
    };

    const handleScroll = () => {
      const term = terminalRef.current;
      if (!term || isWritingRef.current) {
        return;
      }

      const buffer = term.buffer.active;
      const viewportY = term.viewportY;
      const rows = term.rows;
      const bufferLength = buffer.length;

      const now = Date.now();

      // Calculate scroll metrics for infinite scroll callback
      const scrollTop = viewportY;
      const scrollHeight = bufferLength;
      const clientHeight = rows;

      // Call infinite scroll callback if provided
      if (
        callbacksRef.current?.onInfiniteScroll &&
        now - lastInfiniteScrollCallRef.current >= 100
      ) {
        callbacksRef.current.onInfiniteScroll({
          scrollTop,
          scrollHeight,
          clientHeight,
        });
        lastInfiniteScrollCallRef.current = now;
      }

      // Only trigger top/bottom callbacks if there's actually scrollable content
      const hasScrollableContent = bufferLength > rows;

      // In ghostty-web/xterm.js, viewportY is inverted:
      // viewportY === 0 means at BOTTOM (most recent lines)
      // viewportY === maxScroll means at TOP (oldest lines in scrollback)
      const maxScroll = bufferLength - rows;

      // Detect if at top (scrolled all the way up to oldest lines)
      const isAtTop = viewportY >= maxScroll;
      if (
        hasScrollableContent &&
        isAtTop &&
        now - lastTopCallRef.current >= 1000
      ) {
        if (callbacksRef.current?.onTopReached) {
          callbacksRef.current.onTopReached();
        }
        lastTopCallRef.current = now;
      }

      // Detect if at bottom (scrolled all the way down to most recent lines)
      const isAtBottom = viewportY === 0;
      if (
        hasScrollableContent &&
        isAtBottom &&
        now - lastBottomCallRef.current >= 1000
      ) {
        if (callbacksRef.current?.onBottomReached) {
          callbacksRef.current.onBottomReached();
        }
        lastBottomCallRef.current = now;
      }
    };

    async function initTerminal() {
      try {
        await ensureWasmInit();

        const term = new Terminal({
          cursorBlink: false,
          fontSize: 11,
          fontFamily: 'Monaco, Menlo, "Courier New", monospace',
          theme: options?.theme || {
            background: '#1e293b',
            foreground: '#dddddd',
          },
          scrollback: 10000000,
          rows: 24,
          cols: 80,
        });

        if (!containerRef.current) {
          return;
        }

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.open(containerRef.current);
        fitAddon.fit();
        fitAddon.observeResize();

        terminalRef.current = term;
        fitAddonRef.current = fitAddon;
        initializedRef.current = true;

        // Register scroll event handler
        term.onScroll(handleScroll);

        // Prevent terminal from capturing keyboard events and focus (read-only view)
        if (containerRef.current) {
          const textarea = containerRef.current.querySelector('textarea');
          const canvas = containerRef.current.querySelector('canvas');

          if (textarea) {
            textarea.setAttribute('disabled', 'true');
            textarea.setAttribute('tabindex', '-1');
            textarea.setAttribute('aria-hidden', 'true');
            textarea.style.pointerEvents = 'none';
            textarea.addEventListener('focus', (e) => {
              (e.target as HTMLElement).blur();
            });
          }

          if (canvas) {
            canvas.setAttribute('tabindex', '-1');
            canvas.style.outline = 'none';
            canvas.addEventListener('focus', (e) => {
              (e.target as HTMLElement).blur();
            });
            // Prevent canvas from capturing keyboard events
            canvas.addEventListener(
              'keydown',
              (e) => {
                e.stopPropagation();
                e.preventDefault();
              },
              true,
            );
          }
        }

        window.addEventListener('resize', handleResize);

        // Mark as fresh terminal so we don't call reset() on first write
        isFreshTerminalRef.current = true;

        setTerminalInitialized(true);
      } catch (error) {
        options?.onInitError?.();
      }
    }

    initTerminal();

    return () => {
      window.removeEventListener('resize', handleResize);
      if (terminalRef.current) {
        terminalRef.current.dispose();
        terminalRef.current = null;
      }
      initializedRef.current = false;
      // Clear last written logs so they get rewritten when terminal reinitializes
      lastWrittenLogsRef.current = '';
    };
  }, [containerRef, options]);

  // Update terminal content when logs change
  useEffect(() => {
    if (!terminalRef.current || !terminalInitialized) {
      return;
    }

    const term = terminalRef.current;

    // Check if logs have actually changed
    if (logs === lastWrittenLogsRef.current) {
      return;
    }

    // Lock to prevent fit from running during reset/write
    isWritingRef.current = true;

    // Detect if logs were appended (starts with previous logs)
    const isAppend =
      lastWrittenLogsRef.current.length > 0 &&
      logs &&
      logs.startsWith(lastWrittenLogsRef.current);

    if (isAppend) {
      // Only write the new part
      const newLogs = logs.slice(lastWrittenLogsRef.current.length);

      if (newLogs) {
        const lines = newLogs.split('\n');
        for (let i = 0; i < lines.length; i++) {
          const line = lines[i];
          const isLastLine = i === lines.length - 1;
          // TODO: We use \x1b[0m (reset) and \x1b[K (clear to end of line) as workarounds:
          // 1. \x1b[0m resets any lingering ANSI state (colors, styles) from previous lines
          // 2. \x1b[K clears garbage data due to ghostty-web memory pollution bug
          // When terminal instances are disposed and recreated, WASM memory isn't properly cleared,
          // causing old buffer data to appear in new terminal instances.
          // This should be reported to ghostty-web maintainers as a memory management bug:
          // - dispose() doesn't fully clear WASM buffers
          // - graphemeBufferPtr is never freed (see free() implementation)
          // - Sequential terminal instances show memory pollution from previous instances
          term.write('\x1b[0m' + line + '\x1b[K' + (isLastLine ? '' : '\r\n'));
        }
      }
    } else {
      // Logs completely replaced - reset and write all
      // Skip reset() for freshly initialized terminals to avoid WASM memory pollution
      if (!isFreshTerminalRef.current) {
        term.reset();
      }

      // Write actual log content, split by newlines
      if (logs) {
        const lines = logs.split('\n');
        for (let i = 0; i < lines.length; i++) {
          const line = lines[i];
          const isLastLine = i === lines.length - 1;
          term.write('\x1b[0m' + line + '\x1b[K' + (isLastLine ? '' : '\r\n'));
        }
      }
    }

    lastWrittenLogsRef.current = logs;
    isFreshTerminalRef.current = false;

    // Unlock after a brief delay to ensure write is fully processed
    setTimeout(() => {
      isWritingRef.current = false;
    }, 100);
  }, [logs, terminalInitialized]);

  return { terminalInitialized };
}
