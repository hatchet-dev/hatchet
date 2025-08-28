import { Cross2Icon } from '@radix-ui/react-icons';
import { Table } from '@tanstack/react-table';

import { Button } from '@/components/v1/ui/button';
import { DataTableViewOptions } from './data-table-view-options';

import { DataTableFacetedFilter } from './data-table-faceted-filter';
import { Input } from '@/components/v1/ui/input.tsx';
import { Spinner } from '@/components/v1/ui/loading';
import { flattenDAGsKey } from '@/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns';

export interface FilterOption {
  label: string;
  value: string;
  icon?: React.ComponentType<{ className?: string }>;
}

export enum ToolbarType {
  Checkbox = 'checkbox',
  Radio = 'radio',
  KeyValue = 'key-value',
  Array = 'array',
  Switch = 'switch',
}

export type ToolbarFilters = {
  columnId: string;
  title: string;
  type?: ToolbarType;
  options?: FilterOption[];
}[];

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
  filters: ToolbarFilters;
  actions: JSX.Element[];
  setSearch?: (search: string) => void;
  search?: string;
  showColumnToggle?: boolean;
  isLoading?: boolean;
  onReset?: () => void;
  hideFlatten?: boolean;
}

export function DataTableToolbar<TData>({
  table,
  filters,
  actions,
  setSearch,
  search,
  showColumnToggle,
  isLoading = false,
  onReset,
  hideFlatten,
}: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters?.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div 
        className="flex flex-1 items-center space-x-2 overflow-x-auto pr-4 min-w-0 [&::-webkit-scrollbar]:h-1 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-gray-400/50 [&::-webkit-scrollbar-thumb]:rounded-full" 
        style={{
          scrollbarWidth: 'thin',
          scrollbarColor: 'rgba(156, 163, 175, 0.5) transparent',
          scrollbarGutter: 'stable both-edges'
        }}
      >
        {setSearch && (
          <Input
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-8 w-[150px] lg:w-[250px] flex-shrink-0"
          />
        )}
        {filters
          .filter((filter) => {
            if (hideFlatten && filter.columnId === flattenDAGsKey) {
              return false;
            }

            return true;
          })
          .map((filter) => {
            return (
              <DataTableFacetedFilter
                key={filter.columnId}
                column={table.getColumn(filter.columnId)}
                title={filter.title}
                type={filter.type}
                options={filter.options}
              />
            );
          })}
        {isFiltered && (
          <Button
            variant="outline"
            onClick={() => {
              if (onReset) {
                onReset();
              } else {
                table.resetColumnFilters();
              }
            }}
            className="h-8 px-2 lg:px-3 flex-shrink-0"
          >
            Reset
            <Cross2Icon className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <div className="flex flex-row gap-4 items-center flex-shrink-0">
        {isLoading && <Spinner />}
        {actions && actions.length > 0 && actions}
        {showColumnToggle && <DataTableViewOptions table={table} />}
      </div>
    </div>
  );
}
