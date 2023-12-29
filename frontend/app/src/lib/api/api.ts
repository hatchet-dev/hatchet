import { Api } from './generated/Api';
import qs from 'qs';

const api = new Api({
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});

export default api;
