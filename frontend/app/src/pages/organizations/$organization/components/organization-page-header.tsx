import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import { Input } from '@/components/v1/ui/input';
import { CheckIcon, PencilIcon, XMarkIcon } from '@heroicons/react/24/outline';
import React from 'react';

type OrganizationPageHeaderProps = {
  orgId: string;
  orgName: string;
  isEditingName: boolean;
  editedName: string;
  onEditedNameChange: (next: string) => void;
  onStartEdit: () => void;
  onCancelEdit: () => void;
  onSaveEdit: () => void;
  onClose: () => void;
  saving?: boolean;
};

export function OrganizationPageHeader({
  orgId,
  orgName,
  isEditingName,
  editedName,
  onEditedNameChange,
  onStartEdit,
  onCancelEdit,
  onSaveEdit,
  onClose,
  saving,
}: OrganizationPageHeaderProps) {
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      onSaveEdit();
    } else if (e.key === 'Escape') {
      onCancelEdit();
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            {isEditingName ? (
              <>
                <Input
                  value={editedName}
                  onChange={(e) => onEditedNameChange(e.target.value)}
                  onKeyDown={onKeyDown}
                  className="h-10 w-[min(520px,90vw)] px-3 text-2xl font-bold"
                  autoFocus
                  disabled={saving}
                />
                <Button
                  size="sm"
                  onClick={onSaveEdit}
                  disabled={saving || !editedName.trim()}
                  aria-label="Save organization name"
                >
                  <CheckIcon className="size-4" />
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={onCancelEdit}
                  disabled={saving}
                  aria-label="Cancel"
                >
                  <XMarkIcon className="size-4" />
                </Button>
              </>
            ) : (
              <>
                <h1 className="truncate text-2xl font-semibold tracking-tight">
                  {orgName}
                </h1>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={onStartEdit}
                  className="h-8 w-8 p-0"
                  disabled={saving}
                  aria-label="Rename organization"
                >
                  <PencilIcon className="size-4" />
                </Button>
              </>
            )}
          </div>

          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted-foreground">
            <Badge variant="secondary" className="font-normal">
              Organization settings
            </Badge>
            <span className="flex items-center gap-2">
              <span className="font-mono text-xs">{orgId}</span>
              <CopyToClipboard text={orgId} />
            </span>
          </div>
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          className="h-8 w-8 shrink-0 p-0"
          aria-label="Close organization"
        >
          <XMarkIcon className="size-4" />
        </Button>
      </div>
    </div>
  );
}


