import { useState } from 'react';
import { Button } from '../ui/button';
import { CheckIcon, CopyIcon } from 'lucide-react';

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
        navigator.clipboard.writeText(JSON.stringify(workflowConfig));
        setCopySuccess(true);
        setTimeout(() => setCopySuccess(false), 2000);
      }}
    >
      {copySuccess ? (
        <>
          <CheckIcon className="w-3 h-3 mr-2" />
          Copied!
        </>
      ) : (
        <>
          <CopyIcon className="w-3 h-3 mr-2" />
          Copy Workflow Config
        </>
      )}
    </Button>
  );
}
