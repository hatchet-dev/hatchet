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
import { TagIcon } from '@heroicons/react/24/outline';
import { memo } from 'react';

export interface AdditionalMetadataClick {
  key: string;
  value: any;
}

interface AdditionalMetadataProps {
  metadata: object;
  onClick?: (click: AdditionalMetadataClick) => void;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  title?: string;
  align?: 'start' | 'center' | 'end';
}

export const AdditionalMetadata = memo(
  function AdditionalMetadata({
    metadata,
    onClick,
    isOpen,
    onOpenChange,
    title = 'Metadata',
    align = 'end',
  }: AdditionalMetadataProps) {
    const metadataEntries = Object.entries(metadata || {});

    if (metadataEntries.length === 0) {
      return null;
    }

    const metadataCount = metadataEntries.length;

    return (
      <div className="flex max-w-32 items-center justify-start">
        <Popover open={isOpen} onOpenChange={onOpenChange}>
          <PopoverTrigger asChild>
            <div className="flex cursor-pointer items-center gap-1 rounded px-2 py-1 transition-colors hover:bg-muted/50">
              <TagIcon className="size-3 flex-shrink-0 text-muted-foreground" />
              <span className="text-xs font-medium text-muted-foreground">
                {metadataCount}
              </span>
            </div>
          </PopoverTrigger>
          <PopoverContent
            className="w-80 p-0 shadow-md shadow-slate-800/30"
            align={align}
          >
            <div className="p-3">
              <div className="mb-3 flex items-center gap-2 border-b pb-2">
                <TagIcon className="size-4 text-muted-foreground" />
                <span className="text-sm font-medium">
                  {title} ({metadataCount}{' '}
                  {metadataCount === 1 ? 'item' : 'items'})
                </span>
              </div>
              <div className="max-h-60 space-y-2 overflow-y-auto">
                {metadataEntries
                  .sort(([a], [b]) =>
                    a.toLowerCase().localeCompare(b.toLowerCase()),
                  )
                  .map(([key, value]) => (
                    <div
                      key={key}
                      className="group flex cursor-pointer flex-col gap-1 rounded border p-2 transition-colors hover:bg-muted/30"
                      onClick={() => onClick?.({ key, value })}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-medium tracking-wide text-muted-foreground">
                          {key}
                        </span>
                      </div>
                      <div className="break-all text-sm">
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div className="truncate">
                                {getDisplayValue(value)}
                              </div>
                            </TooltipTrigger>
                            <TooltipContent side="left" className="max-w-xs">
                              <pre className="whitespace-pre-wrap text-xs">
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
  },
  (prevProps, nextProps) => {
    return (
      prevProps.isOpen === nextProps.isOpen &&
      prevProps.onClick === nextProps.onClick &&
      prevProps.onOpenChange === nextProps.onOpenChange &&
      JSON.stringify(prevProps.metadata) === JSON.stringify(nextProps.metadata)
    );
  },
);

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
