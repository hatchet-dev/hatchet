import { Button } from '../ui/button';
import { CheckIcon, CopyIcon } from 'lucide-react';
import { useState } from 'react';

export function CopyWorkflowConfigButton({
  workflowConfig,
}: {
  workflowConfig: object | undefined;
}) {
  const [copySuccess, setCopySuccess] = useState(false);

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => {
        navigator.clipboard.writeText(JSON.stringify(workflowConfig ?? {}));
        setCopySuccess(true);
        setTimeout(() => setCopySuccess(false), 2000);
      }}
    >
      {copySuccess ? (
        <>
          <CheckIcon className="mr-2 size-3" />
          Copied!
        </>
      ) : (
        <>
          <CopyIcon className="mr-2 size-3" />
          Copy Config
        </>
      )}
    </Button>
  );
}
