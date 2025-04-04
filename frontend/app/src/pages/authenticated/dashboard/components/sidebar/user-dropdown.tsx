import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import useUser from '@/hooks/use-user';

export function UserBlock() {
  const { data: user } = useUser();

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
      <div className="grid flex-1 text-left text-sm leading-tight">
        <span className="truncate font-semibold">{name}</span>
        <span className="truncate text-xs">{user?.email}</span>
      </div>
    </>
  );
}
