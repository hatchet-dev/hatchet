import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { Button } from '@/components/v1/ui/button';
import { SNSIntegration } from '@/lib/api';
import { CheckIcon } from '@heroicons/react/24/outline';
import { CopyIcon } from '@radix-ui/react-icons';
import { useState } from 'react';

export function CopyIngestURL({ ingestUrl }: { ingestUrl: string }) {
  const [successCopy, setSuccessCopy] = useState(false);

  return (
    <Button
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
    </Button>
  );
}

export function SNSActions({
  integration,
  onDeleteClick,
}: {
  integration: SNSIntegration;
  onDeleteClick: (integration: SNSIntegration) => void;
}) {
  return (
    <TableRowActions
      row={integration}
      actions={[
        {
          label: 'Delete',
          onClick: () => onDeleteClick(integration),
        },
      ]}
    />
  );
}
