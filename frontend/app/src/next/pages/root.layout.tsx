import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';
import { ThemeProvider } from '@/next/components/theme-provider';
import { ApiConnectionError } from '@/next/components/errors/api-connection-error';
import useApiMeta from '@/next/hooks/use-api-meta';
import {
  PropsWithChildren,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react';
import { Outlet } from 'react-router-dom';
import { Toaster } from '@/next/components/ui/toaster';
import { ToastProvider } from '@/next/hooks/utils/use-toast';
import { SidePanel } from '../components/ui/sheet/side-sheet.layout';
import {
  SideSheetContext,
  useSideSheetState,
} from '@/next/hooks/use-side-sheet';
import { SidebarProvider } from '@/next/components/ui/sidebar';
import { DocsContext, useDocsState } from '../hooks/use-docs-sheet';
import { SidePanelProvider, useSidePanel } from '../hooks/use-side-panel';

function usePersistentPanelWidth(defaultWidth: number = 67) {
  const [leftPanelWidth, setLeftPanelWidth] = useState(() => {
    try {
      const savedWidth = localStorage.getItem('leftPanelWidth');
      return savedWidth ? parseFloat(savedWidth) : defaultWidth;
    } catch {
      return defaultWidth;
    }
  });

  const updateLeftPanelWidth = useCallback((width: number) => {
    const clampedWidth = Math.min(Math.max(width, 20), 80);
    setLeftPanelWidth(clampedWidth);

    try {
      localStorage.setItem('leftPanelWidth', clampedWidth.toString());
    } catch {
      // Silently fail if localStorage is not available
    }
  }, []);

  return [leftPanelWidth, updateLeftPanelWidth] as const;
}

function RootContent({ children }: PropsWithChildren) {
  const meta = useApiMeta();

  const [leftPanelWidth, setLeftPanelWidth] = usePersistentPanelWidth(67);
  const [isDragging, setIsDragging] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const { isOpen: isRightPanelOpen } = useSidePanel();

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
      if (!isRightPanelOpen) {
        return;
      }
      setIsDragging(true);
      e.preventDefault();
    },
    [isRightPanelOpen],
  );

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging || !containerRef.current) {
        return;
      }

      const containerRect = containerRef.current.getBoundingClientRect();
      const newLeftWidth =
        ((e.clientX - containerRect.left) / containerRect.width) * 100;

      setLeftPanelWidth(newLeftWidth);
    },
    [isDragging, setLeftPanelWidth],
  );

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);

      return () => {
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [isDragging, handleMouseMove, handleMouseUp]);

  useEffect(() => {
    if (isRightPanelOpen && leftPanelWidth === 100) {
      try {
        const savedWidth = localStorage.getItem('leftPanelWidth');
        if (savedWidth) {
          setLeftPanelWidth(parseFloat(savedWidth));
        }
      } catch {
        setLeftPanelWidth(67);
      }
    }
  }, [isRightPanelOpen, leftPanelWidth, setLeftPanelWidth]);

  return (
    <>
      {meta.isLoading ? (
        <CenterStageLayout>
          <div className="flex h-screen w-full items-center justify-center"></div>
        </CenterStageLayout>
      ) : meta.hasFailed ? (
        <ApiConnectionError
          retryInterval={meta.refetchInterval}
          errorMessage={meta.hasFailed.message}
        />
      ) : (
        <div
          ref={containerRef}
          className="h-screen flex flex-1 relative overflow-hidden"
        >
          {/* Left panel - main content */}
          <div
            className="h-full overflow-auto flex-shrink-0"
            style={{
              width: isRightPanelOpen ? `${leftPanelWidth}%` : '100%',
            }}
          >
            {children ?? <Outlet />}
          </div>

          {/* Resize handle */}
          {isRightPanelOpen && (
            <div
              className="relative w-1 flex-shrink-0 group cursor-col-resize"
              onMouseDown={handleMouseDown}
            >
              <div className="absolute inset-0 -left-2 -right-2 w-5" />

              <div
                className={`
                  h-full w-full transition-colors duration-150
                  ${
                    isDragging
                      ? 'bg-blue-500'
                      : 'bg-border hover:bg-blue-400/50 group-hover:bg-blue-400/50'
                  }
                `}
              />

              <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity duration-150">
                <div className="flex flex-col space-y-1">
                  <div className="w-0.5 h-0.5 bg-white/60 rounded-full" />
                  <div className="w-0.5 h-0.5 bg-white/60 rounded-full" />
                  <div className="w-0.5 h-0.5 bg-white/60 rounded-full" />
                </div>
              </div>
            </div>
          )}

          {/* Right panel - side sheet */}
          {isRightPanelOpen && (
            <div className="flex flex-col overflow-hidden flex-1">
              <SidePanel />
            </div>
          )}
        </div>
      )}
    </>
  );
}

function Root({ children }: PropsWithChildren) {
  const docsState = useDocsState();
  const sideSheetState = useSideSheetState();

  return (
    <ToastProvider>
      <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
        <SidebarProvider>
          <SideSheetContext.Provider value={sideSheetState}>
            <DocsContext.Provider value={docsState}>
              <SidePanelProvider>
                <Toaster />
                <RootContent>{children}</RootContent>
              </SidePanelProvider>
            </DocsContext.Provider>
          </SideSheetContext.Provider>
        </SidebarProvider>
      </ThemeProvider>
    </ToastProvider>
  );
}

export default Root;
