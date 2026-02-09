import { Api } from './generated/Api';
import { Api as CloudApi } from './generated/cloud/Api';
import qs from 'qs';

const api = new Api({
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});

export default api;

export const cloudApi = new CloudApi({
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});
