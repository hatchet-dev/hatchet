import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { MonthlyComputeCost } from '@/lib/api/generated/cloud/data-contracts';
import { CurrencyDollarIcon } from '@heroicons/react/24/outline';

interface MonthlyUsageCardProps {
  computeCost?: MonthlyComputeCost;
  isLoading: boolean;
}

export function MonthlyUsageCard({
  computeCost,
  isLoading,
}: MonthlyUsageCardProps) {
  if (isLoading) {
    return (
      <Card className="w-full">
        <CardHeader className="pb-2">
          <CardTitle className="text-md flex items-center font-medium">
            <CurrencyDollarIcon className="mr-2 h-5 w-5 text-primary" />
            Monthly Usage
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-sm text-muted-foreground">Loading...</div>
        </CardContent>
      </Card>
    );
  }

  if (!computeCost) {
    return null;
  }

  const { cost, hasCreditsRemaining, creditsRemaining } = computeCost;

  // If cost is negative or has credits remaining, show credits
  const showingCredits = cost < 0 || hasCreditsRemaining;
  const amount = Math.abs(
    showingCredits && creditsRemaining !== undefined ? creditsRemaining : cost,
  );
  const formattedAmount = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(amount);

  return (
    <Card className="w-full">
      <CardHeader className="pb-2">
        <CardTitle className="text-md flex items-center font-medium">
          <CurrencyDollarIcon className="mr-2 h-5 w-5 text-primary" />
          Monthly Usage
        </CardTitle>
        <CardDescription>Current billing period</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center">
          <div
            className={`text-xl font-bold ${showingCredits ? 'text-green-500' : 'text-foreground'}`}
          >
            {showingCredits ? '+ ' : '- '}
            {formattedAmount}
          </div>
          <div className="ml-2 text-sm text-muted-foreground">
            {showingCredits ? 'credits remaining' : 'used this month'}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
