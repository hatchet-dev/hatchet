import { Sheet, SheetContent, SheetHeader, SheetTitle } from './sheet';
import { useIsMobile } from '@/next/hooks/use-mobile';
import { Cross2Icon } from '@radix-ui/react-icons';

interface InfoSheetProps {
  isOpen: boolean;
  onClose: () => void;
  title: React.ReactNode;
  children: React.ReactNode;
  variant?: 'overlay' | 'push';
}

export function InfoSheet({
  isOpen,
  onClose,
  title,
  children,
  variant = 'push',
}: InfoSheetProps) {
  const isMobile = useIsMobile();

  // If using push variant, render as a side panel instead of using Sheet
  if (variant === 'push' && !isMobile) {
    return (
      <div
        className={`
            border-l border-border
          ${isOpen ? 'lg:w-[600px] md:w-[400px] w-[300px]' : 'w-0 overflow-hidden'}
        `}
      >
        {isOpen && (
          <div className="h-full flex flex-col">
            <div className="flex justify-between items-center p-4 border-b">
              <h2 className="text-lg font-semibold truncate pr-2">{title}</h2>
              <button
                onClick={onClose}
                className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 flex-shrink-0"
              >
                <Cross2Icon className="h-4 w-4" />
                <span className="sr-only">Close</span>
              </button>
            </div>
            <div className="flex-1 overflow-y-auto p-4">{children}</div>
          </div>
        )}
      </div>
    );
  }

  // Fall back to the overlay variant if not using push
  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side="right"
        className="p-4 md:p-6 w-full max-w-[300px] sm:max-w-[400px] lg:max-w-[600px]"
      >
        <SheetHeader className="mb-4 pr-8">
          <SheetTitle className="truncate">{title}</SheetTitle>
        </SheetHeader>
        <div className="h-[calc(100vh-120px)] w-full relative overflow-hidden">
          {children}
        </div>
      </SheetContent>
    </Sheet>
  );
}
