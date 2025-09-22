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

export type LabeledRefetchInterval = {
  label: string;
  value: number | false;
};
export type RefetchIntervalOption = 'off' | '5s' | '10s' | '30s' | '1m' | '5m';

export const RefetchInterval = {
  off: {
    label: 'Off',
    value: false,
  },
  '5s': {
    label: '5s',
    value: 5000,
  },
  '10s': {
    label: '10s',
    value: 10000,
  },
  '30s': {
    label: '30s',
    value: 30000,
  },
  '1m': {
    label: '1m',
    value: 60000,
  },
  '5m': {
    label: '5m',
    value: 300000,
  },
} as const satisfies Record<RefetchIntervalOption, LabeledRefetchInterval>;
