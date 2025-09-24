import React, { useEffect, useMemo, useState } from 'react';
import TimeAgo from 'timeago-react';
import {
  PortalTooltip,
  PortalTooltipTrigger,
  PortalTooltipContent,
  PortalTooltipProvider,
} from '@/components/v1/ui/portal-tooltip';

interface RelativeDateProps {
  date?: Date | string;
  future?: boolean;
}

const RelativeDate: React.FC<RelativeDateProps> = ({
  date = '',
  future = false,
}) => {
  const formattedDate = useMemo(
    () => (typeof date === 'string' ? new Date(date) : date),
    [date],
  );

  const [countdown, setCountdown] = useState('');

  useEffect(() => {
    if (future) {
      const updateCountdown = () => {
        const currentDate = new Date();
        const timeDiff = formattedDate.getTime() - currentDate.getTime();

        if (timeDiff <= 0) {
          setCountdown('');
          return;
        }

        const days = Math.floor(timeDiff / (1000 * 3600 * 24));
        const hours = Math.floor(
          (timeDiff % (1000 * 3600 * 24)) / (1000 * 3600),
        );
        const minutes = Math.floor((timeDiff % (1000 * 3600)) / (1000 * 60));
        const seconds = Math.floor((timeDiff % (1000 * 60)) / 1000);

        const countdownParts = [];
        if (days > 0) {
          countdownParts.push(`${days}d`);
        }
        if (hours > 0 || days > 0) {
          countdownParts.push(`${hours}h`);
        }
        if (minutes > 0 || hours > 0 || days > 0) {
          countdownParts.push(`${minutes}m`);
        }
        countdownParts.push(`${seconds}s`);

        setCountdown(countdownParts.join(' '));
      };

      updateCountdown();
      const countdownInterval = setInterval(updateCountdown, 1000);

      return () => {
        clearInterval(countdownInterval);
      };
    }
  }, [formattedDate, future]);

  if (date == '0001-01-01T00:00:00Z') {
    return null;
  }

  return (
    <PortalTooltipProvider>
      <PortalTooltip>
        <PortalTooltipTrigger
          asChild
          onFocusCapture={(e) => {
            e.stopPropagation();
          }}
        >
          <span>
            {future && countdown ? (
              <>{countdown}</>
            ) : (
              <TimeAgo datetime={formattedDate} />
            )}
          </span>
        </PortalTooltipTrigger>
        <PortalTooltipContent side="top">
          {formattedDate.toLocaleString()}
        </PortalTooltipContent>
      </PortalTooltip>
    </PortalTooltipProvider>
  );
};

export default RelativeDate;
