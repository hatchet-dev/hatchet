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
import { OrganizationMember } from '@/lib/api/generated/cloud/data-contracts';
import {
  UserMinusIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';

interface DeleteMemberModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  member: OrganizationMember | null;
  organizationName: string;
  onSuccess: () => void;
}

export function DeleteMemberModal({
  open,
  onOpenChange,
  member,
  organizationName,
  onSuccess,
}: DeleteMemberModalProps) {
  const { handleApiError } = useApiError({});

  const deleteMemberMutation = useMutation({
    mutationFn: async () => {
      if (!member) {
        return;
      }
      await cloudApi.organizationMemberDelete(member.metadata.id, {
        emails: [member.email],
      });
    },
    onSuccess: () => {
      onSuccess();
      onOpenChange(false);
    },
    onError: handleApiError,
  });

  const handleDelete = () => {
    if (member) {
      deleteMemberMutation.mutate();
    }
  };

  if (!member) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserMinusIcon className="h-5 w-5 text-red-500" />
            Remove Member
          </DialogTitle>
          <DialogDescription>
            Are you sure you want to remove this member from {organizationName}?
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-start gap-3 p-4 bg-yellow-50 border border-yellow-200 rounded-md">
            <ExclamationTriangleIcon className="h-5 w-5 text-yellow-600 mt-0.5 flex-shrink-0" />
            <div className="text-sm">
              <p className="font-medium text-yellow-800 mb-1">
                This action cannot be undone
              </p>
              <p className="text-yellow-700">
                <span className="font-medium">{member.email}</span> will lose
                access to this organization and all its tenants immediately.
              </p>
            </div>
          </div>

          <div className="bg-gray-50 rounded-md p-3">
            <div className="text-sm">
              <div className="font-medium text-gray-700 mb-1">
                Member Details:
              </div>
              <div className="text-gray-600">
                <div>Email: {member.email}</div>
                <div>Role: {member.memberType}</div>
                <div>
                  Joined:{' '}
                  {new Date(member.metadata.createdAt).toLocaleDateString()}
                </div>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={deleteMemberMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMemberMutation.isPending}
            >
              {deleteMemberMutation.isPending ? 'Removing...' : 'Remove Member'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
