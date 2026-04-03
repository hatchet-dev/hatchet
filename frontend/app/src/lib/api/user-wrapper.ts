import useControlPlane from '@/hooks/use-control-plane';
import api, { controlPlaneApi } from '@/lib/api/api';
import { useMemo } from 'react';

type UserUpdateLoginRequest = Parameters<typeof api.userUpdateLogin>[0];
type UserCreateRequest = Parameters<typeof api.userCreate>[0];
type UserUpdatePasswordRequest = Parameters<typeof api.userUpdatePassword>[0];

export function useUserApi() {
  const { isControlPlaneEnabled } = useControlPlane();

  return useMemo(
    () => ({
      // ── Queries ────────────────────────────────────────────────────────────

      userGetCurrentQuery: () => ({
        queryKey: ['user:get'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserGetCurrent()
              : api.userGetCurrent())
          ).data,
      }),

      userUpdateGoogleOauthStartQuery: () => ({
        queryKey: ['user:google-oauth:start'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserUpdateGoogleOauthStart()
              : api.userUpdateGoogleOauthStart())
          ).data,
      }),

      userUpdateGoogleOauthCallbackQuery: () => ({
        queryKey: ['user:google-oauth:callback'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserUpdateGoogleOauthCallback()
              : api.userUpdateGoogleOauthCallback())
          ).data,
      }),

      userUpdateGithubOauthStartQuery: () => ({
        queryKey: ['user:github-oauth:start'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserUpdateGithubOauthStart()
              : api.userUpdateGithubOauthStart())
          ).data,
      }),

      userUpdateGithubOauthCallbackQuery: () => ({
        queryKey: ['user:github-oauth:callback'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserUpdateGithubOauthCallback()
              : api.userUpdateGithubOauthCallback())
          ).data,
      }),

      // ── Mutations ──────────────────────────────────────────────────────────

      userUpdateLoginMutation: () => ({
        mutationKey: ['user:update:login'] as const,
        mutationFn: async (data: UserUpdateLoginRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserUpdateLogin(data)
              : api.userUpdateLogin(data))
          ).data,
      }),

      userCreateMutation: () => ({
        mutationKey: ['user:create'] as const,
        mutationFn: async (data: UserCreateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserCreate(data)
              : api.userCreate(data))
          ).data,
      }),

      userUpdateLogoutMutation: () => ({
        mutationKey: ['user:update:logout'] as const,
        mutationFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserUpdateLogout()
              : api.userUpdateLogout())
          ).data,
      }),

      userUpdatePasswordMutation: () => ({
        mutationKey: ['user:update:password'] as const,
        mutationFn: async (data: UserUpdatePasswordRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.cloudUserUpdatePassword(data)
              : api.userUpdatePassword(data))
          ).data,
      }),
    }),
    [isControlPlaneEnabled],
  );
}
