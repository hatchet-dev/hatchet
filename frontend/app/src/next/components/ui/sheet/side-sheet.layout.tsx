import { Sheet, SheetContent, SheetHeader, SheetTitle } from '../sheet';
import {  useSideSheet } from '@/next/hooks/use-side-sheet';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { Cross2Icon, ExternalLinkIcon } from '@radix-ui/react-icons';
import { RunDetailSheet } from '@/next/pages/authenticated/dashboard/runs/detail-sheet/run-detail-sheet';
import { useMemo, useCallback  } from 'react';
import { cn } from '@/lib/utils';
import { WorkerDetails } from '@/next/pages/authenticated/dashboard/worker-services/components/worker-details';
import { useSidebar } from '@/next/components/ui/sidebar';

interface SideSheetProps {
  variant?: 'overlay' | 'push';
}

interface SideSheetContent {
  component: React.ReactNode;
  title: React.ReactNode;
  actions?: React.ReactNode;
}

export function SideSheetComponent({
  variant = 'push',
}: SideSheetProps) {
  const isMobile = useIsMobile();
  const { toggleExpand, sheet, close } = useSideSheet();
  const { isCollapsed } = useSidebar();

  const isOpen = useMemo(() => !!sheet.openProps, [sheet.openProps]);

  const onClose = useCallback(() => {
    close();
  }, [close]);

  const content = useMemo<SideSheetContent | undefined>(() => {
    if (sheet.openProps?.type === 'task-detail') {
      return {
        component: <RunDetailSheet
          isOpen={isOpen}
          onClose={onClose}
          {...sheet.openProps.props}
        />,
        title: "Run Detail",
        actions: <>
          <a
            href={sheet.openProps?.props.detailsLink}
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
            title="Open in new tab"
          >
            <ExternalLinkIcon className="h-4 w-4" />
            <span className="sr-only">Open in new tab</span>
          </a>
        </>
      }
    }

    if (sheet.openProps?.type === 'worker-detail') {
      return {
        component: <WorkerDetails 
          {...sheet.openProps.props}  
        />,
        title: "Worker Detail",
        actions: <>
          <a
            // href={ROUTES.workerServices.detail(sheet.openProps?.props.serviceName, sheet.openProps?.props.workerId)}
            target="_blank" 
            rel="noopener noreferrer"
            className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
            title="Open in new tab"
          >
            <ExternalLinkIcon className="h-4 w-4" />
            <span className="sr-only">Open in new tab</span>
          </a>
        </>
      }
    }

    return undefined;
  }, [sheet]);

  // If using push variant, render as a side panel instead of using Sheet
  if (variant === 'push' && !isMobile) {
    return (
      <div
        className={`
          h-full min-h-screen bg-background border-l border-border
          transition-all duration-300 ease-in-out relative
          ${isOpen ? (sheet.isExpanded ? 'lg:w-[800px] md:w-[600px] w-[400px]' : 'lg:w-[500px] md:w-[350px] w-[250px]') : 'w-0 overflow-hidden'}
        `}
      >
        {isOpen && (
          <>
            <button
              onClick={toggleExpand}
              className={cn(
                "absolute inset-y-0 -left-2 z-20 w-4 transition-w ease-linear after:absolute after:inset-y-0 after:left-1/2 after:w-[2px] hover:after:bg-border",
                sheet.isExpanded ? "cursor-e-resize" : "cursor-w-resize"
              )}
              title="Toggle width"
            />
            <div className="h-full min-h-screen flex flex-col overflow-hidden">
              <div className={cn(
                'flex justify-between items-center border-b shrink-0 transition-h duration-300',
                isMobile ? 'h-16 px-4' : isCollapsed ? 'h-12 px-8' : 'h-16 px-8'
              )}>
                <h2 className="text-lg font-semibold truncate pr-2">
                  {content?.title}
                </h2>
                <div className="flex items-center gap-2">
                  {content?.actions}
                  <button
                    onClick={onClose}
                    className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
                  >
                    <Cross2Icon className="h-4 w-4" />
                    <span className="sr-only">Close</span>
                  </button>
                </div>
              </div>
              <div className="flex-1 overflow-y-auto relative">
                {content?.component}
              </div>
            </div>
          </>
        )}
      </div>
    );
  }

  // Fall back to the overlay variant if not using push
  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side="right"
        className={`p-4 md:p-6 ${isOpen ? (sheet.isExpanded ? 'w-[min(800px,90vw)]' : 'w-[min(500px,90vw)]') : 'w-0 overflow-hidden'} h-screen`}
      >
        <SheetHeader className="mb-4 pr-8">
          <div className="flex justify-between items-center">
            <SheetTitle className="truncate">{content?.title}</SheetTitle>
            {content?.actions}
          </div>
        </SheetHeader>
        <div className="flex-1 relative overflow-hidden">
          {content?.component}
        </div>
      </SheetContent>
    </Sheet>
  );
}
