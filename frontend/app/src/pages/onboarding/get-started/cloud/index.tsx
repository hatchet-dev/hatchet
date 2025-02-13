import { Loading } from '@/components/ui/loading';
import { useTenant } from '@/lib/atoms';
import { UserContextType, MembershipsContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Step, Steps } from '@/components/ui/steps';

export default function GetStarted() {
  const ctx = useOutletContext<UserContextType & MembershipsContextType>();
  const { user, memberships } = ctx;
  const { tenant: currTenant } = useTenant();

  // TODO: get whether this is a cloud instance.

  if (!user || !memberships || !currTenant) {
    return <Loading />;
  }

  return (
    <div className="flex flex-col items-center w-full h-full overflow-auto">
      <div className="container mx-auto px-4 py-8 lg:px-8 lg:py-12 max-w-4xl">
        <div className="flex flex-col justify-center space-y-4">
          <div className="flex flex-row justify-between mt-10">
            <h1 className="text-3xl font-bold">Quickstart</h1>
            <a href="/">
              <Button variant="outline">Skip Quickstart</Button>
            </a>
          </div>

          <p className="text-gray-600 dark:text-gray-300 text-sm">
            Get started by deploying your first worker.
          </p>

          <Steps className="mt-6">
            <Step title="Choose your runtime" collapsible>
              <div className="grid gap-4">
                <div className="text-sm text-muted-foreground"></div>
              </div>
            </Step>
            <Step title="Deploy your worker" collapsible disabled>
              <div className="grid gap-4">
                <div className="text-sm text-muted-foreground">Testing</div>
              </div>
            </Step>
            <Step title="Trigger a workflow" collapsible disabled>
              <div className="grid gap-4">
                <div className="text-sm text-muted-foreground">Testing</div>
              </div>
            </Step>
          </Steps>
        </div>
      </div>
    </div>
  );
}
