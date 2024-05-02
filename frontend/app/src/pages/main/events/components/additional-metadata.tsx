import { Badge } from '@/components/ui/badge';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

const MAX_METADATA_LENGTH = 2;
export function AdditionalMetadata({ metadata }: { metadata: object }) {
  const metadataEntries = Object.entries(metadata || {});
  const visibleEntries = metadataEntries.slice(0, MAX_METADATA_LENGTH);
  const hiddenEntries = metadataEntries.slice(MAX_METADATA_LENGTH);

  return (
    <div className="flex flex-row gap-2 items-center justify-start">
      {visibleEntries.map(([key, value]) => (
        <TooltipProvider key={key}>
          <Tooltip>
            <TooltipTrigger>
              <Badge
                className="mr-2 truncate cursor-default font-normal"
                variant="secondary"
              >
                {`${key}: ${value?.substring(0, 10)}${value.length > 10 ? '...' : ''}`}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              {key}: {value}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ))}
      {hiddenEntries.length > 0 && (
        <Popover>
          <PopoverTrigger>
            <Badge className="cursor-pointer font-normal" variant="secondary">
              + {hiddenEntries.length} more
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-3 m-0 bg-background rounded"
            align="end"
          >
            <div className="flex flex-col gap-2 p-0">
              {metadataEntries.map(([key, value]) => (
                <Badge
                  className="mr-2 truncate font-normal text-sm"
                  title={`${key}:${value}`}
                  variant="secondary"
                  key={key}
                >
                  {`${key}: ${value}`}
                </Badge>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}
