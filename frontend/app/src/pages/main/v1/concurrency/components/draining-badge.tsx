import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';

export function DrainingBadge() {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="shrink-0 cursor-help rounded-full border border-dashed border-border px-1.5 text-[10.5px] font-medium leading-4 text-muted-foreground">
            Draining
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs text-xs leading-relaxed">
          This strategy is used only by out-of-date workflows.
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
