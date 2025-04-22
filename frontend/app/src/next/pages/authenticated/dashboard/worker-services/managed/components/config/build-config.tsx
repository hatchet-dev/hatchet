import { Input } from '@/next/components/ui/input';
import { Button } from '@/next/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { TrashIcon, PlusIcon } from '@radix-ui/react-icons';
import {
  ManagedWorkerSecret,
  UpdateManagedWorkerSecretRequest,
} from '@/lib/api/generated/cloud/data-contracts';
import { useState, useMemo } from 'react';
import { BulkSecretsDialog } from './bulk-secrets-dialog';
import { ClipboardDocumentIcon } from '@heroicons/react/24/outline';

interface SecretsEditorProps {
  secrets: UpdateManagedWorkerSecretRequest;
  setSecrets: React.Dispatch<
    React.SetStateAction<UpdateManagedWorkerSecretRequest>
  >;
  original: {
    directSecrets: ManagedWorkerSecret[];
    globalSecrets: ManagedWorkerSecret[];
  };
}

export function SecretsEditor({
  original = { directSecrets: [], globalSecrets: [] },
  secrets = { add: [], update: [], delete: [] },
  setSecrets,
}: SecretsEditorProps) {
  const [isBulkDialogOpen, setIsBulkDialogOpen] = useState(false);

  // Function to check if a key is duplicated
  const isKeyDuplicated = useMemo(() => {
    const allKeys = new Map<string, number>();

    // Count occurrences of each key
    original.directSecrets.forEach((secret) => {
      if (!secrets.delete?.includes(secret.id)) {
        const update = secrets.update?.find((s) => s.id === secret.id);
        const key = update ? update.key : secret.key;
        allKeys.set(key, (allKeys.get(key) || 0) + 1);
      }
    });

    secrets.add?.forEach((secret) => {
      const key = secret.key;
      allKeys.set(key, (allKeys.get(key) || 0) + 1);
    });

    return (key: string) => (allKeys.get(key) || 0) > 1;
  }, [original.directSecrets, secrets]);

  const handleAddSecret = () => {
    setSecrets((prev) => ({
      ...prev,
      add: [...(prev.add || []), { key: '', value: '' }],
    }));
  };

  const handleBulkAddSecrets = (
    newSecrets: { key: string; value: string }[],
  ) => {
    setSecrets((prev) => ({
      ...prev,
      add: [...(prev.add || []), ...newSecrets],
    }));
  };

  const handleBulkUpdateSecrets = (
    updatedSecrets: { id: string; key: string; value: string }[],
  ) => {
    setSecrets((prev) => {
      // Remove any existing updates for these secrets
      const existingUpdates =
        prev.update?.filter(
          (update) => !updatedSecrets.some((s) => s.id === update.id),
        ) || [];

      return {
        ...prev,
        update: [...existingUpdates, ...updatedSecrets],
      };
    });
  };

  const handleDeleteAddSecret = (index: number) => {
    setSecrets((prev) => ({
      ...prev,
      add: prev.add?.filter((_, i) => i !== index),
    }));
  };

  const handleUpdateAddSecret = (
    index: number,
    field: 'key' | 'value',
    value: string,
  ) => {
    setSecrets((prev) => ({
      ...prev,
      add: prev.add?.map((secret, i) =>
        i === index
          ? {
              ...secret,
              [field]: field === 'key' ? value.replace(/\s+/g, '_') : value,
            }
          : secret,
      ),
    }));
  };

  const handleUpdateExistingSecret = (
    id: string,
    field: 'key' | 'value',
    value: string,
  ) => {
    setSecrets((prev) => {
      // Find the original secret to get its current values
      const originalSecret = original.directSecrets.find((s) => s.id === id);
      if (!originalSecret) {
        return prev;
      }

      // Check if this secret is already in the update array
      const existingUpdateIndex =
        prev.update?.findIndex((s) => s.id === id) ?? -1;

      if (existingUpdateIndex >= 0) {
        if (field === 'value' && value === '') {
          return {
            ...prev,
            update: prev.update?.filter((secret) => secret.id !== id),
          };
        }

        // Update existing update
        return {
          ...prev,
          update: prev.update?.map((secret) =>
            secret.id === id
              ? {
                  ...secret,
                  key:
                    field === 'key' ? value.replace(/\s+/g, '_') : secret.key,
                  value: field === 'value' ? value : secret.value,
                }
              : secret,
          ),
        };
      } else {
        // Create new update entry
        return {
          ...prev,
          update: [
            ...(prev.update || []),
            {
              id,
              key:
                field === 'key'
                  ? value.replace(/\s+/g, '_')
                  : originalSecret.key,
              value: field === 'value' ? value : '',
            },
          ],
        };
      }
    });
  };

  const handleDeleteExistingSecret = (id: string) => {
    setSecrets((prev) => ({
      ...prev,
      delete: prev.delete?.includes(id)
        ? prev.delete.filter((deleteId) => deleteId !== id)
        : [...(prev.delete || []), id],
    }));
  };

  const handleRemoveFromDelete = (ids: string[]) => {
    setSecrets((prev) => ({
      ...prev,
      delete: prev.delete?.filter((id) => !ids.includes(id)),
    }));
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Environment Variables</CardTitle>
        <CardDescription>
          Add environment variables that will be available to the worker service
          at runtime. These are encrypted at rest and can only be accessed by
          the worker service.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Existing secrets */}
        {original.directSecrets.map((secret) => {
          const update = secrets.update?.find((s) => s.id === secret.id);
          const currentKey = update ? update.key : secret.key;
          const isDuplicated = isKeyDuplicated(currentKey);

          return (
            <div key={secret.id} className="flex items-center gap-2">
              <Input
                value={currentKey}
                onChange={(e) =>
                  handleUpdateExistingSecret(secret.id, 'key', e.target.value)
                }
                className={`w-48 ${secrets.delete?.includes(secret.id) ? 'line-through' : ''} ${isDuplicated ? 'border-red-500' : ''}`}
                disabled={secrets.delete?.includes(secret.id)}
              />
              <Input
                placeholder={secret.hint}
                value={update ? update.value : ''}
                onChange={(e) =>
                  handleUpdateExistingSecret(secret.id, 'value', e.target.value)
                }
                className={`flex-1 ${secrets.delete?.includes(secret.id) ? 'line-through' : ''}`}
                disabled={secrets.delete?.includes(secret.id)}
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleDeleteExistingSecret(secret.id)}
              >
                <TrashIcon className="h-4 w-4" />
              </Button>
            </div>
          );
        })}

        {/* New secrets */}
        {(secrets.add || []).map((secret, index) => {
          const isDuplicated = isKeyDuplicated(secret.key);
          return (
            <div key={index} className="flex items-center gap-2">
              <Input
                placeholder="Key"
                value={secret.key}
                onChange={(e) =>
                  handleUpdateAddSecret(index, 'key', e.target.value)
                }
                className={`w-48 ${isDuplicated ? 'border-red-500' : ''}`}
              />
              <Input
                placeholder="Value"
                value={secret.value}
                onChange={(e) =>
                  handleUpdateAddSecret(index, 'value', e.target.value)
                }
                className="flex-1"
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleDeleteAddSecret(index)}
              >
                <TrashIcon className="h-4 w-4" />
              </Button>
            </div>
          );
        })}

        {/* Add new secret buttons */}
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={handleAddSecret}
            className="flex-1"
          >
            <PlusIcon className="h-4 w-4 mr-2" />
            Add Secret
          </Button>
          <Button variant="outline" onClick={() => setIsBulkDialogOpen(true)}>
            <ClipboardDocumentIcon className="h-4 w-4 mr-2" />
            From Clipboard
          </Button>
        </div>
      </CardContent>

      <BulkSecretsDialog
        open={isBulkDialogOpen}
        onOpenChange={setIsBulkDialogOpen}
        onAddSecrets={handleBulkAddSecrets}
        onUpdateSecrets={handleBulkUpdateSecrets}
        onRemoveFromDelete={handleRemoveFromDelete}
        original={original}
        pendingUpdates={secrets.update || []}
        pendingDeletes={secrets.delete || []}
      />
    </Card>
  );
}
