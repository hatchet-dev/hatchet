import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import { useAppContext } from '@/providers/app-context';
import { appRoutes } from '@/router';
import {
  ArrowRightIcon,
  InformationCircleIcon,
} from '@heroicons/react/24/outline';
import { Link } from '@tanstack/react-router';

export default function ResourceLimits() {
  const { getCurrentOrganization } = useAppContext();
  const currentOrg = getCurrentOrganization();

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <Alert>
          <InformationCircleIcon className="size-4" />
          <AlertTitle>Billing & Limits Moved</AlertTitle>
          <AlertDescription className="flex items-center justify-between">
            <span>
              Billing and resource limits are now managed at the organization
              level.
            </span>
            {currentOrg && (
              <Link
                to={appRoutes.organizationsRoute.to}
                params={{ organization: currentOrg.metadata.id }}
                search={{ tab: 'billing' }}
              >
                <Button variant="outline" size="sm">
                  Go to Organization Billing
                  <ArrowRightIcon className="ml-2 size-4" />
                </Button>
              </Link>
            )}
          </AlertDescription>
        </Alert>
      </div>
    </div>
  );
}
