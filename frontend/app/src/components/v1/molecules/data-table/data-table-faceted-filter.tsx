import * as React from 'react';
import { Column } from '@tanstack/react-table';

import { ToolbarType } from './data-table-toolbar';
import { Combobox } from '../combobox/combobox';
import { Label } from '../../ui/label';
import { Checkbox } from '../../ui/checkbox';

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
  const value = column?.getFilterValue();

  if (type === ToolbarType.Switch) {
    return (
      <div className="flex items-center space-x-2">
        <Checkbox
          id="toolbar-switch"
          onCheckedChange={(e) => column?.setFilterValue(e.valueOf() === true)}
        />
        <Label htmlFor="toolbar-switch" className="text-sm">
          Flatten
        </Label>
      </div>
    );
  }

  return (
    <Combobox
      values={typeof value === 'string' ? [value] : (value as string[])}
      title={title}
      type={type}
      options={options}
      setValues={(values) => column?.setFilterValue(values)}
    />
  );
}
