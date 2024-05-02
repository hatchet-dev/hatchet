import { Badge } from '@/components/ui/badge';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { useState } from 'react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

const MAX_METADATA_LENGTH = 2;
export function AdditionalMetadata({ metadata }: { metadata: object }) {
  const [showAll, setShowAll] = useState(false);

  const metadataEntries = Object.entries(metadata || {});
  const visibleEntries = showAll
    ? metadataEntries
    : metadataEntries.slice(0, MAX_METADATA_LENGTH);
  const hiddenEntries = showAll
    ? []
    : metadataEntries.slice(MAX_METADATA_LENGTH);

  const handleShowAll = () => {
    setShowAll(true);
  };

  return (
    <div className="flex flex-row gap-2 items-center justify-start">
      {visibleEntries.map(([key, value]) => (
        <TooltipProvider key={key}>
          <Tooltip>
            <TooltipTrigger>
              <Badge className="mr-2 truncate" title={`${key}:${value}`}>
                {`${key}:${value?.substring(0, 10)}${value.length > 10 ? '...' : ''}`}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>{value}</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ))}
      {hiddenEntries.length > 0 && (
        <Popover>
          <PopoverTrigger>
            <Badge className="cursor-pointer" onClick={handleShowAll}>
              + {hiddenEntries.length} more
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            <div className="flex flex-col gap-2">
              {hiddenEntries.map(([key, value]) => (
                <TooltipProvider key={key}>
                  <Tooltip>
                    <TooltipTrigger>
                      <Badge
                        className="mr-2 truncate"
                        title={`${key}:${value}`}
                      >
                        {`${key}:${value.substring(0, 10)}${value.length > 10 ? '...' : ''}`}
                      </Badge>
                    </TooltipTrigger>
                    <TooltipContent>{value}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}
