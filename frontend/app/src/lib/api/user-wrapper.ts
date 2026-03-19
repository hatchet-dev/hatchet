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
      userGetCurrent: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserGetCurrent()
          : api.userGetCurrent(),

      userUpdateLogin: (data: UserUpdateLoginRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserUpdateLogin(data)
          : api.userUpdateLogin(data),

      userCreate: (data: UserCreateRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserCreate(data)
          : api.userCreate(data),

      userUpdateLogout: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserUpdateLogout()
          : api.userUpdateLogout(),

      userUpdatePassword: (data: UserUpdatePasswordRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserUpdatePassword(data)
          : api.userUpdatePassword(data),

      userUpdateGoogleOauthStart: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserUpdateGoogleOauthStart()
          : api.userUpdateGoogleOauthStart(),

      userUpdateGoogleOauthCallback: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserUpdateGoogleOauthCallback()
          : api.userUpdateGoogleOauthCallback(),

      userUpdateGithubOauthStart: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserUpdateGithubOauthStart()
          : api.userUpdateGithubOauthStart(),

      userUpdateGithubOauthCallback: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.cloudUserUpdateGithubOauthCallback()
          : api.userUpdateGithubOauthCallback(),
    }),
    [isControlPlaneEnabled],
  );
}
