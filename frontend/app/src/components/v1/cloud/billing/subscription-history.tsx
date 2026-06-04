import { Badge, BadgeProps } from '@/components/v1/ui/badge';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import {
  OrganizationBillingStateSubscription,
  SubscriptionPlan,
  SubscriptionStatus,
} from '@/lib/api/generated/control-plane/data-contracts';

interface SubscriptionHistoryProps {
  history?: OrganizationBillingStateSubscription[];
  plans?: SubscriptionPlan[];
}

const statusLabels: Record<SubscriptionStatus, string> = {
  [SubscriptionStatus.Current]: 'Current',
  [SubscriptionStatus.Upcoming]: 'Upcoming',
  [SubscriptionStatus.Past]: 'Past',
};

const statusVariants: Record<SubscriptionStatus, BadgeProps['variant']> = {
  [SubscriptionStatus.Current]: 'successful',
  [SubscriptionStatus.Upcoming]: 'inProgress',
  [SubscriptionStatus.Past]: 'queued',
};

// Upcoming is shown first (the next state), then the active plan, then past history.
const statusOrder: Record<SubscriptionStatus, number> = {
  [SubscriptionStatus.Upcoming]: 0,
  [SubscriptionStatus.Current]: 1,
  [SubscriptionStatus.Past]: 2,
};

function sortHistory(
  history: OrganizationBillingStateSubscription[],
): OrganizationBillingStateSubscription[] {
  return [...history].sort((a, b) => {
    const orderA = a.status ? statusOrder[a.status] : statusOrder.past;
    const orderB = b.status ? statusOrder[b.status] : statusOrder.past;
    if (orderA !== orderB) {
      return orderA - orderB;
    }
    const startA = a.startedAt ? new Date(a.startedAt).getTime() : 0;
    const startB = b.startedAt ? new Date(b.startedAt).getTime() : 0;
    return startB - startA;
  });
}

function formatDate(value?: string) {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '—';
  }
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function SubscriptionHistory({
  history,
  plans,
}: SubscriptionHistoryProps) {
  const planName = (subscription: OrganizationBillingStateSubscription) =>
    plans?.find((plan) => plan.planCode === subscription.planCode)?.name ??
    subscription.planCode;

  const sortedHistory = history ? sortHistory(history) : [];

  return (
    <Card
      variant="light"
      className="bg-transparent ring-1 ring-border/50 border-none"
    >
      <CardHeader className="p-4 border-b border-border/50">
        <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
          Plan History
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        {sortedHistory.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="px-4">Plan</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Started</TableHead>
                <TableHead>Ended</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedHistory.map((subscription, index) => (
                <TableRow
                  key={`${subscription.planCode}-${subscription.startedAt}-${index}`}
                >
                  <TableCell className="px-4 font-medium text-foreground">
                    {planName(subscription)}
                  </TableCell>
                  <TableCell>
                    {subscription.status ? (
                      <Badge variant={statusVariants[subscription.status]}>
                        {statusLabels[subscription.status]}
                      </Badge>
                    ) : (
                      '—'
                    )}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(subscription.startedAt)}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(subscription.endsAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <p className="p-4 text-sm text-muted-foreground">
            No plan history yet. Plan changes will appear here.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
