import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { useTenant } from '@/lib/atoms';
import { Link } from 'react-router-dom';
import { useState } from 'react';
import { ArrowLeftIcon } from '@radix-ui/react-icons';
import { CpuChipIcon } from '@heroicons/react/24/outline';

export default function DemoTemplate() {
  const { tenant } = useTenant();
  const [deploying, setDeploying] = useState(false);
  const [deployed, setDeployed] = useState(false);

  const handleDeploy = async () => {
    setDeploying(true);
    
    // TODO: Implement the actual deployment logic here
    // This would typically involve an API call to deploy the demo template
    
    // Simulate deployment process
    setTimeout(() => {
      setDeploying(false);
      setDeployed(true);
    }, 2000);
  };

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center mb-4">
          <Link to="/managed-workers" className="mr-4">
            <Button variant="ghost" size="icon">
              <ArrowLeftIcon className="h-4 w-4" />
            </Button>
          </Link>
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Deploy Demo Template
          </h2>
        </div>
        <Separator className="my-4" />
        
        <div className="max-w-3xl mx-auto">
          {!deployed ? (
            <div className="border rounded-lg bg-card p-8 shadow-sm">
              <div className="flex items-start space-x-4">
                <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                  <CpuChipIcon className="h-5 w-5 text-primary" />
                </div>
                <div className="flex-1">
                  <h3 className="text-xl font-medium mb-2">Demo Template</h3>
                  <p className="text-muted-foreground mb-4">
                    This demo template will deploy a sample workflow with managed compute resources that you can use to explore the features of our platform without adding a payment method.
                  </p>
                  
                  <div className="bg-muted/30 p-4 rounded-lg mb-6">
                    <h4 className="font-medium mb-2">Demo Includes:</h4>
                    <ul className="space-y-2">
                      <li className="flex items-start">
                        <span className="text-primary mr-2 mt-0.5">-</span>
                        <span>Sample workflow with 3 steps</span>
                      </li>
                      <li className="flex items-start">
                        <span className="text-primary mr-2 mt-0.5">-</span>
                        <span>1 managed compute worker (limited resources)</span>
                      </li>
                      <li className="flex items-start">
                        <span className="text-primary mr-2 mt-0.5">-</span>
                        <span>Active for 24 hours</span>
                      </li>
                      <li className="flex items-start">
                        <span className="text-primary mr-2 mt-0.5">-</span>
                        <span>No payment method required</span>
                      </li>
                    </ul>
                  </div>
                  
                  <div className="border-t pt-4">
                    <div className="flex justify-between items-center">
                      <div className="text-sm text-muted-foreground">
                        {deploying ? 'Deploying demo template...' : 'Ready to deploy'}
                      </div>
                      <Button 
                        onClick={handleDeploy} 
                        disabled={deploying}
                        className="min-w-32"
                      >
                        {deploying ? 'Deploying...' : 'Deploy Demo'}
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="border rounded-lg bg-card p-8 shadow-sm">
              <div className="flex items-center justify-center flex-col text-center py-6">
                <div className="h-16 w-16 rounded-full bg-green-500/10 flex items-center justify-center mb-4">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-8 w-8 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <h3 className="text-xl font-medium mb-2">Demo Template Deployed!</h3>
                <p className="text-muted-foreground mb-6 max-w-md">
                  Your demo template has been successfully deployed. You can now explore the managed compute features.
                </p>
                <div className="flex gap-4">
                  <Link to="/managed-workers">
                    <Button variant="outline">
                      View Managed Workers
                    </Button>
                  </Link>
                  <Link to="/workflows">
                    <Button>
                      View Demo Workflow
                    </Button>
                  </Link>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
} 