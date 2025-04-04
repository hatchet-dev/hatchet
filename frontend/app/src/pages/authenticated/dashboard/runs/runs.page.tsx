import { Button } from '@/components/ui/button';
import useUser from '@/hooks/use-user';

export default function RunsPage() {
  const { data: user, logout } = useUser();

  return (
    <div>
      {user?.email}
      <Button onClick={() => logout.mutate()} loading={logout.isPending}>
        Logout
      </Button>
    </div>
  );
}
