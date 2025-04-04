import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
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
          <CardTitle className="text-md font-medium flex items-center">
            <CurrencyDollarIcon className="h-5 w-5 mr-2 text-primary" />
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
  const formattedAmount = Math.abs(
    showingCredits && creditsRemaining !== undefined ? creditsRemaining : cost,
  ).toFixed(2);

  return (
    <Card className="w-full">
      <CardHeader className="pb-2">
        <CardTitle className="text-md font-medium flex items-center">
          <CurrencyDollarIcon className="h-5 w-5 mr-2 text-primary" />
          Monthly Usage
        </CardTitle>
        <CardDescription>Current billing period</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center">
          <div
            className={`text-xl font-bold ${showingCredits ? 'text-green-500' : 'text-foreground'}`}
          >
            {showingCredits && '+ '}${formattedAmount}
          </div>
          <div className="ml-2 text-sm text-muted-foreground">
            {showingCredits ? 'credits remaining' : 'used this month'}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
