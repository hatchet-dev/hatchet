import { Button } from '@/next/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { ChevronDown } from 'lucide-react';
import { cn } from '@/next/lib/utils';
import { usePagination } from '../../../hooks/utils/use-pagination';
import { PaginationItem } from './pagination-link';

interface PageSizeSelectorProps {
  options?: number[];
  className?: string;
}

export function PageSizeSelector({
  options,
  className,
}: PageSizeSelectorProps) {
  const { pageSize, setPageSize, pageSizeOptions } = usePagination();
  const availableOptions = options || pageSizeOptions || [10, 50, 100, 500];

  return (
    <PaginationItem className={className}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="h-8 w-[120px] flex items-center justify-between"
          >
            {pageSize} per page
            <ChevronDown className="h-4 w-4 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-[110px]">
          {availableOptions.map((size) => (
            <DropdownMenuItem
              key={size}
              onClick={() => setPageSize(size)}
              className={cn(
                'flex items-center justify-between',
                size === pageSize && 'bg-accent',
              )}
            >
              {size} per page
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </PaginationItem>
  );
}
