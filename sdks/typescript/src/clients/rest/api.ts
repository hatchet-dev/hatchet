import qs from 'qs';
import { AxiosRequestConfig } from 'axios';
import { Api } from './generated/Api';
import { HATCHET_VERSION } from '@hatchet/version';

const api = (serverUrl: string, token: string, axiosOpts?: AxiosRequestConfig) => {
  return new Api({
    baseURL: serverUrl,
    headers: {
      Authorization: `Bearer ${token}`,
      'User-Agent': `hatchet-typescript-sdk/${HATCHET_VERSION}`,
    },
    paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
    ...axiosOpts,
  });
};

export default api;
