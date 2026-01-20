// @ts-ignore - ghostty-web doesn't have TypeScript declarations
import { init, Terminal, FitAddon } from 'ghostty-web';
import { useEffect, useRef, RefObject } from 'react';

export interface TerminalScrollCallbacks {
  onTopReached?: () => void;
  onBottomReached?: () => void;
  onInfiniteScroll?: (scrollMetrics: {
    scrollTop: number;
    scrollHeight: number;
    clientHeight: number;
  }) => void;
}

export function useTerminal(
  containerRef: RefObject<HTMLDivElement>,
  logs: string,
  options?: {
    autoScroll?: boolean;
    callbacks?: TerminalScrollCallbacks;
  },
) {
  const terminalRef = useRef<any>(null);
  const fitAddonRef = useRef<any>(null);
  const initializedRef = useRef(false);
  const lastTopCallRef = useRef<number>(0);
  const lastBottomCallRef = useRef<number>(0);
  const lastInfiniteScrollCallRef = useRef<number>(0);
  const firstMountRef = useRef<boolean>(true);
  const logsRef = useRef<string>(logs);
  const isWritingLogsRef = useRef<boolean>(false);

  // Initialize terminal once
  useEffect(() => {
    if (!containerRef.current || initializedRef.current) return;

    let term: any;
    let fitAddon: any;
    let isCleanedUp = false;

    const handleResize = () => {
      if (fitAddon && !isCleanedUp) {
        fitAddon.fit();
      }
    };

    const handleScroll = () => {
      if (!term || isCleanedUp || isWritingLogsRef.current) return;

      const buffer = term.buffer.active;
      const viewportY = term.viewportY; // Current scroll position (from term, not buffer!)
      const rows = term.rows; // Visible rows
      const bufferLength = buffer.length; // Total buffer lines

      const now = Date.now();

      // Calculate scroll metrics for infinite scroll callback
      const scrollTop = viewportY;
      const scrollHeight = bufferLength;
      const clientHeight = rows;

      // Call infinite scroll callback if provided
      if (
        options?.callbacks?.onInfiniteScroll &&
        now - lastInfiniteScrollCallRef.current >= 100
      ) {
        options.callbacks.onInfiniteScroll({
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
        if (options?.callbacks?.onTopReached) {
          options.callbacks.onTopReached();
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
        if (options?.callbacks?.onBottomReached) {
          options.callbacks.onBottomReached();
        }
        lastBottomCallRef.current = now;
      }
    };

    async function initTerminal() {
      try {
        // Initialize WASM
        await init();

        term = new Terminal({
          cursorBlink: false,
          fontSize: 11, // text-xs
          fontFamily: 'Monaco, Menlo, "Courier New", monospace',
          // Dark theme based on Aardvark Blue theme
          // TODO: add light theme
          theme: {
            background: '#1e293b',
            foreground: '#dddddd',
            cursor: '#007acc',
            cursorAccent: '#bfdbfe',
            selectionBackground: '#bfdbfe',
            selectionForeground: '#000000',
            // ANSI colors (palette 0-15)
            black: '#191919',
            red: '#aa342e',
            green: '#4b8c0f',
            yellow: '#dbba00',
            blue: '#1370d3',
            magenta: '#c43ac3',
            cyan: '#008eb0',
            white: '#bebebe',
            brightBlack: '#525252',
            brightRed: '#f05b50',
            brightGreen: '#95dc55',
            brightYellow: '#ffe763',
            brightBlue: '#60a4ec',
            brightMagenta: '#e26be2',
            brightCyan: '#60b6cb',
            brightWhite: '#f7f7f7',
          },
          scrollback: 10000,
        });

        fitAddon = new FitAddon();
        term.loadAddon(fitAddon);

        if (containerRef.current && !isCleanedUp) {
          term.open(containerRef.current);
          fitAddon.fit();
          fitAddon.observeResize();

          terminalRef.current = term;
          fitAddonRef.current = fitAddon;

          initializedRef.current = true;

          // Write any existing logs immediately after initialization
          if (logsRef.current && logsRef.current.trim()) {
            isWritingLogsRef.current = true;
            term.clear();
            const lines = logsRef.current.split('\n');
            for (const line of lines) {
              if (line) {
                term.write(line + '\r\n');
              }
            }

            // Block scroll events briefly
            // TODO: determine if this is necessary
            setTimeout(() => {
              isWritingLogsRef.current = false;
            }, 150);

            if (options?.autoScroll !== false) {
              setTimeout(() => {
                if (terminalRef.current) {
                  terminalRef.current.scrollToBottom();
                }
              }, 100);
              firstMountRef.current = false;
            }
          }

          // Prevent terminal from capturing keyboard events and focus (read-only view)
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

          // Handle scroll events
          term.onScroll(handleScroll);

          // Handle window resize
          window.addEventListener('resize', handleResize);
        }
      } catch (error) {
        console.error('Failed to initialize terminal:', error);
      }
    }

    initTerminal();

    return () => {
      isCleanedUp = true;
      window.removeEventListener('resize', handleResize);
      if (terminalRef.current) {
        terminalRef.current.dispose();
        terminalRef.current = null;
      }
      if (fitAddonRef.current) {
        fitAddonRef.current = null;
      }
      initializedRef.current = false;
    };
  }, [containerRef, options?.callbacks]);

  // Keep logsRef updated
  useEffect(() => {
    logsRef.current = logs;
  }, [logs]);

  // Update terminal content when logs change
  useEffect(() => {
    if (!terminalRef.current) {
      return;
    }

    isWritingLogsRef.current = true;

    const term = terminalRef.current;
    // viewportY === 0 means at bottom in ghostty-web/xterm.js
    const wasAtBottom = term.viewportY === 0;

    term.clear();

    if (logs) {
      const lines = logs.split('\n');
      for (const line of lines) {
        if (line) {
          term.write(line + '\r\n');
        }
      }
    }

    // Block scroll events briefly to prevent false triggers during write
    setTimeout(() => {
      isWritingLogsRef.current = false;
    }, 150);

    // Auto-scroll to bottom if enabled and conditions are met
    if (options?.autoScroll !== false) {
      if (firstMountRef.current || wasAtBottom) {
        setTimeout(() => {
          if (terminalRef.current) {
            terminalRef.current.scrollToBottom();
          }
        }, 100);
        firstMountRef.current = false;
      }
    }
  }, [logs, options?.autoScroll]);
}
