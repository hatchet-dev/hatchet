import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/v1/ui/dialog';
import { ReviewedButtonTemp } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import { Textarea } from '@/components/v1/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Filter } from 'lucide-react';
import { FilterOption } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { createFilterSchema, CreateFilterFormData } from '../schemas';
import { V1CreateFilterRequest } from '@/lib/api';

interface FilterCreateFormProps {
  isOpen: boolean;
  onClose: () => void;
  workflowNameFilters: FilterOption[];
  onCreate: (data: V1CreateFilterRequest) => Promise<void>;
  isCreating: boolean;
}

function FilterCreateForm({
  isOpen,
  onClose,
  workflowNameFilters,
  onCreate,
  isCreating,
}: FilterCreateFormProps) {
  const [payloadError, setPayloadError] = useState<string | null>(null);

  const form = useForm<CreateFilterFormData>({
    resolver: zodResolver(createFilterSchema),
    defaultValues: {
      workflowId: '',
      scope: '',
      expression: '',
      payload: '',
    },
  });

  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors },
  } = form;

  const onSubmit = async (data: CreateFilterFormData) => {
    try {
      const payloadText = data.payload.trim() || '{}';
      const payloadObj = JSON.parse(payloadText);
      setPayloadError(null);

      await onCreate({
        ...data,
        payload: payloadObj,
      });
      reset();
      onClose();
    } catch (error) {
      if (error instanceof SyntaxError) {
        setPayloadError('The filter payload must be valid JSON');
        return;
      }
      console.error('Failed to create filter:', error);
    }
  };

  const handleClose = () => {
    reset();
    setPayloadError(null);
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Filter className="size-5" />
            Create New Filter
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="create-workflow">Workflow *</Label>
            <Controller
              name="workflowId"
              control={control}
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a workflow" />
                  </SelectTrigger>
                  <SelectContent>
                    {workflowNameFilters.map((workflow) => (
                      <SelectItem key={workflow.value} value={workflow.value}>
                        {workflow.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
            {errors.workflowId && (
              <p className="text-sm text-red-600">
                {errors.workflowId.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-scope">Scope *</Label>
            <Input
              id="create-scope"
              {...register('scope')}
              placeholder="e.g., event, step, workflow"
            />
            {errors.scope && (
              <p className="text-sm text-red-600">{errors.scope.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-expression">Expression *</Label>
            <Textarea
              id="create-expression"
              {...register('expression')}
              placeholder="Enter your filter expression..."
              className="min-h-[100px] font-mono"
            />
            {errors.expression && (
              <p className="text-sm text-red-600">
                {errors.expression.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-payload">Payload (JSON)</Label>
            <Textarea
              id="create-payload"
              {...register('payload')}
              placeholder='{"key": "value"} or leave empty for {}'
              className="min-h-[80px] font-mono text-sm"
              onChange={(e) => {
                register('payload').onChange(e);
                setPayloadError(null);
              }}
            />
            {payloadError && (
              <p className="text-sm text-red-600">{payloadError}</p>
            )}
          </div>
        </div>

        <DialogFooter>
          <ReviewedButtonTemp variant="outline" onClick={handleClose}>
            Cancel
          </ReviewedButtonTemp>
          <ReviewedButtonTemp
            onClick={handleSubmit(onSubmit)}
            disabled={isCreating}
          >
            {isCreating ? 'Creating...' : 'Create Filter'}
          </ReviewedButtonTemp>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function FilterCreateButton({
  workflowNameFilters,
  onCreate,
  isCreating,
}: Omit<FilterCreateFormProps, 'isOpen' | 'onClose'>) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <ReviewedButtonTemp onClick={() => setIsOpen(true)} variant="cta">
        Create Filter
      </ReviewedButtonTemp>
      <FilterCreateForm
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        workflowNameFilters={workflowNameFilters}
        onCreate={onCreate}
        isCreating={isCreating}
      />
    </>
  );
}
