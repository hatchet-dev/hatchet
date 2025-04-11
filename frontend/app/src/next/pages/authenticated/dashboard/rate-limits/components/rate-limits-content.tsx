import { Card, CardContent } from '@/next/components/ui/card';
import { Button } from '@/next/components/ui/button';
import { Badge } from '@/next/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { Separator } from '@/next/components/ui/separator';
import { MoreHorizontal, RefreshCw, Lock } from 'lucide-react';
import useRateLimits from '@/next/hooks/use-ratelimits';
import docs from '@/next/docs-meta-data';
import { DocsButton } from '@/next/components/ui/docs-button';

export function RateLimitsContent() {
  const { data: rateLimits, isLoading } = useRateLimits();

  // Format the rate limit period display
  const formatWindow = (rateLimit: any) => {
    return `${rateLimit.limitValue} per ${rateLimit.window.toLowerCase()}`;
  };

  // Get the status badge
  const getStatusBadge = (tokens: number, limitValue: number) => {
    const percentage = (tokens / limitValue) * 100;
    if (percentage > 75) {
      return (
        <Badge
          variant="outline"
          className="bg-green-50 text-green-700 border-green-200"
        >
          Healthy
        </Badge>
      );
    } else if (percentage > 25) {
      return (
        <Badge
          variant="outline"
          className="bg-yellow-50 text-yellow-700 border-yellow-200"
        >
          Limited
        </Badge>
      );
    } else {
      return (
        <Badge
          variant="outline"
          className="bg-red-50 text-red-700 border-red-200"
        >
          Exhausted
        </Badge>
      );
    }
  };

  // Format date
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-semibold leading-tight text-foreground">
            Rate Limits
          </h2>
          <div className="flex flex-row items-center gap-2"></div>
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Rate limits help you control API usage and prevent abuse.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6 mt-6">
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold">
                {rateLimits?.length || 0}
              </div>
              <p className="text-sm text-muted-foreground">Total Rate Limits</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-green-600">
                {rateLimits?.filter(
                  (limit: any) => limit.tokens > limit.limitValue / 2,
                )?.length || 0}
              </div>
              <p className="text-sm text-muted-foreground">Healthy Limits</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-red-600">
                {rateLimits?.filter(
                  (limit: any) => limit.tokens <= limit.limitValue / 2,
                )?.length || 0}
              </div>
              <p className="text-sm text-muted-foreground">Limited Resources</p>
            </CardContent>
          </Card>
        </div>
        <Separator className="my-6" />
        {isLoading ? (
          <div className="flex justify-center items-center h-64">
            <RefreshCw className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Key</TableHead>
                  <TableHead>Limit</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last Refill</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {!rateLimits || rateLimits.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-24 text-center">
                      <div className="flex flex-col items-center justify-center gap-4 py-8">
                        <p className="text-md">No rate limits found.</p>
                        <p className="text-sm text-muted-foreground">
                          Create a new rate limit to get started.
                        </p>
                        <DocsButton doc={docs.home['rate-limits']} />
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  rateLimits.map((limit: any) => (
                    <TableRow key={limit.key}>
                      <TableCell className="font-medium">
                        <div className="flex items-center">
                          <Lock className="h-4 w-4 mr-2 text-muted-foreground" />
                          {limit.key}
                        </div>
                      </TableCell>
                      <TableCell>{formatWindow(limit)}</TableCell>
                      <TableCell>
                        {getStatusBadge(limit.tokens, limit.limitValue)}
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {formatDate(limit.lastRefill)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon">
                                <MoreHorizontal className="h-4 w-4" />
                                <span className="sr-only">Open menu</span>
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end"></DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </div>
  );
}
