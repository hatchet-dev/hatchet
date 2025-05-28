import { Sheet, SheetContent, SheetHeader, SheetTitle } from '../sheet';
import { useSideSheet } from '@/next/hooks/use-side-sheet';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { Cross2Icon, ExternalLinkIcon } from '@radix-ui/react-icons';
import { RunDetailSheet } from '@/next/pages/authenticated/dashboard/runs/detail-sheet/run-detail-sheet';
import { useMemo, useCallback } from 'react';
import { cn } from '@/lib/utils';
import { WorkerDetails } from '@/next/pages/authenticated/dashboard/workers/components/worker-details';
import { useSidebar } from '@/next/components/ui/sidebar';
import { Button } from '../button';

interface SideSheetProps {
  variant?: 'overlay' | 'push';
  onClose: () => void;
}

interface SideSheetContent {
  component: React.ReactNode;
  title: React.ReactNode;
  actions?: React.ReactNode;
}

export function SideSheetComponent({
  variant = 'push',
  onClose,
}: SideSheetProps) {
  const isMobile = useIsMobile();
  const { sheet } = useSideSheet();
  const { isCollapsed } = useSidebar();

  const isOpen = useMemo(() => !!sheet.openProps, [sheet.openProps]);

  const content = useMemo<SideSheetContent | undefined>(() => {
    if (sheet.openProps?.type === 'task-detail') {
      return {
        component: (
          <RunDetailSheet
            isOpen={isOpen}
            onClose={onClose}
            {...sheet.openProps.props}
          />
        ),
        title: 'Run Detail',
        actions: (
          <>
            {/* <a
            href={sheet.openProps?.props.detailsLink}
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
            title="Open in new tab"
          >
            <ExternalLinkIcon className="h-4 w-4" />
            <span className="sr-only">Open in new tab</span>
          </a> */}
          </>
        ),
      };
    }

    if (sheet.openProps?.type === 'worker-detail') {
      return {
        component: <WorkerDetails {...sheet.openProps.props} />,
        title: 'Worker Detail',
        actions: (
          <>
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
        ),
      };
    }

    return undefined;
  }, [sheet, isOpen, onClose]);

  if (variant === 'push' && !isMobile) {
    return (
      <div>
        {isOpen && content && (
          <div className="flex flex-col h-screen">
            <div
              className={cn(
                'flex flex-row w-full justify-between items-center border-b bg-background',
                isMobile
                  ? 'h-16 px-4'
                  : isCollapsed
                    ? 'h-12 px-4'
                    : 'h-12 px-4',
              )}
            >
              <h2 className="text-lg font-semibold truncate pr-2">
                {content.title}
              </h2>
              <div className="flex items-center gap-2">
                {content.actions}
                <Button
                  variant="ghost"
                  onClick={onClose}
                  className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
                >
                  <Cross2Icon className="h-4 w-4" />
                  <span className="sr-only">Close</span>
                </Button>
              </div>
            </div>

            {content.component}
          </div>
        )}

        {(!isOpen || !content) && (
          <div className="flex-1 flex items-center justify-center text-muted-foreground">
            <p>No content selected</p>
          </div>
        )}
      </div>
    );
  }

  // return (
  //   <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
  //     <SheetContent
  //       side="right"
  //       className="w-[min(500px,90vw)] sm:w-[min(800px,90vw)]"
  //     >
  //       {content && (
  //         <>
  //           <SheetHeader>
  //             <div className="flex justify-between items-center">
  //               <SheetTitle className="truncate">{content.title}</SheetTitle>
  //               <div className="flex items-center gap-2">{content.actions}</div>
  //             </div>
  //           </SheetHeader>
  //           <div className="flex-1 overflow-y-auto mt-4">
  //             {content.component}
  //           </div>
  //         </>
  //       )}
  //     </SheetContent>
  //   </Sheet>
  // );
}
