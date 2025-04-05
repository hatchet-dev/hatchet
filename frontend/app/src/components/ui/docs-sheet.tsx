import { Sheet, SheetContent, SheetHeader, SheetTitle } from './sheet';
import { DocsSheet } from '@/hooks/use-docs-sheet';
import { Cross2Icon } from '@radix-ui/react-icons';

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
  // If using push variant, render as a side panel instead of using Sheet
  if (variant === 'push') {
    return (
      <div
        className={`
          h-full min-h-screen bg-background border-l border-border
          transition-all duration-300 ease-in-out
          ${sheet.isOpen ? 'w-[600px]' : 'w-0 overflow-hidden'}
        `}
      >
        {sheet.isOpen && (
          <div className="h-full min-h-screen flex flex-col p-6 overflow-hidden">
            <div className="flex justify-between items-center mb-4 shrink-0">
              <h2 className="text-lg font-semibold">{sheet.title}</h2>
              <button
                onClick={onClose}
                className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              >
                <Cross2Icon className="h-4 w-4" />
                <span className="sr-only">Close</span>
              </button>
            </div>
            <div className="flex-1 overflow-hidden relative">
              {sheet.url && (
                <iframe
                  src={sheet.url}
                  className="absolute inset-0 w-full h-full rounded-md border"
                  title={`Documentation: ${sheet.title}`}
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
        className="w-full max-w-[600px] sm:max-w-[600px] lg:max-w-[600px]"
      >
        <SheetHeader className="mb-4">
          <SheetTitle>{sheet.title}</SheetTitle>
        </SheetHeader>
        <div className="h-[calc(100vh-120px)] w-full relative">
          {sheet.url && (
            <iframe
              src={sheet.url}
              className="absolute inset-0 w-full h-full rounded-md border"
              title={`Documentation: ${sheet.title}`}
            />
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
