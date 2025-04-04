import { CenterStageLayout } from '@/components/layouts/center-stage.layout';
import { Button } from '@/components/ui/button';
import useUser from '@/hooks/use-user';

export default function OnboardingNewPage() {
  const { logout } = useUser();

  return (
    <CenterStageLayout>
      <div>OnboardingNew</div>
      <Button onClick={() => logout.mutate()} loading={logout.isPending}>
        Logout
      </Button>
    </CenterStageLayout>
  );
}
