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
  serviceName: string;
  serviceId: string;
  onDelete: (serviceId: string) => Promise<void>;
  type: 'update' | 'create';
  actions?: React.ReactNode;
}

export function DangerZone({
  serviceName,
  serviceId,
  onDelete,
  type,
  actions,
}: DangerZoneProps) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(serviceId);
    } catch (error) {
      console.error('Failed to delete service:', error);
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
          Deleting a service will immediately tear down all workers in this
          service.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Button variant="destructive" onClick={() => setShowDeleteDialog(true)}>
          Delete Service
        </Button>
        {actions}
      </CardContent>

      <DestructiveDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete Service"
        description={`Are you sure you want to delete the service "${serviceName}"? This action cannot be undone, and will immediately tear down all workers in this service.`}
        confirmationText={serviceName}
        confirmButtonText="Delete Service"
        isLoading={isDeleting}
        onConfirm={handleDelete}
      />
    </Card>
  );
}
