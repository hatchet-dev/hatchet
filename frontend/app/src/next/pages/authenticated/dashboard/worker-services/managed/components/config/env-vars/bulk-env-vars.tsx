import { useState } from 'react';
import { Textarea } from '@/next/components/ui/textarea';
import { Button } from '@/next/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog';
import { Alert, AlertDescription } from '@/next/components/ui/alert';
import { FaExclamationTriangle } from 'react-icons/fa';
import { ManagedWorkerSecret } from '@/lib/api/generated/cloud/data-contracts';

interface BulkSecretsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAddSecrets: (secrets: { key: string; value: string }[]) => void;
  onUpdateSecrets: (
    secrets: { id: string; key: string; value: string }[],
  ) => void;
  onRemoveFromDelete: (ids: string[]) => void;
  original: {
    directSecrets: ManagedWorkerSecret[];
    globalSecrets: ManagedWorkerSecret[];
  };
  pendingUpdates: { id: string; key: string; value: string }[];
  pendingDeletes: string[];
}

export function BulkSecretsDialog({
  open,
  onOpenChange,
  onAddSecrets,
  onUpdateSecrets,
  onRemoveFromDelete,
  original,
  pendingUpdates,
  pendingDeletes,
}: BulkSecretsDialogProps) {
  const [text, setText] = useState('');
  const [error, setError] = useState<string | null>(null);

  const parseSecrets = (input: string): { key: string; value: string }[] => {
    const secrets: { key: string; value: string }[] = [];
    const lines = input.split('\n').filter((line) => line.trim());

    for (const line of lines) {
      // Try to parse as KEY=value format
      const envMatch = line.match(/^([^=]+)=(.*)$/);
      if (envMatch) {
        const [, key, value] = envMatch;
        secrets.push({
          key: key.trim().replace(/^["']|["']$/g, ''),
          value: value.trim().replace(/^["']|["']$/g, ''),
        });
        continue;
      }

      // Try to parse as CSV format (key,value)
      const csvMatch = line.match(/^([^,]+),(.*)$/);
      if (csvMatch) {
        const [, key, value] = csvMatch;
        secrets.push({
          key: key.trim().replace(/^["']|["']$/g, ''),
          value: value.trim().replace(/^["']|["']$/g, ''),
        });
        continue;
      }

      throw new Error(`Invalid format in line: ${line}`);
    }

    return secrets;
  };

  const handleSubmit = () => {
    try {
      setError(null);
      const parsedSecrets = parseSecrets(text);

      // Group secrets into updates and additions
      const updates: { id: string; key: string; value: string }[] = [];
      const additions: { key: string; value: string }[] = [];
      const secretsToUndelete: string[] = [];

      for (const secret of parsedSecrets) {
        // First check pending updates
        const pendingUpdate = pendingUpdates.find((u) => u.key === secret.key);
        if (pendingUpdate) {
          updates.push({ ...pendingUpdate, value: secret.value });
          continue;
        }

        // Then check original secrets
        const originalSecret = original.directSecrets.find(
          (s) => s.key === secret.key,
        );
        if (originalSecret) {
          // If this secret was marked for deletion, we'll undelete it
          if (pendingDeletes.includes(originalSecret.id)) {
            secretsToUndelete.push(originalSecret.id);
          }

          updates.push({
            id: originalSecret.id,
            key: secret.key,
            value: secret.value,
          });
          continue;
        }

        // If not found in either, it's a new secret
        additions.push(secret);
      }

      // Apply updates and additions
      if (updates.length > 0) {
        onUpdateSecrets(updates);
      }
      if (additions.length > 0) {
        onAddSecrets(additions);
      }
      if (secretsToUndelete.length > 0) {
        onRemoveFromDelete(secretsToUndelete);
      }

      setText('');
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid format');
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Bulk Add Environment Variables</DialogTitle>
          <DialogDescription>
            Existing keys will be updated, new keys will be added.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <Textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={`KEY1=value1\nKEY2=value2\n\nor\n\nkey1,value1\nkey2,value2`}
            className="min-h-[200px] font-mono"
          />

          {error && (
            <Alert variant="destructive">
              <FaExclamationTriangle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit}>Add Variables</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
