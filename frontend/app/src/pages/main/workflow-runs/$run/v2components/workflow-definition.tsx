import { Button } from '@/components/v1/ui/button';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { appRoutes } from '@/router';
import { Squares2X2Icon } from '@heroicons/react/24/outline';
import { Link } from '@tanstack/react-router';

export const WorkflowDefinitionLink = ({
  workflowId,
}: {
  workflowId: string;
}) => {
  const { tenantId } = useCurrentTenantId();

  return (
    <Link
      to={appRoutes.tenantWorkflowRoute.to}
      params={{ tenant: tenantId, workflow: workflowId }}
      target="_blank"
      rel="noreferrer"
    >
      <Button
        size="sm"
        variant="outline"
        leftIcon={<Squares2X2Icon className="size-4" />}
      >
        Workflow
      </Button>
    </Link>
  );
};
