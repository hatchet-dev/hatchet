import { Badge } from '@/components/v1/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { formatCron } from '@/lib/cron';
import { RiPulseAiLine, RiTimeLine } from 'react-icons/ri';

export type Trigger =
  { type: 'event'; eventKey: string } | { type: 'cron'; cron: string };

export const triggerKey = (trigger: Trigger) =>
  trigger.type === 'event'
    ? `event:${trigger.eventKey}`
    : `cron:${trigger.cron}`;

export const triggerDescription = (trigger: Trigger) =>
  trigger.type === 'event'
    ? `Triggered by the ${trigger.eventKey} event`
    : `Runs ${formatCron(trigger.cron)}`;

export function TriggerBadge({ trigger }: { trigger: Trigger }) {
  const Icon = trigger.type === 'event' ? RiPulseAiLine : RiTimeLine;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex items-center gap-1.5">
            <Icon
              className="size-3.5 shrink-0 text-muted-foreground"
              aria-hidden="true"
            />
            <Badge
              variant="secondary"
              className="whitespace-nowrap px-2 font-mono font-normal"
            >
              {trigger.type === 'event' ? trigger.eventKey : trigger.cron}
            </Badge>
          </span>
        </TooltipTrigger>
        <TooltipContent side="top">
          {triggerDescription(trigger)}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
