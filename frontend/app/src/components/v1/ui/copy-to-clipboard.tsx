import { Button } from './button';
import { cn } from '@/lib/utils';
import { CheckIcon } from '@heroicons/react/24/outline';
import { CopyIcon } from '@radix-ui/react-icons';
import React, { useState } from 'react';

type Props = {
  text: string;
  className?: string;
  withText?: boolean;
  onCopy?: () => void;
};

const CopyToClipboard: React.FC<Props> = ({
  text,
  className,
  withText,
  onCopy,
}) => {
  const [successCopy, setSuccessCopy] = useState(false);

  return (
    <Button
      className={cn(
        className,
        withText
          ? 'mt-2 flex cursor-pointer flex-row items-center gap-2'
          : 'h-6 w-6 cursor-pointer p-0',
      )}
      variant={withText ? 'default' : 'ghost'}
      onClick={() => {
        navigator.clipboard.writeText(text);
        setSuccessCopy(true);
        // eslint-disable-next-line @typescript-eslint/no-unused-expressions
        onCopy && onCopy();
        setTimeout(() => {
          setSuccessCopy(false);
        }, 2000);
      }}
    >
      {successCopy ? (
        <CheckIcon className="h-4 w-4" />
      ) : (
        <CopyIcon className="h-4 w-4" />
      )}
      {withText && (successCopy ? 'Copied' : 'Copy to clipboard')}
    </Button>
  );
};

export default CopyToClipboard;
