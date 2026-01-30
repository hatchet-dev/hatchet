import { CreateTokenModal } from './create-token-modal';
import { DeleteTokenModal } from './delete-token-modal';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Loading } from '@/components/v1/ui/loading';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { cloudApi } from '@/lib/api/api';
import {
  ManagementToken,
  Organization,
} from '@/lib/api/generated/cloud/data-contracts';
import { PlusIcon, KeyIcon } from '@heroicons/react/24/outline';
import { EllipsisVerticalIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';
import { useState } from 'react';

interface TokensTabProps {
  organization: Organization;
  orgId: string;
}

export function TokensTab({ organization, orgId }: TokensTabProps) {
  const [showCreateTokenModal, setShowCreateTokenModal] = useState(false);
  const [tokenToDelete, setTokenToDelete] = useState<ManagementToken | null>(
    null,
  );

  const managementTokensQuery = useQuery({
    queryKey: ['management-tokens:list', orgId],
    queryFn: async () => {
      const result = await cloudApi.managementTokenList(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  const formatExpirationDate = (expiresDate: string) => {
    try {
      const expires = new Date(expiresDate);
      const now = new Date();
      if (expires < now) {
        return 'expired';
      }
      return `in ${formatDistanceToNow(expires)}`;
    } catch {
      return new Date(expiresDate).toLocaleDateString();
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">Management Tokens</h3>
          <p className="text-sm text-muted-foreground">
            API tokens for managing this organization
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowCreateTokenModal(true)}
          leftIcon={<PlusIcon className="size-4" />}
        >
          Create Token
        </Button>
      </div>

      {managementTokensQuery.isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loading />
        </div>
      ) : managementTokensQuery.data?.rows &&
        managementTokensQuery.data.rows.length > 0 ? (
        <div className="space-y-4">
          <div className="hidden md:block">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Expiry</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {managementTokensQuery.data.rows.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-sm">{token.id}</span>
                        <CopyToClipboard text={token.id} />
                      </div>
                    </TableCell>
                    <TableCell className="font-medium">{token.name}</TableCell>
                    <TableCell>
                      {formatExpirationDate(token.expiresAt)}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                          >
                            <EllipsisVerticalIcon className="size-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => setTokenToDelete(token)}
                          >
                            <TrashIcon className="mr-2 size-4" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="space-y-4 md:hidden">
            {managementTokensQuery.data.rows.map((token) => (
              <div key={token.id} className="space-y-3 rounded-lg border p-4">
                <div className="flex items-center justify-between">
                  <h4 className="font-medium">{token.name}</h4>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">
                      {formatExpirationDate(token.expiresAt)}
                    </Badge>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                        >
                          <EllipsisVerticalIcon className="size-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() => setTokenToDelete(token)}
                        >
                          <TrashIcon className="mr-2 size-4" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="font-medium text-muted-foreground">
                      Token ID:
                    </span>
                    <div className="mt-1 flex items-center gap-2">
                      <span className="font-mono text-sm">{token.id}</span>
                      <CopyToClipboard text={token.id} />
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <div className="py-8 text-center">
          <KeyIcon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
          <h3 className="mb-2 text-lg font-medium">No Management Tokens</h3>
          <p className="mb-4 text-muted-foreground">
            Create API tokens to manage this organization programmatically.
          </p>
          <Button
            onClick={() => setShowCreateTokenModal(true)}
            leftIcon={<PlusIcon className="size-4" />}
          >
            Create Token
          </Button>
        </div>
      )}

      <CreateTokenModal
        open={showCreateTokenModal}
        onOpenChange={setShowCreateTokenModal}
        organizationId={orgId}
        organizationName={organization.name}
        onSuccess={() => managementTokensQuery.refetch()}
      />

      {tokenToDelete && (
        <DeleteTokenModal
          open={!!tokenToDelete}
          onOpenChange={(open) => !open && setTokenToDelete(null)}
          token={tokenToDelete}
          organizationName={organization.name}
          onSuccess={() => managementTokensQuery.refetch()}
        />
      )}
    </div>
  );
}
