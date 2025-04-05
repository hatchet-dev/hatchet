import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import useUser from '@/hooks/use-user';
import useTenant from '@/hooks/use-tenant';
import { Skeleton } from '@/components/ui/skeleton';
import { useIsMobile } from '@/hooks/use-mobile';

export function UserBlock() {
  const { data: user } = useUser();
  const isMobile = useIsMobile();

  const name = user?.name || user?.email;
  const initials = name
    ?.split(' ')
    .slice(0, 2) // Take at most 2 initials
    .map((n) => n[0])
    .join('');

  return (
    <>
      <Avatar className="h-8 w-8 rounded-lg">
        {/* <AvatarImage src={user?.avatar} alt={user?.name} /> */}
        <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
      </Avatar>
      {!isMobile && (
        <div className="grid flex-1 text-left text-sm leading-tight">
          <span className="truncate font-semibold">{name}</span>
          <span className="truncate text-xs">{user?.email}</span>
        </div>
      )}
    </>
  );
}

export function TenantBlock() {
  const { tenant } = useTenant();

  const name = tenant?.name;
  const initials = name
    ?.split(' ')
    .slice(0, 2) // Take at most 2 initials
    .map((n) => n[0])
    .join('');

  return (
    <>
      <Avatar className="h-8 w-8 rounded-lg">
        {/* <AvatarImage src={user?.avatar} alt={user?.name} /> */}
        <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
      </Avatar>
      <div className="grid flex-1 text-left text-sm leading-tight">
        <span className="truncate font-semibold">
          {name || <Skeleton className="h-4 w-24" />}
        </span>
      </div>
    </>
  );
}
