import {
  TenantInvite,
  TenantMemberRole,
} from '@/lib/api/generated/data-contracts';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';
import { MailIcon } from 'lucide-react';
import { Button } from '@/next/components/ui/button';
import { Skeleton } from '@/next/components/ui/skeleton';
import useCan from '@/next/hooks/use-can';
import { members } from '@/next/lib/can/features/members.permissions';

interface InvitesTableProps {
  data: TenantInvite[];
  isLoading: boolean;
  onRevokeClick?: (invite: TenantInvite) => void;
  emptyState?: React.ReactNode;
}

export function InvitesTable({
  data,
  isLoading,
  onRevokeClick,
  emptyState,
}: InvitesTableProps) {
  const { can } = useCan();

  if (isLoading) {
    return <InvitesTableSkeleton />;
  }

  if (data.length === 0 && emptyState) {
    return emptyState;
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Invited On</TableHead>
            <TableHead>Expires</TableHead>
            <TableHead className="w-[100px] text-right"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((invite) => (
            <TableRow key={invite.metadata.id}>
              <TableCell className="font-medium">
                <div className="flex items-center gap-2">
                  <MailIcon className="h-4 w-4 text-muted-foreground" />
                  {invite.email}
                </div>
              </TableCell>
              <TableCell>{formatRole(invite.role)}</TableCell>
              <TableCell>{formatDate(invite.metadata.createdAt)}</TableCell>
              <TableCell>{formatDate(invite.expires)}</TableCell>
              <TableCell className="text-right">
                {onRevokeClick && can(members.invite(invite.role)) && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRevokeClick(invite)}
                    className="h-8 px-2 lg:px-3"
                  >
                    Revoke
                  </Button>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function InvitesTableSkeleton() {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Invited On</TableHead>
            <TableHead>Expires</TableHead>
            <TableHead className="w-[100px] text-right"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: 3 }).map((_, i) => (
            <TableRow key={i}>
              <TableCell>
                <Skeleton className="h-5 w-[200px]" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-[100px]" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-[120px]" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-[80px]" />
              </TableCell>
              <TableCell className="text-right">
                <Skeleton className="h-8 w-[80px] ml-auto" />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function formatDate(date?: string) {
  if (!date) {
    return '-';
  }
  return new Date(date).toLocaleDateString();
}

function formatRole(role: TenantMemberRole) {
  switch (role) {
    case TenantMemberRole.ADMIN:
      return 'Admin';
    case TenantMemberRole.MEMBER:
      return 'Member';
    case TenantMemberRole.OWNER:
      return 'Owner';
    default:
      return role;
  }
}
