import { Button, ButtonProps } from '../ui/button';
import { CheckIcon, CopyIcon } from 'lucide-react';
import { useState } from 'react';

export function CopyWorkflowConfigButton({
  workflowConfig,
  size = 'sm',
}: {
  workflowConfig: object | undefined;
  size?: ButtonProps['size'];
}) {
  const [copySuccess, setCopySuccess] = useState(false);

  return (
    <Button
      variant="outline"
      size={size}
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
