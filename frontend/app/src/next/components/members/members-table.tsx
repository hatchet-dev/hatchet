import {
  TenantInvite,
  TenantMember,
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
import { User } from 'lucide-react';
import { Button } from '@/next/components/ui/button';
import { Skeleton } from '@/next/components/ui/skeleton';
import useCan from '@/next/hooks/use-can';
import { members } from '@/next/lib/can/features/members.permissions';
import useUser from '@/next/hooks/use-user';
import { Badge } from '@/next/components/ui/badge';
import { useState } from 'react';
import useMembers from '@/next/hooks/use-members';
import { RemoveMemberForm } from '@/next/pages/authenticated/dashboard/settings/team/components/remove-member-form';
import { RevokeInviteForm } from '@/next/pages/authenticated/dashboard/settings/team/components/revoke-invite-form';
import { Separator } from '@radix-ui/react-separator';
import { InvitesTable } from '@/next/pages/authenticated/dashboard/settings/team/components/invites-table';

interface MembersTableProps {
  emptyState?: React.ReactNode;
}

export function MembersTable({ emptyState }: MembersTableProps) {
  const { can } = useCan();
  const { data: user } = useUser();
  const { data, isLoading, invites, isLoadingInvites, refetch } = useMembers();
  const [removeMember, setRemoveMember] = useState<TenantMember | null>(null);
  const [revokeInvite, setRevokeInvite] = useState<TenantInvite | null>(null);

  if (isLoading) {
    return <MembersTableSkeleton />;
  }

  if (data.length === 0 && emptyState) {
    return emptyState;
  }

  return (
    <>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>User</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Joined</TableHead>
              <TableHead className="w-[100px] text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((member) => (
              <TableRow key={member.metadata.id}>
                <TableCell className="font-medium">
                  <div className="flex items-center gap-2">
                    <User className="h-4 w-4 text-muted-foreground" />
                    {member.user.name || '-'}
                    {member.user.email === user?.email && (
                      <Badge variant="outline" className="ml-2">
                        You
                      </Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell>{member.user.email}</TableCell>
                <TableCell>{RoleMap[member.role]}</TableCell>
                <TableCell>{formatDate(member.metadata.createdAt)}</TableCell>
                <TableCell className="text-right">
                  {can(members.remove(member)) && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setRemoveMember(member)}
                      className="h-8 px-2 lg:px-3"
                    >
                      Remove
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {invites && invites.length > 0 ? (
        <>
          <Separator className="my-8" />

          <h3 className="text-xl font-semibold leading-tight text-foreground mb-4">
            Pending Invitations
          </h3>
          <InvitesTable
            data={invites}
            isLoading={isLoadingInvites}
            onRevokeClick={(invite) => {
              setRevokeInvite(invite);
            }}
            emptyState={
              <div className="flex flex-col items-center justify-center gap-4 py-8">
                <p className="text-sm text-muted-foreground">
                  No pending invitations.
                </p>
              </div>
            }
          />
        </>
      ) : null}

      {removeMember ? (
        <RemoveMemberForm
          member={removeMember}
          close={async () => {
            setRemoveMember(null);
            await refetch();
          }}
        />
      ) : null}

      {revokeInvite ? (
        <RevokeInviteForm
          invite={revokeInvite}
          close={() => {
            setRevokeInvite(null);
          }}
        />
      ) : null}
    </>
  );
}

function MembersTableSkeleton() {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>User</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Joined</TableHead>
            <TableHead className="w-[100px] text-right"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: 5 }).map((_, key) => (
            <TableRow key={key}>
              <TableCell>
                <Skeleton className="h-5 w-[150px]" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-[200px]" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-[100px]" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-[120px]" />
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

const RoleMap: Record<TenantMemberRole, string> = {
  [TenantMemberRole.ADMIN]: 'Admin',
  [TenantMemberRole.MEMBER]: 'Member',
  [TenantMemberRole.OWNER]: 'Owner',
};
