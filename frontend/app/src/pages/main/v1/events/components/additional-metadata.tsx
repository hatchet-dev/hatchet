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
import { TagIcon, EyeIcon } from '@heroicons/react/24/outline';

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

  if (metadataEntries.length === 0) {
    return null;
  }

  const metadataCount = metadataEntries.length;
  const hasMultipleItems = metadataCount > 1;

  return (
    <div className="flex items-center justify-start max-w-32">
      <Popover>
        <PopoverTrigger asChild>
          <div className="flex items-center gap-1 cursor-pointer hover:bg-muted/50 rounded px-2 py-1 transition-colors">
            <TagIcon className="h-3 w-3 text-muted-foreground flex-shrink-0" />
            <span className="text-xs text-muted-foreground font-medium">
              {metadataCount}
            </span>
            {hasMultipleItems && (
              <EyeIcon className="h-3 w-3 text-muted-foreground flex-shrink-0" />
            )}
          </div>
        </PopoverTrigger>
        <PopoverContent className="w-80 p-0" align="end">
          <div className="p-3">
            <div className="flex items-center gap-2 mb-3 pb-2 border-b">
              <TagIcon className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium text-sm">
                Metadata ({metadataCount}{' '}
                {metadataCount === 1 ? 'item' : 'items'})
              </span>
            </div>
            <div className="space-y-2 max-h-60 overflow-y-auto">
              {metadataEntries.map(([key, value]) => (
                <div
                  key={key}
                  className="group flex flex-col gap-1 p-2 rounded border hover:bg-muted/30 transition-colors cursor-pointer"
                  onClick={() => onClick?.({ key, value })}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-muted-foreground tracking-wide">
                      {key}
                    </span>
                  </div>
                  <div className="text-sm break-all">
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="truncate">
                            {getDisplayValue(value)}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent side="left" className="max-w-xs">
                          <pre className="text-xs whitespace-pre-wrap">
                            {JSON.stringify(value, null, 2)}
                          </pre>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}

const getDisplayValue = (value: any): string => {
  if (value === null) {
    return 'null';
  }
  if (value === undefined) {
    return 'undefined';
  }
  if (typeof value === 'string') {
    return value.length > 50 ? `${value.substring(0, 50)}...` : value;
  }
  if (typeof value === 'object') {
    const jsonStr = JSON.stringify(value);
    return jsonStr.length > 50 ? `${jsonStr.substring(0, 50)}...` : jsonStr;
  }
  return String(value);
};
