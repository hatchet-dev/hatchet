import { Button } from '@/components/ui/button';
import { Link } from 'react-router-dom';
import { CpuChipIcon } from '@heroicons/react/24/outline';

interface BillingRequiredProps {
  tenant: any;
  billing: any;
  manageClicked: () => Promise<void>;
  portalLoading: boolean;
}

export function BillingRequired({
  tenant,
  billing,
  manageClicked,
  portalLoading,
}: BillingRequiredProps) {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="border rounded-lg bg-card p-12 shadow-sm">
          <div className="flex flex-col items-center text-center max-w-md mx-auto">
            <div className="h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center mb-6">
              <CpuChipIcon className="h-8 w-8 text-primary" />
            </div>

            <h3 className="text-2xl font-semibold mb-2">
              Ready to supercharge your workflows?
            </h3>

            <p className="text-muted-foreground mb-6">
              Unlock Managed Compute by adding a payment method. No commitment
              required - you only pay for what you use!
            </p>

            <div className="flex justify-center mb-8 bg-muted/30 p-6 rounded-lg">
              <div className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm text-left">
                <div className="flex items-start">
                  <span className="text-primary mr-2 mt-0.5">-</span>
                  <span>Auto-scaling workers based on slots</span>
                </div>
                <div className="flex items-start">
                  <span className="text-primary mr-2 mt-0.5">-</span>
                  <span>Zero infrastructure headaches</span>
                </div>
                <div className="flex items-start">
                  <span className="text-primary mr-2 mt-0.5">-</span>
                  <span>Multiple regions and machine types</span>
                </div>
                <div className="flex items-start">
                  <span className="text-primary mr-2 mt-0.5">-</span>
                  <span>Pay only for what you use</span>
                </div>
              </div>
            </div>

            <div className="flex flex-col gap-4 w-full">
              <Button
                onClick={manageClicked}
                disabled={portalLoading}
                className="min-w-40 py-6 px-8 text-base"
                size="lg"
              >
                {portalLoading ? 'Loading...' : 'Add Payment Method →'}
              </Button>

              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">Or</span>
                </div>
              </div>

              <Link to="/managed-workers/demo-template" className="w-full">
                <Button
                  variant="outline"
                  className="min-w-40 py-6 px-8 text-base w-full"
                  size="lg"
                >
                  Try Demo Template
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
} 