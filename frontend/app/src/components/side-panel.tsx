import { useSidePanel } from '@/hooks/use-side-panel';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';
import { Cross2Icon } from '@radix-ui/react-icons';
import { Button } from './v1/ui/button';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { cn } from '@/lib/utils';

const DEFAULT_PANEL_WIDTH = 650;
const MIN_PANEL_WIDTH = 200;

export function SidePanel() {
  const { content: maybeContent, isOpen, close } = useSidePanel();
  const [panelWidth, setPanelWidth] = useLocalStorageState(
    'sidePanelWidth',
    DEFAULT_PANEL_WIDTH,
  );

  const [isResizing, setIsResizing] = useState(false);
  const [startX, setStartX] = useState(0);
  const [startWidth, setStartWidth] = useState(0);
  const panelRef = useRef<HTMLDivElement>(null);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsResizing(true);
      setStartX(e.clientX);
      setStartWidth(panelWidth);
    },
    [panelWidth],
  );

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) {
        return;
      }

      const deltaX = startX - e.clientX;
      const newWidth = Math.max(MIN_PANEL_WIDTH, startWidth + deltaX);

      console.log({
        startX,
        currentX: e.clientX,
        deltaX,
        startWidth,
        newWidth,
        actualChange: newWidth - startWidth,
      });

      setPanelWidth(newWidth);
    },
    [isResizing, startX, startWidth, setPanelWidth],
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

  if (!maybeContent || !isOpen) {
    return null;
  }

  return (
    <div
      ref={panelRef}
      className="flex flex-col border-l border-border bg-background relative flex-shrink-0"
      style={{ width: panelWidth }}
    >
      <div
        className={cn(
          'absolute left-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-blue-500/20 transition-colors z-10',
          isResizing && 'bg-blue-500/30',
        )}
        onMouseDown={handleMouseDown}
      />

      <div className="flex flex-row w-full justify-between items-center border-b bg-background h-16 px-4 md:px-6">
        <h2 className="text-lg font-semibold truncate pr-2">
          {maybeContent.title}
        </h2>
        <div className="flex items-center gap-2">
          {!maybeContent.isDocs && maybeContent.actions}
          <Button
            variant="ghost"
            onClick={close}
            className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
          >
            <Cross2Icon className="h-4 w-4" />
            <span className="sr-only">Close</span>
          </Button>
        </div>
      </div>

      <div className="flex-1  p-4 overflow-auto">{maybeContent.component}</div>
    </div>
  );
}
