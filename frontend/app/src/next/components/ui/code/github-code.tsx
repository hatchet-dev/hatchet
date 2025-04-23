import React, { useCallback } from 'react';
import { Code, CodeProps } from './code';
import { useQuery } from '@tanstack/react-query';
import { useToast } from '@/next/hooks/utils/use-toast';

interface GithubCodeProps extends Omit<CodeProps, 'value'> {
  repo: string;
  path: string;
  branch?: string;
}

export function GithubCode({
  repo,
  path,
  branch = 'main',
  ...props
}: GithubCodeProps) {
  const { toast } = useToast();

  const fetchGithubCode = useCallback(
    async (repo: string, path: string, branch: string): Promise<string> => {
      const response = await fetch(
        `https://raw.githubusercontent.com/${repo}/${branch}/${path}`,
      );

      if (!response.ok) {
        toast({
          title: 'Error fetching code',
          description: `Failed to fetch code: ${response.statusText}`,
          variant: 'destructive',
        });
      }

      return response.text();
    },
    [toast],
  );

  const {
    data: code,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['github-code', repo, path, branch],
    queryFn: () => fetchGithubCode(repo, path, branch),
    retry: false,
  });

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return (
      <div className="text-red-500">
        Error: {error instanceof Error ? error.message : 'Failed to fetch code'}
      </div>
    );
  }

  return <Code value={code || ''} {...props} title={props.title || path} />;
}
