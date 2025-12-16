import { useState } from 'react';
import { Button, ReviewedButtonTemp } from '../ui/button';
import { CheckIcon, CopyIcon } from 'lucide-react';

export function CopyWorkflowConfigButton({
  workflowConfig,
}: {
  workflowConfig: object | undefined;
}) {
  const [copySuccess, setCopySuccess] = useState(false);

  return (
    <ReviewedButtonTemp
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
          <CheckIcon className="size-3 mr-2" />
          Copied!
        </>
      ) : (
        <>
          <CopyIcon className="size-3 mr-2" />
          Copy Config
        </>
      )}
    </ReviewedButtonTemp>
  );
}
