import { V1TaskStatus, V1TaskSummary, queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useState, useCallback } from 'react';
import LoggingComponent from '@/components/v1/cloud/logging/logs';
import { Button } from '@/components/v1/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Label } from '@radix-ui/react-label';
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  DoubleArrowLeftIcon,
  DoubleArrowRightIcon,
} from '@radix-ui/react-icons';

const LOGS_PER_PAGE = 100;

interface LogsPaginationProps {
  pageIndex: number;
  pageSize: number;
  totalPages: number;
  hasNextPage: boolean;
  hasPrevPage: boolean;
  isLoading: boolean;
  onFirstPage: () => void;
  onPrevPage: () => void;
  onNextPage: () => void;
  onLastPage: () => void;
  onPageSizeChange: (pageSize: number) => void;
}

function LogsPagination({
  pageIndex,
  pageSize,
  totalPages,
  hasNextPage,
  hasPrevPage,
  isLoading,
  onFirstPage,
  onPrevPage,
  onNextPage,
  onLastPage,
  onPageSizeChange,
}: LogsPaginationProps) {
  return (
    <div className="flex items-center justify-between px-2 mt-4 border-t pt-4">
      <div className="flex-1 text-sm text-gray-600 dark:text-gray-400"></div>

      <div className="flex items-center space-x-6 lg:space-x-8">
        <div className="flex items-center space-x-2">
          <Label
            className="text-sm font-medium text-gray-600 dark:text-gray-400"
            htmlFor="logs-per-page"
            id="logs-per-page-label"
          >
            Rows per page
          </Label>
          <Select
            value={`${pageSize}`}
            onValueChange={(value) => onPageSizeChange(Number(value))}
          >
            <SelectTrigger
              className="h-8 w-[70px]"
              id="logs-per-page"
              aria-labelledby="logs-per-page-label"
            >
              <SelectValue placeholder={pageSize} />
            </SelectTrigger>
            <SelectContent side="top">
              {[50, 100, 200, 500].map((size) => (
                <SelectItem key={size} value={`${size}`}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex w-[100px] items-center justify-center text-sm font-medium">
          Page {pageIndex + 1} of {totalPages}
        </div>

        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            className="hidden h-8 w-8 p-0 lg:flex"
            onClick={onFirstPage}
            disabled={!hasPrevPage || isLoading}
          >
            <span className="sr-only">Go to first page</span>
            <DoubleArrowLeftIcon className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-8 w-8 p-0"
            onClick={onPrevPage}
            disabled={!hasPrevPage || isLoading}
          >
            <span className="sr-only">Go to previous page</span>
            <ChevronLeftIcon className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-8 w-8 p-0"
            onClick={onNextPage}
            disabled={!hasNextPage || isLoading}
          >
            <span className="sr-only">Go to next page</span>
            <ChevronRightIcon className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="hidden h-8 w-8 p-0 lg:flex"
            onClick={onLastPage}
            disabled={!hasNextPage || isLoading}
          >
            <span className="sr-only">Go to last page</span>
            <DoubleArrowRightIcon className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}

export function StepRunLogs({ taskRun }: { taskRun: V1TaskSummary }) {
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(LOGS_PER_PAGE);

  const getLogsQuery = useQuery({
    ...queries.v1Tasks.getLogs(taskRun?.metadata.id || '', {
      limit: pageSize,
      offset: pageIndex * pageSize,
    }),
    enabled: !!taskRun,
    refetchInterval: () => {
      if (taskRun?.status === V1TaskStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const pagination = getLogsQuery.data?.pagination;
  const totalPages = pagination?.num_pages || 1;
  const currentApiPage = pagination?.current_page || 0;
  const hasNextPage =
    pagination?.next_page !== undefined &&
    pagination.next_page > currentApiPage;
  const hasPrevPage = pageIndex > 0;

  const handleFirstPage = useCallback(() => {
    setPageIndex(0);
  }, []);

  const handlePrevPage = useCallback(() => {
    setPageIndex((prev) => Math.max(0, prev - 1));
  }, []);

  const handleNextPage = useCallback(() => {
    setPageIndex((prev) => prev + 1);
  }, []);

  const handleLastPage = useCallback(() => {
    setPageIndex(Math.max(0, totalPages - 1));
  }, [totalPages]);

  const handlePageSizeChange = useCallback((newSize: number) => {
    setPageSize(newSize);
    setPageIndex(0);
  }, []);

  return (
    <div className="my-4">
      <LoggingComponent
        logs={
          getLogsQuery.data?.rows?.map((row) => ({
            timestamp: row.createdAt,
            line: row.message,
            instance: taskRun.displayName,
          })) || []
        }
        onTopReached={() => {}}
        onBottomReached={() => {}}
      />

      <LogsPagination
        pageIndex={pageIndex}
        pageSize={pageSize}
        totalPages={totalPages}
        hasNextPage={hasNextPage}
        hasPrevPage={hasPrevPage}
        isLoading={getLogsQuery.isFetching}
        onFirstPage={handleFirstPage}
        onPrevPage={handlePrevPage}
        onNextPage={handleNextPage}
        onLastPage={handleLastPage}
        onPageSizeChange={handlePageSizeChange}
      />
    </div>
  );
}
