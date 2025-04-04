import { Cross2Icon } from '@radix-ui/react-icons';
import { Table } from '@tanstack/react-table';

import { Button } from '@/components/ui/button';
import { DataTableViewOptions } from './data-table-view-options';

import { DataTableFacetedFilter } from './data-table-faceted-filter';
import { Input } from '@/components/ui/input.tsx';
import { Spinner } from '@/components/ui/loading';

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
}

export function DataTableToolbar<TData>({
  table,
  filters,
  actions,
  setSearch,
  search,
  showColumnToggle,
  isLoading = false,
}: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters?.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2">
        {setSearch && (
          <Input
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-8 w-[150px] lg:w-[250px]"
          />
        )}
        {filters.map((filter) => (
          <DataTableFacetedFilter
            key={filter.columnId}
            column={table.getColumn(filter.columnId)}
            title={filter.title}
            type={filter.type}
            options={filter.options}
          />
        ))}
        {isFiltered && (
          <Button
            variant="ghost"
            onClick={() => table.resetColumnFilters()}
            className="h-8 px-2 lg:px-3"
          >
            Reset
            <Cross2Icon className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <div className="flex flex-row gap-4 items-center">
        {isLoading && <Spinner />}
        {actions && actions.length > 0 && actions}
        {showColumnToggle && <DataTableViewOptions table={table} />}
      </div>
    </div>
  );
}
