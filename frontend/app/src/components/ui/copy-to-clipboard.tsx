import React, { useState,useEffect,useRef } from 'react';
import { Button } from './button';
import { CopyIcon } from '@radix-ui/react-icons';
import { CheckIcon } from '@heroicons/react/24/outline';
import { cn } from '@/lib/utils';
import ClipboardJS from 'clipboard'

type Props = {
  text: string;
  className?: string;
  withText?: boolean;
};

const CopyToClipboard: React.FC<Props> = ({ text, className, withText }) => {
  const [successCopy, setSuccessCopy] = useState(false);

  const copyRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if(copyRef.current){
      const clipboard   =  new ClipboardJS(copyRef.current)
   
      return () => {
        clipboard.destroy();
      };
    }
   
  }, [])
 
  return (
    <Button
      className={cn(
        className,
        withText
          ? 'cursor-pointer flex flex-row gap-2 items-center mt-2'
          : 'w-6 h-6 p-0 cursor-pointer',
      )}
      ref={copyRef}
      data-clipboard-text={text}
      variant={withText ? 'default' : 'ghost'}
      onClick={() => {
        // navigator.clipboard.writeText(text);
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
      {withText && (successCopy ? 'Copied' : 'Copy to clipboard')}
    </Button>
  );
};

export default CopyToClipboard;
