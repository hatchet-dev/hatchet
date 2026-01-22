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
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
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
          scrollback: 10 * 1024 * 1024, // 10MB (these are bytes, not lines)
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
        console.error(
          'Failed to initialize terminal in useTerminal hook:',
          error,
        );
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [containerRef, options?.theme]);

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

    const isInitialLoad = isFreshTerminalRef.current;

    // Detect if older logs were appended (newest-first means older logs are at the end)
    // If new logs string starts with the old string, we can just append the difference
    const isAppendingOlderLogs =
      !isInitialLoad &&
      lastWrittenLogsRef.current.length > 0 &&
      logs.startsWith(lastWrittenLogsRef.current);

    if (isAppendingOlderLogs) {
      // Save scroll position relative to the TOP of the buffer
      // viewportY is distance from bottom, so we convert to distance from top
      const buffer = term.buffer.active;
      const oldBufferLength = buffer.length;
      const oldMaxScroll = Math.max(0, oldBufferLength - term.rows);
      const distanceFromTop = oldMaxScroll - term.viewportY;

      // Only write the new (older) logs that were appended
      const newLogs = logs.slice(lastWrittenLogsRef.current.length);
      if (newLogs) {
        const lines = newLogs.split('\n');
        for (let i = 0; i < lines.length; i++) {
          const line = lines[i];
          const isLastLine = i === lines.length - 1;
          term.write('\x1b[0m' + line + '\x1b[K' + (isLastLine ? '' : '\r\n'));
        }
      }

      // Restore scroll position - keep isWritingRef true until scroll is restored
      // to prevent scroll handler from triggering more fetches
      lastWrittenLogsRef.current = logs;
      isFreshTerminalRef.current = false;

      setTimeout(() => {
        // Scroll to the target position
        term.scrollToTop();
        if (distanceFromTop > 0) {
          term.scrollLines(-distanceFromTop);
        }
        // Only release the writing lock after scroll is restored
        isWritingRef.current = false;
      }, 0);

      return; // Exit early since we handled everything above
    } else {
      // Full rewrite needed (initial load or new logs prepended)
      // Skip reset() for freshly initialized terminals to avoid WASM memory pollution
      if (!isInitialLoad) {
        term.reset();
      }

      // Write all log content
      if (logs) {
        const lines = logs.split('\n');
        for (let i = 0; i < lines.length; i++) {
          const line = lines[i];
          const isLastLine = i === lines.length - 1;
          // We use \x1b[0m (reset) and \x1b[K (clear to end of line) as workarounds:
          // 1. \x1b[0m resets any lingering ANSI state (colors, styles) from previous lines
          // 2. \x1b[K clears garbage data due to ghostty-web memory pollution bug
          term.write('\x1b[0m' + line + '\x1b[K' + (isLastLine ? '' : '\r\n'));
        }
      }

      // Only scroll to top on initial load to show newest logs
      // Use requestAnimationFrame to ensure content is rendered before scrolling
      if (isInitialLoad) {
        requestAnimationFrame(() => {
          term.scrollToTop();
        });
      }
    }

    lastWrittenLogsRef.current = logs;
    isFreshTerminalRef.current = false;

    isWritingRef.current = false;
  }, [logs, terminalInitialized]);

  return { terminalInitialized };
}
