import { useFilterDetails, useFilters } from '../hooks/use-filters';
import { updateFilterSchema, UpdateFilterFormData } from '../schemas';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Textarea } from '@/components/v1/ui/textarea';
import { useSidePanel } from '@/hooks/use-side-panel';
import { zodResolver } from '@hookform/resolvers/zod';
import { Trash2Icon, EditIcon, SaveIcon, XIcon } from 'lucide-react';
import { useCallback, useState } from 'react';
import { useForm } from 'react-hook-form';

interface FilterDetailViewProps {
  filterId: string;
}

export function FilterDetailView({ filterId }: FilterDetailViewProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [payloadError, setPayloadError] = useState<string | null>(null);

  const { close } = useSidePanel();

  const { filter, isLoading } = useFilterDetails(filterId);

  const { workflowIdToName, mutations } = useFilters({
    key: `detail-${filter?.metadata.id}`,
  });

  const form = useForm<UpdateFilterFormData>({
    resolver: zodResolver(updateFilterSchema),
    defaultValues: {
      expression: filter?.expression,
      scope: filter?.scope,
      payload: JSON.stringify(filter?.payload || {}, null, 2),
    },
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = form;

  const handleEdit = () => {
    reset({
      expression: filter?.expression,
      scope: filter?.scope,
      payload: JSON.stringify(filter?.payload || {}, null, 2),
    });
    setPayloadError(null);
    setIsEditing(true);
  };

  const onSubmit = useCallback(
    async (data: UpdateFilterFormData) => {
      if (!filter) {
        return;
      }

      try {
        let payloadObj;
        if (data.payload !== undefined) {
          try {
            const payloadText = data.payload.trim() || '{}';
            payloadObj = JSON.parse(payloadText);
            setPayloadError(null);
          } catch (error) {
            if (error instanceof SyntaxError) {
              setPayloadError('The filter payload must be valid JSON');
              return;
            }
          }
        }

        await mutations.update.perform(filter.metadata.id, {
          ...data,
          payload: payloadObj,
        });
        setIsEditing(false);
      } catch (error) {
        console.error('Failed to update filter:', error);
      }
    },
    [mutations.update, filter],
  );

  const handleCancel = () => {
    setIsEditing(false);
    setPayloadError(null);
    reset();
  };

  const handleDelete = async () => {
    if (!filter) {
      return;
    }

    try {
      await mutations.delete.perform(filter.metadata.id);
      setShowDeleteDialog(false);
      close();
    } catch (error) {
      console.error('Failed to delete filter:', error);
    }
  };

  if (!filter || isLoading) {
    return <div>Loading...</div>;
  }

  return (
    <>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex gap-2 pt-2">
            {!isEditing ? (
              <Button
                variant="outline"
                size="sm"
                onClick={handleEdit}
                leftIcon={<EditIcon className="size-4" />}
              >
                Edit
              </Button>
            ) : (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCancel}
                  leftIcon={<XIcon className="size-4" />}
                >
                  Cancel
                </Button>
                <Button
                  size="sm"
                  onClick={handleSubmit(onSubmit)}
                  disabled={mutations.update.isPending}
                  leftIcon={<SaveIcon className="size-4" />}
                >
                  {mutations.update.isPending ? 'Saving...' : 'Save'}
                </Button>
              </>
            )}
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setShowDeleteDialog(true)}
              leftIcon={<Trash2Icon className="size-4" />}
            >
              Delete
            </Button>
          </div>
        </div>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="filter-id">Filter ID</Label>
            <Input
              id="filter-id"
              value={filter.metadata.id}
              disabled
              className="bg-muted disabled:cursor-text"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="workflow">Workflow</Label>
            <Input
              id="workflow"
              value={workflowIdToName[filter.workflowId] || filter.workflowId}
              disabled
              className="bg-muted disabled:cursor-text"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="scope">Scope</Label>
            {isEditing ? (
              <>
                <Input
                  id="scope"
                  {...register('scope')}
                  placeholder="Enter scope"
                />
                {errors.scope && (
                  <p className="text-sm text-red-600">{errors.scope.message}</p>
                )}
              </>
            ) : (
              <Input
                id="scope"
                value={filter.scope}
                disabled
                className="bg-muted disabled:cursor-text"
              />
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="expression">Expression</Label>
            {isEditing ? (
              <>
                <Textarea
                  id="expression"
                  {...register('expression')}
                  placeholder="Enter filter expression"
                  className="min-h-[100px] font-mono"
                />
                {errors.expression && (
                  <p className="text-sm text-red-600">
                    {errors.expression.message}
                  </p>
                )}
              </>
            ) : (
              <Textarea
                id="expression"
                value={filter.expression}
                disabled
                className="min-h-[100px] bg-muted font-mono disabled:cursor-text"
              />
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="payload">Payload (JSON)</Label>
            {isEditing ? (
              <>
                <Textarea
                  id="payload"
                  {...register('payload')}
                  placeholder='{"key": "value"} or leave empty for {}'
                  className="min-h-[120px] font-mono text-sm"
                  onChange={(e) => {
                    register('payload').onChange(e);
                    setPayloadError(null);
                  }}
                />
                {payloadError && (
                  <p className="text-sm text-red-600">{payloadError}</p>
                )}
              </>
            ) : (
              <Textarea
                id="payload"
                value={JSON.stringify(filter.payload || {}, null, 2)}
                disabled
                className="min-h-[120px] bg-muted font-mono text-sm disabled:cursor-text"
              />
            )}
          </div>
        </div>
      </div>

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Filter</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this filter? This action cannot be
              undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={mutations.delete.isPending}
            >
              {mutations.delete.isPending ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
