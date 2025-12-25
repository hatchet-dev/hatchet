import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';

export function CreateOrganizationDialog({
  open,
  onOpenChange,
  orgName,
  setOrgName,
  createOrganizationLoading,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgName: string;
  setOrgName: (name: string) => void;
  createOrganizationLoading: boolean;
  onCreate: (name: string) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create New Organization</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label htmlFor="org-name" className="text-sm font-medium">
              Organization Name
            </label>
            <Input
              id="org-name"
              value={orgName}
              onChange={(e) => setOrgName(e.target.value)}
              placeholder="Enter organization name"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && orgName.trim()) {
                  onCreate(orgName.trim());
                }
              }}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              onOpenChange(false);
              setOrgName('');
            }}
            disabled={createOrganizationLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={() => {
              if (!orgName.trim()) {
                return;
              }
              onCreate(orgName.trim());
            }}
            disabled={!orgName.trim() || createOrganizationLoading}
          >
            {createOrganizationLoading ? 'Creating...' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
