import { Button } from '../ui/button';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { useSidePanel } from '@/hooks/use-side-panel';
import { cn } from '@/lib/utils';
import {
  Cross2Icon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from '@radix-ui/react-icons';
import React, { useCallback, useEffect, useRef, useState } from 'react';

const DEFAULT_PANEL_WIDTH = 650;
const MIN_PANEL_WIDTH = 350;

export function SidePanel() {
  const {
    content: maybeContent,
    isOpen,
    close,
    canGoBack,
    canGoForward,
    goBack,
    goForward,
  } = useSidePanel();
  const [storedPanelWidth, setStoredPanelWidth] = useLocalStorageState(
    'sidePanelWidth',
    DEFAULT_PANEL_WIDTH,
  );

  const [isResizing, setIsResizing] = useState(false);
  const [startX, setStartX] = useState(0);
  const [startWidth, setStartWidth] = useState(0);
  const panelRef = useRef<HTMLDivElement>(null);

  const panelWidth = isOpen ? storedPanelWidth : 0;

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsResizing(true);
      setStartX(e.clientX);
      setStartWidth(storedPanelWidth);
    },
    [storedPanelWidth],
  );

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) {
        return;
      }

      const deltaX = startX - e.clientX;
      const newWidth = Math.max(MIN_PANEL_WIDTH, startWidth + deltaX);

      if (newWidth < MIN_PANEL_WIDTH) {
        return;
      }

      setStoredPanelWidth(newWidth);
    },
    [isResizing, startX, startWidth, setStoredPanelWidth],
  );

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = 'col-resize';

      return () => {
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
        document.body.style.cursor = '';
      };
    }
  }, [isResizing, handleMouseMove, handleMouseUp]);

  return (
    <div
      ref={panelRef}
      data-cy="side-panel"
      className={cn(
        'relative flex flex-shrink-0 flex-col overflow-hidden border-l border-border bg-background h-full',
        !isResizing && 'transition-all duration-300 ease-in-out',
      )}
      style={{
        width: panelWidth,
      }}
    >
      {maybeContent && isOpen && (
        <>
          <div
            className={cn(
              'absolute bottom-0 left-0 top-0 z-10 w-1 cursor-col-resize transition-colors hover:bg-blue-500/20',
              isResizing && 'bg-blue-500/30',
            )}
            onMouseDown={handleMouseDown}
          />

          <div className="sticky top-0 z-20 flex w-full flex-row items-center justify-between bg-background px-4 pb-2 pt-4">
            <div className="flex flex-row items-center gap-x-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={goBack}
                disabled={!canGoBack}
                className="flex-shrink-0 rounded-sm border opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              >
                <ChevronLeftIcon className="size-4" />
                <span className="sr-only">Go Back</span>
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={goForward}
                disabled={!canGoForward}
                className="flex-shrink-0 rounded-sm border opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              >
                <ChevronRightIcon className="size-4" />
                <span className="sr-only">Go Forward</span>
              </Button>
            </div>
            <div>
              <Button
                variant="ghost"
                onClick={close}
                className="flex-shrink-0 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              >
                <Cross2Icon className="size-4" />
                <span className="sr-only">Close</span>
              </Button>
            </div>
          </div>

          <div
            data-cy="side-panel-content"
            className={cn(
              'side-panel-content flex-1 overflow-auto p-4',
              isResizing && 'pointer-events-none',
            )}
          >
            {maybeContent.component}
          </div>
        </>
      )}
    </div>
  );
}
