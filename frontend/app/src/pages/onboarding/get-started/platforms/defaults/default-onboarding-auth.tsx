import { Button } from '@/components/ui/button';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import api, { CreateAPITokenRequest } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';

export const DefaultOnboardingAuth: React.FC<{
  tenantId: string;
  tokenGenerated: () => void;
}> = ({ tenantId, tokenGenerated }) => {
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();

  const { handleApiError } = useApiError({});

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenantId],
    mutationFn: async (data: CreateAPITokenRequest) => {
      const res = await api.apiTokenCreate(tenantId, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedToken(data.token);
    },
    onError: handleApiError,
  });

  if (generatedToken) {
    return (
      <div>
        <p className="mb-4 text-muted-foreground">
          Set the following token as an environment variable in your worker.
        </p>
        <p className="mb-4 text-muted-foreground">
          This is the only time we will show you this auth token. Make sure to
          copy it somewhere safe and do not share it with others. You can manage
          your auth tokens from the settings page.
        </p>
        <div className="rounded-lg mb-6">
          <CodeHighlighter
            language="plaintext"
            className="text-sm"
            wrapLines={false}
            code={'export HATCHET_CLIENT_TOKEN="' + generatedToken + '"'}
            copy
          />
        </div>

        <Button
          onClick={() => tokenGenerated()}
          variant="outline"
          className="mt-2"
        >
          Continue
        </Button>
      </div>
    );
  }

  return (
    <div>
      <p className="mb-4 text-muted-foreground">
        Before you can start your worker, you need to generate an auth token.
      </p>
      <Button
        onClick={() => createTokenMutation.mutate({ name: 'default' })}
        className="mr-2"
        variant="outline"
      >
        Generate Auth Token
      </Button>
    </div>
  );
};
