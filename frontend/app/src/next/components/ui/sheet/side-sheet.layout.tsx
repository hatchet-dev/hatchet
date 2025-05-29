import { Sheet, SheetContent, SheetHeader, SheetTitle } from '../sheet';
import { useSideSheet } from '@/next/hooks/use-side-sheet';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { Cross2Icon } from '@radix-ui/react-icons';
import {
  RunDetailSheet,
  RunDetailSheetSerializableProps,
} from '@/next/pages/authenticated/dashboard/runs/detail-sheet/run-detail-sheet';
import { useMemo, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { Button } from '../button';
import { useDocs } from '@/next/hooks/use-docs-sheet';
import { useSidePanel } from '@/next/hooks/use-side-panel';

interface SideSheetProps {
  variant?: 'overlay' | 'push';
  onClose: () => void;
  isOpen: boolean;
  props: SidePanelProps;
}

interface SideSheetContent {
  component: React.ReactNode;
  title: React.ReactNode;
  actions?: React.ReactNode;
}

type SidePanelProps =
  | {
      type: 'docs';
      url: string;
      title: string;
    }
  | {
      type: 'task-detail';
      props: RunDetailSheetSerializableProps;
    };

type SidePanelContent = {
  content: React.ReactNode;
  title: string;
  actions?: React.ReactNode;
};

const useSidePanelContent = (
  props: SidePanelProps,
): SidePanelContent | null => {
  const { sheet } = useSideSheet();

  switch (props.type) {
    case 'task-detail':
      if (
        !sheet ||
        !sheet.openProps ||
        sheet.openProps.type !== 'task-detail'
      ) {
        return null;
      }

      return {
        content: <RunDetailSheet {...sheet.openProps.props} />,
        title: 'Run Detail',
      };
    case 'docs':
      return {
        content: (
          <iframe
            src={props.url}
            className="absolute inset-0 w-full h-full rounded-md border"
            title={`Documentation: ${props.title}`}
            loading="lazy"
          />
        ),
        title: props.title,
      };
  }
};

export function SideSheetComponent({
  variant = 'push',
  onClose,
}: SideSheetProps) {
  const isMobile = useIsMobile();
  const { sheet } = useSideSheet();
  const { close: closeDocsSheet } = useDocs();

  const isOpen = useMemo(() => !!sheet.openProps, [sheet.openProps]);

  useEffect(() => {
    if (isOpen) {
      closeDocsSheet();
    }
  }, [isOpen]);

  const content = useMemo<SideSheetContent | undefined>(() => {
    if (sheet.openProps?.type === 'task-detail') {
      return {
        component: <RunDetailSheet {...sheet.openProps.props} />,
        title: 'Run Detail',
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
                isMobile ? 'h-16 px-4' : 'h-12 px-4',
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
      </div>
    );
  }

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side="right"
        className="w-[min(500px,90vw)] sm:w-[min(800px,90vw)]"
      >
        {content && (
          <>
            <SheetHeader>
              <div className="flex justify-between items-center">
                <SheetTitle className="truncate">{content.title}</SheetTitle>
                <div className="flex items-center gap-2">{content.actions}</div>
              </div>
            </SheetHeader>
            <div className="flex-1 overflow-y-auto mt-4">
              {content.component}
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

export function SidePanel() {
  const { content: maybeContent, isOpen, close } = useSidePanel();

  if (!maybeContent) {
    return null;
  }

  const { component, title, actions } = maybeContent;

  return (
    <div>
      {isOpen && (
        <div className="flex flex-col h-screen">
          <div
            className={
              'flex flex-row w-full justify-between items-center border-b bg-background h-16 px-4 md:p-12'
            }
          >
            <h2 className="text-lg font-semibold truncate pr-2">{title}</h2>
            <div className="flex items-center gap-2">
              {actions}
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

          {component}
        </div>
      )}
    </div>
  );
}
