// import { ColumnDef } from '@tanstack/react-table';
// import { DataTableColumnHeader } from '@/components/molecules/data-table/data-table-column-header';
// import { WorkerAffinity } from '@/lib/api';

// export const affinityColumns: ColumnDef<WorkerAffinity>[] = [
//   {
//     accessorKey: 'key',
//     header: ({ column }) => (
//       <DataTableColumnHeader column={column} title="key" />
//     ),
//     cell: ({ row }) => {
//       return (
//         <div className="pl-0 cursor-pointer hover:underline min-w-fit whitespace-nowrap">
//           {row.original.key}
//         </div>
//       );
//     },
//     enableSorting: false,
//     enableHiding: false,
//   },
//   {
//     accessorKey: 'value',
//     header: ({ column }) => (
//       <DataTableColumnHeader column={column} title="value" />
//     ),
//     cell: ({ row }) => {
//       return (
//         <div className="pl-0 cursor-pointer hover:underline min-w-fit whitespace-nowrap">
//           {row.original.value}
//         </div>
//       );
//     },
//     enableSorting: false,
//     enableHiding: false,
//   },
//   {
//     accessorKey: 'comparator',
//     header: ({ column }) => (
//       <DataTableColumnHeader column={column} title="Comparator" />
//     ),
//     cell: ({ row }) => {
//       return (
//         <div className="pl-0 cursor-pointer hover:underline min-w-fit whitespace-nowrap">
//           {row.original.comparator}
//         </div>
//       );
//     },
//     enableSorting: false,
//     enableHiding: false,
//   },
//   {
//     accessorKey: 'weight',
//     header: ({ column }) => (
//       <DataTableColumnHeader column={column} title="Weight" />
//     ),
//     cell: ({ row }) => {
//       return (
//         <div className="pl-0 cursor-pointer hover:underline min-w-fit whitespace-nowrap">
//           {row.original.weight}
//         </div>
//       );
//     },
//     enableSorting: false,
//     enableHiding: false,
//   },
//   {
//     accessorKey: 'Required',
//     header: ({ column }) => (
//       <DataTableColumnHeader column={column} title="Required" />
//     ),
//     cell: ({ row }) => {
//       return (
//         <div className="pl-0 cursor-pointer hover:underline min-w-fit whitespace-nowrap">
//           {row.original.required ? 'Yes' : 'No'}
//         </div>
//       );
//     },
//     enableSorting: false,
//     enableHiding: false,
//   },
// ];
