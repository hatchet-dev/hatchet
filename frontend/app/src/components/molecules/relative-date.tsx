import React from 'react';
import TimeAgo from 'timeago-react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

interface RelativeDateProps {
  date: Date | string;
}

const RelativeDate: React.FC<RelativeDateProps> = ({ date }) => {
  const formattedDate = typeof date === 'string' ? new Date(date) : date;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <TimeAgo datetime={formattedDate} />
        </TooltipTrigger>
        <TooltipContent>{formattedDate.toLocaleString()}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export default RelativeDate;
