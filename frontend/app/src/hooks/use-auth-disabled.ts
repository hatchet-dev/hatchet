import useApiMeta from '@/pages/auth/hooks/use-api-meta';

// True when the instance is an authdisabled build. authDisabled only exists on APIMeta
// (not APIControlPlaneMetadata), so narrow before reading it.
export default function useAuthDisabled(): boolean {
  const { meta } = useApiMeta();
  return Boolean(meta && 'authDisabled' in meta && meta.authDisabled);
}
