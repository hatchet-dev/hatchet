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
      <Button size={'sm'} className="gap-2 px-2 py-2" variant="outline">
        <Squares2X2Icon className="h-4 w-4" />
        Workflow
      </Button>
    </Link>
  );
};
