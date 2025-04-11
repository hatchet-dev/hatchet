import { MoreHorizontal } from 'lucide-react';
import { usePagination } from '../../../hooks/use-pagination';
import {
  PaginationItem,
  PaginationLink,
  PaginationPrevious,
  PaginationNext,
} from './pagination-link';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { cn } from '@/next/lib/utils';

const PaginationEllipsis = () => (
  <PaginationLink className="w-9 h-9" disabled>
    <MoreHorizontal className="h-4 w-4" />
    <span className="sr-only">More pages</span>
  </PaginationLink>
);

interface PageSelectorProps {
  variant?: 'default' | 'dropdown';
}

export function PageSelector({ variant = 'default' }: PageSelectorProps) {
  const { currentPage, numPages, setCurrentPage } = usePagination();

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  const generatePagination = () => {
    // Always show first and last page
    // Show one page before and after current page
    // Show ellipsis when needed
    const pages: (number | 'ellipsis')[] = [];

    if (numPages <= 7) {
      // If 7 or fewer pages, show all
      return Array.from({ length: numPages }, (_, i) => i + 1);
    }

    // Always add first page
    pages.push(1);

    if (currentPage > 3) {
      pages.push('ellipsis');
    }

    // Add pages around current page
    for (
      let i = Math.max(2, currentPage - 1);
      i <= Math.min(numPages - 1, currentPage + 1);
      i++
    ) {
      pages.push(i);
    }

    if (currentPage < numPages - 2) {
      pages.push('ellipsis');
    }

    // Always add last page
    if (numPages > 1) {
      pages.push(numPages);
    }

    return pages;
  };

  if (variant === 'dropdown') {
    return (
      <div className="flex items-center">
        <PaginationItem className="flex items-center text-sm">
          <span className="flex items-center text-sm gap-1">
            Page
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <PaginationLink
                  variant="content"
                  className="h-9 p-2 flex items-center justify-center gap-0.5"
                >
                  {currentPage} of {numPages}
                </PaginationLink>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="center"
                className="min-w-[3rem] max-h-[300px] overflow-y-auto"
              >
                {Array.from({ length: numPages }, (_, i) => i + 1).map(
                  (page) => (
                    <DropdownMenuItem
                      key={page}
                      onClick={() => handlePageChange(page)}
                      className={cn(
                        'flex items-center justify-center px-2',
                        page === currentPage && 'bg-accent',
                      )}
                    >
                      {page}
                    </DropdownMenuItem>
                  ),
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </span>
        </PaginationItem>
        <PaginationItem>
          <PaginationPrevious
            onClick={() => handlePageChange(currentPage - 1)}
            disabled={currentPage <= 1}
          />
        </PaginationItem>
        <PaginationItem>
          <PaginationNext
            onClick={() => handlePageChange(currentPage + 1)}
            disabled={currentPage >= numPages}
          />
        </PaginationItem>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <PaginationItem>
        <PaginationPrevious
          onClick={() => handlePageChange(currentPage - 1)}
          disabled={currentPage <= 1}
        />
      </PaginationItem>
      {generatePagination().map((page, index) => (
        <PaginationItem key={index}>
          {page === 'ellipsis' ? (
            <PaginationEllipsis />
          ) : (
            <PaginationLink
              isActive={currentPage === page}
              onClick={() => handlePageChange(page)}
            >
              {page}
            </PaginationLink>
          )}
        </PaginationItem>
      ))}
      <PaginationItem>
        <PaginationNext
          onClick={() => handlePageChange(currentPage + 1)}
          disabled={currentPage >= numPages}
        />
      </PaginationItem>
    </div>
  );
}
