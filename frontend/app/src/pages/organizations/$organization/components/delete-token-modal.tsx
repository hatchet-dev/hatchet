import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import { ManagementToken } from '@/lib/api/generated/cloud/data-contracts';
import {
  KeyIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';

interface DeleteTokenModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  token: ManagementToken | null;
  organizationName: string;
  onSuccess: () => void;
}

export function DeleteTokenModal({
  open,
  onOpenChange,
  token,
  organizationName,
  onSuccess,
}: DeleteTokenModalProps) {
  const { handleApiError } = useApiError({});

  const deleteTokenMutation = useMutation({
    mutationFn: async () => {
      if (!token) {
        return;
      }
      await cloudApi.managementTokenDelete(token.id);
    },
    onSuccess: () => {
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const handleDelete = () => {
    if (token) {
      deleteTokenMutation.mutate();
    }
  };

  if (!token) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <XCircleIcon className="h-5 w-5 text-red-500" />
            Delete Management Token
          </DialogTitle>
          <DialogDescription>
            Are you sure you want to delete this management token from{' '}
            {organizationName}?
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-start gap-3 p-4 bg-red-50 border border-red-200 rounded-md">
            <ExclamationTriangleIcon className="h-5 w-5 text-red-600 mt-0.5 flex-shrink-0" />
            <div className="text-sm">
              <p className="font-medium text-red-800 mb-1">
                This action cannot be undone
              </p>
              <p className="text-red-700">
                Any applications or services using this token will immediately
                lose access and may break functionality.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-3 p-4 bg-yellow-50 border border-yellow-200 rounded-md">
            <ExclamationTriangleIcon className="h-5 w-5 text-yellow-600 mt-0.5 flex-shrink-0" />
            <div className="text-sm">
              <p className="font-medium text-yellow-800 mb-1">
                Potential consequences
              </p>
              <ul className="text-yellow-700 list-disc list-inside space-y-1">
                <li>API integrations may fail</li>
                <li>Automated deployments could break</li>
                <li>CI/CD pipelines might stop working</li>
                <li>Third-party tools will lose access</li>
              </ul>
            </div>
          </div>

          <div className="bg-gray-50 rounded-md p-3">
            <div className="text-sm">
              <div className="font-medium text-gray-700 mb-1">
                Token Details:
              </div>
              <div className="text-gray-600">
                <div className="flex items-center gap-2">
                  <KeyIcon className="h-4 w-4" />
                  <span>{token.name}</span>
                </div>
                <div className="mt-1">
                  <span className="text-xs">Duration: {token.duration}</span>
                </div>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={deleteTokenMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteTokenMutation.isPending}
            >
              {deleteTokenMutation.isPending ? 'Deleting...' : 'Delete Token'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
