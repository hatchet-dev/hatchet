import { TaskRunActionButton } from './actions';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { ChevronDownIcon } from '@radix-ui/react-icons';

// Replay, plus a chevron menu with "Replay as new" when the caller provides
// it. Without onReplayAsNew it renders the plain Replay button.
export function ReplaySplitButton({
  externalId,
  replayDisabled,
  onReplayAsNew,
  replayAsNewDisabledReason,
}: {
  externalId: string;
  replayDisabled: boolean;
  onReplayAsNew?: () => void;
  replayAsNewDisabledReason?: string;
}) {
  return (
    <div className="flex items-center">
      <TaskRunActionButton
        actionType="replay"
        paramOverrides={{ externalIds: [externalId] }}
        disabled={replayDisabled}
        showModal={false}
        showLabel
        className={onReplayAsNew ? 'rounded-r-none' : undefined}
      />
      {onReplayAsNew && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size="sm"
              variant="outline"
              className="rounded-l-none border-l-0 px-2"
              aria-label="More replay options"
            >
              <ChevronDownIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              disabled={!!replayAsNewDisabledReason}
              onSelect={onReplayAsNew}
              className="flex-col items-start"
            >
              Replay as new
              {replayAsNewDisabledReason && (
                <span className="text-xs text-muted-foreground">
                  {replayAsNewDisabledReason}
                </span>
              )}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}
