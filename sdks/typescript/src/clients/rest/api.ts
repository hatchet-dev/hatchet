import qs from 'qs';
import { AxiosRequestConfig } from 'axios';
import { Api } from './generated/Api';

const api = (serverUrl: string, token: string, axiosOpts?: AxiosRequestConfig) => {
  return new Api({
    baseURL: serverUrl,
    headers: {
      Authorization: `Bearer ${token}`,
    },
    paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
    ...axiosOpts,
  });
};

export default api;
