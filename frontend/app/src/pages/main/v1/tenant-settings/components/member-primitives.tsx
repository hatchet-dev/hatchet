import { Badge } from '@/components/v1/ui/badge';

// Display formatting for member/tenant role enum values: "OWNER" -> "Owner".
// The uppercase values stay untouched on the wire.
export function formatMemberRole(role: string): string {
  if (!role) {
    return role;
  }

  return role.charAt(0).toUpperCase() + role.slice(1).toLowerCase();
}

export function RoleBadge({ role }: { role: string }) {
  return <Badge variant="outline">{formatMemberRole(role)}</Badge>;
}

export function MemberEmail({ email }: { email?: string }) {
  return <span className="text-[13px] text-muted-foreground">{email}</span>;
}
