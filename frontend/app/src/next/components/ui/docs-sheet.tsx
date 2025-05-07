import { Sheet, SheetContent, SheetHeader, SheetTitle } from './sheet';
import { DocsSheet } from '@/next/hooks/use-docs-sheet';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { Cross2Icon, ExternalLinkIcon } from '@radix-ui/react-icons';

interface DocsSheetProps {
  sheet: DocsSheet;
  onClose: () => void;
  variant?: 'overlay' | 'push';
}

export function DocsSheetComponent({
  sheet,
  onClose,
  variant = 'push',
}: DocsSheetProps) {
  const isMobile = useIsMobile();

  // If using push variant, render as a side panel instead of using Sheet
  if (variant === 'push' && !isMobile) {
    return (
      <div
        className={`
          h-full min-h-screen bg-background border-l border-border
          transition-all duration-300 ease-in-out
          ${sheet.isOpen ? 'lg:w-[500px] md:w-[350px] w-[250px]' : 'w-0 overflow-hidden'}
        `}
      >
        {sheet.isOpen && (
          <div className="h-full min-h-screen flex flex-col p-4 md:p-6 overflow-hidden">
            <div className="flex justify-between items-center mb-4 shrink-0">
              <h2 className="text-lg font-semibold truncate pr-2">
                {sheet.title}
              </h2>
              <div className="flex items-center gap-2">
                {sheet.url && (
                  <a
                    href={sheet.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
                    title="Open in new tab"
                  >
                    <ExternalLinkIcon className="h-4 w-4" />
                    <span className="sr-only">Open in new tab</span>
                  </a>
                )}
                <button
                  onClick={onClose}
                  className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
                >
                  <Cross2Icon className="h-4 w-4" />
                  <span className="sr-only">Close</span>
                </button>
              </div>
            </div>
            <div className="flex-1 overflow-hidden relative">
              {sheet.url && (
                <iframe
                  src={sheet.url}
                  className="absolute inset-0 w-full h-full rounded-md border"
                  title={`Documentation: ${sheet.title}`}
                  loading="lazy"
                />
              )}
            </div>
          </div>
        )}
      </div>
    );
  }

  // Fall back to the overlay variant if not using push
  return (
    <Sheet open={sheet.isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side="right"
        className="p-4 md:p-6 w-[min(500px,90vw)] h-screen"
      >
        <SheetHeader className="mb-4 pr-8">
          <div className="flex justify-between items-center">
            <SheetTitle className="truncate">{sheet.title}</SheetTitle>
            {sheet.url && (
              <a
                href={sheet.url}
                target="_blank"
                rel="noopener noreferrer"
                className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
                title="Open in new tab"
              >
                <ExternalLinkIcon className="h-4 w-4" />
                <span className="sr-only">Open in new tab</span>
              </a>
            )}
          </div>
        </SheetHeader>
        <div className="flex-1 relative overflow-hidden">
          {sheet.url && (
            <iframe
              src={sheet.url}
              className="absolute inset-0 w-full h-full rounded-md border"
              title={`Documentation: ${sheet.title}`}
              loading="lazy"
            />
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
