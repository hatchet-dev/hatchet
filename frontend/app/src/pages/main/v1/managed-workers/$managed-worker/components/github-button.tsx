import { Button } from '@/components/v1/ui/button';
import { ManagedWorkerBuildConfig } from '@/lib/api/generated/cloud/data-contracts';
import { GitHubLogoIcon } from '@radix-ui/react-icons';

export default function GithubButton({
  buildConfig,
  commitSha,
  prefix,
}: {
  buildConfig: ManagedWorkerBuildConfig;
  commitSha?: string;
  prefix?: string;
}) {
  return (
    <div className="text-sm w-fit flex flex-row items-center gap-2 text-gray-700 dark:text-gray-300">
      {prefix}
      <a
        href={getHref(buildConfig, commitSha)}
        target="_blank"
        rel="noreferrer"
      >
        <Button variant="link" className="flex items-center gap-1" size={'xs'}>
          <GitHubLogoIcon className="w-4 h-4" />
          {commitSha
            ? commitSha.substring(0, 7)
            : `${buildConfig.githubRepository.repo_owner}/${buildConfig.githubRepository.repo_name}`}
        </Button>
      </a>
    </div>
  );
}

function getHref(buildConfig: ManagedWorkerBuildConfig, commitSha?: string) {
  const root = `https://github.com/${buildConfig.githubRepository.repo_owner}/${buildConfig.githubRepository.repo_name}`;
  return commitSha ? root + '/commit/' + commitSha : root;
}
