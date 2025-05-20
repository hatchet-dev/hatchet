import { Avatar, AvatarFallback } from '@/next/components/ui/avatar';
import useUser from '@/next/hooks/use-user';
import { useTenant } from '@/next/hooks/use-tenant';
import { Skeleton } from '@/next/components/ui/skeleton';
import { Tenant } from '@/lib/api';
import { UserIcon } from 'lucide-react';
interface UserBlockProps {
  variant?: 'default' | 'compact';
}

export function UserBlock({ variant = 'default' }: UserBlockProps) {
  const { data: user } = useUser();

  const name = user?.name || user?.email;
  const initials = name
    ?.split(' ')
    .slice(0, 2) // Take at most 2 initials
    .map((n) => n[0])
    .join('');

  return (
    <>
      <div className="flex size-6 items-center justify-center rounded-full bg-primary text-primary-foreground">
        {initials?.[0]?.toUpperCase() || <UserIcon className="w-4 h-4" />}
      </div>
      {variant === 'default' && (
        <div className="grid flex-1 text-left text-sm leading-tight">
          <span className="truncate font-semibold">{name}</span>
          <span className="truncate text-xs">{user?.email}</span>
        </div>
      )}
    </>
  );
}

interface TenantBlockProps extends UserBlockProps {
  tenant?: Partial<Tenant>;
  tagline?: JSX.Element;
}

export function TenantBlock({
  tenant,
  tagline,
  variant = 'default',
}: TenantBlockProps) {
  const { tenant: currentTenant } = useTenant();

  const activeTenant = tenant || currentTenant;

  const name = activeTenant?.name;
  const initials = name
    ?.split(' ')
    .slice(0, 2) // Take at most 2 initials
    .map((n) => n[0])
    .join('');

  return (
    <>
      <Avatar className="h-8 w-8 rounded-lg">
        <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
      </Avatar>
      {variant === 'default' && (
        <div className="grid flex-1 items-center text-left text-sm leading-tight">
          <span className="truncate font-semibold">
            {name || <Skeleton className="h-4 w-24" />}
          </span>
          {tagline && <span className="truncate text-xs">{tagline}</span>}
        </div>
      )}
    </>
  );
}
