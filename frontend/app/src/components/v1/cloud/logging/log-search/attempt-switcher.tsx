import { useLogsContext } from './use-logs';
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { ChevronLeftIcon, ChevronRightIcon } from '@radix-ui/react-icons';

export function AttemptSwitcher({ className }: { className?: string }) {
  const { availableAttempts, selectedAttempt, setSelectedAttempt } =
    useLogsContext();

  if (availableAttempts.length <= 1) {
    return null;
  }

  const currentIndex =
    selectedAttempt === null ? 0 : availableAttempts.indexOf(selectedAttempt);

  const currentAttempt = selectedAttempt ?? availableAttempts[0] ?? 1;

  const canGoNewer = currentIndex > 0;
  const canGoOlder = currentIndex < availableAttempts.length - 1;

  const goNewer = () => {
    if (canGoNewer) {
      const newAttempt = availableAttempts[currentIndex - 1];
      setSelectedAttempt(newAttempt);
    }
  };

  const goOlder = () => {
    if (canGoOlder) {
      const newAttempt = availableAttempts[currentIndex + 1];
      setSelectedAttempt(newAttempt);
    }
  };

  return (
    <div className={cn('flex items-center gap-1', className)}>
      <span className="text-xs text-muted-foreground mr-1">Attempt</span>
      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6"
        onClick={goOlder}
        disabled={!canGoOlder}
      >
        <ChevronLeftIcon className="h-3 w-3" />
      </Button>
      <span className="text-xs font-medium min-w-[3ch] text-center">
        {currentAttempt}
      </span>
      <span className="text-xs text-muted-foreground">/</span>
      <span className="text-xs text-muted-foreground min-w-[3ch] text-center">
        {availableAttempts.length}
      </span>
      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6"
        onClick={goNewer}
        disabled={!canGoNewer}
      >
        <ChevronRightIcon className="h-3 w-3" />
      </Button>
    </div>
  );
}
