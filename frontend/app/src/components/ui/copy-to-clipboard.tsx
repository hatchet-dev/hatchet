import React, { useState } from 'react';
import { Button } from './button';
import { CopyIcon } from '@radix-ui/react-icons';
import { CheckIcon } from '@heroicons/react/24/outline';
import { cn } from '@/lib/utils';

type Props = {
  text: string;
  className?: string;
  withText?: boolean;
};

const CopyToClipboard: React.FC<Props> = ({ text, className, withText }) => {
  const [successCopy, setSuccessCopy] = useState(false);

  return (
    <Button
      className={cn(
        className,
        withText ? 'w-6 h-6 p-0 cursor-pointer' : 'cursor-pointer',
      )}
      variant="ghost"
      onClick={() => {
        navigator.clipboard.writeText(text);
        setSuccessCopy(true);

        setTimeout(() => {
          setSuccessCopy(false);
        }, 2000);
      }}
    >
      {successCopy ? (
        <CheckIcon className="w-4 h-4" />
      ) : (
        <CopyIcon className="w-4 h-4" />
      )}
      {withText && (successCopy ? 'Copied' : 'Copy')}
    </Button>
  );
};

export default CopyToClipboard;
