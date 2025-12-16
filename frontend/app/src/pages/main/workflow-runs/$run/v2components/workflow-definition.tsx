import { Button } from '@/components/v1/ui/button';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { Squares2X2Icon } from '@heroicons/react/24/outline';
import { Link } from 'react-router-dom';

export const WorkflowDefinitionLink = ({
  workflowId,
}: {
  workflowId: string;
}) => {
  const { tenantId } = useCurrentTenantId();

  return (
    <Link
      to={`/tenants/${tenantId}/workflows/${workflowId}`}
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
