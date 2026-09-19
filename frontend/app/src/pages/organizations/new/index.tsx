import { NewOrganizationSaverForm } from '@/components/forms/new-organization-saver-form';
import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { Button } from '@/components/v1/ui/button';
import { appRoutes } from '@/router';
import { XMarkIcon } from '@heroicons/react/24/outline';
import { useNavigate } from '@tanstack/react-router';

export default function OrganizationsNew() {
  const navigate = useNavigate();

  return (
    <SetupScreen
      topRight={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => window.history.back()}
          className="h-8 w-8 p-0"
        >
          <XMarkIcon className="size-4" />
        </Button>
      }
    >
      <SetupCard title="Create a new organization">
        <NewOrganizationSaverForm
          afterSave={({ tenant }) =>
            navigate({
              to: appRoutes.tenantOnboardingRoute.to,
              params: { tenant: tenant.id },
            })
          }
        />
      </SetupCard>
    </SetupScreen>
  );
}
