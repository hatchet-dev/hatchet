import BasicLayout from '@/next/components/layouts/basic.layout';
import { RunsTable } from '@/next/components/runs/runs-table/runs-table';
import { FilterProvider } from '@/next/hooks/use-filters';
import { PaginationProvider } from '@/next/hooks/use-pagination';
import { RunsProvider } from '@/next/hooks/use-runs';
import { V1TaskStatus } from '@/next/lib/api';

export default function RunsPage() {
  return (
    <BasicLayout>
      <FilterProvider
        initialFilters={{
          statuses: [
            V1TaskStatus.RUNNING,
            V1TaskStatus.COMPLETED,
            V1TaskStatus.FAILED,
          ],
          is_root_task: true,
        }}
      >
        <PaginationProvider>
          <RunsProvider>
            <RunsTable />
          </RunsProvider>
        </PaginationProvider>
      </FilterProvider>
    </BasicLayout>
  );
}
