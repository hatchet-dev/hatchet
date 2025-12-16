import { ColumnDef } from '@tanstack/react-table';
import { SNSIntegration } from '@/lib/api';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { ReviewedButtonTemp } from '@/components/v1/ui/button';
import { useState } from 'react';
import { CheckIcon } from '@heroicons/react/24/outline';
import { CopyIcon } from '@radix-ui/react-icons';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';

type Props = {
  ingestUrl: string;
};

const CopyIngestURL: React.FC<Props> = ({ ingestUrl }: Props) => {
  const [successCopy, setSuccessCopy] = useState(false);

  return (
    <ReviewedButtonTemp
      variant="icon"
      onClick={() => {
        navigator.clipboard.writeText(ingestUrl);
        setSuccessCopy(true);

        setTimeout(() => {
          setSuccessCopy(false);
        }, 2000);
      }}
    >
      {successCopy ? (
        <CheckIcon className="size-4" />
      ) : (
        <CopyIcon className="size-4" />
      )}
      {successCopy ? 'Copied' : 'Copy ingest URL'}
    </ReviewedButtonTemp>
  );
};

export const columns = ({
  onDeleteClick,
}: {
  onDeleteClick: (row: SNSIntegration) => void;
}): ColumnDef<SNSIntegration>[] => {
  return [
    {
      accessorKey: 'topicArn',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Topic ARN" />
      ),
      cell: ({ row }) => <div>{row.original.topicArn}</div>,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'ingestUrl',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Ingest URL" />
      ),
      cell: ({ row }) => (
        <CopyIngestURL ingestUrl={row.original.ingestUrl || ''} />
      ),
    },
    {
      accessorKey: 'created',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created" />
      ),
      cell: ({ row }) => (
        <div>
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      ),
    },
    {
      id: 'actions',
      cell: ({ row }) => (
        <DataTableRowActions
          row={row}
          actions={[
            {
              label: 'Delete',
              onClick: () => onDeleteClick(row.original),
            },
          ]}
        />
      ),
    },
  ];
};
