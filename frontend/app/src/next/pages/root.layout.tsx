import { CenterStageLayout } from '@/next/components/layouts/center-stage.layout';
import { ThemeProvider } from '@/next/components/theme-provider';
import { ApiConnectionError } from '@/next/components/errors/api-connection-error';
import useApiMeta from '@/next/hooks/use-api-meta';
import {
  PropsWithChildren,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Outlet } from 'react-router-dom';
import { DocsSheetComponent } from '@/next/components/ui/docs-sheet';
import { Toaster } from '@/next/components/ui/toaster';
import { ToastProvider } from '@/next/hooks/utils/use-toast';
import { SideSheetComponent } from '../components/ui/sheet/side-sheet.layout';
import {
  SideSheetContext,
  useSideSheet,
  useSideSheetState,
} from '@/next/hooks/use-side-sheet';
import { SidebarProvider } from '@/next/components/ui/sidebar';
import { DocsContext, useDocsState } from '../hooks/use-docs-sheet';

function RootContent({ children }: PropsWithChildren) {
  const meta = useApiMeta();
  const docsState = useDocsState();
  const sideSheetState = useSideSheetState();

  const [leftPanelWidth, setLeftPanelWidth] = useState(67);
  const [isDragging, setIsDragging] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const { sheet, close } = useSideSheet();

  const onSideSheetClose = useCallback(() => {
    setLeftPanelWidth(67);
    close();
  }, [close]);

  const isRightPanelOpen = useMemo(() => !!sheet.openProps, [sheet.openProps]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
      if (!isRightPanelOpen) return;
      setIsDragging(true);
      e.preventDefault();
    },
    [isRightPanelOpen],
  );

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging || !containerRef.current) return;

      const containerRect = containerRef.current.getBoundingClientRect();
      const newLeftWidth =
        ((e.clientX - containerRect.left) / containerRect.width) * 100;

      const clampedWidth = Math.min(Math.max(newLeftWidth, 20), 80);
      setLeftPanelWidth(clampedWidth);
    },
    [isDragging],
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
    if (isRightPanelOpen) {
      setLeftPanelWidth(67);
    }
  }, [isRightPanelOpen]);

  return (
    <SideSheetContext.Provider value={sideSheetState}>
      <DocsContext.Provider value={docsState}>
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
          <div ref={containerRef} className="flex flex-1 relative">
            <div
              className="flex flex-col flex-1 overflow-hidden"
              style={{
                width: isRightPanelOpen ? `${leftPanelWidth}%` : '100%',
                transition: isRightPanelOpen ? 'none' : 'width 0.3s ease',
              }}
            >
              {children ?? <Outlet />}
            </div>

            {isRightPanelOpen && (
              <div
                className={`w-1 bg-gray-300 hover:bg-gray-400 cursor-col-resize flex-shrink-0 ${
                  isDragging ? 'bg-gray-400' : ''
                }`}
                onMouseDown={handleMouseDown}
              />
            )}

            {isRightPanelOpen && (
              <div
                className="flex flex-col overflow-hidden"
                style={{ width: `${100 - leftPanelWidth}%` }}
              >
                <SideSheetComponent variant="push" onClose={onSideSheetClose} />
                <DocsSheetComponent
                  sheet={docsState.sheet}
                  onClose={docsState.close}
                />
              </div>
            )}
          </div>
        )}
      </DocsContext.Provider>
    </SideSheetContext.Provider>
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
              <Toaster />
              <RootContent>{children}</RootContent>
            </DocsContext.Provider>
          </SideSheetContext.Provider>
        </SidebarProvider>
      </ThemeProvider>
    </ToastProvider>
  );
}

export default Root;
