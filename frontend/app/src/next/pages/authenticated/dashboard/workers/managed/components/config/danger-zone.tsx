import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { Button } from '@/next/components/ui/button';
import { DestructiveDialog } from '@/next/components/ui/dialog/destructive-dialog';
import { useState } from 'react';

interface DangerZoneProps {
  poolName: string;
  poolId: string;
  onDelete: (poolId: string) => Promise<void>;
  type: 'update' | 'create';
  actions?: React.ReactNode;
}

export function DangerZone({
  poolName,
  poolId,
  onDelete,
  type,
  actions,
}: DangerZoneProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(poolId);
    } catch (error) {
      console.error('Failed to delete pool:', error);
    } finally {
      setIsDeleting(false);
      setShowDeleteDialog(false);
    }
  };

  return (
    <Card variant={type === 'update' ? 'borderless' : 'default'}>
      <CardHeader>
        <CardTitle className="text-destructive">Danger Zone</CardTitle>
        <CardDescription>
          Deleting a pool will immediately tear down all workers in this pool.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
          Delete Pool
        </Button>
        {actions}
      </CardContent>

      <DestructiveDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete Pool"
        description={`Are you sure you want to delete the pool "${poolName}"? This action cannot be undone, and will immediately tear down all workers in this pool.`}
        confirmationText={poolName}
        confirmButtonText="Delete Pool"
        isLoading={isDeleting}
        onConfirm={handleDelete}
      />
    </Card>
  );
}
