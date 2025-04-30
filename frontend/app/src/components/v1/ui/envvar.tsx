import React, { useEffect } from 'react';
import { Input } from '@/components/v1/ui/input';
import { Textarea } from './textarea';
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { TrashIcon } from '@radix-ui/react-icons';

export type KeyValueType = {
  key: string;
  value: string;
  hidden: boolean;
  locked: boolean;
  deleted: boolean;
  id?: string; // Optional ID for existing env vars
  isEditing?: boolean; // Whether the value is being edited
  hint?: string; // Hint value for existing secrets
};

type PropsType = {
  label?: string;
  values: KeyValueType[];
  setValues?: (x: KeyValueType[]) => void;
  disabled?: boolean;
  fileUpload?: boolean;
  secretOption?: boolean;
  onDeleteIds?: (ids: string[]) => void; // Callback for deleted IDs
  onAdd?: (values: KeyValueType[]) => void; // Callback for added values
  onUpdate?: (values: KeyValueType[]) => void; // Callback for updated values
};

const EnvGroupArray: React.FC<PropsType> = ({
  label,
  values,
  setValues = () => {},
  disabled,
  secretOption,
  onDeleteIds,
  onAdd,
  onUpdate,
}) => {
  useEffect(() => {
    if (!values) {
      setValues([]);
    }
  }, [setValues, values]);

  const handleValueChange = (index: number, key: string, value: any) => {
    const newValues = [...values];
    const entry = newValues[index];

    // If this is an existing secret and we're editing the value
    if (key === 'value' && entry.hint) {
      // Only set isEditing to true if the value is different from the hint
      // and not empty (which would be the case when no changes are made)
      if (value !== entry.hint && value !== '') {
        newValues[index] = { ...entry, [key]: value, isEditing: true };
      }
    } else {
      newValues[index] = { ...entry, [key]: value };
    }

    setValues(newValues);

    // If this is an existing entry (has an ID), notify about updates
    if (entry.id && onUpdate && value !== entry.hint) {
      onUpdate([newValues[index]]);
    }
  };

  const handleDeleteToggle = (index: number) => {
    const newValues = [...values];
    const entry = newValues[index];
    const newDeletedState = !entry.deleted;

    newValues[index] = { ...entry, deleted: newDeletedState };
    setValues(newValues);

    // If this is an existing env var with an ID, notify parent of deleted IDs
    if (entry.id && onDeleteIds) {
      const deletedIds = values
        .filter((v) => v.id && v.deleted)
        .map((v) => v.id as string);
      onDeleteIds(deletedIds);
    }
  };

  const handleRemove = (index: number) => {
    const entry = values[index];

    if (entry.id) {
      // For existing entries, mark as deleted
      const newValues = [...values];
      newValues[index] = { ...entry, deleted: true };
      setValues(newValues);

      if (onDeleteIds) {
        const deletedIds = values
          .filter((v) => v.id && v.deleted)
          .map((v) => v.id as string);
        onDeleteIds(deletedIds);
      }
    } else {
      // For new entries, remove completely
      const newValues = values.filter((_, i) => i !== index);
      setValues(newValues);

      if (onAdd) {
        onAdd(newValues.filter((v) => !v.id));
      }
    }
  };

  return (
    <>
      {label && <div className="mb-2 text-white">{label}</div>}
      {values?.map((entry: KeyValueType, i: number) => (
        <div
          className={cn(
            'mb-2 flex items-center gap-2',
            entry.deleted && 'opacity-50 [&>*]:line-through',
          )}
          key={i}
        >
          <Input
            placeholder="ex: key"
            value={entry.key}
            onChange={(e) => handleValueChange(i, 'key', e.target.value)}
            disabled={disabled || entry.locked || entry.deleted}
            className={cn(
              'w-64',
              entry.locked && 'bg-gray-200 cursor-not-allowed',
            )}
          />
          {entry.hidden ? (
            <Input
              placeholder={entry.hint}
              value={entry.isEditing ? entry.value : undefined}
              onChange={(e) => handleValueChange(i, 'value', e.target.value)}
              type="password"
              disabled={disabled || entry.locked || entry.deleted}
              className={cn(
                'flex-1',
                entry.locked && 'bg-gray-200 cursor-not-allowed',
                !entry.isEditing && entry.hint && 'text-gray-400',
              )}
            />
          ) : (
            <Textarea
              placeholder={entry.hint}
              value={entry.isEditing ? entry.value : undefined}
              onChange={(e: any) =>
                handleValueChange(i, 'value', e.target.value)
              }
              rows={entry.value?.split('\n').length || 0}
              disabled={disabled || entry.locked || entry.deleted}
              className={cn(
                'flex-1',
                entry.locked && 'bg-gray-200 cursor-not-allowed',
                !entry.isEditing && entry.hint && 'text-gray-400',
              )}
            />
          )}
          {secretOption && (
            <Button
              variant="ghost"
              onClick={() =>
                !entry.locked && handleValueChange(i, 'hidden', !entry.hidden)
              }
              disabled={entry.locked || entry.deleted}
            >
              {entry.hidden ? 'Unlock' : 'Lock'}
            </Button>
          )}
          {!disabled && (
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  if (entry.id) {
                    handleDeleteToggle(i);
                  } else {
                    handleRemove(i);
                  }
                }}
              >
                <TrashIcon className="h-4 w-4" />
              </Button>
            </div>
          )}
        </div>
      ))}
      {!disabled && (
        <div className="flex items-center">
          <Button
            variant="secondary"
            onClick={() => {
              const newEntry = {
                key: '',
                value: '',
                hidden: false,
                locked: false,
                deleted: false,
              };
              const newValues = [...values, newEntry];
              setValues(newValues);

              if (onAdd) {
                onAdd([...values.filter((v) => !v.id), newEntry]);
              }
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
