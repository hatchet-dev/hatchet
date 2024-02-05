import https from 'https';
import Hatchet, { Api } from '../src';

const hatchet = Hatchet.init(
  {
    log_level: 'OFF',
  },
  {},
  {
    // This is needed for the local certificate in the example, but should not be used in production
    httpsAgent: new https.Agent({
      rejectUnauthorized: false,
    }),
  }
);

const api = hatchet.api as Api;

api.workflowList('707d0855-80ab-4e1f-a156-f1c4546cbf52').then((res) => {
  res.data.rows?.forEach((row) => {
    console.log(row);
  });
});
