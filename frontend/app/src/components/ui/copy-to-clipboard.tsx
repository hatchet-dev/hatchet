import React, { useState } from 'react';
import { Button } from './button';
import { CopyIcon } from '@radix-ui/react-icons';
import { CheckCircleIcon } from '@heroicons/react/24/outline';

type Props = {
  text: string;
};

const CopyToClipboard: React.FC<Props> = ({ text }) => {
  const [successCopy, setSuccessCopy] = useState(false);

  return (
    <Button
      className="max-w-fit"
      onClick={() => {
        navigator.clipboard.writeText(text);
        setSuccessCopy(true);

        setTimeout(() => {
          setSuccessCopy(false);
        }, 2000);
      }}
    >
      {successCopy ? (
        <CheckCircleIcon className="w-4 h-4 mr-2" />
      ) : (
        <CopyIcon className="w-4 h-4 mr-2" />
      )}
      {successCopy ? 'Copied!' : 'Copy to clipboard'}
    </Button>
  );
};

export default CopyToClipboard;
