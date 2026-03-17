import https from 'https';
import qs from 'qs';
import { AxiosRequestConfig } from 'axios';
import { Api } from './generated/Api';

function getDefaultAxiosOpts(serverUrl: string): AxiosRequestConfig {
  const opts: AxiosRequestConfig = {};
  if (serverUrl.startsWith('https://') && process.env.NODE_TLS_REJECT_UNAUTHORIZED === '0') {
    opts.httpsAgent = new https.Agent({ rejectUnauthorized: false });
  }
  return opts;
}

const api = (serverUrl: string, token: string, axiosOpts?: AxiosRequestConfig) => new Api({
    baseURL: serverUrl,
    headers: {
      Authorization: `Bearer ${token}`,
    },
    paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
    ...getDefaultAxiosOpts(serverUrl),
    ...axiosOpts,
  });

export default api;
