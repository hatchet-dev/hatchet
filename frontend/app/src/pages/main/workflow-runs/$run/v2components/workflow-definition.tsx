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
      <Button size={'sm'} className="px-2 py-2 gap-2" variant="outline">
        <Squares2X2Icon className="w-4 h-4" />
        Workflow
      </Button>
    </Link>
  );
};
