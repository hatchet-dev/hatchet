import RelativeDate from '@/components/v1/molecules/relative-date';
import { TenantInvite } from '@/lib/api';
import {
  MemberEmail,
  RoleBadge,
} from '@/pages/main/v1/tenant-settings/components/member-primitives';

export function PendingInvitesSection({
  invites,
}: {
  invites: TenantInvite[];
}) {
  if (invites.length === 0) {
    return null;
  }

  return (
    <div className="mt-6 space-y-2">
      <h4 className="text-sm font-medium">Pending Invites</h4>
      <div className="rounded-md border border-border/70">
        <div className="hidden grid-cols-[minmax(0,1.6fr)_120px_140px_140px] gap-3 border-b border-border/70 bg-muted/20 px-4 py-2 text-xs font-medium uppercase tracking-wide text-muted-foreground md:grid">
          <span>Email</span>
          <span>Role</span>
          <span>Created</span>
          <span>Expires</span>
        </div>
        <div>
          {invites.map((invite) => (
            <div
              key={invite.metadata.id}
              className="grid gap-3 border-b border-border/50 px-4 py-3 last:border-b-0 md:grid-cols-[minmax(0,1.6fr)_120px_140px_140px] md:items-center"
            >
              <div>
                <p className="text-xs text-muted-foreground md:hidden">Email</p>
                <MemberEmail email={invite.email} />
              </div>
              <div>
                <p className="text-xs text-muted-foreground md:hidden">Role</p>
                <RoleBadge role={invite.role} />
              </div>
              <div>
                <p className="text-xs text-muted-foreground md:hidden">
                  Created
                </p>
                <RelativeDate date={invite.metadata.createdAt} />
              </div>
              <div>
                <p className="text-xs text-muted-foreground md:hidden">
                  Expires
                </p>
                <RelativeDate date={invite.expires} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
