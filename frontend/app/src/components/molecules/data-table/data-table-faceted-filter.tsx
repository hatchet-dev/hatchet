import * as React from 'react';
import { Column } from '@tanstack/react-table';

import { ToolbarType } from './data-table-toolbar';
import { Combobox } from '../combobox/combobox';

interface DataTableFacetedFilterProps<TData, TValue> {
  column?: Column<TData, TValue>;
  title?: string;
  type?: ToolbarType;
  options?: {
    label: string;
    value: string;
    icon?: React.ComponentType<{ className?: string }>;
  }[];
}

export function DataTableFacetedFilter<TData, TValue>({
  column,
  title,
  type = ToolbarType.Checkbox,
  options,
}: DataTableFacetedFilterProps<TData, TValue>) {
  return (
    <Combobox
      values={column?.getFilterValue() as string[]}
      title={title}
      type={type}
      options={options}
      setValues={(values) => column?.setFilterValue(values)}
    />
  );
}
