import { useSidePanel } from '@/hooks/use-side-panel';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import {
  Cross2Icon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from '@radix-ui/react-icons';
import { Button, ReviewedButtonTemp } from './v1/ui/button';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { cn } from '@/lib/utils';

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
      className={cn(
        'flex flex-col border-l border-border bg-background relative flex-shrink-0 overflow-hidden',
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
              'absolute left-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-blue-500/20 transition-colors z-10',
              isResizing && 'bg-blue-500/30',
            )}
            onMouseDown={handleMouseDown}
          />

          <div className="sticky top-0 z-20 flex flex-row w-full justify-between items-center bg-background px-4 pt-4 pb-2">
            <div className="flex flex-row gap-x-2 items-center">
              <ReviewedButtonTemp
                variant="ghost"
                size="sm"
                onClick={goBack}
                disabled={!canGoBack}
                className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0 border"
              >
                <ChevronLeftIcon className="h-4 w-4" />
                <span className="sr-only">Go Back</span>
              </ReviewedButtonTemp>
              <ReviewedButtonTemp
                variant="ghost"
                size="sm"
                onClick={goForward}
                disabled={!canGoForward}
                className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0 border"
              >
                <ChevronRightIcon className="h-4 w-4" />
                <span className="sr-only">Go Forward</span>
              </ReviewedButtonTemp>
            </div>
            <div>
              <ReviewedButtonTemp
                variant="ghost"
                onClick={close}
                className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
              >
                <Cross2Icon className="h-4 w-4" />
                <span className="sr-only">Close</span>
              </ReviewedButtonTemp>
            </div>
          </div>

          <div
            className={cn(
              'flex-1 p-4 overflow-auto side-panel-content',
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
