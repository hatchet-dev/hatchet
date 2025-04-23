import React from 'react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { Time } from './time';

interface RelativeDateProps {
  date?: string | Date | null | undefined;
}

const RelativeDate: React.FC<RelativeDateProps> = ({ date = '' }) => {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          onFocusCapture={(e) => {
            e.stopPropagation();
          }}
          className="cursor-pointer"
        >
          <Time date={date} variant="timeSince" />
        </TooltipTrigger>
        <TooltipContent className="z-[80] bg-muted">
          <Time
            date={date}
            variant="timestamp"
            className="font-mono text-foreground"
          />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export default RelativeDate;
