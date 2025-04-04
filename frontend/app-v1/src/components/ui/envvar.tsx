import React, { useEffect } from 'react';
import { Input } from '@/components/ui/input';
import { Textarea } from './textarea';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { TrashIcon } from '@radix-ui/react-icons';

export type KeyValueType = {
  key: string;
  value: string;
  hidden: boolean;
  locked: boolean;
  deleted: boolean;
};

type PropsType = {
  label?: string;
  values: KeyValueType[];
  setValues?: (x: KeyValueType[]) => void;
  disabled?: boolean;
  fileUpload?: boolean;
  secretOption?: boolean;
};

const EnvGroupArray: React.FC<PropsType> = ({
  label,
  values,
  setValues = () => {},
  disabled,
  secretOption,
}) => {
  useEffect(() => {
    if (!values) {
      setValues([]);
    }
  }, [setValues, values]);

  const handleValueChange = (index: number, key: string, value: any) => {
    const newValues = [...values];
    newValues[index] = { ...newValues[index], [key]: value };
    setValues(newValues);
  };

  return (
    <>
      {label && <div className="mb-2 text-white">{label}</div>}
      {values?.map((entry: KeyValueType, i: number) => {
        if (!entry.deleted) {
          return (
            <div className="mb-2 flex items-center" key={i}>
              <Input
                placeholder="ex: key"
                value={entry.key}
                onChange={(e) => handleValueChange(i, 'key', e.target.value)}
                disabled={disabled || entry.locked}
                className={cn(
                  'w-64',
                  entry.locked && 'bg-gray-200 cursor-not-allowed',
                )}
              />
              <div className="mx-2" />
              {entry.hidden ? (
                <Input
                  placeholder="ex: value"
                  value={entry.value}
                  onChange={(e) =>
                    handleValueChange(i, 'value', e.target.value)
                  }
                  type="password"
                  disabled={disabled || entry.locked}
                  className={cn(
                    'flex-1',
                    entry.locked && 'bg-gray-200 cursor-not-allowed',
                  )}
                />
              ) : (
                <Textarea
                  placeholder="ex: value"
                  value={entry.value}
                  onChange={(e: any) =>
                    handleValueChange(i, 'value', e.target.value)
                  }
                  rows={entry.value?.split('\n').length || 0}
                  disabled={disabled || entry.locked}
                  className={cn(
                    'flex-1',
                    entry.locked && 'bg-gray-200 cursor-not-allowed',
                  )}
                />
              )}
              {secretOption && (
                <Button
                  variant="ghost"
                  onClick={() =>
                    !entry.locked &&
                    handleValueChange(i, 'hidden', !entry.hidden)
                  }
                  disabled={entry.locked}
                >
                  {entry.hidden ? 'Unlock' : 'Lock'}
                </Button>
              )}
              {!disabled && (
                <Button
                  variant="ghost"
                  className="ml-2"
                  size="sm"
                  onClick={() => {
                    const newValues = values.filter((_, index) => index !== i);
                    setValues(newValues);
                  }}
                >
                  <TrashIcon className="h-4 w-4" />
                </Button>
              )}
            </div>
          );
        }
      })}
      {!disabled && (
        <div className="flex items-center">
          <Button
            variant="secondary"
            onClick={() => {
              const newValues = [
                ...values,
                {
                  key: '',
                  value: '',
                  hidden: false,
                  locked: false,
                  deleted: false,
                },
              ];
              setValues(newValues);
            }}
          >
            Add row
          </Button>
        </div>
      )}
    </>
  );
};

export default EnvGroupArray;
