import { Badge } from '@/components/v1/ui/badge';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';

const MAX_METADATA_LENGTH = 2;

export interface AdditionalMetadataClick {
  key: string;
  value: any;
}

interface AdditionalMetadataProps {
  metadata: object;
  onClick?: (click: AdditionalMetadataClick) => void;
}

export function AdditionalMetadata({
  metadata,
  onClick,
}: AdditionalMetadataProps) {
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
                className="mr-2 truncate cursor-default font-normal cursor-pointer"
                variant="secondary"
                onClick={() => onClick?.({ key, value })}
              >
                {`${key}: ${getValueString(value)}`}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              {key}: {JSON.stringify(value)}
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
                  className="mr-2 truncate font-normal text-sm cursor-pointer"
                  title={`${key}:${value}`}
                  variant="secondary"
                  key={key}
                  onClick={() => onClick?.({ key, value })}
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

const getValueString = (value: any) => {
  const res = JSON.stringify(value).substring(0, 10);

  if (value && value?.length > 10) {
    return `${res}...`;
  }

  return res;
};
