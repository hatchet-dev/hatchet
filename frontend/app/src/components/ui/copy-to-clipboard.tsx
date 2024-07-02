import React, { useState } from 'react';
import { Button } from './button';
import { CopyIcon } from '@radix-ui/react-icons';
import { CheckIcon } from '@heroicons/react/24/outline';
import { CopyToClipboard as Copy } from 'react-copy-to-clipboard';
import { cn } from '@/lib/utils';

type Props = {
  text: string;
  className?: string;
  withText?: boolean;
};

const CopyToClipboard: React.FC<Props> = ({ text, className, withText }) => {
  const [successCopy, setSuccessCopy] = useState(false);

  return (
    <Copy
      text={text}
      onCopy={() => {
        setSuccessCopy(true);

        setTimeout(() => {
          setSuccessCopy(false);
        }, 2000);
      }}
    >
      <Button
        className={cn(
          className,
          withText
            ? 'cursor-pointer flex flex-row gap-2 items-center mt-2'
            : 'w-6 h-6 p-0 cursor-pointer',
        )}
        variant={withText ? 'default' : 'ghost'}
      >
        {successCopy ? (
          <CheckIcon className="w-4 h-4" />
        ) : (
          <CopyIcon className="w-4 h-4" />
        )}
        {withText && (successCopy ? 'Copied' : 'Copy to clipboard')}
      </Button>
    </Copy>
  );
};

export default CopyToClipboard;
