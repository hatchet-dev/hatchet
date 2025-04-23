import { Button } from '@/next/components/ui/button';
import {
  CalendarIcon,
  CpuChipIcon,
  CurrencyDollarIcon,
} from '@heroicons/react/24/outline';
import { ROUTES } from '@/next/lib/routes';

export function CloudOnly() {
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

            {/* Pricing Information */}
            <div className="border rounded-lg p-4 mb-6 bg-muted/10 w-full">
              <div className="flex items-start">
                <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center mr-3">
                  <CurrencyDollarIcon className="h-5 w-5 text-primary" />
                </div>
                <div className="text-left flex-1">
                  <h4 className="font-medium">
                    Transparent Pay as You Go Pricing
                  </h4>
                  {PricingTable()}
                </div>
              </div>
            </div>

            <div className="flex justify-center mb-8 bg-muted/30 p-6 rounded-lg">
              <div className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm text-left">
                <div className="flex items-start">
                  <span className="text-primary mr-2 flex items-center">•</span>
                  <span>Auto-scaling workers based on slots</span>
                </div>
                <div className="flex items-start">
                  <span className="text-primary mr-2 flex items-center">•</span>
                  <span>Zero infrastructure headaches</span>
                </div>
                <div className="flex items-start">
                  <span className="text-primary mr-2 flex items-center">•</span>
                  <span>Multiple regions and machine types</span>
                </div>
                <div className="flex items-start">
                  <span className="text-primary mr-2 flex items-center">•</span>
                  <span>No cold starts</span>
                </div>
              </div>
            </div>

            <div className="flex flex-col gap-4 w-full">
              <Button
                onClick={() => window.open(ROUTES.common.pricing, '_blank')}
                className="min-w-40 py-6 px-8 text-base"
                size="lg"
              >
                Create a Hatchet Cloud Account →
              </Button>

              <div className="relative mt-4">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">
                    Have questions or requirements?
                  </span>
                </div>
              </div>

              <a
                href="https://hatchet.run/office-hours"
                target="_blank"
                rel="noopener noreferrer"
                className="w-full"
              >
                <Button
                  variant="ghost"
                  className="min-w-40 py-6 px-8 text-base w-full flex items-center justify-center gap-2"
                  size="lg"
                >
                  <CalendarIcon className="h-5 w-5" />
                  Book a Call with Our Team
                </Button>
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
export function PricingTable() {
  return (
    <div className="mt-2 grid gap-2 text-sm">
      <div className="flex justify-between">
        <span className="text-muted-foreground">Shared CPU:</span>
        <span className="font-medium">$0.01/CPU/hour</span>
      </div>
      <div className="flex justify-between">
        <span className="text-muted-foreground">Performance CPU:</span>
        <span className="font-medium">$0.02/CPU/hour</span>
      </div>
      <div className="flex justify-between">
        <span className="text-muted-foreground">Memory:</span>
        <span className="font-medium">$0.01/GB/hour</span>
      </div>
    </div>
  );
}
